package logger

import (
	"coffee-and-running/src/config"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// NewLogger creates a new zap logger based on the provided configuration
func NewLogger(cfg *config.LoggerConfig) (*zap.Logger, error) {
	// Parse log level
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %s: %w", cfg.Level, err)
	}

	// Create encoder config
	encoderConfig := getEncoderConfig(cfg)

	// Create encoder based on format
	var encoder zapcore.Encoder
	switch strings.ToLower(cfg.Format) {
	case "json":
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	case "console", "":
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	default:
		return nil, fmt.Errorf("unsupported log format: %s", cfg.Format)
	}

	// Create writer syncer based on output
	writeSyncer, err := getWriteSyncer(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create write syncer: %w", err)
	}

	// Create core
	core := zapcore.NewCore(encoder, writeSyncer, level)

	// Create logger options
	options := getLoggerOptions(cfg)

	// Create logger
	logger := zap.New(core, options...)

	return logger, nil
}

// getEncoderConfig returns the encoder configuration based on the logger config
func getEncoderConfig(cfg *config.LoggerConfig) zapcore.EncoderConfig {
	var encoderConfig zapcore.EncoderConfig

	if cfg.Development {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		// Development defaults
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	} else {
		encoderConfig = zap.NewProductionEncoderConfig()
		// Production defaults
		encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		encoderConfig.EncodeTime = zapcore.EpochTimeEncoder
		encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	}

	// Override time encoding for console format in development
	if cfg.Development && strings.ToLower(cfg.Format) == "console" {
		encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	return encoderConfig
}

// getWriteSyncer returns the appropriate write syncer based on output configuration
func getWriteSyncer(cfg *config.LoggerConfig) (zapcore.WriteSyncer, error) {
	switch strings.ToLower(cfg.Output) {
	case "stdout", "":
		return zapcore.AddSync(os.Stdout), nil
	case "stderr":
		return zapcore.AddSync(os.Stderr), nil
	case "file":
		if cfg.File == "" {
			return nil, fmt.Errorf("file path is required when output is 'file'")
		}

		// Use lumberjack for log rotation
		lumberjackLogger := &lumberjack.Logger{
			Filename:   cfg.File,
			MaxSize:    cfg.MaxSize,    // MB
			MaxBackups: cfg.MaxBackups, // number of backups
			MaxAge:     cfg.MaxAge,     // days
			Compress:   cfg.Compress,   // compress rotated files
		}

		return zapcore.AddSync(lumberjackLogger), nil
	default:
		return nil, fmt.Errorf("unsupported output type: %s", cfg.Output)
	}
}

// getLoggerOptions returns logger options based on configuration
func getLoggerOptions(cfg *config.LoggerConfig) []zap.Option {
	var options []zap.Option

	// Add caller info unless disabled
	if !cfg.DisableCaller {
		options = append(options, zap.AddCaller())
	}

	// Add stack trace unless disabled
	if !cfg.DisableStacktrace {
		options = append(options, zap.AddStacktrace(zapcore.ErrorLevel))
	}

	// Development mode
	if cfg.Development {
		options = append(options, zap.Development())
	}

	return options
}
