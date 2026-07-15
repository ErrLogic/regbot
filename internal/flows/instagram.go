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
// just before the final submit when DryRun is set), returning the created
// account. All UI elements are referenced by locator name.
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
		{Name: "enter email", Run: func(ctx context.Context) error {
			return typeByLocator(ctx, driver, loc, "email_field", email, wait)
		}},
		{Name: "tap next", Run: func(ctx context.Context) error {
			return tapByLocator(ctx, driver, loc, "next_button", wait)
		}},
		{Name: "wait confirm email screen", Run: func(ctx context.Context) error {
			_, err := loc.Resolve(ctx, driver, "confirm_email_screen", wait)
			return err
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
			return typeByLocator(ctx, driver, loc, "confirmation_code_field", code, wait)
		}},
		{Name: "submit otp", Run: func(ctx context.Context) error {
			return tapByLocator(ctx, driver, loc, "confirm_code_button", wait)
		}},
		{Name: "set full name", Run: func(ctx context.Context) error {
			return typeByLocator(ctx, driver, loc, "full_name_field", fullName, wait)
		}},
		{Name: "set password", Run: func(ctx context.Context) error {
			return typeByLocator(ctx, driver, loc, "password_field", password, wait)
		}},
		{Name: "set username", Run: func(ctx context.Context) error {
			u, err := UniqueUsername(f.Cfg.UsernamePrefix, func(name string) (bool, error) {
				if err := typeByLocator(ctx, driver, loc, "username_field", name, wait); err != nil {
					return false, err
				}
				// The username is taken if the "not available" error is shown.
				return isPresent(ctx, driver, loc, "username_taken_error", probe), nil
			}, maxUsernameTries)
			if err != nil {
				return err
			}
			username = u
			return nil
		}},
		{Name: "set birthday", Run: func(ctx context.Context) error {
			// The date-picker interaction is device-specific; here we advance the
			// birthday screen. TODO: apply the generated date via the picker.
			logger.Debug("generated birthday", zap.Time("birthday", birthday))
			return tapByLocator(ctx, driver, loc, "birthday_next", wait)
		}},
		{Name: "dismiss optional screens", Run: func(ctx context.Context) error {
			dismissIfPresent(ctx, driver, loc, "skip_button", probe)
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
		Platform:  PlatformInstagram,
		Email:     email,
		Username:  username,
		Password:  password,
		CreatedAt: time.Now().UTC(),
		Status:    status,
	}, nil
}

// compile-time check.
var _ PlatformFlow = InstagramFlow{}
