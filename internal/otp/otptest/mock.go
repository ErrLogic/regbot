// Package otptest provides a mock OTPProvider for use in flow and core tests.
package otptest

import (
	"context"
	"time"
)

// Mock is a configurable OTPProvider for tests. It records the arguments it was
// called with and returns the preconfigured Code/Err.
type Mock struct {
	Code string
	Err  error

	Called     bool
	GotEmail   string
	GotTimeout time.Duration
}

// GetCode records the call and returns the configured code or error.
func (m *Mock) GetCode(_ context.Context, targetEmail string, timeout time.Duration) (string, error) {
	m.Called = true
	m.GotEmail = targetEmail
	m.GotTimeout = timeout
	return m.Code, m.Err
}
