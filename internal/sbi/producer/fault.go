package producer

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nextranet/gateway/c-plane/internal/context"
	"github.com/nextranet/gateway/c-plane/internal/logger"
	"github.com/nextranet/gateway/c-plane/internal/models"
	"github.com/nextranet/gateway/c-plane/pkg/factory"
	"github.com/nextranet/gateway/c-plane/pkg/service"
)

// GetFaults returns a list of faults
func GetFaults(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get query parameters
		deviceID := c.Query("deviceId")
		status := c.Query("status")
		severity := c.Query("severity")
		channel := c.Query("channel")

		// Get all faults from GenieACS
		cfg := factory.GetConfig()
		genieService := service.NewGenieACSService(cfg.GenieACS, appContext)

		faults, err := genieService.GetFaults("")
		if err != nil {
			logger.ProducerLog.Errorf("Failed to get faults from GenieACS: %v", err)
			// Fall back to context data
			faults = appContext.GetActiveFaults()
		}

		// Add faults to context
		for _, fault := range faults {
			appContext.AddFault(fault)
		}

		// Apply filters
		filteredFaults := make([]*models.Fault, 0)

		for _, fault := range faults {
			// Filter by device ID
			if deviceID != "" && fault.DeviceID != deviceID {
				continue
			}

			// Filter by status
			if status != "" && fault.Status != status {
				continue
			}

			// Filter by severity
			if severity != "" && fault.Severity != severity {
				continue
			}

			// Filter by channel
			if channel != "" && fault.Channel != channel {
				continue
			}

			filteredFaults = append(filteredFaults, fault)
		}

		// Apply pagination
		page := 1
		pageSize := 20

		if p := c.Query("page"); p != "" {
			if val, err := strconv.Atoi(p); err == nil && val > 0 {
				page = val
			}
		}

		if ps := c.Query("pageSize"); ps != "" {
			if val, err := strconv.Atoi(ps); err == nil && val > 0 && val <= 100 {
				pageSize = val
			}
		}

		start := (page - 1) * pageSize
		end := start + pageSize

		if start > len(filteredFaults) {
			start = len(filteredFaults)
		}
		if end > len(filteredFaults) {
			end = len(filteredFaults)
		}

		paginatedFaults := filteredFaults[start:end]

		c.JSON(http.StatusOK, gin.H{
			"faults":   paginatedFaults,
			"total":    len(filteredFaults),
			"page":     page,
			"pageSize": pageSize,
		})
	}
}

// GetFault returns a single fault by ID
func GetFault(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		faultID := c.Param("faultId")
		if faultID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Fault ID is required",
			})
			return
		}

		// Check context first
		fault, exists := appContext.GetFault(faultID)
		if !exists {
			// Try to fetch from GenieACS
			cfg := factory.GetConfig()
			genieService := service.NewGenieACSService(cfg.GenieACS, appContext)

			faults, err := genieService.GetFaults("")
			if err != nil {
				logger.ProducerLog.Errorf("Failed to get faults from GenieACS: %v", err)
				c.JSON(http.StatusNotFound, gin.H{
					"error": "Fault not found",
				})
				return
			}

			// Search for the fault
			for _, f := range faults {
				if f.ID == faultID {
					fault = f
					appContext.AddFault(f)
					exists = true
					break
				}
			}

			if !exists {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "Fault not found",
				})
				return
			}
		}

		// Get device information
		var device *models.Device
		if fault.DeviceID != "" {
			device, _ = appContext.GetDevice(fault.DeviceID)
		}

		response := gin.H{
			"fault": fault,
		}

		if device != nil {
			response["device"] = gin.H{
				"id":           device.ID,
				"serialNumber": device.DeviceID.SerialNumber,
				"modelName":    device.DeviceID.ModelName,
				"manufacturer": device.DeviceID.Manufacturer,
			}
		}

		c.JSON(http.StatusOK, response)
	}
}

