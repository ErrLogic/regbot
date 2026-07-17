package main

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ErrLogic/regbot/internal/otp"
)

func TestMapExit(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"success", nil, exitSuccess},
		{"config error", usageErrorf("bad config: %w", errors.New("x")), exitConfigError},
		{"interrupted", fmt.Errorf("run: %w", context.Canceled), exitInterrupted},
		{"deadline", fmt.Errorf("run: %w", context.DeadlineExceeded), exitInterrupted},
		{"otp not found", fmt.Errorf("step: %w", otp.ErrCodeNotFound), exitOTPNotFound},
		{"automation", errors.New("element not found"), exitAutomation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapExit(tt.err); got != tt.want {
				t.Errorf("mapExit(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

func TestRootCommandHasRegister(t *testing.T) {
	root := newRootCmd()
	reg, _, err := root.Find([]string{"register", "instagram"})
	if err != nil {
		t.Fatalf("find register instagram: %v", err)
	}
	if reg.Name() != "instagram" {
		t.Errorf("resolved command = %q", reg.Name())
	}
}
