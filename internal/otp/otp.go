package otp

import (
	"context"
	"errors"
	"time"
)

// ErrCodeNotFound indicates no verification code arrived within the timeout. It
// is exposed so callers can map it to a distinct exit code.
var ErrCodeNotFound = errors.New("otp: verification code not found")

// OTPProvider retrieves a one-time verification code for targetEmail, waiting up
// to timeout for it to arrive. Implementations decouple flows from any specific
// OTP source (Gmail app, IMAP, etc.).
type OTPProvider interface {
	GetCode(ctx context.Context, targetEmail string, timeout time.Duration) (string, error)
}
