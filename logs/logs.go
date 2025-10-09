package logs

import (
	"os"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	globalLogger     *zap.SugaredLogger
	globalLoggerOnce sync.Once
	logLevel         = zap.NewAtomicLevelAt(zapcore.InfoLevel)
)

// LogConfig configures the logger behavior
type LogConfig struct {
	Level      string // debug, info, warn, error
	Format     string // json, console
	OutputPath string // stdout, stderr, or file path
}

// DefaultLogConfig returns a default logging configuration
func DefaultLogConfig() LogConfig {
	return LogConfig{
		Level:      "info",
		Format:     "console",
		OutputPath: "stdout",
	}
}

// InitLogger initializes the global logger with the provided configuration
// This should be called once at application startup
func InitLogger(config LogConfig) error {
	var err error
	globalLoggerOnce.Do(func() {
		// Parse log level
		var level zapcore.Level
		err = level.UnmarshalText([]byte(config.Level))
		if err != nil {
			level = zapcore.InfoLevel
		}
		logLevel.SetLevel(level)

		// Configure encoder
		var encoderConfig zapcore.EncoderConfig
		if config.Format == "json" {
			encoderConfig = zap.NewProductionEncoderConfig()
		} else {
			encoderConfig = zap.NewDevelopmentEncoderConfig()
			encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
			encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		}

		// Configure output
		var output zapcore.WriteSyncer
		switch config.OutputPath {
		case "stdout":
			output = zapcore.AddSync(os.Stdout)
		case "stderr":
			output = zapcore.AddSync(os.Stderr)
		default:
			file, fileErr := os.OpenFile(config.OutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if fileErr != nil {
				err = fileErr
				return
			}
			output = zapcore.AddSync(file)
		}

		// Create encoder
		var encoder zapcore.Encoder
		if config.Format == "json" {
			encoder = zapcore.NewJSONEncoder(encoderConfig)
		} else {
			encoder = zapcore.NewConsoleEncoder(encoderConfig)
		}

		// Create core
		core := zapcore.NewCore(encoder, output, logLevel)

		// Create logger
		logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
		globalLogger = logger.Sugar()
	})
	return err
}

// GetLogger returns the global logger instance
// If the logger hasn't been initialized, it will be initialized with default config
func GetLogger() *zap.SugaredLogger {
	if globalLogger == nil {
		_ = InitLogger(DefaultLogConfig())
	}
	return globalLogger
}

// SetLogLevel changes the global log level dynamically
func SetLogLevel(level string) error {
	var lvl zapcore.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		return err
	}
	logLevel.SetLevel(lvl)
	return nil
}

// GetTraceID generates a unique trace ID for request tracking
func GetTraceID() string {
	uuidWithHyphen := uuid.New()
	return uuidWithHyphen.String()
}

// Sync flushes any buffered log entries
// Should be called before application exit
func Sync() {
	if globalLogger != nil {
		_ = globalLogger.Sync()
	}
}
