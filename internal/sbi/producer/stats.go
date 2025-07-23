package producer

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nextranet/gateway/c-plane/internal/context"
	"github.com/nextranet/gateway/c-plane/internal/models"
)

// GetOverviewStats returns overall system statistics
func GetOverviewStats(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get device statistics
		deviceStats := appContext.GetDeviceStats()

		// Get fault statistics
		faults := appContext.GetActiveFaults()
		faultStats := calculateFaultStats(faults)

		// Get GenieACS status
		genieStatus := appContext.GetGenieACSStatus()

		// Build overview response
		overview := gin.H{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"devices": gin.H{
				"total":    deviceStats.TotalDevices,
				"online":   deviceStats.OnlineDevices,
				"offline":  deviceStats.OfflineDevices,
				"byVendor": deviceStats.DevicesByVendor,
				"byModel":  deviceStats.DevicesByModel,
			},
			"faults": gin.H{
				"total":    faultStats.Total,
				"active":   faultStats.Active,
				"critical": faultStats.Critical,
				"major":    faultStats.Major,
				"minor":    faultStats.Minor,
				"warning":  faultStats.Warning,
			},

			"system": gin.H{
				"cwmpConnected": genieStatus.CWMPConnected,
				"nbiConnected":  genieStatus.NBIConnected,
				"fsConnected":   genieStatus.FSConnected,
				"lastCheck":     genieStatus.LastCheck,
			},
		}

		c.JSON(http.StatusOK, overview)
	}
}

// GetDeviceStats returns detailed device statistics
func GetDeviceStats(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats := appContext.GetDeviceStats()
		devices := appContext.GetAllDevices()

		// Calculate additional statistics
		vendorModels := make(map[string]map[string]int)
		connectionTypes := map[string]int{
			"connected":    0,
			"disconnected": 0,
			"unknown":      0,
		}

		lastSeenDistribution := map[string]int{
			"last5min":   0,
			"last30min":  0,
			"last1hour":  0,
			"last24hour": 0,
			"older":      0,
		}

		now := time.Now()

		for _, device := range devices {
			// Vendor-Model mapping
			if device.DeviceID.Manufacturer != "" && device.DeviceID.ModelName != "" {
				if vendorModels[device.DeviceID.Manufacturer] == nil {
					vendorModels[device.DeviceID.Manufacturer] = make(map[string]int)
				}
				vendorModels[device.DeviceID.Manufacturer][device.DeviceID.ModelName]++
			}

			// Connection status
			connectionTypes[device.Status.ConnectionStatus]++

			// Last seen distribution
			if !device.Status.LastSeen.IsZero() {
				timeSince := now.Sub(device.Status.LastSeen)
				switch {
				case timeSince <= 5*time.Minute:
					lastSeenDistribution["last5min"]++
				case timeSince <= 30*time.Minute:
					lastSeenDistribution["last30min"]++
				case timeSince <= time.Hour:
					lastSeenDistribution["last1hour"]++
				case timeSince <= 24*time.Hour:
					lastSeenDistribution["last24hour"]++
				default:
					lastSeenDistribution["older"]++
				}
			}
		}

		// Find top vendors
		topVendors := getTopEntries(stats.DevicesByVendor, 5)

		// Find top models
		topModels := getTopEntries(stats.DevicesByModel, 5)

		response := gin.H{
			"summary": gin.H{
				"total":   stats.TotalDevices,
				"online":  stats.OnlineDevices,
				"offline": stats.OfflineDevices,
			},
			"byVendor":             stats.DevicesByVendor,
			"byModel":              stats.DevicesByModel,
			"vendorModels":         vendorModels,
			"connectionStatus":     connectionTypes,
			"lastSeenDistribution": lastSeenDistribution,
			"topVendors":           topVendors,
			"topModels":            topModels,
			"timestamp":            time.Now().UTC().Format(time.RFC3339),
		}

		c.JSON(http.StatusOK, response)
	}
}

// Helper structures and functions

type faultStatsResult struct {
	Total    int
	Active   int
	Critical int
	Major    int
	Minor    int
	Warning  int
	Info     int
}

func calculateFaultStats(faults []*models.Fault) faultStatsResult {
	result := faultStatsResult{
		Total: len(faults),
	}

	for _, fault := range faults {
		if fault.Status == models.FaultStatusActive {
			result.Active++
		}

		switch fault.Severity {
		case models.SeverityCritical:
			result.Critical++
		case models.SeverityMajor:
			result.Major++
		case models.SeverityMinor:
			result.Minor++
		case models.SeverityWarning:
			result.Warning++
		case models.SeverityInfo:
			result.Info++
		}
	}

	return result
}

func getTopEntries(data map[string]int, limit int) []gin.H {
	type entry struct {
		Key   string
		Value int
	}

	entries := make([]entry, 0, len(data))
	for k, v := range data {
		entries = append(entries, entry{Key: k, Value: v})
	}

	// Sort by value descending
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].Value > entries[i].Value {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Take top N
	if limit > len(entries) {
		limit = len(entries)
	}

	result := make([]gin.H, 0, limit)
	for i := 0; i < limit; i++ {
		result = append(result, gin.H{
			"name":  entries[i].Key,
			"count": entries[i].Value,
		})
	}

	return result
}
