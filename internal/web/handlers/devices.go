package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nextranet/gateway/c-plane/internal/context"
	"github.com/nextranet/gateway/c-plane/internal/logger"
	"github.com/nextranet/gateway/c-plane/internal/models"
	"github.com/nextranet/gateway/c-plane/internal/web/templates"
	"github.com/nextranet/gateway/c-plane/pkg/factory"
	"github.com/nextranet/gateway/c-plane/pkg/service"
)

// Devices renders the devices list page
func Devices(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get query parameters for filtering
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

		if page < 1 {
			page = 1
		}
		if pageSize < 1 || pageSize > 100 {
			pageSize = 20
		}

		// Build filter from query parameters
		filter := &models.DeviceFilter{
			Pagination: &models.PaginationOptions{
				Page:     page,
				PageSize: pageSize,
				SortBy:   c.DefaultQuery("sortBy", "lastInform"),
				SortDir:  c.DefaultQuery("sortDir", "desc"),
			},
		}

		// Apply filters
		if search := c.Query("search"); search != "" {
			filter.Search = search
		}

		if vendor := c.Query("vendor"); vendor != "" {
			filter.Manufacturer = vendor
		}

		if model := c.Query("model"); model != "" {
			filter.ModelName = model
		}

		if status := c.Query("status"); status != "" {
			online := status == "online"
			filter.Online = &online
		}

		if tags := c.Query("tags"); tags != "" {
			filter.Tags = strings.Split(tags, ",")
		}

		// Get devices from GenieACS
		cfg := factory.GetConfig()
		genieService := service.NewGenieACSService(cfg.GenieACS, appContext)

		devices, err := genieService.GetDevices(filter)
		if err != nil {
			logger.WebLog.Errorf("Failed to get devices: %v", err)
			// Fall back to cached data
			devices = appContext.GetFilteredDevices(filter)
		}

		// Convert to display format
		displayDevices := make([]*templates.DeviceDisplay, 0, len(devices))
		for _, device := range devices {
			displayDevice := &templates.DeviceDisplay{
				Device:       device,
				StatusClass:  getDeviceStatusClass(device.Status.Online),
				StatusText:   getDeviceStatusText(device.Status.Online),
				LastSeenText: formatLastSeen(device.Status.LastSeen),
				TagList:      getTagList(device.Tags),
			}

			displayDevices = append(displayDevices, displayDevice)
		}

		// Get available filters
		vendors := getUniqueVendors(devices)
		models := getUniqueModels(devices)

		// Calculate pagination
		totalPages := (len(devices) + pageSize - 1) / pageSize

		// Get theme
		theme := c.GetString("theme")
		if theme == "" {
			theme = "dark"
		}

		// Prepare data for template
		data := templates.DevicesPageData{
			BasePageData: templates.BasePageData{
				Title:       "Devices",
				Theme:       theme,
				CurrentPath: "/devices",
			},
			Devices:       displayDevices,
			TotalCount:    len(devices),
			FilteredCount: len(displayDevices),
			CurrentPage:   page,
			PageSize:      pageSize,
			TotalPages:    totalPages,
			Filters: templates.DeviceFilters{
				Search: filter.Search,

				Vendor: filter.Manufacturer,
				Model:  filter.ModelName,
				Status: c.Query("status"),
				Tags:   filter.Tags,
			},

			Vendors: vendors,
			Models:  models,
		}

		// Render the devices page
		component := templates.DevicesPage(data)
		c.Header("Content-Type", "text/html; charset=utf-8")

		if err := component.Render(c.Request.Context(), c.Writer); err != nil {
			logger.WebLog.Errorf("Failed to render devices page: %v", err)
			c.String(http.StatusInternalServerError, "Failed to render page")
			return
		}
	}
}

