package flows

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/ErrLogic/regbot/internal/locators"
)

func loadTikTokLocators(t *testing.T) locators.Map {
	t.Helper()
	m, err := locators.Load(filepath.Join("..", "..", "locators"), "tiktok")
	if err != nil {
		t.Fatalf("load tiktok locators: %v", err)
	}
	return m
}

func TestTikTokRegister(t *testing.T) {
	tests := []struct {
		name       string
		dryRun     bool
		wantStatus string
		// wantSignUpCount: sign_up_button taps once; the finish button shares the
		// "Sign up" selector, so a completed (non-dry-run) run shows it twice.
		wantSignUpCount int
	}{
		{name: "happy path", dryRun: false, wantStatus: "success", wantSignUpCount: 2},
		{name: "dry run skips submit", dryRun: true, wantStatus: "dry-run", wantSignUpCount: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &flowRecorder{}
			driver := recordingDriver(t, rec)
			provider := &recProvider{rec: rec, code: "551234"}
			loc := loadTikTokLocators(t)

			flow := TikTokFlow{
				Cfg: FlowConfig{
					PasswordLength: 16,
					UsernamePrefix: "user",
					ElementWait:    500 * time.Millisecond,
					ProbeWait:      100 * time.Millisecond,
					OTPTimeout:     time.Second,
					Retry:          RetryPolicy{Attempts: 1},
					DryRun:         tt.dryRun,
				},
				Logger: zap.NewNop(),
			}

			acct, err := flow.Register(context.Background(), driver, provider, "tester@gmail.com", loc)
			if err != nil {
				t.Fatalf("Register: %v", err)
			}

			if acct.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q", acct.Status, tt.wantStatus)
			}
			if acct.Platform != PlatformTikTok {
				t.Errorf("platform = %q", acct.Platform)
			}
			if !strings.HasPrefix(acct.Username, "user_") {
				t.Errorf("nickname/username = %q", acct.Username)
			}
			if len(acct.Password) != 16 {
				t.Errorf("password length = %d", len(acct.Password))
			}

			events := rec.snapshot()

			// The email field (EditText) is entered before "Send code" (which uses
			// the "Continue" selector), and OTP is retrieved after that.
			iEmail := indexOf(events, "android.widget.EditText")
			iSend := indexOf(events, `"Continue"`)
			iOTP := indexOf(events, "OTP_GETCODE")
			if iOTP < 0 {
				t.Fatal("GetCode was never called")
			}
			if iEmail < 0 {
				t.Fatal("email never entered")
			}
			if iEmail >= iSend || iSend >= iOTP {
				t.Errorf("ordering wrong: email=%d send=%d otp=%d (%v)", iEmail, iSend, iOTP, events)
			}
			// A code field (EditText) is entered after OTP.
			if indexAfter(events, "android.widget.EditText", iOTP) < 0 {
				t.Errorf("no code field entered after OTP: %v", events)
			}

			// Final submit presence via the shared "Sign up" selector count.
			got := countContains(events, `new UiSelector().textContains("Sign up")`)
			if got != tt.wantSignUpCount {
				t.Errorf(`"Sign up" taps = %d, want %d (%v)`, got, tt.wantSignUpCount, events)
			}
		})
	}
}

func TestTikTokRegisterSSO(t *testing.T) {
	tests := []struct {
		name       string
		dryRun     bool
		wantStatus string
	}{
		{name: "sso happy path", dryRun: false, wantStatus: "success"},
		{name: "sso dry run", dryRun: true, wantStatus: "dry-run"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &flowRecorder{}
			driver := recordingDriver(t, rec)
			provider := &recProvider{rec: rec, code: "000000"}
			loc := loadTikTokLocators(t)

			flow := TikTokFlow{
				Cfg: FlowConfig{
					PasswordLength: 16,
					UsernamePrefix: "user",
					ElementWait:    500 * time.Millisecond,
					ProbeWait:      100 * time.Millisecond,
					OTPTimeout:     time.Second,
					Retry:          RetryPolicy{Attempts: 1},
					DryRun:         tt.dryRun,
					UseSSO:         true,
				},
				Logger: zap.NewNop(),
			}

			acct, err := flow.Register(context.Background(), driver, provider, "ssouser@gmail.com", loc)
			if err != nil {
				t.Fatalf("Register (SSO): %v", err)
			}

			if acct.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q", acct.Status, tt.wantStatus)
			}
			if acct.Platform != PlatformTikTok {
				t.Errorf("platform = %q", acct.Platform)
			}
			// SSO accounts carry no local password.
			if acct.Password != "" {
				t.Errorf("SSO password should be empty, got %q", acct.Password)
			}
			if acct.Email != "ssouser@gmail.com" {
				t.Errorf("email = %q", acct.Email)
			}

			events := rec.snapshot()
			// OTP must never be called in the SSO flow.
			if indexOf(events, "OTP_GETCODE") >= 0 {
				t.Errorf("SSO flow must not call OTP provider: %v", events)
			}
			// The "Continue" SSO button must be tapped on a real (non-dry-run) run.
			tappedContinue := indexOf(events, "Continue") >= 0 ||
				indexOf(events, "@text='Continue'") >= 0
			if !tt.dryRun && !tappedContinue {
				t.Errorf("expected SSO Continue tap on non-dry-run: %v", events)
			}
		})
	}
}
