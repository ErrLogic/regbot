package tiktok

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/ErrLogic/regbot/internal/appium"
	"github.com/ErrLogic/regbot/internal/config"
	"github.com/ErrLogic/regbot/internal/flows"
	"github.com/ErrLogic/regbot/internal/job"
	"github.com/ErrLogic/regbot/internal/locators"
	"github.com/ErrLogic/regbot/internal/otp"
	"github.com/ErrLogic/regbot/internal/otp/gmailapp"
	"github.com/ErrLogic/regbot/internal/otp/notification"
)

const defaultProbeWait = 2 * time.Second

// Tap is an alias for flows.TapByLocator.
var Tap = flows.TapByLocator

// Type is an alias for flows.TypeByLocator.
var Type = flows.TypeByLocator

// Register drives the TikTok registration flow by delegating to the shared,
// real-device-verified flows.TikTokFlow (which handles onboarding, tutorial
// overlays, profile navigation, email OTP, and Google SSO). This keeps the
// worker path and the CLI path on a single implementation.
func Register(
	ctx context.Context,
	driver *appium.Driver,
	loc locators.Map,
	cfg config.Config,
	params job.RegisterParams,
	logFunc func(string, string, string),
) error {
	flow := flows.TikTokFlow{
		Cfg:    flowConfigFrom(cfg, params),
		Logger: loggerFor(logFunc),
	}
	provider := otpProvider(cfg, driver)
	_, err := flow.Register(ctx, driver, provider, params.Email, loc)
	return err
}

// flowConfigFrom builds a flows.FlowConfig from the run config and job params.
func flowConfigFrom(cfg config.Config, params job.RegisterParams) flows.FlowConfig {
	probe := 3 * time.Second
	if cfg.Timeouts.ElementWait < probe {
		probe = cfg.Timeouts.ElementWait
	}
	backoff := cfg.Timeouts.StepBackoff
	if backoff <= 0 {
		backoff = time.Second
	}
	return flows.FlowConfig{
		PasswordLength: cfg.Account.PasswordLength,
		UsernamePrefix: cfg.Account.UsernamePrefix,
		ElementWait:    cfg.Timeouts.ElementWait,
		ProbeWait:      probe,
		OTPTimeout:     cfg.OTP.WaitTimeout,
		Retry:          flows.RetryPolicy{Attempts: cfg.Timeouts.StepRetry + 1, Backoff: backoff},
		DryRun:         params.DryRun,
		UseSSO:         params.UseSSO || cfg.Account.UseGoogleSSO,
	}
}

// otpProvider builds the notification-first OTP provider (Gmail app fallback),
// matching the CLI path. The Gmail fallback uses the gmail locator set.
func otpProvider(cfg config.Config, driver *appium.Driver) otp.OTPProvider {
	gmailLoc, err := locators.Load(cfg.Paths.LocatorsDir, "gmail")
	if err != nil {
		// Notification-only provider still works without Gmail locators.
		p, nerr := notification.New(cfg.Device.UDID, cfg.OTP.CodeRegex, nil)
		if nerr != nil {
			return &nopProvider{}
		}
		return p
	}
	gmailProvider := gmailapp.New(driver, gmailLoc, gmailapp.Config{
		GmailPackage:    cfg.Apps.GmailPackage,
		ReturnPackage:   cfg.Apps.TikTokPackage,
		SenderAllowlist: cfg.OTP.SenderAllowlist,
		CodeRegex:       cfg.OTP.CodeRegex,
		PollInterval:    cfg.OTP.PollInterval,
		ElementWait:     cfg.Timeouts.ElementWait,
	})
	provider, err := notification.New(cfg.Device.UDID, cfg.OTP.CodeRegex, gmailProvider)
	if err != nil {
		return gmailProvider
	}
	return provider
}

// nopProvider is a last-resort OTP provider that always fails; used only when
// both notification and Gmail providers cannot be constructed.
type nopProvider struct{}

func (nopProvider) GetCode(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", otp.ErrCodeNotFound
}

// loggerFor adapts a (level, step, message) log function to a *zap.Logger the
// flow can use. Since the flow logs through zap, we build a no-op logger and
// rely on the flow's own step logging via the sink; callers still get progress
// through the step names emitted by runSteps.
func loggerFor(_ func(string, string, string)) *zap.Logger {
	return zap.NewNop()
}
