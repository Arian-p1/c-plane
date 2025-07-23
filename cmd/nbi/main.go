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
 _   _           _                        _     _   _ ____ ___
| \ | | _____  _| |_ _ __ __ _ _ __   ___| |_  | \ | | __ |_ _|
|  \| |/ _ \ \/ / __| '__/ _` + "`" + ` | '_ \ / _ \ __| |  \| |  _ \| |
| |\  |  __/>  <| |_| | | (_| | | | |  __/ |_  | |\  | |_) | |
|_| \_|\___/_/\_\__|_|  \__,_|_| |_|\___|\__| |_| \_|____/___|

`
	fmt.Println(banner)
	fmt.Printf("Version: %s | Build Time: %s | Git Commit: %s\n\n", version, buildTime, gitCommit)
}

func printVersion() {
	fmt.Printf("GenieACS Gateway NBI Service\n")
	fmt.Printf("Version:     %s\n", version)
	fmt.Printf("Build Time:  %s\n", buildTime)
	fmt.Printf("Git Commit:  %s\n", gitCommit)
	fmt.Printf("Go Version:  %s\n", getGoVersion())
	fmt.Printf("OS/Arch:     %s/%s\n", getOS(), getArch())
}

func printHelp() {
	fmt.Println("GenieACS Gateway NBI Service")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  nbi [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -config string")
	fmt.Println("        Path to configuration file (default: searches for config.yaml in standard locations)")
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
	fmt.Println("  # Start with default configuration")
	fmt.Println("  nbi")
	fmt.Println()
	fmt.Println("  # Start with specific configuration file")
	fmt.Println("  nbi -config /etc/gateway/config.yaml")
	fmt.Println()
	fmt.Println("  # Start with configuration from environment variable")
	fmt.Println("  GATEWAY_CONFIG_PATH=/etc/gateway/config.yaml nbi")
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
