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
