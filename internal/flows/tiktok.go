package flows

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/ErrLogic/regbot/internal/appium"
	"github.com/ErrLogic/regbot/internal/locators"
	"github.com/ErrLogic/regbot/internal/otp"
)

const maxTikTokSkips = 3

// maxTutorialSwipes bounds the "Swipe up for more" tutorial dismissal loop.
// Tutorials usually end after 2-3 swipes; keep the cap low to avoid wasted
// PageSource round-trips (which are slow on TikTok).
const maxTutorialSwipes = 4

// TikTokFlow registers an account on TikTok.
type TikTokFlow struct {
	Cfg    FlowConfig
	Logger *zap.Logger
	Sink   FailureSink
}

// Register drives the TikTok registration flow. It selects the Google SSO path
// when FlowConfig.UseSSO is set, otherwise the email + OTP path.
func (f TikTokFlow) Register(
	ctx context.Context,
	driver *appium.Driver,
	provider otp.OTPProvider,
	email string,
	loc locators.Map,
) (Account, error) {
	if f.Cfg.UseSSO {
		return f.registerSSO(ctx, driver, email, loc)
	}
	return f.registerEmail(ctx, driver, provider, email, loc)
}

// registerEmail drives the TikTok email-registration flow.
func (f TikTokFlow) registerEmail(
	ctx context.Context,
	driver *appium.Driver,
	provider otp.OTPProvider,
	email string,
	loc locators.Map,
) (Account, error) {
	logger := f.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	wait := f.Cfg.ElementWait
	probe := f.Cfg.ProbeWait
	if probe <= 0 {
		probe = defaultProbeWait
	}

	password, err := GeneratePassword(f.Cfg.PasswordLength)
	if err != nil {
		return Account{}, err
	}
	nickname, err := GenerateUsername(f.Cfg.UsernamePrefix)
	if err != nil {
		return Account{}, err
	}
	birthday, err := AdultBirthday()
	if err != nil {
		return Account{}, err
	}
	_ = birthday

	var code string

	steps := []Step{
		// ── Onboarding phase (all best-effort — screens vary between sessions) ──
		{Name: "dismiss google sheet", Run: func(ctx context.Context) error {
			dismissIfPresent(ctx, driver, loc, "dismiss_sheet", probe)
			return nil
		}},
		{Name: "agree terms", Run: func(ctx context.Context) error {
			// Terms dialog may or may not appear. If present, tap the Agree button.
			if dismissIfPresent(ctx, driver, loc, "agree_terms", wait) {
				time.Sleep(1 * time.Second)
				// Second privacy dialog may appear after the first.
				dismissIfPresent(ctx, driver, loc, "agree_terms", probe)
			}
			return nil
		}},
		{Name: "allow notifications", Run: func(ctx context.Context) error {
			dismissIfPresent(ctx, driver, loc, "allow_button", probe)
			return nil
		}},
		{Name: "skip interests", Run: func(ctx context.Context) error {
			dismissIfPresent(ctx, driver, loc, "skip_button", wait)
			return nil
		}},

		// ── Tutorial overlay phase ──
		{Name: "clear tutorial overlays", Run: func(ctx context.Context) error {
			clearTikTokTutorials(ctx, driver, logger)
			return nil
		}},

		// ── Navigate to sign-up ──
		{Name: "go to profile tab", Run: func(ctx context.Context) error {
			for i := 0; i < 3; i++ {
				_ = driver.Tap(ctx, 972, 1827)
				time.Sleep(2 * time.Second)
				dismissIfPresent(ctx, driver, loc, "dismiss_sheet", probe)
				time.Sleep(500 * time.Millisecond)
				src, _ := driver.PageSource(ctx)
				if strings.Contains(src, "Login") || strings.Contains(src, "Edit profile") {
					return nil
				}
			}
			return nil
		}},
		{Name: "tap login to reach sign-up", Run: func(ctx context.Context) error {
			// TikTok may not show "Sign up" directly on profile; route through Login page.
			src, _ := driver.PageSource(ctx)
			if strings.Contains(src, "Sign up") {
				return nil // already on sign-up screen
			}
			// Tap "Login" or "Log into existing account".
			if !dismissIfPresent(ctx, driver, loc, "login_button", wait) {
				// Fallback: tap by text.
				_ = tapByLocator(ctx, driver, loc, "login_button", wait)
			}
			time.Sleep(2 * time.Second)
			return nil
		}},
		{Name: "tap sign up", Run: func(ctx context.Context) error {
			return tapByLocator(ctx, driver, loc, "sign_up_button", wait)
		}},
		{Name: "choose phone or email", Run: func(ctx context.Context) error {
			return tapByLocator(ctx, driver, loc, "use_phone_or_email", wait)
		}},
		{Name: "choose email tab", Run: func(ctx context.Context) error {
			return tapByLocator(ctx, driver, loc, "email_tab", wait)
		}},
		{Name: "set birthday", Run: func(ctx context.Context) error {
			logger.Debug("generated birthday", zap.Time("birthday", birthday))
			return tapByLocator(ctx, driver, loc, "birthday_next", wait)
		}},
		{Name: "enter email", Run: func(ctx context.Context) error {
			return typeByLocator(ctx, driver, loc, "email_field", email, wait)
		}},
		{Name: "send code", Run: func(ctx context.Context) error {
			return tapByLocator(ctx, driver, loc, "send_code_button", wait)
		}},
		{Name: "retrieve otp", Run: func(ctx context.Context) error {
			c, err := provider.GetCode(ctx, email, f.Cfg.OTPTimeout)
			if err != nil {
				return err
			}
			code = c
			return nil
		}},
		{Name: "enter otp", Run: func(ctx context.Context) error {
			return typeByLocator(ctx, driver, loc, "code_field", code, wait)
		}},
		{Name: "submit otp", Run: func(ctx context.Context) error {
			return tapByLocator(ctx, driver, loc, "next_button", wait)
		}},
		{Name: "set password", Run: func(ctx context.Context) error {
			return typeByLocator(ctx, driver, loc, "password_field", password, wait)
		}},
		{Name: "set nickname", Run: func(ctx context.Context) error {
			return typeByLocator(ctx, driver, loc, "nickname_field", nickname, wait)
		}},
		{Name: "skip contacts", Run: func(ctx context.Context) error {
			for i := 0; i < maxTikTokSkips; i++ {
				if !dismissIfPresent(ctx, driver, loc, "skip_button", probe) {
					break
				}
			}
			return nil
		}},
		{Name: "finalise", Run: func(ctx context.Context) error {
			if f.Cfg.DryRun {
				logger.Info("dry-run: skipping final submit")
				return nil
			}
			return tapByLocator(ctx, driver, loc, "finish_button", wait)
		}},
	}

	if err := runSteps(ctx, logger, f.Sink, f.Cfg.Retry, steps...); err != nil {
		return Account{}, err
	}

	status := "success"
	if f.Cfg.DryRun {
		status = "dry-run"
	}
	return Account{
		Platform:  PlatformTikTok,
		Email:     email,
		Username:  nickname,
		Password:  password,
		CreatedAt: time.Now().UTC(),
		Status:    status,
	}, nil
}

