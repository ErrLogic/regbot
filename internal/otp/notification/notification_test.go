package notification

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ErrLogic/regbot/internal/otp"
)

// mockProvider is a stand-in Gmail fallback provider.
type mockProvider struct {
	code   string
	err    error
	called bool
}

func (m *mockProvider) GetCode(_ context.Context, _ string, _ time.Duration) (string, error) {
	m.called = true
	return m.code, m.err
}

func TestNewInvalidRegex(t *testing.T) {
	if _, err := New("dev1", "[invalid(", nil); err == nil {
		t.Fatal("expected error for invalid regex")
	}
}

func TestGetCodeDelegatesToGmailWhenNoSerial(t *testing.T) {
	// With an empty serial (e.g. no device / test env), the provider must skip
	// the ADB notification path and delegate straight to the Gmail fallback.
	gmail := &mockProvider{code: "483920"}
	p, err := New("", `\d{6}`, gmail)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	code, err := p.GetCode(context.Background(), "user@gmail.com", 2*time.Second)
	if err != nil {
		t.Fatalf("GetCode: %v", err)
	}
	if code != "483920" {
		t.Errorf("code = %q, want 483920", code)
	}
	if !gmail.called {
		t.Error("Gmail fallback should have been called")
	}
}

func TestGetCodeNoSerialNoGmailFails(t *testing.T) {
	// Empty serial and no Gmail fallback: the ADB path yields nothing and the
	// call must fail with ErrCodeNotFound (never hang).
	p, err := New("", `\d{6}`, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err = p.GetCode(ctx, "user@gmail.com", 500*time.Millisecond)
	if err == nil {
		t.Fatal("expected an error when no device and no fallback")
	}
}

func TestGetCodePropagatesGmailError(t *testing.T) {
	gmail := &mockProvider{err: otp.ErrCodeNotFound}
	p, err := New("", `\d{6}`, gmail)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = p.GetCode(context.Background(), "user@gmail.com", time.Second)
	if !errors.Is(err, otp.ErrCodeNotFound) {
		t.Fatalf("want ErrCodeNotFound from fallback, got %v", err)
	}
}

func TestIsLikelyCode(t *testing.T) {
	cases := []struct {
		src, code string
		want      bool
	}{
		{"Your code is 483920", "483920", true},
		{"Copyright 2025 TikTok", "2025", false}, // year-like, filtered
		{"code 12", "12", false},                 // too short
		{"verify 55123", "55123", true},
	}
	for _, c := range cases {
		if got := isLikelyCode(c.src, c.code); got != c.want {
			t.Errorf("isLikelyCode(%q,%q) = %v, want %v", c.src, c.code, got, c.want)
		}
	}
}
