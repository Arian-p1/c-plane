package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/nextranet/gateway/c-plane/internal/context"
	"github.com/nextranet/gateway/c-plane/internal/logger"
	"github.com/nextranet/gateway/c-plane/internal/models"
	"github.com/nextranet/gateway/c-plane/internal/web/templates"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development, restrict in production
		return true
	},
}

// Faults renders the faults/alarms page
func Faults(appContext *context.Context) gin.HandlerFunc {
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
		filter := &models.FaultFilter{
			DeviceID:  c.Query("deviceId"),
			Severity:  c.Query("severity"),
			Status:    c.Query("status"),
			Channel:   c.Query("channel"),
			TimeRange: c.Query("timeRange"),
		}

		// Get faults from context
		allFaults := appContext.GetActiveFaults()

		// Apply filters
		filteredFaults := filterFaults(allFaults, filter)

		// Calculate pagination
		totalCount := len(filteredFaults)
		totalPages := (totalCount + pageSize - 1) / pageSize

		// Get page of results
		start := (page - 1) * pageSize
		end := start + pageSize
		if end > totalCount {
			end = totalCount
		}

		var pageFaults []*models.Fault
		if start < totalCount {
			pageFaults = filteredFaults[start:end]
		}

		// Convert to display format
		displayFaults := make([]*templates.FaultDisplay, 0, len(pageFaults))
		for _, fault := range pageFaults {
			displayFault := &templates.FaultDisplay{
				Fault:          fault,
				DeviceName:     getDeviceName(fault.DeviceSerial, appContext),
				SeverityClass:  getSeverityClass(fault.Severity),
				StatusClass:    getStatusClass(fault.Status),
				TimeAgoText:    formatTimeAgo(fault.Timestamp),
				CanAcknowledge: fault.Status == models.FaultStatusActive,
				CanResolve:     fault.Status == models.FaultStatusActive || fault.Status == models.FaultStatusAcknowledged,
			}
			displayFaults = append(displayFaults, displayFault)
		}

		// Calculate severity stats
		severityStats := make(map[string]int)
		activeCount := 0
		acknowledgedCount := 0

		for _, fault := range allFaults {
			severityStats[fault.Severity]++
			switch fault.Status {
			case models.FaultStatusActive:
				activeCount++
			case models.FaultStatusAcknowledged:
				acknowledgedCount++
			}
		}

		// Get theme
		theme := c.GetString("theme")
		if theme == "" {
			theme = "dark"
		}

		// Prepare data for template
		data := templates.FaultsPageData{
			BasePageData: templates.BasePageData{
				Title:       "Faults & Alarms",
				Theme:       theme,
				CurrentPath: "/faults",
			},
			Faults:            displayFaults,
			TotalCount:        totalCount,
			ActiveCount:       activeCount,
			AcknowledgedCount: acknowledgedCount,
			CurrentPage:       page,
			PageSize:          pageSize,
			TotalPages:        totalPages,
			Filters: templates.FaultFilters{
				DeviceID:  filter.DeviceID,
				Severity:  filter.Severity,
				Status:    filter.Status,
				Channel:   filter.Channel,
				TimeRange: filter.TimeRange,
			},
			SeverityStats: severityStats,
		}

		// Render the faults page
		component := templates.FaultsPage(data)
		c.Header("Content-Type", "text/html; charset=utf-8")

		if err := component.Render(c.Request.Context(), c.Writer); err != nil {
			logger.WebLog.Errorf("Failed to render faults page: %v", err)
			c.String(http.StatusInternalServerError, "Failed to render page")
			return
		}
	}
}

// Helper functions for fault processing

func filterFaults(faults []*models.Fault, filter *models.FaultFilter) []*models.Fault {
	if filter == nil {
		return faults
	}

	filtered := make([]*models.Fault, 0)
	for _, fault := range faults {
		// Apply device filter
		if filter.DeviceID != "" {
			if !strings.Contains(strings.ToLower(fault.DeviceSerial), strings.ToLower(filter.DeviceID)) {
				continue
			}
		}

		// Apply severity filter
		if filter.Severity != "" && fault.Severity != filter.Severity {
			continue
		}

		// Apply status filter
		if filter.Status != "" && fault.Status != filter.Status {
			continue
		}

		// Apply time range filter
		if filter.TimeRange != "" {
			if !isWithinTimeRange(fault.Timestamp, filter.TimeRange) {
				continue
			}
		}

		filtered = append(filtered, fault)
	}

	return filtered
}

func getDeviceName(serial string, appContext *context.Context) string {
	if device, exists := appContext.GetDeviceBySerial(serial); exists {
		if device.DeviceID.ModelName != "" {
			return fmt.Sprintf("%s %s", device.DeviceID.Manufacturer, device.DeviceID.ModelName)
		}
		return device.DeviceID.Manufacturer
	}
	return serial
}

