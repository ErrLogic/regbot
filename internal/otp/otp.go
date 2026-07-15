package otp

import (
	"context"
	"time"
)

// OTPProvider retrieves a one-time verification code for targetEmail, waiting up
// to timeout for it to arrive. Implementations decouple flows from any specific
// OTP source (Gmail app, IMAP, etc.).
type OTPProvider interface {
	GetCode(ctx context.Context, targetEmail string, timeout time.Duration) (string, error)
}
