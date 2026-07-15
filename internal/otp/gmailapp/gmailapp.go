package gmailapp

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ErrLogic/regbot/internal/appium"
	"github.com/ErrLogic/regbot/internal/locators"
)

// Default timings used when the corresponding Config field is zero.
const (
	defaultPollInterval = 5 * time.Second
	defaultElementWait  = 15 * time.Second
	defaultGmailPackage = "com.google.android.gm"
	inboxProbeWait      = 2 * time.Second
	maxBackToInbox      = 3
)

// errNoMatch signals that no allow-listed email or code was visible yet, so the
// provider should keep polling rather than fail.
var errNoMatch = errors.New("no matching verification email yet")

// Config configures the Gmail-app OTP provider.
type Config struct {
	// GmailPackage is the Gmail app package (default com.google.android.gm).
	GmailPackage string
	// ReturnPackage is the app to foreground again after reading the code
	// (typically the target platform's package).
	ReturnPackage string
	// SenderAllowlist matches the sender text of the verification email.
	SenderAllowlist []string
	// CodeRegex extracts the code from the email body (default \d{6}).
	CodeRegex string
	// PollInterval is how often the inbox is re-scanned.
	PollInterval time.Duration
	// ElementWait bounds each Gmail element lookup.
	ElementWait time.Duration
}

// ScreenshotSink persists a screenshot captured on failure.
type ScreenshotSink func(name string, png []byte)

// Option configures a GmailAppProvider.
type Option func(*GmailAppProvider)

// WithScreenshotSink registers a sink that receives a screenshot on failure.
func WithScreenshotSink(sink ScreenshotSink) Option {
	return func(p *GmailAppProvider) { p.sink = sink }
}

// GmailAppProvider reads verification codes from the on-device Gmail app by
// switching apps through the shared Appium driver. It never opens a second
// Appium session.
type GmailAppProvider struct {
	driver *appium.Driver
	loc    locators.Map
	cfg    Config
	sink   ScreenshotSink
}

// New constructs a GmailAppProvider bound to an existing Appium driver.
func New(driver *appium.Driver, loc locators.Map, cfg Config, opts ...Option) *GmailAppProvider {
	if cfg.GmailPackage == "" {
		cfg.GmailPackage = defaultGmailPackage
	}
	if cfg.CodeRegex == "" {
		cfg.CodeRegex = `\d{6}`
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = defaultPollInterval
	}
	if cfg.ElementWait <= 0 {
		cfg.ElementWait = defaultElementWait
	}
	p := &GmailAppProvider{driver: driver, loc: loc, cfg: cfg}
	for _, o := range opts {
		o(p)
	}
	return p
}

// GetCode switches to Gmail, finds the latest verification email whose sender
// matches the allow-list, extracts the code, and switches back to the return
// package. It polls until the code is found or timeout elapses.
func (p *GmailAppProvider) GetCode(ctx context.Context, targetEmail string, timeout time.Duration) (string, error) {
	re, err := regexp.Compile(p.cfg.CodeRegex)
	if err != nil {
		return "", fmt.Errorf("otp: compile code regex %q: %w", p.cfg.CodeRegex, err)
	}

	if err := p.driver.LaunchApp(ctx, p.cfg.GmailPackage); err != nil {
		return "", p.failf(ctx, "otp: launch gmail: %v", err)
	}
	defer p.restore(ctx)

	p.normalizeToInbox(ctx)
	p.pullToRefresh(ctx)

	deadline := time.Now().Add(timeout)
	for {
		code, err := p.findCode(ctx, re)
		if err == nil {
			return code, nil
		}
		if fatal := p.classify(err); fatal != nil {
			return "", p.failf(ctx, "otp: %v", fatal)
		}
		if time.Now().After(deadline) {
			return "", p.failf(ctx, "otp: verification code for %s not found within %s", targetEmail, timeout)
		}
		select {
		case <-ctx.Done():
			return "", p.failf(ctx, "otp: %v", ctx.Err())
		case <-time.After(p.cfg.PollInterval):
		}
		p.pullToRefresh(ctx)
	}
}

