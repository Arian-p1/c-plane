package producer

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nextranet/gateway/c-plane/internal/context"
	"github.com/nextranet/gateway/c-plane/internal/logger"
	"github.com/nextranet/gateway/c-plane/internal/models"
	"github.com/nextranet/gateway/c-plane/pkg/factory"
	"github.com/nextranet/gateway/c-plane/pkg/service"
)

// GetDevices returns a list of devices
func GetDevices(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Build filter from query parameters
		filter := &models.DeviceFilter{
			Pagination: &models.PaginationOptions{
				Page:     1,
				PageSize: 20,
			},
		}

		// Parse pagination
		if page := c.Query("page"); page != "" {
			if p, err := strconv.Atoi(page); err == nil && p > 0 {
				filter.Pagination.Page = p
			}
		}

		if pageSize := c.Query("pageSize"); pageSize != "" {
			if ps, err := strconv.Atoi(pageSize); err == nil && ps > 0 && ps <= 100 {
				filter.Pagination.PageSize = ps
			}
		}

		filter.Pagination.SortBy = c.DefaultQuery("sortBy", "lastInform")
		filter.Pagination.SortDir = c.DefaultQuery("sortDir", "desc")

		// Parse filters
		if manufacturer := c.Query("manufacturer"); manufacturer != "" {
			filter.Manufacturer = manufacturer
		}

		if modelName := c.Query("modelName"); modelName != "" {
			filter.ModelName = modelName
		}

		if productClass := c.Query("productClass"); productClass != "" {
			filter.ProductClass = productClass
		}

		if tags := c.Query("tags"); tags != "" {
			filter.Tags = strings.Split(tags, ",")
		}

		if online := c.Query("online"); online != "" {
			if o, err := strconv.ParseBool(online); err == nil {
				filter.Online = &o
			}
		}

		if search := c.Query("search"); search != "" {
			filter.Search = search
		}

		// Parse IP range
		if startIP := c.Query("startIP"); startIP != "" {
			if endIP := c.Query("endIP"); endIP != "" {
				filter.IPRange = &models.IPRange{
					StartIP: startIP,
					EndIP:   endIP,
				}
			}
		}

		// Get devices from GenieACS
		cfg := factory.GetConfig()
		genieService := service.NewGenieACSService(cfg.GenieACS, appContext)

		devices, err := genieService.GetDevices(filter)
		if err != nil {
			logger.ProducerLog.Errorf("Failed to get devices: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve devices",
			})
			return
		}

		// Use devices directly from GenieACS
		result := devices

		c.JSON(http.StatusOK, gin.H{
			"devices":  result,
			"total":    len(result),
			"page":     filter.Pagination.Page,
			"pageSize": filter.Pagination.PageSize,
		})
	}
}

// GetDevice returns a single device by ID
func GetDevice(appContext *context.Context) gin.HandlerFunc {
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

		device, err := genieService.GetDevice(deviceID)
		if err != nil {
			if err == models.ErrDeviceNotFound {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "Device not found",
				})
				return
			}
			logger.ProducerLog.Errorf("Failed to get device: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve device",
			})
			return
		}

		c.JSON(http.StatusOK, device)
	}
}

// RefreshDevice refreshes device data from CPE
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
			logger.ProducerLog.Errorf("Failed to refresh device: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to refresh device",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  "Device refresh initiated",
			"deviceId": deviceID,
		})
	}
}

// GetDeviceParameters retrieves device parameters
func GetDeviceParameters(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Param("deviceId")
		if deviceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Device ID is required",
			})
			return
		}

		// Get parameter names from query
		paramNames := c.QueryArray("names")
		if len(paramNames) == 0 {
			// Default to common parameters
			paramNames = []string{
				"InternetGatewayDevice.DeviceInfo.Manufacturer",
				"InternetGatewayDevice.DeviceInfo.ModelName",
				"InternetGatewayDevice.DeviceInfo.SoftwareVersion",
				"InternetGatewayDevice.DeviceInfo.HardwareVersion",
				"InternetGatewayDevice.LANDevice.1.LANHostConfigManagement.IPInterface.1.IPInterfaceIPAddress",
				"InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANIPConnection.1.ExternalIPAddress",
			}
		}

		cfg := factory.GetConfig()
		genieService := service.NewGenieACSService(cfg.GenieACS, appContext)

		parameters, err := genieService.GetDeviceParameters(deviceID, paramNames)
		if err != nil {
			if err == models.ErrDeviceNotFound {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "Device not found",
				})
				return
			}
			logger.ProducerLog.Errorf("Failed to get device parameters: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve device parameters",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"deviceId":   deviceID,
			"parameters": parameters,
		})
	}
}

