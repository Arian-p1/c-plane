package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	log         *logrus.Logger
	AppLog      *logrus.Entry
	InitLog     *logrus.Entry
	ConfigLog   *logrus.Entry
	ContextLog  *logrus.Entry
	ConsumerLog *logrus.Entry
	ProducerLog *logrus.Entry
	GinLog      *logrus.Entry
	HTTPLog     *logrus.Entry
	SBILog      *logrus.Entry
	WebLog      *logrus.Entry
	GenieACSLog *logrus.Entry
)

func init() {
	log = logrus.New()
	log.SetReportCaller(false)

	AppLog = log.WithFields(logrus.Fields{"component": "APP"})
	InitLog = log.WithFields(logrus.Fields{"component": "INIT"})
	ConfigLog = log.WithFields(logrus.Fields{"component": "CONFIG"})
	ContextLog = log.WithFields(logrus.Fields{"component": "CONTEXT"})
	ConsumerLog = log.WithFields(logrus.Fields{"component": "CONSUMER"})
	ProducerLog = log.WithFields(logrus.Fields{"component": "PRODUCER"})
	GinLog = log.WithFields(logrus.Fields{"component": "GIN"})
	HTTPLog = log.WithFields(logrus.Fields{"component": "HTTP"})
	SBILog = log.WithFields(logrus.Fields{"component": "SBI"})
	WebLog = log.WithFields(logrus.Fields{"component": "WEB"})
	GenieACSLog = log.WithFields(logrus.Fields{"component": "GENIEACS"})
}

type Config struct {
	Level           string
	ReportCaller    bool
	File            string
	RotationCount   int
	RotationTime    string
	RotationMaxAge  int
	RotationMaxSize int
}

func SetLogLevel(levelStr string) {
	level, err := logrus.ParseLevel(levelStr)
	if err != nil {
		log.Warnf("Invalid log level [%s], using default level [info]", levelStr)
		level = logrus.InfoLevel
	}
	log.SetLevel(level)
}

func SetReportCaller(enable bool) {
	log.SetReportCaller(enable)
	if enable {
		log.SetFormatter(&logrus.TextFormatter{
			ForceColors:     true,
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339Nano,
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				s := strings.Split(f.Function, ".")
				funcname := s[len(s)-1]
				filename := filepath.Base(f.File)
				return funcname, fmt.Sprintf("%s:%d", filename, f.Line)
			},
		})
	}
}

func InitLogger(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("logger config is nil")
	}

	// Set log level
	SetLogLevel(cfg.Level)

	// Set report caller
	SetReportCaller(cfg.ReportCaller)

	// Set formatter
	if cfg.File == "" {
		// Console output with colors
		log.SetFormatter(&logrus.TextFormatter{
			ForceColors:     true,
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
		log.SetOutput(os.Stdout)
	} else {
		// File output without colors
		log.SetFormatter(&logrus.TextFormatter{
			ForceColors:     false,
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})

		// Create log directory if it doesn't exist
		logDir := filepath.Dir(cfg.File)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		// Setup log rotation
		rotateLogger := &lumberjack.Logger{
			Filename:   cfg.File,
			MaxSize:    cfg.RotationMaxSize, // megabytes
			MaxBackups: cfg.RotationCount,
			MaxAge:     cfg.RotationMaxAge, // days
			Compress:   true,
		}

		log.SetOutput(rotateLogger)
	}

	InitLog.Infof("Logger initialized with level: %s", cfg.Level)
	return nil
}

// GetLogger returns the base logger instance
func GetLogger() *logrus.Logger {
	return log
}

// WithFields creates a new logger entry with the given fields
func WithFields(fields logrus.Fields) *logrus.Entry {
	return log.WithFields(fields)
}

// Helper functions for different log levels
func Debugf(format string, args ...interface{}) {
	log.Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	log.Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	log.Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	log.Errorf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	log.Fatalf(format, args...)
}

func Panicf(format string, args ...interface{}) {
	log.Panicf(format, args...)
}

// GinLogger returns a gin-compatible logger middleware
func GinLogger() func(c interface{}) {
	return func(c interface{}) {
		// This is a placeholder - actual implementation would depend on gin context
		GinLog.Info("Request processed")
	}
}

// GetLoggerWithField returns a logger with a specific field
func GetLoggerWithField(field, value string) *logrus.Entry {
	return log.WithField(field, value)
}

// GetLoggerWithFields returns a logger with multiple fields
func GetLoggerWithFields(fields map[string]interface{}) *logrus.Entry {
	return log.WithFields(fields)
}
