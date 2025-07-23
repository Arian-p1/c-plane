package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nextranet/gateway/c-plane/config"
	appContext "github.com/nextranet/gateway/c-plane/internal/context"
	"github.com/nextranet/gateway/c-plane/internal/logger"
	"github.com/nextranet/gateway/c-plane/internal/models"
)

// GenieACSService provides integration with GenieACS
type GenieACSService struct {
	config     *config.GenieACS
	appContext *appContext.Context
	httpClient *http.Client
}

// NewGenieACSService creates a new GenieACS service instance
func NewGenieACSService(cfg *config.GenieACS, ctx *appContext.Context) *GenieACSService {
	return &GenieACSService{
		config:     cfg,
		appContext: ctx,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// Initialize initializes the GenieACS service
func (s *GenieACSService) Initialize() error {
	logger.GenieACSLog.Info("Initializing GenieACS service...")

	// Test connections
	if err := s.testConnections(); err != nil {
		return fmt.Errorf("failed to connect to GenieACS: %w", err)
	}

	logger.GenieACSLog.Info("GenieACS service initialized successfully")
	return nil
}

// StartMonitoring starts monitoring GenieACS services
func (s *GenieACSService) StartMonitoring(ctx context.Context) {
	logger.GenieACSLog.Info("Starting GenieACS monitoring...")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Initial check
	s.checkStatus()

	for {
		select {
		case <-ctx.Done():
			logger.GenieACSLog.Info("Stopping GenieACS monitoring...")
			return
		case <-ticker.C:
			s.checkStatus()
		}
	}
}

// checkStatus checks the status of GenieACS services
func (s *GenieACSService) checkStatus() {
	status := appContext.GenieACSStatus{
		CWMPConnected: s.checkCWMPConnection(),
		NBIConnected:  s.checkNBIConnection(),
		FSConnected:   s.checkFSConnection(),
		LastCheck:     time.Now(),
	}

	s.appContext.UpdateGenieACSStatus(status)
}

// testConnections tests all GenieACS connections
func (s *GenieACSService) testConnections() error {
	var lastErr error

	if !s.checkCWMPConnection() {
		lastErr = fmt.Errorf("CWMP service not available")
		logger.GenieACSLog.Error(lastErr)
	}

	if !s.checkNBIConnection() {
		lastErr = fmt.Errorf("NBI service not available")
		logger.GenieACSLog.Error(lastErr)
	}

	if !s.checkFSConnection() {
		lastErr = fmt.Errorf("FS service not available")
		logger.GenieACSLog.Error(lastErr)
	}

	return lastErr
}

// checkCWMPConnection checks if CWMP service is available
func (s *GenieACSService) checkCWMPConnection() bool {
	req, err := http.NewRequest("GET", s.config.CWMPURL, nil)
	if err != nil {
		return false
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode != 0
}

// checkNBIConnection checks if NBI service is available
func (s *GenieACSService) checkNBIConnection() bool {
	req, err := http.NewRequest("GET", s.config.NBIURL+"/devices?limit=1", nil)
	if err != nil {
		return false
	}

	s.addAuth(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized
}

// checkFSConnection checks if FS service is available
func (s *GenieACSService) checkFSConnection() bool {
	req, err := http.NewRequest("GET", s.config.FSURL, nil)
	if err != nil {
		return false
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode != 0
}

// Device Operations

// GetDevices retrieves devices from GenieACS
func (s *GenieACSService) GetDevices(filter *models.DeviceFilter) ([]*models.Device, error) {
	query := s.buildDeviceQuery(filter)

	req, err := http.NewRequest("GET", s.config.NBIURL+"/devices"+query, nil)
	if err != nil {
		return nil, err
	}

	s.addAuth(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch devices: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var genieDevices []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&genieDevices); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	devices := make([]*models.Device, 0, len(genieDevices))
	for _, gd := range genieDevices {
		device := s.convertGenieDevice(gd)
		devices = append(devices, device)
	}

	return devices, nil
}

// GetDevice retrieves a single device from GenieACS
func (s *GenieACSService) GetDevice(deviceID string) (*models.Device, error) {
	// Build query parameter: {"_id":"deviceID"}
	query := fmt.Sprintf(`{"_id":"%s"}`, deviceID)
	encodedQuery := url.QueryEscape(query)

	req, err := http.NewRequest("GET", s.config.NBIURL+"/devices?query="+encodedQuery, nil)
	if err != nil {
		return nil, err
	}

	s.addAuth(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch device: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var genieDevices []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&genieDevices); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(genieDevices) == 0 {
		return nil, models.ErrDeviceNotFound
	}

	return s.convertGenieDevice(genieDevices[0]), nil
}

// RefreshDevice refreshes device data from GenieACS
func (s *GenieACSService) RefreshDevice(deviceID string) error {
	task := map[string]interface{}{
		"name":       "refreshObject",
		"objectName": "",
	}

	return s.CreateTask(deviceID, task)
}

// Task Operations

// CreateTask creates a new task for a device
func (s *GenieACSService) CreateTask(deviceID string, task map[string]interface{}) error {
	body, err := json.Marshal(task)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", s.config.NBIURL+"/devices/"+url.QueryEscape(deviceID)+"/tasks", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	s.addAuth(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create task: %s", string(body))
	}

	return nil
}

// GetTasks retrieves tasks for a device
func (s *GenieACSService) GetTasks(deviceID string) ([]*models.Task, error) {
	req, err := http.NewRequest("GET", s.config.NBIURL+"/tasks?device="+url.QueryEscape(deviceID), nil)
	if err != nil {
		return nil, err
	}

	s.addAuth(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tasks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var genieTasks []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&genieTasks); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	tasks := make([]*models.Task, 0, len(genieTasks))
	for _, gt := range genieTasks {
		task := s.convertGenieTask(gt)
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// DeleteTask deletes a task
func (s *GenieACSService) DeleteTask(taskID string) error {
	req, err := http.NewRequest("DELETE", s.config.NBIURL+"/tasks/"+url.QueryEscape(taskID), nil)
	if err != nil {
		return err
	}

	s.addAuth(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete task: status %d", resp.StatusCode)
	}

	return nil
}

// Fault Operations

// GetFaults retrieves faults from GenieACS
func (s *GenieACSService) GetFaults(deviceID string) ([]*models.Fault, error) {
	query := ""
	if deviceID != "" {
		query = "?device=" + url.QueryEscape(deviceID)
	}

	req, err := http.NewRequest("GET", s.config.NBIURL+"/faults"+query, nil)
	if err != nil {
		return nil, err
	}

	s.addAuth(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch faults: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var genieFaults []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&genieFaults); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	faults := make([]*models.Fault, 0, len(genieFaults))
	for _, gf := range genieFaults {
		fault := s.convertGenieFault(gf)
		faults = append(faults, fault)
	}

	return faults, nil
}

// DeleteFault deletes a fault
func (s *GenieACSService) DeleteFault(faultID string) error {
	req, err := http.NewRequest("DELETE", s.config.NBIURL+"/faults/"+url.QueryEscape(faultID), nil)
	if err != nil {
		return err
	}

	s.addAuth(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete fault: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete fault: status %d", resp.StatusCode)
	}

	return nil
}

// Parameter Operations

// GetDeviceParameters retrieves device parameters
func (s *GenieACSService) GetDeviceParameters(deviceID string, parameterNames []string) (map[string]models.Parameter, error) {
	projection := make(map[string]int)
	for _, name := range parameterNames {
		projection[name] = 1
	}

	query := url.Values{}
	query.Add("projection", s.encodeProjection(projection))

	req, err := http.NewRequest("GET", s.config.NBIURL+"/devices/"+url.QueryEscape(deviceID)+"?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}

	s.addAuth(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch parameters: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var genieDevice map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&genieDevice); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return s.extractParameters(genieDevice), nil
}

// SetDeviceParameters sets device parameters
func (s *GenieACSService) SetDeviceParameters(deviceID string, parameters map[string]interface{}) error {
	tasks := []map[string]interface{}{}

	for path, value := range parameters {
		task := map[string]interface{}{
			"name": "setParameterValues",
			"parameterValues": []interface{}{
				[]interface{}{path, value},
			},
		}
		tasks = append(tasks, task)
	}

	for _, task := range tasks {
		if err := s.CreateTask(deviceID, task); err != nil {
			return err
		}
	}

	return nil
}

// GetDeviceConfig retrieves the current configuration for a device
func (s *GenieACSService) GetDeviceConfig(deviceID string) (string, error) {
	// Get device information from GenieACS
	device, err := s.GetDevice(deviceID)
	if err != nil {
		return "", fmt.Errorf("failed to get device: %v", err)
	}

	// Create a simple XML configuration with device information
	config := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"
	config += "<device-configuration>\n"
	config += "  <device-info>\n"
	config += fmt.Sprintf("    <serial-number>%s</serial-number>\n", device.DeviceID.SerialNumber)
	config += fmt.Sprintf("    <manufacturer>%s</manufacturer>\n", device.DeviceID.Manufacturer)
	config += fmt.Sprintf("    <model>%s</model>\n", device.DeviceID.ModelName)
	config += fmt.Sprintf("    <product-class>%s</product-class>\n", device.DeviceID.ProductClass)
	config += fmt.Sprintf("    <oui>%s</oui>\n", device.DeviceID.OUI)
	config += fmt.Sprintf("    <hardware-version>%s</hardware-version>\n", device.DeviceID.HardwareVersion)
	config += fmt.Sprintf("    <software-version>%s</software-version>\n", device.DeviceID.SoftwareVersion)
	config += fmt.Sprintf("    <ip-address>%s</ip-address>\n", device.DeviceID.IPAddress)
	config += fmt.Sprintf("    <external-ip>%s</external-ip>\n", device.DeviceID.ExternalIPAddress)
	config += "  </device-info>\n"
	config += "  <status>\n"
	config += fmt.Sprintf("    <online>%t</online>\n", device.Status.Online)
	config += fmt.Sprintf("    <last-inform>%s</last-inform>\n", device.Status.LastSeen.Format("2006-01-02T15:04:05Z"))
	config += fmt.Sprintf("    <last-boot>%s</last-boot>\n", device.LastBoot.Format("2006-01-02T15:04:05Z"))
	config += "  </status>\n"
	config += "</device-configuration>"

	return config, nil
}

// SetDeviceParameter sets a single parameter on a device (wrapper for SetDeviceParameters)
func (s *GenieACSService) SetDeviceParameter(deviceID, parameter string, value interface{}) error {
	params := map[string]interface{}{
		parameter: value,
	}
	return s.SetDeviceParameters(deviceID, params)
}

// AddDeviceTag adds a tag to a device
func (s *GenieACSService) AddDeviceTag(deviceID, tag string) error {
	// Create a task to add tag
	task := map[string]interface{}{
		"name": "addTag",
		"tag":  tag,
	}

	return s.CreateTask(deviceID, task)
}

// RemoveDeviceTag removes a tag from a device
func (s *GenieACSService) RemoveDeviceTag(deviceID, tag string) error {
	// Create a task to remove tag
	task := map[string]interface{}{
		"name": "removeTag",
		"tag":  tag,
	}

	return s.CreateTask(deviceID, task)
}

// Helper functions

// addAuth adds authentication to the request if configured
func (s *GenieACSService) addAuth(req *http.Request) {
	if s.config.Username != "" && s.config.Password != "" {
		req.SetBasicAuth(s.config.Username, s.config.Password)
	}
}

// buildDeviceQuery builds query string for device filtering
func (s *GenieACSService) buildDeviceQuery(filter *models.DeviceFilter) string {
	if filter == nil {
		return ""
	}

	query := url.Values{}

	// Add filters
	filters := []string{}

	if filter.Manufacturer != "" {
		filters = append(filters, fmt.Sprintf(`_deviceId._Manufacturer:"%s"`, filter.Manufacturer))
	}

	if filter.ModelName != "" {
		filters = append(filters, fmt.Sprintf(`_deviceId._ModelName:"%s"`, filter.ModelName))
	}

	if filter.ProductClass != "" {
		filters = append(filters, fmt.Sprintf(`_deviceId._ProductClass:"%s"`, filter.ProductClass))
	}

	if len(filters) > 0 {
		query.Add("query", "{"+strings.Join(filters, ",")+"}")
	}

	// Add pagination
	if filter.Pagination != nil {
		limit := filter.Pagination.PageSize
		if limit == 0 {
			limit = 20
		}
		skip := (filter.Pagination.Page - 1) * limit

		query.Add("limit", fmt.Sprintf("%d", limit))
		query.Add("skip", fmt.Sprintf("%d", skip))

		// TODO: GenieACS sort parameter causes 400 error - disable for now
		// if filter.Pagination.SortBy != "" {
		// 	sort := filter.Pagination.SortBy
		// 	if filter.Pagination.SortDir == "desc" {
		// 		sort = "-" + sort
		// 	}
		// 	query.Add("sort", sort)
		// }
	}

	if len(query) > 0 {
		return "?" + query.Encode()
	}

	return ""
}

// convertGenieDevice converts GenieACS device format to internal model
func (s *GenieACSService) convertGenieDevice(genieDevice map[string]interface{}) *models.Device {
	device := &models.Device{
		DeviceInfo: genieDevice,
		Parameters: make(map[string]models.Parameter),
		Tags:       make(map[string]bool),
		Status: models.DeviceStatus{
			Online: true, // Will be updated based on last inform
		},
	}

	// Extract device ID
	if deviceID, ok := genieDevice["_id"].(string); ok {
		device.ID = deviceID
	}

	// Extract device identification
	if deviceIdMap, ok := genieDevice["_deviceId"].(map[string]interface{}); ok {
		device.DeviceID = models.DeviceID{
			Manufacturer:     s.getString(deviceIdMap, "_Manufacturer"),
			OUI:              s.getString(deviceIdMap, "_OUI"),
			ProductClass:     s.getString(deviceIdMap, "_ProductClass"),
			SerialNumber:     s.getString(deviceIdMap, "_SerialNumber"),
			HardwareVersion:  s.getString(deviceIdMap, "_HardwareVersion"),
			SoftwareVersion:  s.getString(deviceIdMap, "_SoftwareVersion"),
			ModelName:        s.getString(deviceIdMap, "_ModelName"),
			ProvisioningCode: s.getString(deviceIdMap, "_ProvisioningCode"),
		}
	}

	// Extract timestamps
	if lastInform, ok := genieDevice["_lastInform"].(string); ok {
		if t, err := time.Parse(time.RFC3339, lastInform); err == nil {
			device.LastInform = t
			device.Status.LastSeen = t

			// Update online status based on last inform
			if time.Since(t) > 5*time.Minute {
				device.Status.Online = false
				device.Status.ConnectionStatus = "offline"
			} else {
				device.Status.ConnectionStatus = "online"
			}
		}
	}

	if lastBoot, ok := genieDevice["_lastBoot"].(string); ok {
		if t, err := time.Parse(time.RFC3339, lastBoot); err == nil {
			device.LastBoot = t
		}
	}

	if lastBootstrap, ok := genieDevice["_lastBootstrap"].(string); ok {
		if t, err := time.Parse(time.RFC3339, lastBootstrap); err == nil {
			device.LastBootstrap = t
		}
	}

	if registered, ok := genieDevice["_registered"].(string); ok {
		if t, err := time.Parse(time.RFC3339, registered); err == nil {
			device.LastRegistered = t
		}
	}

	// Extract tags
	if tags, ok := genieDevice["_tags"].([]interface{}); ok {
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				device.Tags[tagStr] = true
			}
		}
	}

	// Extract parameters
	device.Parameters = s.extractParameters(genieDevice)

	// Extract IP addresses
	if ipAddr := s.getParameterValue(genieDevice, "InternetGatewayDevice.LANDevice.1.LANHostConfigManagement.IPInterface.1.IPInterfaceIPAddress"); ipAddr != "" {
		device.DeviceID.IPAddress = ipAddr
	}

	if extIPAddr := s.getParameterValue(genieDevice, "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANIPConnection.1.ExternalIPAddress"); extIPAddr != "" {
		device.DeviceID.ExternalIPAddress = extIPAddr
	}

	return device
}

// convertGenieTask converts GenieACS task format to internal model
func (s *GenieACSService) convertGenieTask(genieTask map[string]interface{}) *models.Task {
	task := &models.Task{
		Args: make(map[string]interface{}),
	}

	if id, ok := genieTask["_id"].(string); ok {
		task.ID = id
	}

	if device, ok := genieTask["device"].(string); ok {
		task.DeviceID = device
	}

	if name, ok := genieTask["name"].(string); ok {
		task.Name = name
	}

	if status, ok := genieTask["status"].(string); ok {
		task.Status = status
	}

	if timestamp, ok := genieTask["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
			task.Timestamp = t
		}
	}

	if provisions, ok := genieTask["provisions"].([]interface{}); ok {
		task.Provisions = provisions
	}

	if retries, ok := genieTask["retries"].(float64); ok {
		task.Retries = int(retries)
	}

	return task
}

// convertGenieFault converts GenieACS fault format to internal model
func (s *GenieACSService) convertGenieFault(genieFault map[string]interface{}) *models.Fault {
	fault := &models.Fault{
		Tags: []string{},
	}

	if id, ok := genieFault["_id"].(string); ok {
		fault.ID = id
	}

	if device, ok := genieFault["device"].(string); ok {
		fault.DeviceID = device
	}

	if channel, ok := genieFault["channel"].(string); ok {
		fault.Channel = channel
	}

	if code, ok := genieFault["code"].(string); ok {
		fault.Code = code
	}

	if message, ok := genieFault["message"].(string); ok {
		fault.Message = message
	}

	if detail, ok := genieFault["detail"].(string); ok {
		fault.Detail = detail
	}

	if timestamp, ok := genieFault["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
			fault.Timestamp = t
		}
	}

	if retries, ok := genieFault["retries"].(float64); ok {
		fault.Retries = int(retries)
	}

	// Set default severity based on code
	fault.Severity = s.determineFaultSeverity(fault.Code)

	// Set status as active by default
	fault.Status = models.FaultStatusActive

	return fault
}

// extractParameters extracts parameters from GenieACS device data
func (s *GenieACSService) extractParameters(genieDevice map[string]interface{}) map[string]models.Parameter {
	params := make(map[string]models.Parameter)

	for key, value := range genieDevice {
		if strings.HasPrefix(key, "_") || key == "Downloads" {
			continue
		}

		if paramMap, ok := value.(map[string]interface{}); ok {
			param := models.Parameter{
				Path: key,
			}

			if val, ok := paramMap["_value"]; ok {
				param.Value = val
			}

			if valType, ok := paramMap["_type"].(string); ok {
				param.Type = valType
			}

			if writable, ok := paramMap["_writable"].(bool); ok {
				param.Writable = writable
			}

			if timestamp, ok := paramMap["_timestamp"].(string); ok {
				if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
					param.LastUpdate = t
				}
			}

			params[key] = param
		}
	}

	return params
}

// getString safely extracts a string value from a map
func (s *GenieACSService) getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// getParameterValue extracts a parameter value as string
func (s *GenieACSService) getParameterValue(genieDevice map[string]interface{}, path string) string {
	if param, ok := genieDevice[path].(map[string]interface{}); ok {
		if val, ok := param["_value"]; ok {
			return fmt.Sprintf("%v", val)
		}
	}
	return ""
}

// encodeProjection encodes projection for GenieACS query
func (s *GenieACSService) encodeProjection(projection map[string]int) string {
	parts := []string{}
	for key, val := range projection {
		parts = append(parts, fmt.Sprintf(`"%s":%d`, key, val))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// determineFaultSeverity determines fault severity based on code
func (s *GenieACSService) determineFaultSeverity(code string) string {
	// Map common fault codes to severities
	switch {
	case strings.Contains(code, "9001"): // Request denied
		return models.SeverityWarning
	case strings.Contains(code, "9002"): // Internal error
		return models.SeverityMajor
	case strings.Contains(code, "9003"): // Invalid arguments
		return models.SeverityMinor
	case strings.Contains(code, "9004"): // Resources exceeded
		return models.SeverityMajor
	case strings.Contains(code, "9005"): // Invalid parameter name
		return models.SeverityMinor
	case strings.Contains(code, "9006"): // Invalid parameter type
		return models.SeverityMinor
	case strings.Contains(code, "9007"): // Invalid parameter value
		return models.SeverityMinor
	case strings.Contains(code, "9008"): // Attempt to set non-writable parameter
		return models.SeverityWarning
	case strings.Contains(code, "9009"): // Notification request rejected
		return models.SeverityWarning
	case strings.Contains(code, "9010"): // Download failure
		return models.SeverityMajor
	case strings.Contains(code, "9011"): // Upload failure
		return models.SeverityMajor
	case strings.Contains(code, "9012"): // File transfer authentication failure
		return models.SeverityCritical
	default:
		return models.SeverityInfo
	}
}
