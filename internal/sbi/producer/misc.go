package producer

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/nextranet/gateway/c-plane/internal/context"
	"github.com/nextranet/gateway/c-plane/internal/logger"
	"github.com/nextranet/gateway/c-plane/internal/models"
	"github.com/nextranet/gateway/c-plane/pkg/factory"
	"github.com/nextranet/gateway/c-plane/pkg/service"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development, restrict in production
		return true
	},
}

// GetSystemStatus returns system status information
func GetSystemStatus(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		genieStatus := appContext.GetGenieACSStatus()
		deviceStats := appContext.GetDeviceStats()

		status := gin.H{
			"status": "operational",
			"services": gin.H{
				"nbi": gin.H{
					"status":    "running",
					"connected": genieStatus.NBIConnected,
				},
				"cwmp": gin.H{
					"status":    "running",
					"connected": genieStatus.CWMPConnected,
				},
				"fs": gin.H{
					"status":    "running",
					"connected": genieStatus.FSConnected,
				},
			},
			"metrics": gin.H{
				"totalDevices":  deviceStats.TotalDevices,
				"onlineDevices": deviceStats.OnlineDevices,
				"activeFaults":  deviceStats.ActiveFaults,
			},
			"lastCheck": genieStatus.LastCheck,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}

		// Determine overall status
		if !genieStatus.NBIConnected || !genieStatus.CWMPConnected {
			status["status"] = "degraded"
		}

		c.JSON(http.StatusOK, status)
	}
}

// GetSystemConfig returns system configuration
func GetSystemConfig(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := factory.GetConfig()

		// Sanitize sensitive information
		sanitizedConfig := gin.H{
			"info":   cfg.Info,
			"logger": cfg.Logger,
			"nbi": gin.H{
				"scheme":      cfg.NBI.Scheme,
				"bindingIPv4": cfg.NBI.BindingIPv4,
				"bindingIPv6": cfg.NBI.BindingIPv6,
				"port":        cfg.NBI.Port,
			},
			"ui": gin.H{
				"scheme":      cfg.UI.Scheme,
				"bindingIPv4": cfg.UI.BindingIPv4,
				"bindingIPv6": cfg.UI.BindingIPv6,
				"port":        cfg.UI.Port,
				"theme":       cfg.UI.Theme,
			},
			"genieacs": gin.H{
				"cwmpUrl": cfg.GenieACS.CWMPURL,
				"nbiUrl":  cfg.GenieACS.NBIURL,
				"fsUrl":   cfg.GenieACS.FSURL,
				"timeout": cfg.GenieACS.Timeout,
			},
		}

		c.JSON(http.StatusOK, sanitizedConfig)
	}
}

// UpdateSystemConfig updates system configuration
func UpdateSystemConfig(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		var updates map[string]interface{}
		if err := c.ShouldBindJSON(&updates); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request body",
			})
			return
		}

		cfg := factory.GetConfig()

		// Apply updates (limited to safe fields)
		if logLevel, ok := updates["logLevel"].(string); ok {
			cfg.Logger.Level = logLevel
			logger.SetLogLevel(logLevel)
		}

		if theme, ok := updates["theme"].(string); ok {
			if cfg.UI != nil {
				cfg.UI.Theme = theme
			}
		}

		// Save configuration
		if err := factory.SaveConfig(cfg, factory.GetConfigPath()); err != nil {
			logger.ProducerLog.Errorf("Failed to save config: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to save configuration",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Configuration updated successfully",
		})
	}
}

// GetTasks returns all tasks
func GetTasks(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Query("deviceId")
		status := c.Query("status")

		cfg := factory.GetConfig()
		genieService := service.NewGenieACSService(cfg.GenieACS, appContext)

		// Get tasks from GenieACS
		tasks, err := genieService.GetTasks(deviceID)
		if err != nil {
			logger.ProducerLog.Errorf("Failed to get tasks: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve tasks",
			})
			return
		}

		// Filter by status if provided
		if status != "" {
			filteredTasks := make([]*models.Task, 0)
			for _, task := range tasks {
				if task.Status == status {
					filteredTasks = append(filteredTasks, task)
				}
			}
			tasks = filteredTasks
		}

		c.JSON(http.StatusOK, gin.H{
			"tasks": tasks,
			"total": len(tasks),
		})
	}
}

// GetTask returns a single task by ID
func GetTask(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		taskID := c.Param("taskId")
		if taskID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Task ID is required",
			})
			return
		}

		// TODO: GenieACS doesn't provide a direct API to get a single task
		// We need to get all tasks and filter
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "Get single task not implemented",
		})
	}
}

