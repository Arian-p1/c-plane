package sbi

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nextranet/gateway/c-plane/internal/context"
	"github.com/nextranet/gateway/c-plane/internal/logger"
	"github.com/nextranet/gateway/c-plane/internal/sbi/producer"
)

// InitRouter initializes the SBI router with all routes
func InitRouter(router *gin.Engine, appContext *context.Context) {
	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Health check
		v1.GET("/health", healthCheck(appContext))

		// Device routes
		devices := v1.Group("/devices")
		{
			devices.GET("", producer.GetDevices(appContext))
			devices.GET("/:deviceId", producer.GetDevice(appContext))
			devices.POST("/:deviceId/refresh", producer.RefreshDevice(appContext))
			devices.GET("/:deviceId/parameters", producer.GetDeviceParameters(appContext))
			devices.PUT("/:deviceId/parameters", producer.SetDeviceParameters(appContext))
			devices.GET("/:deviceId/tasks", producer.GetDeviceTasks(appContext))
			devices.POST("/:deviceId/tasks", producer.CreateDeviceTask(appContext))
			devices.GET("/:deviceId/faults", producer.GetDeviceFaults(appContext))
			devices.POST("/:deviceId/reboot", producer.RebootDevice(appContext))
			devices.POST("/:deviceId/factory-reset", producer.FactoryResetDevice(appContext))
			devices.PUT("/:deviceId/tags", producer.UpdateDeviceTags(appContext))
		}

		// Fault routes
		faults := v1.Group("/faults")
		{
			faults.GET("", producer.GetFaults(appContext))
			faults.GET("/:faultId", producer.GetFault(appContext))
			faults.PUT("/:faultId/acknowledge", producer.AcknowledgeFault(appContext))
			faults.PUT("/:faultId/resolve", producer.ResolveFault(appContext))
			faults.DELETE("/:faultId", producer.DeleteFault(appContext))
		}

		// Task routes
		tasks := v1.Group("/tasks")
		{
			tasks.GET("", producer.GetTasks(appContext))
			tasks.GET("/:taskId", producer.GetTask(appContext))
			tasks.DELETE("/:taskId", producer.DeleteTask(appContext))
			tasks.POST("/:taskId/retry", producer.RetryTask(appContext))
		}

		// Statistics routes
		stats := v1.Group("/stats")
		{
			stats.GET("/overview", producer.GetOverviewStats(appContext))
			stats.GET("/devices", producer.GetDeviceStats(appContext))
			stats.GET("/faults", producer.GetFaultStats(appContext))

		}

		// System routes
		system := v1.Group("/system")
		{
			system.GET("/status", producer.GetSystemStatus(appContext))
			system.GET("/config", producer.GetSystemConfig(appContext))
			system.PUT("/config", producer.UpdateSystemConfig(appContext))
		}

		// Bulk operations
		bulk := v1.Group("/bulk")
		{
			bulk.POST("/devices/refresh", producer.BulkRefreshDevices(appContext))
			bulk.POST("/devices/reboot", producer.BulkRebootDevices(appContext))
			bulk.PUT("/devices/parameters", producer.BulkSetParameters(appContext))
			bulk.PUT("/devices/tags", producer.BulkUpdateTags(appContext))
		}

		// Export routes
		export := v1.Group("/export")
		{
			export.GET("/devices", producer.ExportDevices(appContext))
			export.GET("/faults", producer.ExportFaults(appContext))
		}
	}

	// WebSocket endpoint for real-time updates
	router.GET("/ws", producer.WebSocketHandler(appContext))
}

// LoggerMiddleware creates a logger middleware for Gin
func LoggerMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Custom log format
		var statusColor, methodColor, resetColor string
		if param.IsOutputColor() {
			statusColor = param.StatusCodeColor()
			methodColor = param.MethodColor()
			resetColor = param.ResetColor()
		}

		if param.Latency > time.Minute {
			param.Latency = param.Latency - param.Latency%time.Second
		}

		logger.HTTPLog.Infof("%s %3d %s| %13v | %15s |%s %-7s %s %#v",
			statusColor, param.StatusCode, resetColor,
			param.Latency,
			param.ClientIP,
			methodColor, param.Method, resetColor,
			param.Path,
		)

		return ""
	})
}

// CORSMiddleware creates a CORS middleware
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Allow all origins in development, restrict in production
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}

		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// AuthMiddleware creates an authentication middleware
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for API key or Bearer token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Missing authorization header",
			})
			c.Abort()
			return
		}

		// Check Bearer token
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			// TODO: Validate token
			if token == "" {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "Invalid token",
				})
				c.Abort()
				return
			}
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization format",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
	// Simple in-memory rate limiter
	// In production, use Redis or similar
	type client struct {
		count    int
		lastSeen time.Time
	}

	clients := make(map[string]*client)

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()

		if cl, exists := clients[clientIP]; exists {
			if now.Sub(cl.lastSeen) > time.Minute {
				cl.count = 0
			}
			cl.count++
			cl.lastSeen = now

			if cl.count > requestsPerMinute {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": "Rate limit exceeded",
				})
				c.Abort()
				return
			}
		} else {
			clients[clientIP] = &client{
				count:    1,
				lastSeen: now,
			}
		}

		c.Next()
	}
}

// ErrorHandlerMiddleware creates an error handler middleware
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Handle any errors that occurred during request processing
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			logger.SBILog.Errorf("Request error: %v", err)

			// Return appropriate error response
			status := c.Writer.Status()
			if status == http.StatusOK {
				status = http.StatusInternalServerError
			}

			c.JSON(status, gin.H{
				"error": err.Error(),
			})
		}
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		c.Set("requestID", requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)

		c.Next()
	}
}

// healthCheck returns a health check handler
func healthCheck(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := appContext.GetGenieACSStatus()

		healthy := status.CWMPConnected && status.NBIConnected

		response := gin.H{
			"status":    "ok",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"services": gin.H{
				"cwmp": status.CWMPConnected,
				"nbi":  status.NBIConnected,
				"fs":   status.FSConnected,
			},
		}

		if !healthy {
			response["status"] = "degraded"
		}

		statusCode := http.StatusOK
		if !healthy {
			statusCode = http.StatusServiceUnavailable
		}

		c.JSON(statusCode, response)
	}
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	// Simple implementation - in production use UUID
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Intn(1000000))
}