func getSeverityClass(severity string) string {
	switch severity {
	case models.SeverityCritical:
		return "text-red-600"
	case models.SeverityMajor:
		return "text-orange-600"
	case models.SeverityMinor:
		return "text-yellow-600"
	case models.SeverityWarning:
		return "text-yellow-500"
	default:
		return "text-blue-500"
	}
}

func getStatusClass(status string) string {
	switch status {
	case models.FaultStatusActive:
		return "text-red-600"
	case models.FaultStatusAcknowledged:
		return "text-yellow-600"
	case models.FaultStatusResolved:
		return "text-green-600"
	default:
		return "text-gray-600"
	}
}

func formatTimeAgo(timestamp time.Time) string {
	duration := time.Since(timestamp)
	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		return fmt.Sprintf("%d minutes ago", int(duration.Minutes()))
	case duration < 24*time.Hour:
		return fmt.Sprintf("%d hours ago", int(duration.Hours()))
	default:
		return fmt.Sprintf("%d days ago", int(duration.Hours()/24))
	}
}

func isWithinTimeRange(timestamp time.Time, timeRange string) bool {
	now := time.Now()
	switch timeRange {
	case "1h":
		return now.Sub(timestamp) <= time.Hour
	case "24h":
		return now.Sub(timestamp) <= 24*time.Hour
	case "7d":
		return now.Sub(timestamp) <= 7*24*time.Hour
	case "30d":
		return now.Sub(timestamp) <= 30*24*time.Hour
	default:
		return true
	}
}

// AcknowledgeFault handles fault acknowledgment
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
			AcknowledgedBy string `json:"acknowledgedBy"`
			Notes          string `json:"notes"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request body",
			})
			return
		}

		err := appContext.AcknowledgeFault(faultID, req.AcknowledgedBy)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to acknowledge fault",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Fault acknowledged successfully",
		})
	}
}

// ResolveFault handles fault resolution
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
			ResolvedBy string `json:"resolvedBy"`
			Resolution string `json:"resolution"`
			Notes      string `json:"notes"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request body",
			})
			return
		}

		err := appContext.ResolveFault(faultID, req.ResolvedBy)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to resolve fault",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Fault resolved successfully",
		})
	}
}

// RecentFaults returns recent faults for AJAX updates
func RecentFaults(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := 10
		if l := c.Query("limit"); l != "" {
			if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 50 {
				limit = val
			}
		}

		faults := appContext.GetActiveFaults()
		if len(faults) > limit {
			faults = faults[:limit]
		}

		c.JSON(http.StatusOK, gin.H{
			"faults": faults,
			"total":  len(faults),
		})
	}
}

// GetDeviceFilters returns saved device filters
func GetDeviceFilters(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Placeholder - return default filters
		filters := []templates.FilterPreset{
			{
				ID:      "offline",
				Name:    "Offline Devices",
				Filters: map[string]interface{}{"status": "offline"},
				Default: false,
			},
			{
				ID:      "critical",
				Name:    "Devices with Critical Faults",
				Filters: map[string]interface{}{"hasCriticalFaults": true},
				Default: false,
			},
		}

		c.JSON(http.StatusOK, gin.H{
			"filters": filters,
		})
	}
}

// SaveDeviceFilter saves a device filter preset
func SaveDeviceFilter(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		var filter templates.FilterPreset
		if err := c.ShouldBindJSON(&filter); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request body",
			})
			return
		}

		// Placeholder - save filter to storage
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Filter saved successfully",
			"filter":  filter,
		})
	}
}

// DeleteDeviceFilter deletes a device filter preset
func DeleteDeviceFilter(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		filterID := c.Param("filterId")
		if filterID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Filter ID is required",
			})
			return
		}

		// Placeholder - delete filter from storage
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Filter deleted successfully",
		})
	}
}

// WebSocketHandler handles WebSocket connections for real-time updates
func WebSocketHandler(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.WebLog.Errorf("WebSocket upgrade failed: %v", err)
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
						logger.WebLog.Errorf("WebSocket read error: %v", err)
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
				genieStatus := appContext.GetGenieACSStatus()

				update := gin.H{
					"type": "stats_update",
					"data": gin.H{
						"devices": gin.H{
							"total":   stats.TotalDevices,
							"online":  stats.OnlineDevices,
							"offline": stats.OfflineDevices,
						},
						"faults": gin.H{
							"active":   stats.ActiveFaults,
							"critical": stats.CriticalFaults,
						},
						"system": gin.H{
							"cwmpConnected": genieStatus.CWMPConnected,
							"nbiConnected":  genieStatus.NBIConnected,
							"fsConnected":   genieStatus.FSConnected,
						},
						"timestamp": time.Now().UTC().Format(time.RFC3339),
					},
				}

				if err := conn.WriteJSON(update); err != nil {
					logger.WebLog.Errorf("WebSocket write error: %v", err)
					return
				}

			case msg, ok := <-clientMsg:
				if !ok {
					return
				}
				// Handle client messages (e.g., subscription requests)
				logger.WebLog.Debugf("Received WebSocket message: %s", string(msg))
			}
		}
	}
}
