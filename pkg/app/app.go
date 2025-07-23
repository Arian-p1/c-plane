package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nextranet/gateway/c-plane/config"
	appContext "github.com/nextranet/gateway/c-plane/internal/context"
	"github.com/nextranet/gateway/c-plane/internal/logger"
	"github.com/nextranet/gateway/c-plane/internal/sbi"
	"github.com/nextranet/gateway/c-plane/internal/web"
	"github.com/nextranet/gateway/c-plane/pkg/factory"
	"github.com/nextranet/gateway/c-plane/pkg/service"
)

// App represents the main application
type App struct {
	cfg        *config.Config
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	nbiServer  *http.Server
	uiServer   *http.Server
	appContext *appContext.Context
}

// New creates a new App instance
func New(cfgPath string) (*App, error) {
	// Load configuration
	cfg, err := factory.InitConfigFactory(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger
	if err := logger.InitLogger(&logger.Config{
		Level:           cfg.Logger.Level,
		ReportCaller:    cfg.Logger.ReportCaller,
		File:            cfg.Logger.File,
		RotationCount:   cfg.Logger.RotationCount,
		RotationTime:    cfg.Logger.RotationTime,
		RotationMaxAge:  cfg.Logger.RotationMaxAge,
		RotationMaxSize: cfg.Logger.RotationMaxSize,
	}); err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Create context
	ctx, cancel := context.WithCancel(context.Background())

	// Get application context
	appCtx := appContext.GetContext()
	appCtx.SetConfig(cfg)

	app := &App{
		cfg:        cfg,
		ctx:        ctx,
		cancel:     cancel,
		appContext: appCtx,
	}

	return app, nil
}

// Start starts the application services
func (a *App) Start() error {
	logger.InitLog.Info("Starting Nextranet Gateway services...")

	// Initialize GenieACS service
	genieService := service.NewGenieACSService(a.cfg.GenieACS, a.appContext)
	if err := genieService.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize GenieACS service: %w", err)
	}

	// Start GenieACS monitoring
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		genieService.StartMonitoring(a.ctx)
	}()

	// Start NBI server
	if a.cfg.NBI != nil {
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			if err := a.startNBI(); err != nil {
				logger.InitLog.Errorf("NBI server error: %v", err)
			}
		}()
	}

	// Start UI server
	if a.cfg.UI != nil {
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			if err := a.startUI(); err != nil {
				logger.InitLog.Errorf("UI server error: %v", err)
			}
		}()
	}

	// Wait for all services to be ready
	time.Sleep(2 * time.Second)

	logger.InitLog.Info("All services started successfully")

	// Setup signal handling
	a.setupSignalHandling()

	return nil
}

// startNBI starts the NBI (North Bound Interface) server
func (a *App) startNBI() error {
	logger.InitLog.Info("Starting NBI server...")

	// Set Gin mode
	// if a.cfg.Logger.Level == "debug" {
	// 	gin.SetMode(gin.DebugMode)
	// } else {
	// 	gin.SetMode(gin.ReleaseMode)
	// }

	gin.SetMode(gin.DebugMode)
	// Create Gin router
	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(sbi.LoggerMiddleware())
	router.Use(sbi.CORSMiddleware())

	// Initialize SBI routes
	sbi.InitRouter(router, a.appContext)

	// Determine binding address
	bindAddr := fmt.Sprintf("%s:%d", a.cfg.NBI.BindingIPv4, a.cfg.NBI.Port)
	if a.cfg.NBI.BindingIPv6 != "" {
		bindAddr = fmt.Sprintf("[%s]:%d", a.cfg.NBI.BindingIPv6, a.cfg.NBI.Port)
	}

	// Create HTTP server
	a.nbiServer = &http.Server{
		Addr:         bindAddr,
		Handler:      router,
		ReadTimeout:  a.cfg.NBI.ReadTimeout,
		WriteTimeout: a.cfg.NBI.WriteTimeout,
	}

	logger.InitLog.Infof("NBI server listening on %s", bindAddr)

	// Start server
	if a.cfg.NBI.Scheme == "https" && a.cfg.NBI.TLS != nil {
		return a.nbiServer.ListenAndServeTLS(a.cfg.NBI.TLS.Cert, a.cfg.NBI.TLS.Key)
	}
	return a.nbiServer.ListenAndServe()
}

// startUI starts the UI server
func (a *App) startUI() error {
	logger.InitLog.Info("Starting UI server...")

	// Set Gin mode
	if a.cfg.Logger.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create Gin router
	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(web.LoggerMiddleware())

	// Initialize web routes
	web.InitRouter(router, a.appContext)

	// Determine binding address
	bindAddr := fmt.Sprintf("%s:%d", a.cfg.UI.BindingIPv4, a.cfg.UI.Port)
	if a.cfg.UI.BindingIPv6 != "" {
		bindAddr = fmt.Sprintf("[%s]:%d", a.cfg.UI.BindingIPv6, a.cfg.UI.Port)
	}

	// Create HTTP server
	a.uiServer = &http.Server{
		Addr:         bindAddr,
		Handler:      router,
		ReadTimeout:  a.cfg.UI.ReadTimeout,
		WriteTimeout: a.cfg.UI.WriteTimeout,
	}

	logger.InitLog.Infof("UI server listening on %s", bindAddr)

	// Start server
	if a.cfg.UI.Scheme == "https" && a.cfg.UI.TLS != nil {
		return a.uiServer.ListenAndServeTLS(a.cfg.UI.TLS.Cert, a.cfg.UI.TLS.Key)
	}
	return a.uiServer.ListenAndServe()
}

// Stop gracefully stops the application
func (a *App) Stop() {
	logger.InitLog.Info("Stopping application...")

	// Cancel context to stop background tasks
	a.cancel()

	// Shutdown HTTP servers with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown NBI server
	if a.nbiServer != nil {
		logger.InitLog.Info("Shutting down NBI server...")
		if err := a.nbiServer.Shutdown(shutdownCtx); err != nil {
			logger.InitLog.Errorf("NBI server shutdown error: %v", err)
		}
	}

	// Shutdown UI server
	if a.uiServer != nil {
		logger.InitLog.Info("Shutting down UI server...")
		if err := a.uiServer.Shutdown(shutdownCtx); err != nil {
			logger.InitLog.Errorf("UI server shutdown error: %v", err)
		}
	}

	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.InitLog.Info("All services stopped gracefully")
	case <-time.After(35 * time.Second):
		logger.InitLog.Warn("Timeout waiting for services to stop")
	}
}

// setupSignalHandling sets up signal handling for graceful shutdown
func (a *App) setupSignalHandling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	logger.InitLog.Infof("Received signal: %v", sig)

	a.Stop()
}

// Wait blocks until the application is stopped
func (a *App) Wait() {
	a.wg.Wait()
}

// GetConfig returns the application configuration
func (a *App) GetConfig() *config.Config {
	return a.cfg
}

// GetContext returns the application context
func (a *App) GetContext() *appContext.Context {
	return a.appContext
}
