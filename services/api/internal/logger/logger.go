package logger

import (
	"time"

	"github.com/trybyteful/interview-takehome/services/api/internal/apiconfig"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Basic returns a JSON logger and the level that controls it.
func Basic() (*zap.Logger, zap.AtomicLevel, error) {
	lvl := zap.NewAtomicLevel()
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapUTCTimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	config := zap.Config{
		Level:       lvl,
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         "json",
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := config.Build()

	return logger, lvl, err
}

func zapUTCTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.UTC().Format(time.RFC3339Nano))
}

// ConfigureLogger returns logger with the fields that name this system on every line.
func ConfigureLogger(logger *zap.Logger, config apiconfig.APIConfig) *zap.Logger {
	return logger.With(zap.Any("system", map[string]string{
		"name":    "api",
		"version": config.Version,
	}))
}
