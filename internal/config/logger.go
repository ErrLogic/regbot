package config

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
)

// redactedValue is the placeholder substituted for secret values in logs.
const redactedValue = "[REDACTED]"

// NewLogger builds a structured JSON logger that writes to stderr and, if
// cfg.File is set, additionally to that file. The level is taken from cfg.Level
// (debug|info|warn|error).
func NewLogger(cfg LoggingConfig) (*zap.Logger, error) {
	level, err := zap.ParseAtomicLevel(strings.ToLower(strings.TrimSpace(cfg.Level)))
	if err != nil {
		return nil, fmt.Errorf("logging.level: %w", err)
	}

	zcfg := zap.NewProductionConfig()
	zcfg.Level = level
	zcfg.Encoding = "json"

	outputs := []string{"stderr"}
	if cfg.File != "" {
		outputs = append(outputs, cfg.File)
	}
	zcfg.OutputPaths = outputs
	zcfg.ErrorOutputPaths = []string{"stderr"}

	logger, err := zcfg.Build()
	if err != nil {
		return nil, fmt.Errorf("build logger: %w", err)
	}
	return logger, nil
}

// Redacted returns a log field whose value is masked. Use it to reference a
// secret (such as a generated password) in logs without exposing the value.
func Redacted(key string) zap.Field {
	return zap.String(key, redactedValue)
}
