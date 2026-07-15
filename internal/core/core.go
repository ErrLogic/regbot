package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/ErrLogic/regbot/internal/appium"
	"github.com/ErrLogic/regbot/internal/config"
	"github.com/ErrLogic/regbot/internal/flows"
	"github.com/ErrLogic/regbot/internal/locators"
	"github.com/ErrLogic/regbot/internal/otp/gmailapp"
)

// ADB is the subset of adb operations the orchestrator needs (satisfied by
// *adb.Client).
type ADB interface {
	CheckDevice(ctx context.Context) error
	IsInstalled(ctx context.Context, pkg string) (bool, error)
}

// driverFactory opens an Appium session; overridable in tests.
type driverFactory func(ctx context.Context, serverURL string, caps appium.Capabilities) (*appium.Driver, error)

// RegistrationService orchestrates a single registration run: pre-flight, one
// Appium session, locator loading, provider wiring, flow execution, and
// artifacts.
type RegistrationService struct {
	logger    *zap.Logger
	adb       ADB
	newDriver driverFactory
}

// NewService constructs a RegistrationService using the real Appium driver
// factory.
func NewService(logger *zap.Logger, adbClient ADB) *RegistrationService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &RegistrationService{logger: logger, adb: adbClient, newDriver: appium.NewDriver}
}

// Register performs a full registration for platform using email, returning the
// created Account. When dryRun is set, the flow stops before the final submit.
func (s *RegistrationService) Register(
	ctx context.Context,
	platform flows.Platform,
	email string,
	cfg config.Config,
	dryRun bool,
) (flows.Account, error) {
	runID := newRunID()
	logger := s.logger.With(
		zap.String("run_id", runID),
		zap.String("platform", string(platform)),
		zap.String("email", email),
	)
	logger.Info("registration started", zap.Bool("dry_run", dryRun))

	targetPkg, err := targetPackage(cfg, platform)
	if err != nil {
		return flows.Account{}, err
	}

	// Pre-flight.
	if err := s.preflight(ctx, cfg, targetPkg); err != nil {
		return flows.Account{}, err
	}

	// One Appium session for the whole run.
	driver, err := s.newDriver(ctx, cfg.Appium.ServerURL, capsFor(cfg, platform, targetPkg))
	if err != nil {
		return flows.Account{}, fmt.Errorf("open appium session: %w", err)
	}
	defer func() {
		if qerr := driver.Quit(ctx); qerr != nil {
			logger.Warn("quit session", zap.Error(qerr))
		}
	}()

	// Locators.
	targetLoc, gmailLoc, err := loadLocators(cfg.Paths.LocatorsDir, platform)
	if err != nil {
		return flows.Account{}, err
	}

	// Provider + flow, sharing the driver and artifact directory.
	sink := flows.NewArtifactSink(driver, cfg.Paths.ArtifactsDir, runID, logger)
	provider := gmailapp.New(driver, gmailLoc, gmailConfig(cfg, targetPkg),
		gmailapp.WithScreenshotSink(pngSink(cfg.Paths.ArtifactsDir, runID)))
	flow := flowFor(platform, flowConfig(cfg, dryRun), logger, sink)

	acct, err := flow.Register(ctx, driver, provider, email, targetLoc)
	if err != nil {
		s.writeResult(cfg.Paths.ArtifactsDir, runResult{
			RunID:     runID,
			Platform:  string(platform),
			Email:     email,
			Status:    "failed",
			Error:     err.Error(),
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		})
		logger.Error("registration failed", zap.Error(err))
		return flows.Account{}, err
	}

	s.writeResult(cfg.Paths.ArtifactsDir, runResult{
		RunID:     runID,
		Platform:  string(acct.Platform),
		Email:     acct.Email,
		Username:  acct.Username,
		Status:    acct.Status,
		CreatedAt: acct.CreatedAt.Format(time.RFC3339),
	})
	logger.Info("registration complete",
		zap.String("username", acct.Username),
		zap.String("status", acct.Status))
	return acct, nil
}

// preflight verifies a device is present and the required apps are installed.
func (s *RegistrationService) preflight(ctx context.Context, cfg config.Config, targetPkg string) error {
	if s.adb == nil {
		return nil
	}
	if err := s.adb.CheckDevice(ctx); err != nil {
		return fmt.Errorf("preflight: %w", err)
	}
	for _, pkg := range []string{targetPkg, cfg.Apps.GmailPackage} {
		installed, err := s.adb.IsInstalled(ctx, pkg)
		if err != nil {
			return fmt.Errorf("preflight: check %s: %w", pkg, err)
		}
		if !installed {
			return fmt.Errorf("preflight: required app %q is not installed", pkg)
		}
	}
	return nil
}

