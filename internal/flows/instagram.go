package flows

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/ErrLogic/regbot/internal/appium"
	"github.com/ErrLogic/regbot/internal/locators"
	"github.com/ErrLogic/regbot/internal/otp"
)

// maxUsernameTries bounds the "username taken -> regenerate" loop.
const maxUsernameTries = 5

// maxInterstitialDismiss bounds the post-signup "Skip"/"Not now" dismissal loop
// (add photo, find friends, add phone number, tutorials, ...).
const maxInterstitialDismiss = 8

// birthdayYearSwipes is how many downward flings to apply to the year spinner of
// the native date-picker. The picker defaults to the current date (age 0); each
// swipe moves it back roughly one year, so a generous fixed count guarantees an
// adult birthday without needing to read the exact spinner value.
const birthdayYearSwipes = 25

// defaultProbeWait is used when FlowConfig.ProbeWait is unset.
const defaultProbeWait = 2 * time.Second

// InstagramFlow registers an account on Instagram. Construct it with a
// FlowConfig, an optional logger, and an optional failure sink.
type InstagramFlow struct {
	Cfg    FlowConfig
	Logger *zap.Logger
	Sink   FailureSink
}

// Register drives the Instagram email-registration flow to completion (or to
// just before the account-creating "I agree" tap when DryRun is set), returning
// the created account. All UI elements are referenced by locator name.
//
// Screen order verified on a real device against the current signup wizard:
// Get started → Sign up with email → email → confirmation code → password →
// birthday (native date-picker) → full name → username → agree to terms
// (account is created here) → post-signup interstitials.
func (f InstagramFlow) Register(
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
	fullName, err := GenerateFullName()
	if err != nil {
		return Account{}, err
	}
	birthday, err := AdultBirthday()
	if err != nil {
		return Account{}, err
	}

	var code, username string

	steps := []Step{
		{Name: "launch create account", Run: func(ctx context.Context) error {
			return tapByLocator(ctx, driver, loc, "create_new_account", wait)
		}},
		{Name: "switch to email", Run: func(ctx context.Context) error {
			// The first post-welcome screen asks for a mobile number; switch to
			// the email path.
			return tapByLocator(ctx, driver, loc, "switch_to_email", wait)
		}},
		{Name: "enter email", Run: func(ctx context.Context) error {
			if err := typeByLocator(ctx, driver, loc, "email_field", email, wait); err != nil {
				return err
			}
			return tapByLocator(ctx, driver, loc, "next_button", wait)
		}},
		{Name: "confirm email dialog", Run: func(ctx context.Context) error {
			// Some builds interject an "Is this your email?" confirmation.
			dismissIfPresent(ctx, driver, loc, "confirm_email_next", probe)
			return nil
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
			if err := typeByLocator(ctx, driver, loc, "confirmation_code_field", code, wait); err != nil {
				return err
			}
			return tapByLocator(ctx, driver, loc, "confirm_code_button", wait)
		}},
		{Name: "set password", Run: func(ctx context.Context) error {
			if err := typeByLocator(ctx, driver, loc, "password_field", password, wait); err != nil {
				return err
			}
			return tapByLocator(ctx, driver, loc, "next_button", wait)
		}},
		{Name: "set birthday", Run: func(ctx context.Context) error {
			logger.Debug("generated birthday", zap.Time("birthday", birthday))
			if err := setBirthday(ctx, driver, loc, wait, probe); err != nil {
				return err
			}
			return tapByLocator(ctx, driver, loc, "birthday_next", wait)
		}},
		{Name: "set full name", Run: func(ctx context.Context) error {
			if err := typeByLocator(ctx, driver, loc, "full_name_field", fullName, wait); err != nil {
				return err
			}
			return tapByLocator(ctx, driver, loc, "next_button", wait)
		}},
		{Name: "set username", Run: func(ctx context.Context) error {
			// The username screen pre-fills a valid suggestion; type our own and
			// regenerate if the "not available" error appears.
			u, err := UniqueUsername(f.Cfg.UsernamePrefix, func(name string) (bool, error) {
				if err := typeByLocator(ctx, driver, loc, "username_field", name, wait); err != nil {
					return false, err
				}
				return isPresent(ctx, driver, loc, "username_taken_error", probe), nil
			}, maxUsernameTries)
			if err != nil {
				return err
			}
			username = u
			return tapByLocator(ctx, driver, loc, "next_button", wait)
		}},
		{Name: "finalise", Run: func(ctx context.Context) error {
			// Agreeing to the terms is what actually creates the account, so a
			// dry-run stops here, before that tap.
			if f.Cfg.DryRun {
				logger.Info("dry-run: skipping terms acceptance / account creation")
				return nil
			}
			if err := tapByLocator(ctx, driver, loc, "agree_terms_button", wait); err != nil {
				return err
			}
			// Post-signup prompts: device permissions, add photo, find friends,
			// add phone, tutorials.
			dismissInterstitials(ctx, driver, loc, probe)
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
		Platform:  PlatformInstagram,
		Email:     email,
		Username:  username,
		Password:  password,
		CreatedAt: time.Now().UTC(),
		Status:    status,
	}, nil
}

// setBirthday drives the native date-picker on the "What's your birthday?"
// screen. The picker defaults to the current date (age 0), which fails the age
// gate, so it flings the year spinner back a generous fixed number of years and
// confirms with the picker's SET button. If the picker is not already open
// (builds vary), it taps the birthday field first.
func setBirthday(ctx context.Context, driver *appium.Driver, loc locators.Map, wait, probe time.Duration) error {
	if !isPresent(ctx, driver, loc, "birthday_year_picker", probe) {
		// Open the picker by tapping the birthday field.
		dismissIfPresent(ctx, driver, loc, "birthday_field", probe)
	}
	year, err := loc.Resolve(ctx, driver, "birthday_year_picker", wait)
	if err != nil {
		return err
	}
	rect, err := year.Rect(ctx)
	if err != nil {
		return err
	}
	cx := rect.X + rect.Width/2
	yTop := rect.Y + rect.Height/4
	yBottom := rect.Y + (rect.Height*3)/4
	// Downward flings decrement the year (move it into the past).
	for i := 0; i < birthdayYearSwipes; i++ {
		if err := driver.Swipe(ctx, cx, yTop, cx, yBottom, 20); err != nil {
			return err
		}
	}
	return tapByLocator(ctx, driver, loc, "birthday_set", wait)
}

// dismissInterstitials taps through consecutive "Skip"/"Not now"/"No, skip"
// screens until none is shown or the cap is reached.
func dismissInterstitials(ctx context.Context, driver *appium.Driver, loc locators.Map, wait time.Duration) {
	for i := 0; i < maxInterstitialDismiss; i++ {
		tapped := dismissIfPresent(ctx, driver, loc, "skip_button", wait)
		if dismissIfPresent(ctx, driver, loc, "not_now_button", wait) {
			tapped = true
		}
		if !tapped {
			return
		}
	}
}

// compile-time check.
var _ PlatformFlow = InstagramFlow{}