// DeviceDetail renders the device detail page
func DeviceDetail(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Param("deviceId")

		// With:
		deviceID, err := url.QueryUnescape(c.Param("deviceId"))
		if err != nil {
			logger.WebLog.Errorf("Failed to decode device ID: %v", err)
			c.String(http.StatusBadRequest, "Invalid device ID")
			return
		}

		// Get device from GenieACS
		cfg := factory.GetConfig()
		genieService := service.NewGenieACSService(cfg.GenieACS, appContext)
		device, err := genieService.GetDevice(deviceID)
		if err != nil {
			logger.WebLog.Errorf("Failed to get device: %v", err)
			c.String(http.StatusNotFound, "Device not found")
			return
		}

		// Get device parameters
		paramNames := []string{
			"InternetGatewayDevice.DeviceInfo.Manufacturer",
			"InternetGatewayDevice.DeviceInfo.ModelName",
			"InternetGatewayDevice.DeviceInfo.SoftwareVersion",
			"InternetGatewayDevice.DeviceInfo.HardwareVersion",
			"InternetGatewayDevice.DeviceInfo.UpTime",
			"InternetGatewayDevice.LANDevice.1.LANHostConfigManagement.IPInterface.1.IPInterfaceIPAddress",
			"InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANIPConnection.1.ExternalIPAddress",
			"InternetGatewayDevice.ManagementServer.URL",
			"InternetGatewayDevice.ManagementServer.ConnectionRequestURL",
		}

		parameters, _ := genieService.GetDeviceParameters(deviceID, paramNames)

		// Get tasks for device
		tasks, _ := genieService.GetTasks(deviceID)

		// Get faults for device
		faults, _ := genieService.GetFaults(deviceID)

		// Get theme
		theme := c.GetString("theme")
		if theme == "" {
			theme = "dark"
		}

		// Prepare data for template
		data := templates.DeviceDetailData{
			BasePageData: templates.BasePageData{
				Title:       "Device: " + device.DeviceID.SerialNumber,
				Theme:       theme,
				CurrentPath: "/devices",
			},
			Device:     device,
			Parameters: parameters,
			Tasks:      tasks,
			Faults:     faults,

			IsOnline:  device.Status.Online,
			CanManage: true, // Based on user permissions
		}

		// Render the device detail page
		component := templates.DeviceDetailPage(data)
		c.Header("Content-Type", "text/html; charset=utf-8")

		if err := component.Render(c.Request.Context(), c.Writer); err != nil {
			logger.WebLog.Errorf("Failed to render device detail page: %v", err)
			c.String(http.StatusInternalServerError, "Failed to render page")
			return
		}
	}
}

// DeviceStatusUpdate returns device status updates for AJAX
func DeviceStatusUpdate(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		devices := appContext.GetAllDevices()

		statusUpdate := make([]gin.H, 0, len(devices))
		for _, device := range devices {
			statusUpdate = append(statusUpdate, gin.H{
				"id":       device.ID,
				"online":   device.Status.Online,
				"lastSeen": device.Status.LastSeen,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"devices":   statusUpdate,
			"timestamp": appContext.GetGenieACSStatus().LastCheck,
		})
	}
}

// RefreshDevice handles device refresh requests
func RefreshDevice(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Param("deviceId")
		if deviceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Device ID is required",
			})
			return
		}

		cfg := factory.GetConfig()
		genieService := service.NewGenieACSService(cfg.GenieACS, appContext)

		err := genieService.RefreshDevice(deviceID)
		if err != nil {
			logger.WebLog.Errorf("Failed to refresh device: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to refresh device",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Device refresh initiated",
		})
	}
}

// RebootDevice handles device reboot requests
func RebootDevice(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Param("deviceId")
		if deviceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Device ID is required",
			})
			return
		}

		cfg := factory.GetConfig()
		genieService := service.NewGenieACSService(cfg.GenieACS, appContext)

		task := map[string]interface{}{
			"name": "reboot",
		}

		err := genieService.CreateTask(deviceID, task)
		if err != nil {
			logger.WebLog.Errorf("Failed to reboot device: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to reboot device",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Device reboot initiated",
		})
	}
}

// DownloadConfig handles device configuration download requests
func DownloadConfig(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Param("deviceId")
		if deviceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Device ID is required",
			})
			return
		}

		cfg := factory.GetConfig()
		genieService := service.NewGenieACSService(cfg.GenieACS, appContext)

		// Get device configuration
		config, err := genieService.GetDeviceConfig(deviceID)
		if err != nil {
			logger.WebLog.Errorf("Failed to get device config: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get device configuration",
			})
			return
		}

		// Get device info for filename
		device, err := genieService.GetDevice(deviceID)
		if err != nil {
			logger.WebLog.Errorf("Failed to get device info: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get device information",
			})
			return
		}

		// Set download headers
		filename := fmt.Sprintf("config_%s_%d.xml", device.DeviceID.SerialNumber, time.Now().Unix())
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		c.Header("Content-Type", "application/xml")

		// Return configuration
		c.String(http.StatusOK, config)
		logger.WebLog.Infof("Configuration downloaded for device: %s", deviceID)
	}
}

