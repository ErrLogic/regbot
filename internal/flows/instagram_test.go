package flows

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/ErrLogic/regbot/internal/appium"
	"github.com/ErrLogic/regbot/internal/locators"
)

// flowRecorder records the sequence of element selectors requested and the OTP
// call, and simulates the username-taken screen for the first takenTimes probes.
type flowRecorder struct {
	mu         sync.Mutex
	events     []string
	takenTimes int
	takenSeen  int
}

func (r *flowRecorder) record(s string) {
	r.mu.Lock()
	r.events = append(r.events, s)
	r.mu.Unlock()
}

func (r *flowRecorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.events))
	copy(out, r.events)
	return out
}

// takenProbe returns true while the username should still appear taken.
func (r *flowRecorder) takenProbe() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.takenSeen++
	return r.takenSeen <= r.takenTimes
}

// recProvider is a mock OTPProvider that records its invocation into the shared
// event log.
type recProvider struct {
	rec  *flowRecorder
	code string
}

func (p *recProvider) GetCode(_ context.Context, _ string, _ time.Duration) (string, error) {
	p.rec.record("OTP_GETCODE")
	return p.code, nil
}

func recordingDriver(t *testing.T, rec *flowRecorder) *appium.Driver {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path
		switch {
		case path == "/session":
			_, _ = io.WriteString(w, `{"value":{"sessionId":"s1","capabilities":{}}}`)
		case strings.HasSuffix(path, "/element"):
			var body struct {
				Using string `json:"using"`
				Value string `json:"value"`
			}
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &body)
			rec.record(body.Value)

			// The username-taken error uses "isn't available"/"not available"
			// selectors. It is "present" (taken) only while the recorder says so;
			// otherwise report it absent (username available).
			if strings.Contains(body.Value, "available") {
				if rec.takenProbe() {
					_, _ = io.WriteString(w, `{"value":{"element-6066-11e4-a52e-4f735466cecf":"el-1"}}`)
					return
				}
				w.WriteHeader(http.StatusNotFound)
				_, _ = io.WriteString(w, `{"value":{"error":"no such element","message":"absent"}}`)
				return
			}
			_, _ = io.WriteString(w, `{"value":{"element-6066-11e4-a52e-4f735466cecf":"el-1"}}`)
		default: // click, value (send keys)
			_, _ = io.WriteString(w, `{"value":null}`)
		}
	}))
	t.Cleanup(srv.Close)
	d, err := appium.NewDriver(context.Background(), srv.URL, appium.Capabilities{})
	if err != nil {
		t.Fatalf("NewDriver: %v", err)
	}
	return d
}

func loadInstagramLocators(t *testing.T) locators.Map {
	t.Helper()
	m, err := locators.Load(filepath.Join("..", "..", "locators"), "instagram")
	if err != nil {
		t.Fatalf("load instagram locators: %v", err)
	}
	return m
}

func indexOf(events []string, substr string) int {
	for i, e := range events {
		if strings.Contains(e, substr) {
			return i
		}
	}
	return -1
}

// indexAfter returns the index of the first event containing substr at or after
// position from, or -1 if none.
func indexAfter(events []string, substr string, from int) int {
	for i := from; i < len(events); i++ {
		if strings.Contains(events[i], substr) {
			return i
		}
	}
	return -1
}

// countContains counts events that contain substr.
func countContains(events []string, substr string) int {
	n := 0
	for _, e := range events {
		if strings.Contains(e, substr) {
			n++
		}
	}
	return n
}

func TestInstagramRegister(t *testing.T) {
	tests := []struct {
		name       string
		dryRun     bool
		takenTimes int
		wantStatus string
		wantFinish bool
	}{
		{name: "happy path", dryRun: false, takenTimes: 0, wantStatus: "success", wantFinish: true},
		{name: "dry run skips submit", dryRun: true, takenTimes: 0, wantStatus: "dry-run", wantFinish: false},
		{name: "username taken then available", dryRun: false, takenTimes: 2, wantStatus: "success", wantFinish: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &flowRecorder{takenTimes: tt.takenTimes}
			driver := recordingDriver(t, rec)
			provider := &recProvider{rec: rec, code: "483920"}
			loc := loadInstagramLocators(t)

			flow := InstagramFlow{
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

			// Account fields.
			if acct.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q", acct.Status, tt.wantStatus)
			}
			if !strings.HasPrefix(acct.Username, "user_") {
				t.Errorf("username = %q", acct.Username)
			}
			if len(acct.Password) != 16 {
				t.Errorf("password length = %d", len(acct.Password))
			}
			if acct.Email != "tester@gmail.com" || acct.Platform != PlatformInstagram {
				t.Errorf("account = %+v", acct)
			}

			events := rec.snapshot()

			// OTP must be retrieved, after the email "Next" tap.
			iNext := indexOf(events, `"Next"`)
			iOTP := indexOf(events, "OTP_GETCODE")
			if iOTP < 0 {
				t.Fatal("GetCode was never called")
			}
			if iNext < 0 || iNext >= iOTP {
				t.Errorf("ordering wrong: Next(%d) must come before OTP(%d): %v", iNext, iOTP, events)
			}
			// A confirmation-code field (EditText) must be looked up after OTP.
			iCode := indexAfter(events, "EditText", iOTP)
			if iCode < 0 {
				t.Errorf("no confirmation-code field lookup after OTP: %v", events)
			}

			// Final submit presence: agreeing to the terms ("I agree") is what
			// creates the account, so it is skipped on a dry run.
			hasFinish := indexOf(events, "I agree") >= 0
			if hasFinish != tt.wantFinish {
				t.Errorf("finish tapped = %v, want %v", hasFinish, tt.wantFinish)
			}

			// Username retry: the taken-error probe must be checked at least
			// takenTimes+1 times (once per attempt, succeeding on the last).
			if tt.takenTimes > 0 {
				got := countContains(events, "available")
				if got < tt.takenTimes+1 {
					t.Errorf("username taken probes = %d, want >= %d: %v", got, tt.takenTimes+1, events)
				}
			}
		})
	}
}
