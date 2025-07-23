package factory

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nextranet/gateway/c-plane/config"
	"github.com/nextranet/gateway/c-plane/internal/logger"
	"gopkg.in/yaml.v3"
)

var (
	defaultConfig *config.Config
	configPath    string
)

// InitConfigFactory initializes the configuration factory
func InitConfigFactory(cfgPath string) (*config.Config, error) {
	if cfgPath == "" {
		cfgPath = getDefaultConfigPath()
	}

	configPath = cfgPath
	cfg, err := loadConfig(cfgPath)
	if err != nil {
		return nil, err
	}

	// Apply defaults
	applyDefaults(cfg)

	// Validate configuration
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	defaultConfig = cfg
	logger.InitLog.Infof("Configuration loaded from: %s", cfgPath)
	return cfg, nil
}

// GetConfig returns the default configuration
func GetConfig() *config.Config {
	return defaultConfig
}

// GetConfigPath returns the path to the configuration file
func GetConfigPath() string {
	return configPath
}

// loadConfig loads configuration from a YAML file
func loadConfig(path string) (*config.Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables
	content := os.ExpandEnv(string(data))

	cfg := &config.Config{}
	if err := yaml.Unmarshal([]byte(content), cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// applyDefaults applies default values to the configuration
func applyDefaults(cfg *config.Config) {
	// Info defaults
	if cfg.Info == nil {
		cfg.Info = &config.Info{}
	}
	if cfg.Info.Version == "" {
		cfg.Info.Version = "1.0.0"
	}
	if cfg.Info.Description == "" {
		cfg.Info.Description = "Nextranet Gateway"
	}

	// Logger defaults
	if cfg.Logger == nil {
		cfg.Logger = &config.Logger{}
	}
	if cfg.Logger.Level == "" {
		cfg.Logger.Level = "info"
	}
	if cfg.Logger.RotationCount == 0 {
		cfg.Logger.RotationCount = 3
	}
	if cfg.Logger.RotationMaxAge == 0 {
		cfg.Logger.RotationMaxAge = 7
	}
	if cfg.Logger.RotationMaxSize == 0 {
		cfg.Logger.RotationMaxSize = 50
	}

	// NBI defaults
	if cfg.NBI != nil {
		if cfg.NBI.Scheme == "" {
			cfg.NBI.Scheme = "http"
		}
		if cfg.NBI.BindingIPv4 == "" {
			cfg.NBI.BindingIPv4 = "0.0.0.0"
		}
		if cfg.NBI.Port == 0 {
			cfg.NBI.Port = 8080
		}
		if cfg.NBI.ReadTimeout == 0 {
			cfg.NBI.ReadTimeout = 30 * time.Second
		}
		if cfg.NBI.WriteTimeout == 0 {
			cfg.NBI.WriteTimeout = 30 * time.Second
		}
	}

	// UI defaults
	if cfg.UI != nil {
		if cfg.UI.Scheme == "" {
			cfg.UI.Scheme = "http"
		}
		if cfg.UI.BindingIPv4 == "" {
			cfg.UI.BindingIPv4 = "0.0.0.0"
		}
		if cfg.UI.Port == 0 {
			cfg.UI.Port = 8081
		}
		if cfg.UI.ReadTimeout == 0 {
			cfg.UI.ReadTimeout = 30 * time.Second
		}
		if cfg.UI.WriteTimeout == 0 {
			cfg.UI.WriteTimeout = 30 * time.Second
		}
		if cfg.UI.Theme == "" {
			cfg.UI.Theme = "dark"
		}
	}

	// Database defaults
	if cfg.Database != nil {
		if cfg.Database.Type == "" {
			cfg.Database.Type = "mongodb"
		}
		if cfg.Database.URL == "" {
			cfg.Database.URL = "mongodb://localhost:27017"
		}
		if cfg.Database.Name == "" {
			cfg.Database.Name = "genieacs"
		}
		if cfg.Database.Pool == nil {
			cfg.Database.Pool = &config.DBPool{}
		}
		if cfg.Database.Pool.MaxIdleConns == 0 {
			cfg.Database.Pool.MaxIdleConns = 10
		}
		if cfg.Database.Pool.MaxOpenConns == 0 {
			cfg.Database.Pool.MaxOpenConns = 100
		}
		if cfg.Database.Pool.ConnMaxLifetime == 0 {
			cfg.Database.Pool.ConnMaxLifetime = 5 * time.Minute
		}
		if cfg.Database.Pool.ConnMaxIdleTime == 0 {
			cfg.Database.Pool.ConnMaxIdleTime = 1 * time.Minute
		}
	}

	// GenieACS defaults
	if cfg.GenieACS != nil {
		if cfg.GenieACS.CWMPURL == "" {
			cfg.GenieACS.CWMPURL = "http://localhost:7547"
		}
		if cfg.GenieACS.NBIURL == "" {
			cfg.GenieACS.NBIURL = "http://localhost:7557"
		}
		if cfg.GenieACS.FSURL == "" {
			cfg.GenieACS.FSURL = "http://localhost:7567"
		}
		if cfg.GenieACS.Timeout == 0 {
			cfg.GenieACS.Timeout = 30 * time.Second
		}
	}
}

// validateConfig validates the configuration
func validateConfig(cfg *config.Config) error {
	// Validate logger
	if cfg.Logger != nil {
		validLevels := []string{"panic", "fatal", "error", "warn", "warning", "info", "debug", "trace"}
		if !contains(validLevels, strings.ToLower(cfg.Logger.Level)) {
			return fmt.Errorf("invalid log level: %s", cfg.Logger.Level)
		}
	}

	// Validate NBI
	if cfg.NBI != nil {
		if cfg.NBI.Port < 1 || cfg.NBI.Port > 65535 {
			return fmt.Errorf("invalid NBI port: %d", cfg.NBI.Port)
		}
		if cfg.NBI.Scheme != "http" && cfg.NBI.Scheme != "https" {
			return fmt.Errorf("invalid NBI scheme: %s", cfg.NBI.Scheme)
		}
		if cfg.NBI.Scheme == "https" && cfg.NBI.TLS == nil {
			return fmt.Errorf("TLS configuration required for HTTPS scheme")
		}
		if cfg.NBI.TLS != nil {
			if cfg.NBI.TLS.Cert == "" || cfg.NBI.TLS.Key == "" {
				return fmt.Errorf("TLS cert and key are required")
			}
			if _, err := os.Stat(cfg.NBI.TLS.Cert); err != nil {
				return fmt.Errorf("TLS cert file not found: %s", cfg.NBI.TLS.Cert)
			}
			if _, err := os.Stat(cfg.NBI.TLS.Key); err != nil {
				return fmt.Errorf("TLS key file not found: %s", cfg.NBI.TLS.Key)
			}
		}
	}

	// Validate UI
	if cfg.UI != nil {
		if cfg.UI.Port < 1 || cfg.UI.Port > 65535 {
			return fmt.Errorf("invalid UI port: %d", cfg.UI.Port)
		}
		if cfg.UI.Scheme != "http" && cfg.UI.Scheme != "https" {
			return fmt.Errorf("invalid UI scheme: %s", cfg.UI.Scheme)
		}
		if cfg.UI.Scheme == "https" && cfg.UI.TLS == nil {
			return fmt.Errorf("TLS configuration required for HTTPS scheme")
		}
		if cfg.UI.TLS != nil {
			if cfg.UI.TLS.Cert == "" || cfg.UI.TLS.Key == "" {
				return fmt.Errorf("TLS cert and key are required")
			}
			if _, err := os.Stat(cfg.UI.TLS.Cert); err != nil {
				return fmt.Errorf("TLS cert file not found: %s", cfg.UI.TLS.Cert)
			}
			if _, err := os.Stat(cfg.UI.TLS.Key); err != nil {
				return fmt.Errorf("TLS key file not found: %s", cfg.UI.TLS.Key)
			}
		}
		validThemes := []string{"dark", "light"}
		if !contains(validThemes, cfg.UI.Theme) {
			return fmt.Errorf("invalid UI theme: %s", cfg.UI.Theme)
		}
	}

	// Validate Database
	if cfg.Database != nil {
		validTypes := []string{"mongodb", "postgresql", "mysql"}
		if !contains(validTypes, cfg.Database.Type) {
			return fmt.Errorf("invalid database type: %s", cfg.Database.Type)
		}
		if cfg.Database.URL == "" {
			return fmt.Errorf("database URL is required")
		}
		if cfg.Database.Name == "" {
			return fmt.Errorf("database name is required")
		}
	}

	// Validate GenieACS
	if cfg.GenieACS != nil {
		if cfg.GenieACS.CWMPURL == "" {
			return fmt.Errorf("GenieACS CWMP URL is required")
		}
		if cfg.GenieACS.NBIURL == "" {
			return fmt.Errorf("GenieACS NBI URL is required")
		}
		if cfg.GenieACS.FSURL == "" {
			return fmt.Errorf("GenieACS FS URL is required")
		}
	}

	// Validate Zone

	return nil
}

// getDefaultConfigPath returns the default configuration file path
func getDefaultConfigPath() string {
	// Check environment variable
	if path := os.Getenv("GATEWAY_CONFIG_PATH"); path != "" {
		return path
	}

	// Check common locations
	commonPaths := []string{
		"./config.yaml",
		"./config.yml",
		"./conf/config.yaml",
		"./conf/config.yml",
		"/etc/gateway/config.yaml",
		"/etc/gateway/config.yml",
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Default to current directory
	return "config.yaml"
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, s := range slice {
		if s == value {
			return true
		}
	}
	return false
}

// ReloadConfig reloads the configuration from file
func ReloadConfig() (*config.Config, error) {
	if configPath == "" {
		return nil, fmt.Errorf("no configuration path set")
	}
	return InitConfigFactory(configPath)
}

// SaveConfig saves the configuration to file
func SaveConfig(cfg *config.Config, path string) error {
	if path == "" {
		path = configPath
	}
	if path == "" {
		return fmt.Errorf("no configuration path specified")
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal configuration to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	logger.InitLog.Infof("Configuration saved to: %s", path)
	return nil
}
