package web

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nextranet/gateway/c-plane/internal/context"
	"github.com/nextranet/gateway/c-plane/internal/logger"
	"github.com/nextranet/gateway/c-plane/internal/web/handlers"
)

// InitRouter initializes the web UI router with all routes
func InitRouter(router *gin.Engine, appContext *context.Context) {
	// Static files
	router.StaticFS("/static", GetStaticFS())

	// UI routes
	router.GET("/", handlers.RedirectToOverview())
	router.GET("/overview", handlers.Overview(appContext))
	router.GET("/devices", handlers.Devices(appContext))
	router.GET("/devices/:deviceId", handlers.DeviceDetail(appContext))
	router.GET("/files", handlers.Files(appContext))
	router.GET("/faults", handlers.Faults(appContext))

	// AJAX/API routes for UI
	api := router.Group("/api")
	{
		// Real-time data endpoints
		api.GET("/stats/realtime", handlers.RealtimeStats(appContext))
		api.GET("/devices/status", handlers.DeviceStatusUpdate(appContext))
		api.GET("/faults/recent", handlers.RecentFaults(appContext))

		// Device operations
		api.POST("/devices/:deviceId/refresh", handlers.RefreshDevice(appContext))
		api.POST("/devices/:deviceId/reboot", handlers.RebootDevice(appContext))
		api.GET("/devices/:deviceId/config/download", handlers.DownloadConfig(appContext))
		api.POST("/devices/:deviceId/factory-reset", handlers.FactoryReset(appContext))
		api.PUT("/devices/:deviceId/parameters", handlers.UpdateParameter(appContext))
		api.POST("/devices/:deviceId/tags", handlers.AddDeviceTag(appContext))
		api.DELETE("/devices/:deviceId/tags/:tag", handlers.RemoveDeviceTag(appContext))

		// File operations
		api.POST("/files/upload", handlers.UploadFiles(appContext))
		api.GET("/files/:fileId/download", handlers.DownloadFile(appContext))
		api.POST("/files/download-bulk", handlers.DownloadBulkFiles(appContext))
		api.DELETE("/files/:fileId", handlers.DeleteFile(appContext))

		// Fault operations
		api.PUT("/faults/:faultId/acknowledge", handlers.AcknowledgeFault(appContext))
		api.PUT("/faults/:faultId/resolve", handlers.ResolveFault(appContext))

		// Filter presets
		api.GET("/filters/devices", handlers.GetDeviceFilters(appContext))
		api.POST("/filters/devices", handlers.SaveDeviceFilter(appContext))
		api.DELETE("/filters/devices/:filterId", handlers.DeleteDeviceFilter(appContext))
	}

	// WebSocket for real-time updates
	router.GET("/ws", handlers.WebSocketHandler(appContext))

	// Health check for UI
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})
}

// LoggerMiddleware creates a logger middleware for the web UI
func LoggerMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Custom log format for web requests
		var statusColor, methodColor, resetColor string
		if param.IsOutputColor() {
			statusColor = param.StatusCodeColor()
			methodColor = param.MethodColor()
			resetColor = param.ResetColor()
		}

		if param.Latency > time.Minute {
			param.Latency = param.Latency - param.Latency%time.Second
		}

		logger.WebLog.Infof("%s %3d %s| %13v | %15s |%s %-7s %s %#v",
			statusColor, param.StatusCode, resetColor,
			param.Latency,
			param.ClientIP,
			methodColor, param.Method, resetColor,
			param.Path,
		)

		return ""
	})
}

// SecurityMiddleware adds security headers
func SecurityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy for UI
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https:; " +
			"font-src 'self' data:; " +
			"connect-src 'self' ws: wss:; " +
			"frame-ancestors 'none';"

		c.Header("Content-Security-Policy", csp)

		c.Next()
	}
}

// SessionMiddleware handles session management
func SessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get or create session ID
		sessionID := c.GetHeader("X-Session-ID")
		if sessionID == "" {
			sessionID = generateSessionID()
			c.Header("X-Session-ID", sessionID)
		}

		c.Set("sessionID", sessionID)
		c.Next()
	}
}

// ThemeMiddleware handles theme preferences
func ThemeMiddleware(defaultTheme string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get theme from cookie or use default
		theme, err := c.Cookie("theme")
		if err != nil || (theme != "dark" && theme != "light") {
			theme = defaultTheme
		}

		c.Set("theme", theme)
		c.Next()
	}
}

// CacheControlMiddleware sets appropriate cache headers
func CacheControlMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Static assets can be cached
		if c.Request.URL.Path[:7] == "/static" {
			c.Header("Cache-Control", "public, max-age=86400") // 24 hours
		} else {
			// Dynamic content should not be cached
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}

		c.Next()
	}
}

// ErrorHandlerMiddleware handles errors for web UI
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Handle any errors that occurred during request processing
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			logger.WebLog.Errorf("Web UI error: %v", err)

			// Determine if this is an AJAX request
			if c.GetHeader("X-Requested-With") == "XMLHttpRequest" {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
			} else {
				// Render error page
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{
					"error":   err.Error(),
					"title":   "Error",
					"message": "An error occurred while processing your request",
				})
			}
		}
	}
}

// loadTemplates is not needed since we use templ components directly
// Templates are rendered in handlers using templ.Render()

// generateSessionID generates a unique session ID
func generateSessionID() string {
	// Simple implementation - in production use UUID or similar
	return fmt.Sprintf("sess_%d", time.Now().UnixNano())
}

// NotFoundHandler handles 404 errors
func NotFoundHandler(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if this is an API request
		if c.Request.URL.Path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Endpoint not found",
			})
			return
		}

		// Render 404 page for UI requests
		c.HTML(http.StatusNotFound, "404.html", gin.H{
			"title":   "Page Not Found",
			"message": "The page you are looking for does not exist",
		})
	}
}

// RateLimitMiddleware implements rate limiting for web UI
func RateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
	// Simple in-memory rate limiter
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
				c.HTML(http.StatusTooManyRequests, "429.html", gin.H{
					"title":   "Too Many Requests",
					"message": "You have made too many requests. Please try again later.",
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

		// Clean up old entries periodically
		if now.Unix()%60 == 0 {
			for ip, cl := range clients {
				if now.Sub(cl.lastSeen) > 5*time.Minute {
					delete(clients, ip)
				}
			}
		}

		c.Next()
	}
}