// DeleteTask deletes a task
func DeleteTask(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		taskID := c.Param("taskId")
		if taskID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Task ID is required",
			})
			return
		}

		cfg := factory.GetConfig()
		genieService := service.NewGenieACSService(cfg.GenieACS, appContext)

		err := genieService.DeleteTask(taskID)
		if err != nil {
			logger.ProducerLog.Errorf("Failed to delete task: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to delete task",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Task deleted successfully",
			"taskId":  taskID,
		})
	}
}

// RetryTask retries a failed task
func RetryTask(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		taskID := c.Param("taskId")
		if taskID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Task ID is required",
			})
			return
		}

		// TODO: Implement task retry logic
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "Task retry not implemented",
		})
	}
}

// BulkRefreshDevices refreshes multiple devices
func BulkRefreshDevices(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			DeviceIDs []string `json:"deviceIds" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request body",
			})
			return
		}

		cfg := factory.GetConfig()
		genieService := service.NewGenieACSService(cfg.GenieACS, appContext)

		successful := 0
		failed := 0
		errors := make([]string, 0)

		for _, deviceID := range req.DeviceIDs {
			err := genieService.RefreshDevice(deviceID)
			if err != nil {
				failed++
				errors = append(errors, fmt.Sprintf("%s: %v", deviceID, err))
			} else {
				successful++
			}
		}

		response := gin.H{
			"message":    "Bulk refresh completed",
			"successful": successful,
			"failed":     failed,
		}

		if len(errors) > 0 {
			response["errors"] = errors
		}

		statusCode := http.StatusOK
		if failed > 0 && successful == 0 {
			statusCode = http.StatusInternalServerError
		} else if failed > 0 {
			statusCode = http.StatusPartialContent
		}

		c.JSON(statusCode, response)
	}
}

// BulkRebootDevices reboots multiple devices
func BulkRebootDevices(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			DeviceIDs []string `json:"deviceIds" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request body",
			})
			return
		}

		cfg := factory.GetConfig()
		genieService := service.NewGenieACSService(cfg.GenieACS, appContext)

		successful := 0
		failed := 0
		errors := make([]string, 0)

		task := map[string]interface{}{
			"name": "reboot",
		}

		for _, deviceID := range req.DeviceIDs {
			err := genieService.CreateTask(deviceID, task)
			if err != nil {
				failed++
				errors = append(errors, fmt.Sprintf("%s: %v", deviceID, err))
			} else {
				successful++
			}
		}

		response := gin.H{
			"message":    "Bulk reboot initiated",
			"successful": successful,
			"failed":     failed,
		}

		if len(errors) > 0 {
			response["errors"] = errors
		}

		statusCode := http.StatusOK
		if failed > 0 && successful == 0 {
			statusCode = http.StatusInternalServerError
		} else if failed > 0 {
			statusCode = http.StatusPartialContent
		}

		c.JSON(statusCode, response)
	}
}

// BulkSetParameters sets parameters on multiple devices
func BulkSetParameters(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			DeviceIDs  []string               `json:"deviceIds" binding:"required"`
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

		successful := 0
		failed := 0
		errors := make([]string, 0)

		for _, deviceID := range req.DeviceIDs {
			err := genieService.SetDeviceParameters(deviceID, req.Parameters)
			if err != nil {
				failed++
				errors = append(errors, fmt.Sprintf("%s: %v", deviceID, err))
			} else {
				successful++
			}
		}

		response := gin.H{
			"message":    "Bulk parameter update completed",
			"successful": successful,
			"failed":     failed,
		}

		if len(errors) > 0 {
			response["errors"] = errors
		}

		statusCode := http.StatusOK
		if failed > 0 && successful == 0 {
			statusCode = http.StatusInternalServerError
		} else if failed > 0 {
			statusCode = http.StatusPartialContent
		}

		c.JSON(statusCode, response)
	}
}