// AcknowledgeFault acknowledges a fault
func AcknowledgeFault(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		faultID := c.Param("faultId")
		if faultID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Fault ID is required",
			})
			return
		}

		var req struct {
			AcknowledgedBy string `json:"acknowledgedBy" binding:"required"`
			Notes          string `json:"notes,omitempty"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request body",
			})
			return
		}

		// Check if fault exists
		fault, exists := appContext.GetFault(faultID)
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Fault not found",
			})
			return
		}

		// Check if already acknowledged or resolved
		if fault.Status == models.FaultStatusAcknowledged {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Fault is already acknowledged",
			})
			return
		}

		if fault.Status == models.FaultStatusResolved {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Fault is already resolved",
			})
			return
		}

		// Acknowledge the fault
		err := appContext.AcknowledgeFault(faultID, req.AcknowledgedBy)
		if err != nil {
			logger.ProducerLog.Errorf("Failed to acknowledge fault: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to acknowledge fault",
			})
			return
		}

		// Get updated fault
		fault, _ = appContext.GetFault(faultID)

		c.JSON(http.StatusOK, gin.H{
			"message": "Fault acknowledged successfully",
			"fault":   fault,
		})
	}
}

// ResolveFault resolves a fault
func ResolveFault(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		faultID := c.Param("faultId")
		if faultID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Fault ID is required",
			})
			return
		}

		var req struct {
			ResolvedBy string `json:"resolvedBy" binding:"required"`
			Resolution string `json:"resolution,omitempty"`
			Notes      string `json:"notes,omitempty"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request body",
			})
			return
		}

		// Check if fault exists
		fault, exists := appContext.GetFault(faultID)
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Fault not found",
			})
			return
		}

		// Check if already resolved
		if fault.Status == models.FaultStatusResolved {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Fault is already resolved",
			})
			return
		}

		// Resolve the fault
		err := appContext.ResolveFault(faultID, req.ResolvedBy)
		if err != nil {
			logger.ProducerLog.Errorf("Failed to resolve fault: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to resolve fault",
			})
			return
		}

		// Delete fault from GenieACS
		cfg := factory.GetConfig()
		genieService := service.NewGenieACSService(cfg.GenieACS, appContext)

		if err := genieService.DeleteFault(faultID); err != nil {
			logger.ProducerLog.Warnf("Failed to delete fault from GenieACS: %v", err)
			// Continue anyway as fault is marked as resolved
		}

		// Get updated fault
		fault, _ = appContext.GetFault(faultID)

		c.JSON(http.StatusOK, gin.H{
			"message": "Fault resolved successfully",
			"fault":   fault,
		})
	}
}

// DeleteFault deletes a fault
func DeleteFault(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		faultID := c.Param("faultId")
		if faultID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Fault ID is required",
			})
			return
		}

		// Check if fault exists
		fault, exists := appContext.GetFault(faultID)
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Fault not found",
			})
			return
		}

		// Check if fault is active
		if fault.Status == models.FaultStatusActive {
			force := c.Query("force") == "true"
			if !force {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Cannot delete active fault. Use force=true to delete anyway",
				})
				return
			}
		}

		// Delete from GenieACS
		cfg := factory.GetConfig()
		genieService := service.NewGenieACSService(cfg.GenieACS, appContext)

		if err := genieService.DeleteFault(faultID); err != nil {
			logger.ProducerLog.Errorf("Failed to delete fault from GenieACS: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to delete fault",
			})
			return
		}

		// Mark as expired in context
		now := time.Now()
		fault.Status = models.FaultStatusExpired
		fault.Expiry = &now
		appContext.AddFault(fault)

		c.JSON(http.StatusOK, gin.H{
			"message": "Fault deleted successfully",
			"faultId": faultID,
		})
	}
}

// GetFaultStats returns fault statistics
func GetFaultStats(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		faults := appContext.GetActiveFaults()

		// Calculate statistics
		stats := struct {
			Total        int             `json:"total"`
			Active       int             `json:"active"`
			Acknowledged int             `json:"acknowledged"`
			BySeverity   map[string]int  `json:"bySeverity"`
			ByChannel    map[string]int  `json:"byChannel"`
			ByDevice     map[string]int  `json:"byDevice"`
			RecentFaults []*models.Fault `json:"recentFaults"`
		}{
			BySeverity: make(map[string]int),
			ByChannel:  make(map[string]int),
			ByDevice:   make(map[string]int),
		}

		stats.Total = len(faults)

		// Process faults
		for _, fault := range faults {
			// Count by status
			switch fault.Status {
			case models.FaultStatusActive:
				stats.Active++
			case models.FaultStatusAcknowledged:
				stats.Acknowledged++
			}

			// Count by severity
			stats.BySeverity[fault.Severity]++

			// Count by channel
			stats.ByChannel[fault.Channel]++

			// Count by device
			stats.ByDevice[fault.DeviceID]++
		}

		// Get recent faults (last 10)
		if len(faults) > 10 {
			stats.RecentFaults = faults[:10]
		} else {
			stats.RecentFaults = faults
		}

		c.JSON(http.StatusOK, stats)
	}
}