// registerSSO drives the TikTok Google single-sign-on registration flow. It uses
// the on-device Google account (via the Credential Manager "Continue" sheet) and
// requires no email OTP. The email argument is recorded on the account for
// reference only.
func (f TikTokFlow) registerSSO(
	ctx context.Context,
	driver *appium.Driver,
	email string,
	loc locators.Map,
) (Account, error) {
	logger := f.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	wait := f.Cfg.ElementWait
	probe := f.Cfg.ProbeWait
	if probe <= 0 {
		probe = defaultProbeWait
	}

	nickname, err := GenerateUsername(f.Cfg.UsernamePrefix)
	if err != nil {
		return Account{}, err
	}

	steps := []Step{
		// ── Pre-SSO onboarding (best-effort; screens vary) ──
		{Name: "agree terms", Run: func(ctx context.Context) error {
			if dismissIfPresent(ctx, driver, loc, "agree_terms", probe) {
				time.Sleep(1 * time.Second)
				dismissIfPresent(ctx, driver, loc, "agree_terms", probe)
			}
			return nil
		}},
		{Name: "allow notifications", Run: func(ctx context.Context) error {
			dismissIfPresent(ctx, driver, loc, "allow_button", probe)
			return nil
		}},
		{Name: "skip interests", Run: func(ctx context.Context) error {
			dismissIfPresent(ctx, driver, loc, "skip_button", probe)
			return nil
		}},
		{Name: "clear tutorial overlays", Run: func(ctx context.Context) error {
			clearTikTokTutorials(ctx, driver, logger)
			return nil
		}},

		// ── Surface the Google account sheet ──
		{Name: "open google sheet", Run: func(ctx context.Context) error {
			// The Credential Manager sheet usually appears on the profile tab.
			for i := 0; i < 3; i++ {
				if isPresent(ctx, driver, loc, "sso_continue", probe) ||
					isPresent(ctx, driver, loc, "sso_account_row", probe) {
					return nil
				}
				_ = driver.Tap(ctx, 972, 1827) // profile tab
				time.Sleep(2 * time.Second)
			}
			return nil
		}},

		// ── Trigger Google SSO ──
		{Name: "tap continue (google sso)", Run: func(ctx context.Context) error {
			if f.Cfg.DryRun {
				logger.Info("dry-run: skipping google sso submit")
				return nil
			}
			return tapByLocator(ctx, driver, loc, "sso_continue", wait)
		}},

		// ── Post-SSO onboarding ──
		{Name: "post-sso agree terms", Run: func(ctx context.Context) error {
			time.Sleep(2 * time.Second)
			dismissIfPresent(ctx, driver, loc, "agree_terms", probe)
			return nil
		}},
		{Name: "post-sso allow", Run: func(ctx context.Context) error {
			dismissIfPresent(ctx, driver, loc, "allow_button", probe)
			return nil
		}},
		{Name: "post-sso skip", Run: func(ctx context.Context) error {
			for i := 0; i < maxTikTokSkips; i++ {
				if !dismissIfPresent(ctx, driver, loc, "skip_button", probe) {
					break
				}
			}
			return nil
		}},
	}

	if err := runSteps(ctx, logger, f.Sink, f.Cfg.Retry, steps...); err != nil {
		return Account{}, err
	}

	status := "success"
	if f.Cfg.DryRun {
		status = "dry-run"
	}
	return Account{
		Platform:  PlatformTikTok,
		Email:     email,
		Username:  nickname,
		Password:  "", // SSO accounts have no local password
		CreatedAt: time.Now().UTC(),
		Status:    status,
	}, nil
}

// clearTikTokTutorials swipes away the "Swipe up for more" tutorial overlays that
// block the bottom navigation on first launch.
func clearTikTokTutorials(ctx context.Context, driver *appium.Driver, logger *zap.Logger) {
	for i := 0; i < maxTutorialSwipes; i++ {
		src, err := driver.PageSource(ctx)
		if err != nil {
			return
		}
		if !strings.Contains(src, "Swipe up") &&
			!strings.Contains(src, "swipe up") &&
			!strings.Contains(src, "Tap anywhere") {
			return
		}
		logger.Debug("dismissing tutorial overlay", zap.Int("iteration", i+1))
		_ = driver.Swipe(ctx, 540, 1400, 540, 300, 30)
		time.Sleep(1500 * time.Millisecond)
	}
}

var _ PlatformFlow = TikTokFlow{}