// runResult is the per-run summary written to artifacts (never includes the
// password).
type runResult struct {
	RunID     string `json:"run_id"`
	Platform  string `json:"platform"`
	Email     string `json:"email"`
	Username  string `json:"username,omitempty"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
	CreatedAt string `json:"created_at"`
}

// writeResult persists the run result as JSON; failures are logged, not fatal.
func (s *RegistrationService) writeResult(dir string, r runResult) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		s.logger.Warn("create artifacts dir", zap.Error(err))
		return
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		s.logger.Warn("marshal result", zap.Error(err))
		return
	}
	path := filepath.Join(dir, r.RunID+"-result.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		s.logger.Warn("write result", zap.Error(err))
	}
}

// ResolveEmail returns the configured address, or generates a +alias from the
// base address per FR-2/FR-9.
func ResolveEmail(cfg config.EmailConfig) (string, error) {
	if cfg.Address != "" {
		return cfg.Address, nil
	}
	if cfg.BaseAddress == "" {
		return "", fmt.Errorf("email: no address or base_address configured")
	}
	at := strings.LastIndexByte(cfg.BaseAddress, '@')
	if at <= 0 {
		return "", fmt.Errorf("email: invalid base_address %q", cfg.BaseAddress)
	}
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("email: random tag: %w", err)
	}
	tag := cfg.AliasTagPrefix + hex.EncodeToString(b)
	return cfg.BaseAddress[:at] + "+" + tag + cfg.BaseAddress[at:], nil
}

// newRunID returns a timestamped, unique run identifier.
func newRunID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return time.Now().UTC().Format("20060102-150405") + "-" + hex.EncodeToString(b)
}

// targetPackage returns the target app package for the platform.
func targetPackage(cfg config.Config, platform flows.Platform) (string, error) {
	switch platform {
	case flows.PlatformInstagram:
		return cfg.Apps.InstagramPackage, nil
	case flows.PlatformTikTok:
		return cfg.Apps.TikTokPackage, nil
	default:
		return "", fmt.Errorf("unsupported platform %q", platform)
	}
}

// capsFor builds Appium capabilities for the platform.
func capsFor(cfg config.Config, platform flows.Platform, targetPkg string) appium.Capabilities {
	caps := appium.Capabilities{
		PlatformName:      cfg.Device.PlatformName,
		AutomationName:    cfg.Device.AutomationName,
		DeviceName:        cfg.Device.DeviceName,
		UDID:              cfg.Device.UDID,
		AppPackage:        targetPkg,
		NewCommandTimeout: cfg.Appium.NewCommandTimeout,
	}
	if platform == flows.PlatformInstagram {
		caps.AppActivity = cfg.Apps.InstagramActivity
	}
	return caps
}

// loadLocators loads and validates the target and Gmail locator maps.
func loadLocators(dir string, platform flows.Platform) (target, gmail locators.Map, err error) {
	var app string
	var required []string
	switch platform {
	case flows.PlatformInstagram:
		app, required = "instagram", flows.InstagramLocatorNames
	case flows.PlatformTikTok:
		app, required = "tiktok", flows.TikTokLocatorNames
	default:
		return locators.Map{}, locators.Map{}, fmt.Errorf("unsupported platform %q", platform)
	}

	target, err = locators.Load(dir, app)
	if err != nil {
		return locators.Map{}, locators.Map{}, err
	}
	if err := target.Require(required...); err != nil {
		return locators.Map{}, locators.Map{}, err
	}

	gmail, err = locators.Load(dir, "gmail")
	if err != nil {
		return locators.Map{}, locators.Map{}, err
	}
	if err := gmail.Require(gmailapp.RequiredLocators...); err != nil {
		return locators.Map{}, locators.Map{}, err
	}
	return target, gmail, nil
}

// flowConfig maps the config to a flows.FlowConfig.
func flowConfig(cfg config.Config, dryRun bool) flows.FlowConfig {
	probe := 3 * time.Second
	if cfg.Timeouts.ElementWait < probe {
		probe = cfg.Timeouts.ElementWait
	}
	return flows.FlowConfig{
		PasswordLength: cfg.Account.PasswordLength,
		UsernamePrefix: cfg.Account.UsernamePrefix,
		ElementWait:    cfg.Timeouts.ElementWait,
		ProbeWait:      probe,
		OTPTimeout:     cfg.OTP.WaitTimeout,
		Retry:          flows.RetryPolicy{Attempts: cfg.Timeouts.StepRetry + 1, Backoff: time.Second},
		DryRun:         dryRun,
	}
}

// gmailConfig maps the config to a gmailapp.Config.
func gmailConfig(cfg config.Config, returnPkg string) gmailapp.Config {
	return gmailapp.Config{
		GmailPackage:    cfg.Apps.GmailPackage,
		ReturnPackage:   returnPkg,
		SenderAllowlist: cfg.OTP.SenderAllowlist,
		CodeRegex:       cfg.OTP.CodeRegex,
		PollInterval:    cfg.OTP.PollInterval,
		ElementWait:     cfg.Timeouts.ElementWait,
	}
}

// flowFor builds the platform flow.
func flowFor(platform flows.Platform, fc flows.FlowConfig, logger *zap.Logger, sink flows.FailureSink) flows.PlatformFlow {
	switch platform {
	case flows.PlatformTikTok:
		return flows.TikTokFlow{Cfg: fc, Logger: logger, Sink: sink}
	default:
		return flows.InstagramFlow{Cfg: fc, Logger: logger, Sink: sink}
	}
}

// pngSink returns a gmailapp screenshot sink that writes PNGs into the artifacts
// directory.
func pngSink(dir, runID string) gmailapp.ScreenshotSink {
	return func(name string, png []byte) {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return
		}
		_ = os.WriteFile(filepath.Join(dir, runID+"-"+name+".png"), png, 0o600)
	}
}
