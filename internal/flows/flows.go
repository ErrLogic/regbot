package flows

import (
	"context"
	"time"

	"github.com/ErrLogic/regbot/internal/appium"
	"github.com/ErrLogic/regbot/internal/locators"
	"github.com/ErrLogic/regbot/internal/otp"
)

// Platform identifies a supported target platform.
type Platform string

// Supported platforms.
const (
	PlatformInstagram Platform = "instagram"
	PlatformTikTok    Platform = "tiktok"
)

// Account is the result of a successful registration. The Password field is
// sensitive and must never be logged.
type Account struct {
	Platform  Platform  `json:"platform"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"`
}

// FlowConfig holds the per-run settings a platform flow needs. It is set on the
// flow value by the core; the runtime driver/provider/email/locators are passed
// to Register.
type FlowConfig struct {
	// PasswordLength is the generated-password length.
	PasswordLength int
	// UsernamePrefix seeds generated usernames.
	UsernamePrefix string
	// ElementWait bounds each UI element lookup.
	ElementWait time.Duration
	// ProbeWait bounds best-effort presence checks (optional screens, taken
	// username), kept short so absent elements don't stall the flow.
	ProbeWait time.Duration
	// OTPTimeout bounds waiting for the verification code.
	OTPTimeout time.Duration
	// Retry is the per-step retry policy.
	Retry RetryPolicy
	// DryRun stops the flow before the final submission.
	DryRun bool
	// UseSSO selects Google single-sign-on registration (via the on-device
	// Google account) instead of email + OTP. Currently honoured by TikTok.
	UseSSO bool
}

// PlatformFlow registers a single account on one platform. The OTP provider is
// injected by the caller so flows never talk to Gmail directly.
type PlatformFlow interface {
	Register(
		ctx context.Context,
		driver *appium.Driver,
		provider otp.OTPProvider,
		email string,
		loc locators.Map,
	) (Account, error)
}
