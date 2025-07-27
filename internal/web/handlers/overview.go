package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nextranet/gateway/c-plane/internal/context"
	"github.com/nextranet/gateway/c-plane/internal/logger"
	"github.com/nextranet/gateway/c-plane/internal/models"
	"github.com/nextranet/gateway/c-plane/internal/web/templates"
	"github.com/nextranet/gateway/c-plane/pkg/factory"
	"github.com/nextranet/gateway/c-plane/pkg/service"
)

// RedirectToOverview redirects root path to overview
func RedirectToOverview() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/overview")
	}
}

// Overview renders the overview page
func Overview(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if we need to fetch data from GenieACS
		deviceStats := appContext.GetDeviceStats()
		if deviceStats.TotalDevices == 0 {
			// Fetch devices from GenieACS to populate context
			cfg := factory.GetConfig()
			genieService := service.NewGenieACSService(cfg.GenieACS, appContext)

			devices, err := genieService.GetDevices(&models.DeviceFilter{})
			if err != nil {
				logger.WebLog.Errorf("Failed to fetch devices for overview: %v", err)
			} else {
				// Add devices to context
				for _, device := range devices {
					appContext.AddDevice(device)
				}
			}

			// Fetch faults from GenieACS
			faults, err := genieService.GetFaults("")
			if err != nil {
				logger.WebLog.Errorf("Failed to fetch faults for overview: %v", err)
			} else {
				// Add faults to context
				for _, fault := range faults {
					appContext.AddFault(fault)
				}
			}

			// Refresh stats after adding data
			deviceStats = appContext.GetDeviceStats()
		}

		// Add dummy test data if still no data available for demonstration
		if deviceStats.TotalDevices == 0 {
			logger.WebLog.Info("No devices found, adding test data for demonstration")
			// Create some test devices for demonstration
			testDevices := []*models.Device{
				{
					ID: "test-device-1",
					DeviceID: models.DeviceID{
						SerialNumber:      "TD001",
						Manufacturer:      "TestVendor",
						ModelName:         "TestModel-A",
						ProductClass:      "TestClass",
						SoftwareVersion:   "1.0.0",
						HardwareVersion:   "1.0",
						IPAddress:         "192.168.1.100",
						ExternalIPAddress: "203.0.113.1",
					},
					Status: models.DeviceStatus{
						Online:   true,
						LastSeen: time.Now().Add(-5 * time.Minute),
					},
					Tags: make(map[string]bool),
				},
				{
					ID: "test-device-2",
					DeviceID: models.DeviceID{
						SerialNumber:      "TD002",
						Manufacturer:      "", // Empty manufacturer to test "Unknown"
						ModelName:         "TestModel-B",
						ProductClass:      "TestClass",
						SoftwareVersion:   "1.1.0",
						HardwareVersion:   "1.1",
						IPAddress:         "192.168.1.101",
						ExternalIPAddress: "203.0.113.2",
					},
					Status: models.DeviceStatus{
						Online:   false,
						LastSeen: time.Now().Add(-30 * time.Minute),
					},
					Tags: make(map[string]bool),
				},
			}

			// Create some test faults
			testFaults := []*models.Fault{
				{
					ID:           "test-fault-1",
					DeviceID:     "test-device-1",
					DeviceSerial: "TD001",
					Code:         "TEST_001",
					Message:      "Test critical fault",
					Severity:     models.SeverityCritical,
					Status:       models.FaultStatusActive,
					Timestamp:    time.Now().Add(-10 * time.Minute),
				},
				{
					ID:           "test-fault-2",
					DeviceID:     "test-device-2",
					DeviceSerial: "TD002",
					Code:         "TEST_002",
					Message:      "Test minor fault",
					Severity:     models.SeverityMinor,
					Status:       models.FaultStatusActive,
					Timestamp:    time.Now().Add(-5 * time.Minute),
				},
			}

			// Add test data to context
			for _, device := range testDevices {
				appContext.AddDevice(device)
			}
			for _, fault := range testFaults {
				appContext.AddFault(fault)
			}

			// Refresh stats with test data
			deviceStats = appContext.GetDeviceStats()
			logger.WebLog.Infof("Added test data: %d devices, %d faults", len(testDevices), len(testFaults))
		}

		// Get active faults
		faults := appContext.GetActiveFaults()
		faultsBySeverity := make(map[string]int)
		criticalFaults := make([]*models.Fault, 0)

		for _, fault := range faults {
			faultsBySeverity[fault.Severity]++
			if fault.Severity == models.SeverityCritical {
				criticalFaults = append(criticalFaults, fault)
			}
		}

		// Get recent faults (last 5)
		recentFaults := faults
		if len(recentFaults) > 5 {
			recentFaults = recentFaults[:5]
		}

		// Get system status
		genieStatus := appContext.GetGenieACSStatus()

		// Calculate health score (simple implementation)
		healthScore := 100
		if !genieStatus.CWMPConnected {
			healthScore -= 30
		}
		if !genieStatus.NBIConnected {
			healthScore -= 30
		}
		if deviceStats.OfflineDevices > 0 {
			offlinePercentage := float64(deviceStats.OfflineDevices) / float64(deviceStats.TotalDevices) * 100
			healthScore -= int(offlinePercentage * 0.4)
		}
		if len(criticalFaults) > 0 {
			healthScore -= len(criticalFaults) * 5
		}
		if healthScore < 0 {
			healthScore = 0
		}

		// Get theme from context
		theme := c.GetString("theme")
		if theme == "" {
			theme = "dark"
		}

		// Add fallback data for charts if no data is available
		vendorData := deviceStats.DevicesByVendor
		if len(vendorData) == 0 {
			// If we have devices but no vendor breakdown, show as unknown
			if deviceStats.TotalDevices > 0 {
				vendorData = map[string]int{
					"Unknown": deviceStats.TotalDevices,
				}
			} else {
				vendorData = map[string]int{
					"TestVendor": 1,
				}
			}
		}

		severityData := faultsBySeverity
		if len(severityData) == 0 {
			// If we have faults but no severity breakdown, show as unknown
			if deviceStats.ActiveFaults > 0 {
				severityData = map[string]int{
					"critical": deviceStats.CriticalFaults,
					"major":    0,
					"minor":    deviceStats.ActiveFaults - deviceStats.CriticalFaults,
					"warning":  0,
					"info":     0,
				}
			} else {
				severityData = map[string]int{
					"critical": 1,
					"major":    1,
					"minor":    1,
					"warning":  0,
					"info":     0,
				}
			}
		}

		// Log data for debugging
		logger.WebLog.Infof("Overview data: TotalDevices=%d, OnlineDevices=%d, ActiveFaults=%d",
			deviceStats.TotalDevices, deviceStats.OnlineDevices, deviceStats.ActiveFaults)
		logger.WebLog.Infof("Vendor data: %+v", vendorData)
		logger.WebLog.Infof("Severity data: %+v", severityData)

		// Prepare data for template
		data := templates.OverviewData{
			BasePageData: templates.BasePageData{
				Title: "Overview",
				Theme: theme,
			},
			Stats: templates.OverviewStats{
				TotalDevices:   deviceStats.TotalDevices,
				OnlineDevices:  deviceStats.OnlineDevices,
				OfflineDevices: deviceStats.OfflineDevices,
				ActiveFaults:   deviceStats.ActiveFaults,
				CriticalFaults: deviceStats.CriticalFaults,

				HealthScore: healthScore,
			},
			DevicesByVendor: vendorData,

			FaultSeverity:  severityData,
			RecentFaults:   recentFaults,
			CriticalFaults: criticalFaults,

			SystemStatus: templates.SystemStatus{
				CWMPConnected: genieStatus.CWMPConnected,
				NBIConnected:  genieStatus.NBIConnected,
				FSConnected:   genieStatus.FSConnected,
				LastCheck:     genieStatus.LastCheck,
			},
		}

		// Render the overview page using templ
		component := templates.OverviewPage(data)
		c.Header("Content-Type", "text/html; charset=utf-8")

		if err := component.Render(c.Request.Context(), c.Writer); err != nil {
			logger.WebLog.Errorf("Failed to render overview page: %v", err)
			c.String(http.StatusInternalServerError, "Failed to render page")
			return
		}
	}
}

// RealtimeStats returns real-time statistics for AJAX updates
func RealtimeStats(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats := appContext.GetDeviceStats()
		genieStatus := appContext.GetGenieACSStatus()

		// Calculate current metrics
		activeFaults := appContext.GetActiveFaults()
		criticalCount := 0
		for _, fault := range activeFaults {
			if fault.Severity == models.SeverityCritical {
				criticalCount++
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"devices": gin.H{
				"total":   stats.TotalDevices,
				"online":  stats.OnlineDevices,
				"offline": stats.OfflineDevices,
			},
			"faults": gin.H{
				"active":   len(activeFaults),
				"critical": criticalCount,
			},
			"system": gin.H{
				"cwmp": genieStatus.CWMPConnected,
				"nbi":  genieStatus.NBIConnected,
				"fs":   genieStatus.FSConnected,
			},
			"timestamp": genieStatus.LastCheck,
		})
	}
}
