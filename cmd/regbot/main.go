// Command regbot is the CLI entry point for RegBot, an educational tool that
// automates email-based account registration for Instagram and TikTok on an
// Android device using Appium and the on-device Gmail app for OTP retrieval.
//
// This binary is a thin adapter over the internal packages: it parses flags,
// loads configuration, builds the logger, and delegates to internal/core.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/ErrLogic/regbot/internal/otp"
)

// Exit codes per PRD FR-10.
const (
	exitSuccess     = 0
	exitConfigError = 1
	exitAutomation  = 2
	exitOTPNotFound = 3
	exitInterrupted = 130
)

func main() {
	os.Exit(mapExit(newRootCmd().Execute()))
}

// mapExit translates a run error into the appropriate process exit code.
func mapExit(err error) int {
	switch {
	case err == nil:
		return exitSuccess
	case isUsageError(err):
		return exitConfigError
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return exitInterrupted
	case errors.Is(err, otp.ErrCodeNotFound):
		return exitOTPNotFound
	default:
		fmt.Fprintln(os.Stderr, err)
		return exitAutomation
	}
}

// isUsageError reports whether err is (or wraps) a usageError, printing it.
func isUsageError(err error) bool {
	var ue usageError
	if errors.As(err, &ue) {
		fmt.Fprintln(os.Stderr, err)
		return true
	}
	return false
}