// classify returns a non-nil error when err is fatal (should stop polling), or
// nil when it is a transient "not yet" condition worth retrying.
func (p *GmailAppProvider) classify(err error) error {
	switch {
	case errors.Is(err, errNoMatch):
		return nil
	case errors.Is(err, appium.ErrElementNotFound), errors.Is(err, appium.ErrTimeout):
		return nil
	case errors.Is(err, appium.ErrSessionExpired):
		return err
	default:
		return nil // treat unknown UI hiccups as transient and keep polling
	}
}

// findCode looks for an allow-listed email at the top of the inbox, opens it,
// and extracts the code from its body.
func (p *GmailAppProvider) findCode(ctx context.Context, re *regexp.Regexp) (string, error) {
	senderEl, err := p.loc.Resolve(ctx, p.driver, "sender", p.cfg.ElementWait)
	if err != nil {
		return "", err
	}
	sender, err := senderEl.GetText(ctx)
	if err != nil {
		return "", err
	}
	if !senderMatches(sender, p.cfg.SenderAllowlist) {
		return "", errNoMatch
	}

	row, err := p.loc.Resolve(ctx, p.driver, "email_row", p.cfg.ElementWait)
	if err != nil {
		return "", err
	}
	if err := row.Click(ctx); err != nil {
		return "", err
	}

	bodyEl, err := p.loc.Resolve(ctx, p.driver, "message_body", p.cfg.ElementWait)
	if err != nil {
		return "", err
	}
	body, err := bodyEl.GetText(ctx)
	if err != nil {
		return "", err
	}

	code, ok := extractCode(re, body)
	if !ok {
		// Email opened but no code yet; back out and keep polling.
		_ = p.driver.PressBack(ctx)
		return "", errNoMatch
	}
	return code, nil
}

// normalizeToInbox backs out of any open conversation until the inbox list is
// visible (best-effort).
func (p *GmailAppProvider) normalizeToInbox(ctx context.Context) {
	for i := 0; i < maxBackToInbox; i++ {
		if _, err := p.loc.Resolve(ctx, p.driver, "message_list", inboxProbeWait); err == nil {
			return
		}
		_ = p.driver.PressBack(ctx)
	}
}

// pullToRefresh performs a downward swipe to refresh the inbox.
// TODO: coordinates are placeholders; derive from screen size on-device.
func (p *GmailAppProvider) pullToRefresh(ctx context.Context) {
	_ = p.driver.Swipe(ctx, 540, 600, 540, 1600, 20)
}

// restore returns to the configured return package, if any.
func (p *GmailAppProvider) restore(ctx context.Context) {
	if p.cfg.ReturnPackage != "" {
		_ = p.driver.LaunchApp(ctx, p.cfg.ReturnPackage)
	}
}

// failf builds an error and, if a sink is registered, attaches a screenshot.
func (p *GmailAppProvider) failf(ctx context.Context, format string, args ...any) error {
	err := fmt.Errorf(format, args...)
	if p.sink != nil {
		if png, serr := p.driver.Screenshot(ctx); serr == nil {
			p.sink("otp-failure", png)
		}
	}
	return err
}

// extractCode returns the first substring of body matching re.
func extractCode(re *regexp.Regexp, body string) (string, bool) {
	match := re.FindString(body)
	if match == "" {
		return "", false
	}
	return match, true
}

// senderMatches reports whether sender contains any allow-listed token
// (case-insensitive). An empty allow-list matches nothing.
func senderMatches(sender string, allowlist []string) bool {
	s := strings.ToLower(sender)
	for _, token := range allowlist {
		token = strings.ToLower(strings.TrimSpace(token))
		if token != "" && strings.Contains(s, token) {
			return true
		}
	}
	return false
}
