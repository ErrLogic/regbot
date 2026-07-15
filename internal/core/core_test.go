package core

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/ErrLogic/regbot/internal/config"
	"github.com/ErrLogic/regbot/internal/flows"
	"github.com/ErrLogic/regbot/internal/otp"
)

type fakeADB struct {
	checkErr   error
	installed  bool
	installErr error
}

func (f fakeADB) CheckDevice(context.Context) error { return f.checkErr }
func (f fakeADB) IsInstalled(context.Context, string) (bool, error) {
	return f.installed, f.installErr
}

// coreServer simulates the Appium endpoints for both the platform flow and the
// Gmail provider. senderText controls the OTP sender; when failAll is set, every
// element lookup 404s (to force a UI failure).
func coreServer(t *testing.T, senderText string, failAll bool) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path
		switch {
		case path == "/session":
			_, _ = io.WriteString(w, `{"value":{"sessionId":"s1","capabilities":{}}}`)
		case strings.HasSuffix(path, "/screenshot"):
			_, _ = io.WriteString(w, `{"value":"`+base64.StdEncoding.EncodeToString([]byte("PNG"))+`"}`)
		case strings.HasSuffix(path, "/source"):
			_, _ = io.WriteString(w, `{"value":"<hierarchy/>"}`)
		case strings.HasSuffix(path, "/text"):
			switch gmailElementID(path) {
			case "sender":
				_, _ = io.WriteString(w, `{"value":"`+senderText+`"}`)
			case "body":
				_, _ = io.WriteString(w, `{"value":"Your code is 483920."}`)
			default:
				_, _ = io.WriteString(w, `{"value":""}`)
			}
		case strings.HasSuffix(path, "/element"):
			raw, _ := io.ReadAll(r.Body)
			body := string(raw)
			if failAll || strings.Contains(body, "available") {
				w.WriteHeader(http.StatusNotFound)
				_, _ = io.WriteString(w, `{"value":{"error":"no such element","message":"nope"}}`)
				return
			}
			_, _ = io.WriteString(w, `{"value":{"element-6066-11e4-a52e-4f735466cecf":"`+gmailID(body)+`"}}`)
		default:
			_, _ = io.WriteString(w, `{"value":null}`)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func gmailID(value string) string {
	switch {
	case strings.Contains(value, "thread_list_view"):
		return "list"
	case strings.Contains(value, "senders"):
		return "sender"
	case strings.Contains(value, "conversation_list_item"):
		return "row"
	case strings.Contains(value, "mail_body"), strings.Contains(value, "WebView"):
		return "body"
	default:
		return "el"
	}
}

func gmailElementID(path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if p == "element" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func testConfig(t *testing.T, serverURL string) config.Config {
	t.Helper()
	return config.Config{
		Appium: config.AppiumConfig{ServerURL: serverURL, NewCommandTimeout: 60 * time.Second},
		Device: config.DeviceConfig{PlatformName: "Android", AutomationName: "UiAutomator2", DeviceName: "emu"},
		Apps: config.AppsConfig{
			InstagramPackage:  "com.instagram.android",
			InstagramActivity: "com.instagram.mainactivity.MainActivity",
			TikTokPackage:     "com.zhiliaoapp.musically",
			GmailPackage:      "com.google.android.gm",
		},
		Email:    config.EmailConfig{Address: "tester@gmail.com"},
		OTP:      config.OTPConfig{SenderAllowlist: []string{"instagram"}, CodeRegex: `\d{6}`, WaitTimeout: 400 * time.Millisecond, PollInterval: 20 * time.Millisecond},
		Account:  config.AccountConfig{PasswordLength: 16, UsernamePrefix: "user"},
		Timeouts: config.TimeoutsConfig{ElementWait: 400 * time.Millisecond, StepRetry: 0},
		Paths:    config.PathsConfig{LocatorsDir: filepath.Join("..", "..", "locators"), ArtifactsDir: t.TempDir()},
		Logging:  config.LoggingConfig{Level: "info"},
	}
}

func resultFile(t *testing.T, dir string) string {
	t.Helper()
	matches, _ := filepath.Glob(filepath.Join(dir, "*-result.json"))
	if len(matches) != 1 {
		t.Fatalf("expected one result.json, found %v", matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	return string(data)
}

func TestRegisterDryRunSuccess(t *testing.T) {
	cfg := testConfig(t, coreServer(t, "Instagram", false))
	svc := NewService(zap.NewNop(), fakeADB{installed: true})

	acct, err := svc.Register(context.Background(), flows.PlatformInstagram, "tester@gmail.com", cfg, true)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if acct.Status != "dry-run" {
		t.Errorf("status = %q, want dry-run", acct.Status)
	}
	if !strings.HasPrefix(acct.Username, "user_") || len(acct.Password) != 16 {
		t.Errorf("account = %+v", acct)
	}

	result := resultFile(t, cfg.Paths.ArtifactsDir)
	if !strings.Contains(result, `"status": "dry-run"`) {
		t.Errorf("result.json status: %s", result)
	}
	if strings.Contains(result, "password") {
		t.Errorf("result.json must not contain the password: %s", result)
	}
}

func TestRegisterTikTokDryRun(t *testing.T) {
	cfg := testConfig(t, coreServer(t, "TikTok", false))
	cfg.OTP.SenderAllowlist = []string{"tiktok"}
	svc := NewService(zap.NewNop(), fakeADB{installed: true})

	acct, err := svc.Register(context.Background(), flows.PlatformTikTok, "tester@gmail.com", cfg, true)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if acct.Platform != flows.PlatformTikTok || acct.Status != "dry-run" {
		t.Errorf("account = %+v", acct)
	}
}

func TestRegisterAppNotInstalled(t *testing.T) {
	cfg := testConfig(t, coreServer(t, "Instagram", false))
	svc := NewService(zap.NewNop(), fakeADB{installed: false})

	_, err := svc.Register(context.Background(), flows.PlatformInstagram, "tester@gmail.com", cfg, true)
	if err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("want not-installed error, got %v", err)
	}
}

func TestRegisterUIFailureWritesArtifacts(t *testing.T) {
	cfg := testConfig(t, coreServer(t, "Instagram", true)) // every element 404s
	svc := NewService(zap.NewNop(), fakeADB{installed: true})

	_, err := svc.Register(context.Background(), flows.PlatformInstagram, "tester@gmail.com", cfg, false)
	if err == nil {
		t.Fatal("expected UI failure")
	}
	// Screenshot + page source + result.json for the failing step.
	png, _ := filepath.Glob(filepath.Join(cfg.Paths.ArtifactsDir, "*.png"))
	xml, _ := filepath.Glob(filepath.Join(cfg.Paths.ArtifactsDir, "*.xml"))
	if len(png) == 0 || len(xml) == 0 {
		t.Errorf("expected screenshot and page-source artifacts, png=%v xml=%v", png, xml)
	}
	result := resultFile(t, cfg.Paths.ArtifactsDir)
	if !strings.Contains(result, `"status": "failed"`) {
		t.Errorf("result.json should mark failure: %s", result)
	}
}

func TestRegisterOTPTimeout(t *testing.T) {
	// Sender never matches -> provider returns ErrCodeNotFound.
	cfg := testConfig(t, coreServer(t, "Weekly Newsletter", false))
	svc := NewService(zap.NewNop(), fakeADB{installed: true})

	_, err := svc.Register(context.Background(), flows.PlatformInstagram, "tester@gmail.com", cfg, true)
	if !errors.Is(err, otp.ErrCodeNotFound) {
		t.Fatalf("want ErrCodeNotFound, got %v", err)
	}
}

func TestResolveEmail(t *testing.T) {
	got, err := ResolveEmail(config.EmailConfig{Address: "fixed@gmail.com"})
	if err != nil || got != "fixed@gmail.com" {
		t.Fatalf("address passthrough: %q, %v", got, err)
	}

	alias, err := ResolveEmail(config.EmailConfig{BaseAddress: "myuser@gmail.com", AliasTagPrefix: "reg"})
	if err != nil {
		t.Fatalf("alias: %v", err)
	}
	if !strings.HasPrefix(alias, "myuser+reg") || !strings.HasSuffix(alias, "@gmail.com") {
		t.Errorf("alias = %q", alias)
	}

	if _, err := ResolveEmail(config.EmailConfig{}); err == nil {
		t.Error("expected error when neither address nor base is set")
	}
}
