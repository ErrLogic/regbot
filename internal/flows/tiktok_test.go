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
		// wantSignUpCount: sign_up_button always taps once; the finish button
		// shares the "Sign up" selector, so a completed run shows it twice.
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

			// GetCode after "Send code" and before entering the code field.
			iSend := indexOf(events, "Send code")
			iOTP := indexOf(events, "OTP_GETCODE")
			iCode := indexOf(events, "code_field")
			if iOTP < 0 {
				t.Fatal("GetCode was never called")
			}
			if iSend < 0 || iSend >= iOTP || iOTP >= iCode {
				t.Errorf("ordering wrong: send=%d otp=%d code=%d (%v)", iSend, iOTP, iCode, events)
			}

			if indexOf(events, "email_field") < 0 {
				t.Fatal("email never entered")
			}

			// Final submit presence via the shared "Sign up" selector count.
			got := countOf(events, `new UiSelector().textContains("Sign up")`)
			if got != tt.wantSignUpCount {
				t.Errorf(`"Sign up" taps = %d, want %d (%v)`, got, tt.wantSignUpCount, events)
			}
		})
	}
}