// FactoryReset handles device factory reset requests
func FactoryReset(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Param("deviceId")
		if deviceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Device ID is required",
			})
			return
		}

		cfg := factory.GetConfig()
		genieService := service.NewGenieACSService(cfg.GenieACS, appContext)

		task := map[string]interface{}{
			"name": "factoryReset",
		}

		err := genieService.CreateTask(deviceID, task)
		if err != nil {
			logger.WebLog.Errorf("Failed to factory reset device: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to factory reset device",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Factory reset initiated",
		})
	}
}

// UpdateParameter handles device parameter update requests
func UpdateParameter(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Param("deviceId")
		if deviceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Device ID is required",
			})
			return
		}

		var request struct {
			Parameter string      `json:"parameter"`
			Value     interface{} `json:"value"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request format",
			})
			return
		}

		cfg := factory.GetConfig()
		genieService := service.NewGenieACSService(cfg.GenieACS, appContext)

		err := genieService.SetDeviceParameter(deviceID, request.Parameter, request.Value)
		if err != nil {
			logger.WebLog.Errorf("Failed to update parameter: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update parameter",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Parameter updated successfully",
		})
	}
}

// AddDeviceTag handles adding tags to devices
func AddDeviceTag(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Param("deviceId")
		if deviceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Device ID is required",
			})
			return
		}

		var request struct {
			Tag string `json:"tag"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request format",
			})
			return
		}

		cfg := factory.GetConfig()
		genieService := service.NewGenieACSService(cfg.GenieACS, appContext)

		err := genieService.AddDeviceTag(deviceID, request.Tag)
		if err != nil {
			logger.WebLog.Errorf("Failed to add device tag: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to add tag",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Tag added successfully",
		})
	}
}

// RemoveDeviceTag handles removing tags from devices
func RemoveDeviceTag(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Param("deviceId")
		tag := c.Param("tag")

		if deviceID == "" || tag == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Device ID and tag are required",
			})
			return
		}

		cfg := factory.GetConfig()
		genieService := service.NewGenieACSService(cfg.GenieACS, appContext)

		err := genieService.RemoveDeviceTag(deviceID, tag)
		if err != nil {
			logger.WebLog.Errorf("Failed to remove device tag: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to remove tag",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Tag removed successfully",
		})
	}
}

// Helper functions

func getDeviceStatusClass(online bool) string {
	if online {
		return "text-green-500"
	}
	return "text-red-500"
}

func getDeviceStatusText(online bool) string {
	if online {
		return "online"
	}
	return "offline"
}

func getDeviceMapColor(online bool) string {
	if online {
		return "#10b981" // green
	}
	return "#ef4444" // red
}

func formatLastSeen(lastSeen time.Time) string {
	if lastSeen.IsZero() {
		return "Never"
	}

	duration := time.Since(lastSeen)
	switch {
	case duration < time.Minute:
		return "Just now"
	case duration < time.Hour:
		return fmt.Sprintf("%d minutes ago", int(duration.Minutes()))
	case duration < 24*time.Hour:
		return fmt.Sprintf("%d hours ago", int(duration.Hours()))
	default:
		return fmt.Sprintf("%d days ago", int(duration.Hours()/24))
	}
}

func getTagList(tags map[string]bool) []string {
	list := make([]string, 0, len(tags))
	for tag := range tags {
		list = append(list, tag)
	}
	return list
}

func getUniqueVendors(devices []*models.Device) []string {
	vendorMap := make(map[string]bool)
	for _, device := range devices {
		if device.DeviceID.Manufacturer != "" {
			vendorMap[device.DeviceID.Manufacturer] = true
		}
	}

	vendors := make([]string, 0, len(vendorMap))
	for vendor := range vendorMap {
		vendors = append(vendors, vendor)
	}
	return vendors
}

func getUniqueModels(devices []*models.Device) []string {
	modelMap := make(map[string]bool)
	for _, device := range devices {
		if device.DeviceID.ModelName != "" {
			modelMap[device.DeviceID.ModelName] = true
		}
	}

	models := make([]string, 0, len(modelMap))
	for model := range modelMap {
		models = append(models, model)
	}
	return models
}
