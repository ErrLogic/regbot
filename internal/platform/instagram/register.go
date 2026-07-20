package instagram

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

// Register drives the Instagram email-registration flow by delegating to the
// shared, real-device-verified flows.InstagramFlow. Returns the created account
// on success so the caller can persist it.
func Register(
	ctx context.Context,
	driver *appium.Driver,
	loc locators.Map,
	cfg config.Config,
	params job.RegisterParams,
	logFunc func(string, string, string),
) (*flows.Account, error) {
	flow := flows.InstagramFlow{
		Cfg:    flowConfigFrom(cfg, params),
		Logger: zap.NewNop(),
	}
	provider := otpProvider(cfg, driver)
	acct, err := flow.Register(ctx, driver, provider, params.Email, loc)
	if err != nil {
		return nil, err
	}
	logFunc("info", "", "account created: "+acct.Username)
	return &acct, nil
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
	}
}

// otpProvider builds the notification-first OTP provider (Gmail app fallback).
func otpProvider(cfg config.Config, driver *appium.Driver) otp.OTPProvider {
	gmailLoc, err := locators.Load(cfg.Paths.LocatorsDir, "gmail")
	if err != nil {
		p, nerr := notification.New(cfg.Device.UDID, cfg.OTP.CodeRegex, nil)
		if nerr != nil {
			return &nopProvider{}
		}
		return p
	}
	gmailProvider := gmailapp.New(driver, gmailLoc, gmailapp.Config{
		GmailPackage:    cfg.Apps.GmailPackage,
		ReturnPackage:   cfg.Apps.InstagramPackage,
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

// nopProvider is a last-resort OTP provider that always fails.
type nopProvider struct{}

func (nopProvider) GetCode(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", otp.ErrCodeNotFound
}
