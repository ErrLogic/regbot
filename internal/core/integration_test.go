//go:build integration

// Package core integration smoke test.
//
// This test requires a real environment and is skipped unless built with the
// `integration` tag:
//
//	go test -tags=integration ./internal/core/
//
// Prerequisites (manual):
//   - An Appium server with the UiAutomator2 driver running at the configured
//     server_url.
//   - A connected Android device (or emulator) with the target app and Gmail
//     installed, and Gmail signed into an account that receives the code.
//   - A valid config.yaml at the repo root (or REGBOT_ env overrides).
//
// It performs a --dry-run Instagram registration and asserts it reaches the end
// of the flow without submitting.
package core

import (
	"context"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/ErrLogic/regbot/internal/adb"
	"github.com/ErrLogic/regbot/internal/config"
	"github.com/ErrLogic/regbot/internal/flows"
)

func TestIntegrationInstagramDryRun(t *testing.T) {
	cfgPath := os.Getenv("REGBOT_CONFIG")
	if cfgPath == "" {
		cfgPath = "../../config.yaml"
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate config: %v", err)
	}

	logger, _ := zap.NewDevelopment()
	svc := NewService(logger, adb.New(adb.WithSerial(cfg.Device.UDID)))

	email, err := ResolveEmail(cfg.Email)
	if err != nil {
		t.Fatalf("resolve email: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	acct, err := svc.Register(ctx, flows.PlatformInstagram, email, cfg, true)
	if err != nil {
		t.Fatalf("dry-run registration: %v", err)
	}
	if acct.Status != "dry-run" {
		t.Errorf("status = %q, want dry-run", acct.Status)
	}
}
