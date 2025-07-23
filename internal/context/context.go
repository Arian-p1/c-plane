package context

import (
	"sync"
	"time"

	"github.com/nextranet/gateway/c-plane/internal/models"
)

var (
	context *Context
	once    sync.Once
)

// Context holds the application context and state
type Context struct {
	mutex sync.RWMutex

	// Device management
	devices      map[string]*models.Device
	devicesMutex sync.RWMutex

	// Fault management
	faults      map[string]*models.Fault
	faultsMutex sync.RWMutex

	// Cache for device statistics
	statsCache      *models.DeviceStats
	statsCacheMutex sync.RWMutex
	statsCacheTime  time.Time

	// GenieACS connection status
	genieACSStatus GenieACSStatus
	statusMutex    sync.RWMutex

	// Configuration
	config interface{}
}

// GenieACSStatus represents the connection status to GenieACS services
type GenieACSStatus struct {
	CWMPConnected bool      `json:"cwmpConnected"`
	NBIConnected  bool      `json:"nbiConnected"`
	FSConnected   bool      `json:"fsConnected"`
	LastCheck     time.Time `json:"lastCheck"`
	LastError     string    `json:"lastError,omitempty"`
}

// GetContext returns the singleton context instance
func GetContext() *Context {
	once.Do(func() {
		context = &Context{
			devices: make(map[string]*models.Device),
			faults:  make(map[string]*models.Fault),
			statsCache: &models.DeviceStats{
				DevicesByVendor: make(map[string]int),
				DevicesByModel:  make(map[string]int),
			},
		}
	})
	return context
}

// Device Management Functions

// AddDevice adds or updates a device in the context
func (c *Context) AddDevice(device *models.Device) {
	c.devicesMutex.Lock()
	defer c.devicesMutex.Unlock()
	c.devices[device.ID] = device
	c.invalidateStatsCache()
}

// GetDevice retrieves a device by ID
func (c *Context) GetDevice(deviceID string) (*models.Device, bool) {
	c.devicesMutex.RLock()
	defer c.devicesMutex.RUnlock()
	device, exists := c.devices[deviceID]
	return device, exists
}

// GetDeviceBySerial retrieves a device by serial number
func (c *Context) GetDeviceBySerial(serial string) (*models.Device, bool) {
	c.devicesMutex.RLock()
	defer c.devicesMutex.RUnlock()

	for _, device := range c.devices {
		if device.DeviceID.SerialNumber == serial {
			return device, true
		}
	}
	return nil, false
}

// GetAllDevices returns all devices
func (c *Context) GetAllDevices() []*models.Device {
	c.devicesMutex.RLock()
	defer c.devicesMutex.RUnlock()

	devices := make([]*models.Device, 0, len(c.devices))
	for _, device := range c.devices {
		devices = append(devices, device)
	}
	return devices
}

// GetFilteredDevices returns devices matching the filter criteria
func (c *Context) GetFilteredDevices(filter *models.DeviceFilter) []*models.Device {
	c.devicesMutex.RLock()
	defer c.devicesMutex.RUnlock()

	var filtered []*models.Device

	for _, device := range c.devices {
		if matchesFilter(device, filter) {
			filtered = append(filtered, device)
		}
	}

	return filtered
}

// RemoveDevice removes a device from the context
func (c *Context) RemoveDevice(deviceID string) {
	c.devicesMutex.Lock()
	defer c.devicesMutex.Unlock()
	delete(c.devices, deviceID)
	c.invalidateStatsCache()
}

// UpdateDeviceStatus updates the status of a device
func (c *Context) UpdateDeviceStatus(deviceID string, online bool) {
	c.devicesMutex.Lock()
	defer c.devicesMutex.Unlock()

	if device, exists := c.devices[deviceID]; exists {
		device.Status.Online = online
		device.Status.LastSeen = time.Now()
		if online {
			device.Status.ConnectionStatus = "connected"
		} else {
			device.Status.ConnectionStatus = "disconnected"
		}
		c.invalidateStatsCache()
	}
}

// Fault Management Functions

// AddFault adds a new fault
func (c *Context) AddFault(fault *models.Fault) {
	c.faultsMutex.Lock()
	defer c.faultsMutex.Unlock()
	c.faults[fault.ID] = fault
	c.invalidateStatsCache()
}

// GetFault retrieves a fault by ID
func (c *Context) GetFault(faultID string) (*models.Fault, bool) {
	c.faultsMutex.RLock()
	defer c.faultsMutex.RUnlock()
	fault, exists := c.faults[faultID]
	return fault, exists
}

// GetDeviceFaults returns all faults for a specific device
func (c *Context) GetDeviceFaults(deviceID string) []*models.Fault {
	c.faultsMutex.RLock()
	defer c.faultsMutex.RUnlock()

	var deviceFaults []*models.Fault
	for _, fault := range c.faults {
		if fault.DeviceID == deviceID {
			deviceFaults = append(deviceFaults, fault)
		}
	}
	return deviceFaults
}

// GetActiveFaults returns all active faults
func (c *Context) GetActiveFaults() []*models.Fault {
	c.faultsMutex.RLock()
	defer c.faultsMutex.RUnlock()

	var activeFaults []*models.Fault
	for _, fault := range c.faults {
		if fault.Status == models.FaultStatusActive || fault.Status == models.FaultStatusAcknowledged {
			activeFaults = append(activeFaults, fault)
		}
	}
	return activeFaults
}

