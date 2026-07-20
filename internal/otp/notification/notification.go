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

// GetCode reads OTP codes using Gmail app first (reads actual email content,
// always correct), then falls back to ADB notification dump.
func (p *Provider) GetCode(ctx context.Context, targetEmail string, timeout time.Duration) (string, error) {
	// If no device serial, skip directly to Gmail fallback (e.g., in tests).
	if p.serial == "" && p.gmail != nil {
		return p.gmail.GetCode(ctx, targetEmail, timeout)
	}

	// Clear existing notifications to prevent stale codes.
	p.clearNotifications(ctx)

	// Strategy 1 (primary): Gmail app reads the actual email, always correct.
	gmailTimeout := timeout * 3 / 4
	if gmailTimeout < 10*time.Second {
		gmailTimeout = 10 * time.Second
	}
	if p.gmail != nil {
		code, err := p.gmail.GetCode(ctx, targetEmail, gmailTimeout)
		if err == nil {
			return code, nil
		}
	}

	// Strategy 2 (fallback): ADB notification dump for speed.
	remaining := timeout - gmailTimeout
	if remaining < 5*time.Second {
		remaining = 5 * time.Second
	}
	notifyDeadline := time.Now().Add(remaining)

	for !time.Now().After(notifyDeadline) {
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

	return "", fmt.Errorf("%w: all strategies exhausted", otp.ErrCodeNotFound)
}

// clearNotifications dismisses all active notifications so stale OTP codes are
// not picked up from prior attempts. Uses the public `cmd notification` API
// (available since Android 12) and falls back to the legacy service call.
func (p *Provider) clearNotifications(ctx context.Context) {
	args := func(a ...string) []string {
		if p.serial != "" {
			return append([]string{"-s", p.serial}, a...)
		}
		return a
	}
	// Public API (Android 12+): cancel all notifications.
	_ = exec.CommandContext(ctx, p.adbPath, args("shell", "cmd", "notification", "cancel-all")...).Run()
	// Legacy fallback (older Android): dismiss via service call.
	_ = exec.CommandContext(ctx, p.adbPath, args("shell", "service", "call", "notification", "1")...).Run()
}

// fromADB runs `adb shell dumpsys notification` and scans for verification
// codes. Codes are only accepted when they appear near a platform keyword
// (instagram/tiktok), filtering out unrelated numeric strings.
func (p *Provider) fromADB(ctx context.Context) (string, error) {
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

	// Only consider lines that mention the platform — avoids unrelated codes.
	for _, line := range strings.Split(src, "\n") {
		lineLower := strings.ToLower(line)
		if !strings.Contains(lineLower, "instagram") && !strings.Contains(lineLower, "tiktok") {
			continue
		}
		// Also require a verification keyword on the same line or adjacent context.
		if !strings.Contains(lineLower, "code") && !strings.Contains(lineLower, "verify") &&
			!strings.Contains(lineLower, "confirm") && !strings.Contains(lineLower, "security") &&
			!strings.Contains(lineLower, "login") && !strings.Contains(lineLower, "sign in") {
			continue
		}
		matches := p.re.FindAllString(line, -1)
		for _, m := range matches {
			if isLikelyCode(line, m) {
				return m, nil
			}
		}
	}

	// Fallback: scan all text but still require verification keywords nearby.
	hasVerification := strings.Contains(lower, "verification") ||
		strings.Contains(lower, "code") || strings.Contains(lower, "confirm") ||
		strings.Contains(lower, "otp") || strings.Contains(lower, "login") ||
		strings.Contains(lower, "sign in") || strings.Contains(lower, "security")
	if !hasVerification {
		return "", fmt.Errorf("no verification notification found")
	}
	matches := p.re.FindAllString(src, -1)
	for _, m := range matches {
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
