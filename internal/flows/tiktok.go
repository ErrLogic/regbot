package flows

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/ErrLogic/regbot/internal/appium"
	"github.com/ErrLogic/regbot/internal/locators"
	"github.com/ErrLogic/regbot/internal/otp"
)

// maxSkips bounds the interests/contacts skip loop.
const maxSkips = 3

// TikTokFlow registers an account on TikTok. Construct it with a FlowConfig, an
// optional logger, and an optional failure sink.
type TikTokFlow struct {
	Cfg    FlowConfig
	Logger *zap.Logger
	Sink   FailureSink
}

// Register drives the TikTok email-registration flow to completion (or to just
// before the final submit when DryRun is set), returning the created account.
// All UI elements are referenced by locator name.
func (f TikTokFlow) Register(
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

	var code string

	steps := []Step{
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
			// TikTok asks for the birthday before the email. The date-picker
			// interaction is device-specific; here we advance the screen.
			// TODO: apply the generated date via the picker.
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
		{Name: "skip interests and contacts", Run: func(ctx context.Context) error {
			for i := 0; i < maxSkips; i++ {
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

// compile-time check.
var _ PlatformFlow = TikTokFlow{}