// AcknowledgeFault acknowledges a fault
func (c *Context) AcknowledgeFault(faultID, acknowledgedBy string) error {
	c.faultsMutex.Lock()
	defer c.faultsMutex.Unlock()

	fault, exists := c.faults[faultID]
	if !exists {
		return models.ErrFaultNotFound
	}

	now := time.Now()
	fault.Status = models.FaultStatusAcknowledged
	fault.AcknowledgedBy = acknowledgedBy
	fault.AcknowledgedAt = &now

	c.invalidateStatsCache()
	return nil
}

// ResolveFault resolves a fault
func (c *Context) ResolveFault(faultID, resolvedBy string) error {
	c.faultsMutex.Lock()
	defer c.faultsMutex.Unlock()

	fault, exists := c.faults[faultID]
	if !exists {
		return models.ErrFaultNotFound
	}

	now := time.Now()
	fault.Status = models.FaultStatusResolved
	fault.ResolvedBy = resolvedBy
	fault.ResolvedAt = &now

	c.invalidateStatsCache()
	return nil
}

// Statistics Functions

// GetDeviceStats returns cached device statistics
func (c *Context) GetDeviceStats() *models.DeviceStats {
	c.statsCacheMutex.RLock()

	// Check if cache is valid (less than 1 minute old)
	if time.Since(c.statsCacheTime) < time.Minute {
		defer c.statsCacheMutex.RUnlock()
		return c.statsCache
	}

	c.statsCacheMutex.RUnlock()

	// Update cache
	c.updateStatsCache()

	c.statsCacheMutex.RLock()
	defer c.statsCacheMutex.RUnlock()
	return c.statsCache
}

// updateStatsCache updates the statistics cache
func (c *Context) updateStatsCache() {
	c.statsCacheMutex.Lock()
	defer c.statsCacheMutex.Unlock()

	c.devicesMutex.RLock()
	c.faultsMutex.RLock()
	defer c.devicesMutex.RUnlock()
	defer c.faultsMutex.RUnlock()

	stats := &models.DeviceStats{
		TotalDevices:    len(c.devices),
		OnlineDevices:   0,
		OfflineDevices:  0,
		DevicesByVendor: make(map[string]int),
		DevicesByModel:  make(map[string]int),
		ActiveFaults:    0,
		CriticalFaults:  0,
	}

	// Count devices
	for _, device := range c.devices {
		if device.Status.Online {
			stats.OnlineDevices++
		} else {
			stats.OfflineDevices++
		}

		// Count by vendor
		vendor := device.DeviceID.Manufacturer
		if vendor == "" {
			vendor = "Unknown"
		}
		stats.DevicesByVendor[vendor]++

		// Count by model
		model := device.DeviceID.ModelName
		if model == "" {
			model = "Unknown"
		}
		stats.DevicesByModel[model]++
	}

	// Count faults
	for _, fault := range c.faults {
		if fault.Status == models.FaultStatusActive || fault.Status == models.FaultStatusAcknowledged {
			stats.ActiveFaults++
			if fault.Severity == models.SeverityCritical {
				stats.CriticalFaults++
			}
		}
	}

	c.statsCache = stats
	c.statsCacheTime = time.Now()
}

// invalidateStatsCache marks the stats cache as invalid
func (c *Context) invalidateStatsCache() {
	c.statsCacheMutex.Lock()
	defer c.statsCacheMutex.Unlock()
	c.statsCacheTime = time.Time{} // Set to zero time
}

// GenieACS Status Functions

// GetGenieACSStatus returns the current GenieACS connection status
func (c *Context) GetGenieACSStatus() GenieACSStatus {
	c.statusMutex.RLock()
	defer c.statusMutex.RUnlock()
	return c.genieACSStatus
}

// UpdateGenieACSStatus updates the GenieACS connection status
func (c *Context) UpdateGenieACSStatus(status GenieACSStatus) {
	c.statusMutex.Lock()
	defer c.statusMutex.Unlock()
	c.genieACSStatus = status
	c.genieACSStatus.LastCheck = time.Now()
}

// Helper Functions

// matchesFilter checks if a device matches the given filter criteria
func matchesFilter(device *models.Device, filter *models.DeviceFilter) bool {
	if filter == nil {
		return true
	}

	// Check IP range
	if filter.IPRange != nil {
		if !isIPInRange(device.DeviceID.IPAddress, filter.IPRange.StartIP, filter.IPRange.EndIP) {
			return false
		}
	}

	// Check manufacturer
	if filter.Manufacturer != "" && device.DeviceID.Manufacturer != filter.Manufacturer {
		return false
	}

	// Check model
	if filter.ModelName != "" && device.DeviceID.ModelName != filter.ModelName {
		return false
	}

	// Check product class
	if filter.ProductClass != "" && device.DeviceID.ProductClass != filter.ProductClass {
		return false
	}

	// Check online status
	if filter.Online != nil && device.Status.Online != *filter.Online {
		return false
	}

	// Check tags
	if len(filter.Tags) > 0 {
		for _, tag := range filter.Tags {
			if !device.Tags[tag] {
				return false
			}
		}
	}

	return true
}

// isIPInRange checks if an IP address is within the specified range
func isIPInRange(ip, startIP, endIP string) bool {
	// TODO: Implement IP range checking
	// This is a placeholder implementation
	return true
}

// SetConfig sets the application configuration
func (c *Context) SetConfig(config interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.config = config
}

// GetConfig returns the application configuration
func (c *Context) GetConfig() interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.config
}