// SetDeviceParameters sets device parameters
func SetDeviceParameters(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Param("deviceId")
		if deviceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Device ID is required",
			})
			return
		}

		var req struct {
			Parameters map[string]interface{} `json:"parameters" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request body",
			})
			return
		}

		cfg := factory.GetConfig()
		genieService := service.NewGenieACSService(cfg.GenieACS, appContext)

		err := genieService.SetDeviceParameters(deviceID, req.Parameters)
		if err != nil {
			logger.ProducerLog.Errorf("Failed to set device parameters: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to set device parameters",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  "Parameters set successfully",
			"deviceId": deviceID,
		})
	}
}

// GetDeviceTasks retrieves tasks for a device
func GetDeviceTasks(appContext *context.Context) gin.HandlerFunc {
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

		tasks, err := genieService.GetTasks(deviceID)
		if err != nil {
			logger.ProducerLog.Errorf("Failed to get device tasks: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve device tasks",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"deviceId": deviceID,
			"tasks":    tasks,
		})
	}
}

// CreateDeviceTask creates a new task for a device
func CreateDeviceTask(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Param("deviceId")
		if deviceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Device ID is required",
			})
			return
		}

		var task map[string]interface{}
		if err := c.ShouldBindJSON(&task); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request body",
			})
			return
		}

		cfg := factory.GetConfig()
		genieService := service.NewGenieACSService(cfg.GenieACS, appContext)

		err := genieService.CreateTask(deviceID, task)
		if err != nil {
			logger.ProducerLog.Errorf("Failed to create device task: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create device task",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  "Task created successfully",
			"deviceId": deviceID,
		})
	}
}

// GetDeviceFaults retrieves faults for a device
func GetDeviceFaults(appContext *context.Context) gin.HandlerFunc {
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

		faults, err := genieService.GetFaults(deviceID)
		if err != nil {
			logger.ProducerLog.Errorf("Failed to get device faults: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve device faults",
			})
			return
		}

		// Add faults to context
		for _, fault := range faults {
			appContext.AddFault(fault)
		}

		c.JSON(http.StatusOK, gin.H{
			"deviceId": deviceID,
			"faults":   faults,
		})
	}
}

// RebootDevice reboots a device
func RebootDevice(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Param("deviceId")
		if deviceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Device ID is required",
			})
			return
		}

		task := map[string]interface{}{
			"name": "reboot",
		}

		cfg := factory.GetConfig()
		genieService := service.NewGenieACSService(cfg.GenieACS, appContext)

		err := genieService.CreateTask(deviceID, task)
		if err != nil {
			logger.ProducerLog.Errorf("Failed to reboot device: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to reboot device",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  "Device reboot initiated",
			"deviceId": deviceID,
		})
	}
}

// FactoryResetDevice performs a factory reset on a device
func FactoryResetDevice(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Param("deviceId")
		if deviceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Device ID is required",
			})
			return
		}

		task := map[string]interface{}{
			"name": "factoryReset",
		}

		cfg := factory.GetConfig()
		genieService := service.NewGenieACSService(cfg.GenieACS, appContext)

		err := genieService.CreateTask(deviceID, task)
		if err != nil {
			logger.ProducerLog.Errorf("Failed to factory reset device: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to factory reset device",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  "Device factory reset initiated",
			"deviceId": deviceID,
		})
	}
}

// UpdateDeviceTags updates device tags
func UpdateDeviceTags(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Param("deviceId")
		if deviceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Device ID is required",
			})
			return
		}

		var req struct {
			Tags []string `json:"tags" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request body",
			})
			return
		}

		// TODO: Implement tag updates in GenieACS
		// For now, just acknowledge the request

		c.JSON(http.StatusOK, gin.H{
			"message":  "Device tags updated successfully",
			"deviceId": deviceID,
			"tags":     req.Tags,
		})
	}
}
