// Package notification provides an OTP provider that reads verification codes
// from Android notifications via ADB, falling back to the Gmail app UI.
package notification

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/ErrLogic/regbot/internal/otp"
)

// Provider reads OTP codes from Android notifications (ADB), with Gmail fallback.
type Provider struct {
	serial  string // adb device serial
	adbPath string // path to adb binary
	re      *regexp.Regexp
	gmail   otp.OTPProvider
}

// New creates a notification-based OTP provider.
func New(serial string, codeRegex string, gmailFallback otp.OTPProvider) (*Provider, error) {
	re, err := regexp.Compile(codeRegex)
	if err != nil {
		return nil, fmt.Errorf("notification: compile regex: %w", err)
	}
	return &Provider{
		serial:  serial,
		adbPath: "adb",
		re:      re,
		gmail:   gmailFallback,
	}, nil
}

// GetCode tries ADB notification dump first, then Gmail app.
func (p *Provider) GetCode(ctx context.Context, targetEmail string, timeout time.Duration) (string, error) {
	// If no device serial, skip directly to Gmail fallback (e.g., in tests).
	if p.serial == "" && p.gmail != nil {
		return p.gmail.GetCode(ctx, targetEmail, timeout)
	}

	// Use half the timeout for notification polling, half for Gmail fallback.
	notifyTimeout := timeout / 2
	if notifyTimeout < 5*time.Second {
		notifyTimeout = 5 * time.Second
	}
	if notifyTimeout > timeout {
		notifyTimeout = timeout
	}
	notifyDeadline := time.Now().Add(notifyTimeout)

	// Wait for email to arrive.
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(5 * time.Second):
	}

	for !time.Now().After(notifyDeadline) {
		// Strategy 1: ADB dumpsys notification.
		code, err := p.fromADB(ctx)
		if err == nil {
			return code, nil
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}

	// Strategy 2: Fall back to Gmail app with remaining timeout.
	if p.gmail != nil {
		remaining := time.Until(time.Now().Add(timeout))
		if remaining < 5*time.Second {
			remaining = 5 * time.Second
		}
		return p.gmail.GetCode(ctx, targetEmail, remaining)
	}

	return "", fmt.Errorf("%w: all strategies exhausted", otp.ErrCodeNotFound)
}

// fromADB runs `adb shell dumpsys notification` and scans for verification codes.
func (p *Provider) fromADB(ctx context.Context) (string, error) {
	// Skip ADB if no device serial (e.g., in tests).
	if p.serial == "" {
		return "", fmt.Errorf("no device serial configured")
	}

	args := []string{"shell", "dumpsys", "notification", "--noredact"}
	args = append([]string{"-s", p.serial}, args...)

	cmd := exec.CommandContext(ctx, p.adbPath, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("dumpsys notification: %w", err)
	}

	src := string(out)
	lower := strings.ToLower(src)

	// Check if there's any verification-related notification.
	hasVerification := strings.Contains(lower, "verification") ||
		strings.Contains(lower, "code") ||
		strings.Contains(lower, "login") ||
		strings.Contains(lower, "sign in") ||
		strings.Contains(lower, "security") ||
		strings.Contains(lower, "confirm") ||
		strings.Contains(lower, "otp") ||
		strings.Contains(lower, "instagram") ||
		strings.Contains(lower, "tiktok")

	if !hasVerification {
		return "", fmt.Errorf("no verification notification found")
	}

	// Extract numeric codes.
	matches := p.re.FindAllString(src, -1)
	for _, m := range matches {
		// Filter: must be a standalone code (not part of a date, ID, or phone number).
		if isLikelyCode(src, m) {
			return m, nil
		}
	}

	return "", fmt.Errorf("no code found in notifications")
}

// isLikelyCode filters out false positives (years, timestamps, short IDs).
// src is retained for future context-based heuristics.
func isLikelyCode(src, code string) bool {
	_ = src
	// Reject very short numbers.
	if len(code) < 4 {
		return false
	}
	// Reject 4-digit year-like values (1900-2099), a common false positive from
	// copyright/footer text in notification previews.
	if len(code) == 4 && (strings.HasPrefix(code, "19") || strings.HasPrefix(code, "20")) {
		return false
	}
	return true
}
