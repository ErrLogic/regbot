package gmailapp

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ErrLogic/regbot/internal/appium"
	"github.com/ErrLogic/regbot/internal/locators"
)

func TestExtractCode(t *testing.T) {
	re := regexp.MustCompile(`\d{6}`)
	tests := []struct {
		body   string
		want   string
		wantOK bool
	}{
		{"Your Instagram code is 483920. Do not share it.", "483920", true},
		{"Code: 000123 expires soon", "000123", true},
		{"No digits here", "", false},
		{"Only 12345 five digits", "", false},
		{"Ref 7788991 has 7 digits but 445566 is the code", "778899", true}, // first 6-run wins
	}
	for _, tt := range tests {
		got, ok := extractCode(re, tt.body)
		if ok != tt.wantOK || got != tt.want {
			t.Errorf("extractCode(%q) = %q,%v; want %q,%v", tt.body, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestSenderMatches(t *testing.T) {
	allow := []string{"instagram", "tiktok", "no-reply"}
	tests := []struct {
		sender string
		want   bool
	}{
		{"Instagram", true},
		{"security@mail.instagram.com", true},
		{"TikTok", true},
		{"no-reply@tiktok.com", true},
		{"Some Newsletter", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := senderMatches(tt.sender, allow); got != tt.want {
			t.Errorf("senderMatches(%q) = %v; want %v", tt.sender, got, tt.want)
		}
	}
	if senderMatches("instagram", nil) {
		t.Error("empty allow-list should match nothing")
	}
}

// gmailServer simulates the Appium endpoints Gmail navigation touches. The
// sender text and body text are configurable per test.
func gmailServer(t *testing.T, senderText, bodyText string) *appium.Driver {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path
		switch {
		case path == "/session":
			_, _ = io.WriteString(w, `{"value":{"sessionId":"s1","capabilities":{}}}`)
		case strings.HasSuffix(path, "/text"):
			switch elementID(path) {
			case "sender":
				writeValue(w, senderText)
			case "body":
				writeValue(w, bodyText)
			default:
				writeValue(w, "")
			}
		case strings.HasSuffix(path, "/screenshot"):
			writeValue(w, base64.StdEncoding.EncodeToString([]byte{0x89, 0x50}))
		case strings.HasSuffix(path, "/element"):
			raw, _ := io.ReadAll(r.Body)
			id := mapSelectorToID(string(raw))
			if id == "" {
				w.WriteHeader(http.StatusNotFound)
				_, _ = io.WriteString(w, `{"value":{"error":"no such element","message":"nope"}}`)
				return
			}
			_, _ = io.WriteString(w, `{"value":{"element-6066-11e4-a52e-4f735466cecf":"`+id+`"}}`)
		default: // click, actions, execute/sync, press_keycode
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

func writeValue(w http.ResponseWriter, s string) {
	_, _ = io.WriteString(w, `{"value":`+quote(s)+`}`)
}

func quote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

func elementID(path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if p == "element" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func mapSelectorToID(body string) string {
	switch {
	case strings.Contains(body, "thread_list_view"):
		return "list"
	case strings.Contains(body, "senders"):
		return "sender"
	case strings.Contains(body, "conversation_list_item"):
		return "row"
	case strings.Contains(body, "mail_body"), strings.Contains(body, "WebView"):
		return "body"
	default:
		return ""
	}
}

func gmailLocators(t *testing.T) locators.Map {
	t.Helper()
	m, err := locators.Load(filepath.Join("..", "..", "..", "locators"), "gmail")
	if err != nil {
		t.Fatalf("load gmail locators: %v", err)
	}
	return m
}

func TestGetCodeHappyPath(t *testing.T) {
	d := gmailServer(t, "Instagram", "Welcome! Your code is 483920.")
	p := New(d, gmailLocators(t), Config{
		SenderAllowlist: []string{"instagram"},
		PollInterval:    20 * time.Millisecond,
		ElementWait:     200 * time.Millisecond,
		ReturnPackage:   "com.instagram.android",
	})

	code, err := p.GetCode(context.Background(), "tester@gmail.com", time.Second)
	if err != nil {
		t.Fatalf("GetCode: %v", err)
	}
	if code != "483920" {
		t.Errorf("code = %q, want 483920", code)
	}
}

func TestGetCodeTimeoutCapturesScreenshot(t *testing.T) {
	// Sender never matches the allow-list -> GetCode should time out.
	d := gmailServer(t, "Weekly Newsletter", "no code here")

	var sinkName string
	var sinkBytes []byte
	p := New(d, gmailLocators(t), Config{
		SenderAllowlist: []string{"instagram"},
		PollInterval:    20 * time.Millisecond,
		ElementWait:     150 * time.Millisecond,
	}, WithScreenshotSink(func(name string, png []byte) {
		sinkName = name
		sinkBytes = png
	}))

	_, err := p.GetCode(context.Background(), "tester@gmail.com", 200*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "not found within") {
		t.Errorf("unexpected error: %v", err)
	}
	if sinkName != "otp-failure" || len(sinkBytes) == 0 {
		t.Errorf("screenshot sink not invoked: name=%q len=%d", sinkName, len(sinkBytes))
	}
}

func TestGetCodeRespectsContextCancel(t *testing.T) {
	d := gmailServer(t, "Weekly Newsletter", "no code")
	p := New(d, gmailLocators(t), Config{
		SenderAllowlist: []string{"instagram"},
		PollInterval:    50 * time.Millisecond,
		ElementWait:     100 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	if _, err := p.GetCode(ctx, "tester@gmail.com", 5*time.Second); err == nil {
		t.Fatal("expected error on context cancel")
	}
}
