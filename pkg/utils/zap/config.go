package zap

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"strings"
)

var DefaultConfig = zap.Config{
	Encoding:         "console",
	Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
	OutputPaths:      []string{"stderr"},
	ErrorOutputPaths: []string{"stderr"},
	EncoderConfig: zapcore.EncoderConfig{
		MessageKey: "message",

		LevelKey:    "level",
		EncodeLevel: zapcore.CapitalLevelEncoder,

		TimeKey:    "time",
		EncodeTime: zapcore.ISO8601TimeEncoder,

		NameKey:    "name",
		EncodeName: zapcore.FullNameEncoder,
	},
}

func translateLogLevel(s string) zapcore.Level {
	switch strings.ToLower(s) {
	case "debug":
		return zap.DebugLevel
	case "info":
		return zap.InfoLevel
	case "warning":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	default:
		return zap.InfoLevel
	}
}

func DefaultConfigWithLogLevel(l string) zap.Config {
	cfg := DefaultConfig
	cfg.Level = zap.NewAtomicLevelAt(translateLogLevel(l))
	return cfg
}
