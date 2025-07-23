package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nextranet/gateway/c-plane/internal/logger"
	"github.com/nextranet/gateway/c-plane/pkg/app"
)

var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	// Command line flags
	var (
		configPath  = flag.String("config", "", "Path to configuration file")
		showVersion = flag.Bool("version", false, "Show version information")
		showHelp    = flag.Bool("help", false, "Show help information")
		disableNBI  = flag.Bool("no-nbi", false, "Disable NBI service")
		disableUI   = flag.Bool("no-ui", false, "Disable UI service")
		debug       = flag.Bool("debug", false, "Enable debug mode (sets logger level to debug)")
	)

	flag.Parse()

	// Show help
	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	// Show version
	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	// Print banner
	printBanner()

	// Create application instance
	application, err := app.New(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize application: %v\n", err)
		os.Exit(1)
	}

	// Modify config based on flags
	if *disableNBI {
		cfg := application.GetConfig()
		cfg.NBI = nil
		logger.InitLog.Info("NBI service disabled")
	}

	if *disableUI {
		cfg := application.GetConfig()
		cfg.UI = nil
		logger.InitLog.Info("UI service disabled")
	}

	if *debug {
		cfg := application.GetConfig()
		cfg.Logger.Level = "debug"
		logger.InitLog.Info("Debug mode enabled")
	}

	// Start the application
	if err := application.Start(); err != nil {
		logger.InitLog.Fatalf("Failed to start application: %v", err)
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	sig := <-sigChan
	logger.InitLog.Infof("Received signal: %v, shutting down...", sig)

	// Graceful shutdown
	application.Stop()

	logger.InitLog.Info("Application stopped successfully")
}

func printBanner() {
	banner := `
 _   _           _                        _
| \ | | _____  _| |_ _ __ __ _ _ __   ___| |_
|  \| |/ _ \ \/ / __| '__/ _` + "`" + ` | '_ \ / _ \ __|
| |\  |  __/>  <| |_| | | (_| | | | |  __/ |_
|_| \_|\___/_/\_\__|_|  \__,_|_| |_|\___|\__|

`
	fmt.Println(banner)
	fmt.Printf("Version: %s | Build Time: %s | Git Commit: %s\n\n", version, buildTime, gitCommit)
}

func printVersion() {
	fmt.Printf("Nextranet Gateway\n")
	fmt.Printf("Version:     %s\n", version)
	fmt.Printf("Build Time:  %s\n", buildTime)
	fmt.Printf("Git Commit:  %s\n", gitCommit)
	fmt.Printf("Go Version:  %s\n", getGoVersion())
	fmt.Printf("OS/Arch:     %s/%s\n", getOS(), getArch())
}

func printHelp() {
	fmt.Println("Nextranet Gateway - Control Plane for GenieACS")
	fmt.Println()
	fmt.Println("This service provides both NBI (API) and UI interfaces for managing")
	fmt.Println("TR-069/CWMP devices through GenieACS.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  gateway [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -config string")
	fmt.Println("        Path to configuration file (default: searches for config.yaml)")
	fmt.Println("  -no-nbi")
	fmt.Println("        Disable the NBI (API) service")
	fmt.Println("  -no-ui")
	fmt.Println("        Disable the UI service")
	fmt.Println("  -debug")
	fmt.Println("        Enable debug mode (sets logger level to debug)")
	fmt.Println("  -version")
	fmt.Println("        Show version information")
	fmt.Println("  -help")
	fmt.Println("        Show this help message")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  GATEWAY_CONFIG_PATH")
	fmt.Println("        Alternative way to specify configuration file path")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Start both NBI and UI services")
	fmt.Println("  gateway")
	fmt.Println()
	fmt.Println("  # Start with specific configuration")
	fmt.Println("  gateway -config /etc/gateway/config.yaml")
	fmt.Println()
	fmt.Println("  # Start only NBI service")
	fmt.Println("  gateway -no-ui")
	fmt.Println()
	fmt.Println("  # Start only UI service")
	fmt.Println("  gateway -no-nbi")
	fmt.Println()
	fmt.Println("  # Start with debug mode enabled")
	fmt.Println("  gateway -debug")
	fmt.Println()
	fmt.Println("Default Service URLs:")
	fmt.Println("  NBI API:  http://localhost:8080")
	fmt.Println("  Web UI:   http://localhost:8081")
	fmt.Println()
	fmt.Println("Configuration File Locations (searched in order):")
	fmt.Println("  1. Command line -config flag")
	fmt.Println("  2. GATEWAY_CONFIG_PATH environment variable")
	fmt.Println("  3. ./config.yaml")
	fmt.Println("  4. ./config.yml")
	fmt.Println("  5. ./conf/config.yaml")
	fmt.Println("  6. ./conf/config.yml")
	fmt.Println("  7. /etc/gateway/config.yaml")
	fmt.Println("  8. /etc/gateway/config.yml")
}

func getGoVersion() string {
	// This would be populated during build
	return "go1.21"
}

func getOS() string {
	// This would be populated during build
	return "linux"
}

func getArch() string {
	// This would be populated during build
	return "amd64"
}