// BulkUpdateTags updates tags on multiple devices
func BulkUpdateTags(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			DeviceIDs []string `json:"deviceIds" binding:"required"`
			Tags      []string `json:"tags" binding:"required"`
			Operation string   `json:"operation"` // "add", "remove", "replace"
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request body",
			})
			return
		}

		if req.Operation == "" {
			req.Operation = "add"
		}

		successful := 0
		failed := 0
		errors := make([]string, 0)

		for _, deviceID := range req.DeviceIDs {
			device, exists := appContext.GetDevice(deviceID)
			if !exists {
				failed++
				errors = append(errors, fmt.Sprintf("%s: device not found", deviceID))
				continue
			}

			switch req.Operation {
			case "add":
				for _, tag := range req.Tags {
					device.Tags[tag] = true
				}
			case "remove":
				for _, tag := range req.Tags {
					delete(device.Tags, tag)
				}
			case "replace":
				device.Tags = make(map[string]bool)
				for _, tag := range req.Tags {
					device.Tags[tag] = true
				}
			default:
				failed++
				errors = append(errors, fmt.Sprintf("%s: invalid operation", deviceID))
				continue
			}

			appContext.AddDevice(device)
			successful++
		}

		response := gin.H{
			"message":    "Bulk tag update completed",
			"successful": successful,
			"failed":     failed,
			"operation":  req.Operation,
		}

		if len(errors) > 0 {
			response["errors"] = errors
		}

		statusCode := http.StatusOK
		if failed > 0 && successful == 0 {
			statusCode = http.StatusInternalServerError
		} else if failed > 0 {
			statusCode = http.StatusPartialContent
		}

		c.JSON(statusCode, response)
	}
}

// ExportDevices exports devices to CSV
func ExportDevices(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		devices := appContext.GetAllDevices()

		// Set headers for CSV download
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=devices.csv")

		writer := csv.NewWriter(c.Writer)
		defer writer.Flush()

		// Write header
		header := []string{
			"ID", "Serial Number", "Manufacturer", "Model", "Product Class",
			"IP Address", "External IP", "Status", "Last Seen",
			"Software Version", "Hardware Version", "Tags",
		}
		writer.Write(header)

		// Write device data
		for _, device := range devices {

			tags := make([]string, 0, len(device.Tags))
			for tag := range device.Tags {
				tags = append(tags, tag)
			}

			status := "offline"
			if device.Status.Online {
				status = "online"
			}

			row := []string{
				device.ID,
				device.DeviceID.SerialNumber,
				device.DeviceID.Manufacturer,
				device.DeviceID.ModelName,
				device.DeviceID.ProductClass,
				device.DeviceID.IPAddress,
				device.DeviceID.ExternalIPAddress,
				status,
				device.Status.LastSeen.Format(time.RFC3339),
				device.DeviceID.SoftwareVersion,
				device.DeviceID.HardwareVersion,
				strings.Join(tags, ";"),
			}
			writer.Write(row)
		}
	}
}

// ExportFaults exports faults to CSV
func ExportFaults(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		faults := appContext.GetActiveFaults()

		// Set headers for CSV download
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=faults.csv")

		writer := csv.NewWriter(c.Writer)
		defer writer.Flush()

		// Write header
		header := []string{
			"ID", "Device ID", "Channel", "Code", "Message", "Severity",
			"Status", "Timestamp", "Acknowledged By", "Resolved By",
		}
		writer.Write(header)

		// Write fault data
		for _, fault := range faults {
			acknowledgedBy := ""
			if fault.AcknowledgedBy != "" {
				acknowledgedBy = fault.AcknowledgedBy
			}

			resolvedBy := ""
			if fault.ResolvedBy != "" {
				resolvedBy = fault.ResolvedBy
			}

			row := []string{
				fault.ID,
				fault.DeviceID,
				fault.Channel,
				fault.Code,
				fault.Message,
				fault.Severity,
				fault.Status,
				fault.Timestamp.Format(time.RFC3339),
				acknowledgedBy,
				resolvedBy,
			}
			writer.Write(row)
		}
	}
}

// WebSocketHandler handles WebSocket connections for real-time updates
func WebSocketHandler(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.ProducerLog.Errorf("WebSocket upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		// Send initial connection message
		conn.WriteJSON(gin.H{
			"type":    "connected",
			"message": "WebSocket connection established",
		})

		// Create a ticker for periodic updates
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		// Channel for client messages
		clientMsg := make(chan []byte, 10)
		go func() {
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						logger.ProducerLog.Errorf("WebSocket read error: %v", err)
					}
					close(clientMsg)
					return
				}
				clientMsg <- msg
			}
		}()

		for {
			select {
			case <-ticker.C:
				// Send periodic updates
				stats := appContext.GetDeviceStats()
				update := gin.H{
					"type": "stats_update",
					"data": gin.H{
						"totalDevices":  stats.TotalDevices,
						"onlineDevices": stats.OnlineDevices,
						"activeFaults":  stats.ActiveFaults,
						"timestamp":     time.Now().UTC().Format(time.RFC3339),
					},
				}

				if err := conn.WriteJSON(update); err != nil {
					logger.ProducerLog.Errorf("WebSocket write error: %v", err)
					return
				}

			case msg, ok := <-clientMsg:
				if !ok {
					return
				}
				// Handle client messages (e.g., subscription requests)
				logger.ProducerLog.Debugf("Received WebSocket message: %s", string(msg))
			}
		}
	}
}
