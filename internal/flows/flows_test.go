package flows

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/ErrLogic/regbot/internal/appium"
	"github.com/ErrLogic/regbot/internal/locators"
)

// testDriver returns an appium.Driver backed by an httptest server. Element
// lookups succeed when the request body contains hitSubstring; screenshots and
// page source return fixed content.
func testDriver(t *testing.T, hitSubstring string) *appium.Driver {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path
		switch {
		case path == "/session":
			_, _ = io.WriteString(w, `{"value":{"sessionId":"s1","capabilities":{}}}`)
		case strings.HasSuffix(path, "/screenshot"):
			_, _ = io.WriteString(w, `{"value":"`+base64.StdEncoding.EncodeToString([]byte("PNGDATA"))+`"}`)
		case strings.HasSuffix(path, "/source"):
			_, _ = io.WriteString(w, `{"value":"<hierarchy/>"}`)
		case strings.HasSuffix(path, "/element"):
			raw, _ := io.ReadAll(r.Body)
			if hitSubstring != "" && strings.Contains(string(raw), hitSubstring) {
				_, _ = io.WriteString(w, `{"value":{"element-6066-11e4-a52e-4f735466cecf":"el-1"}}`)
				return
			}
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"value":{"error":"no such element","message":"nope"}}`)
		default: // click, value, etc.
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

func testLocators() locators.Map {
	return locators.Map{
		Source: "test",
		Elements: map[string][]locators.Selector{
			"button": {{By: appium.ByID, Selector: "id/button"}},
			"field":  {{By: appium.ByID, Selector: "id/field"}},
			"popup":  {{By: appium.ByID, Selector: "id/popup"}},
		},
	}
}

func TestTapByLocator(t *testing.T) {
	d := testDriver(t, "id/button")
	if err := tapByLocator(context.Background(), d, testLocators(), "button", time.Second); err != nil {
		t.Fatalf("tapByLocator: %v", err)
	}
}

func TestTapByLocatorMissing(t *testing.T) {
	d := testDriver(t, "id/nothing")
	if err := tapByLocator(context.Background(), d, testLocators(), "button", 200*time.Millisecond); err == nil {
		t.Fatal("expected error when element missing")
	}
}

func TestTypeByLocator(t *testing.T) {
	d := testDriver(t, "id/field")
	if err := typeByLocator(context.Background(), d, testLocators(), "field", "hello@example.com", time.Second); err != nil {
		t.Fatalf("typeByLocator: %v", err)
	}
}

func TestDismissIfPresent(t *testing.T) {
	// Present: popup matches -> dismissed.
	d := testDriver(t, "id/popup")
	if !dismissIfPresent(context.Background(), d, testLocators(), "popup", 300*time.Millisecond) {
		t.Error("expected popup to be dismissed")
	}
	// Absent: nothing matches -> not dismissed, no error surfaced.
	d2 := testDriver(t, "id/none")
	if dismissIfPresent(context.Background(), d2, testLocators(), "popup", 200*time.Millisecond) {
		t.Error("expected popup to be absent")
	}
}

func TestIsPresent(t *testing.T) {
	d := testDriver(t, "id/button")
	if !isPresent(context.Background(), d, testLocators(), "button", 300*time.Millisecond) {
		t.Error("button should be present")
	}
	if isPresent(context.Background(), d, testLocators(), "field", 200*time.Millisecond) {
		// field selector id/field does not match hitSubstring id/button
		t.Error("field should be absent")
	}
}

func TestArtifactSinkWritesFiles(t *testing.T) {
	d := testDriver(t, "")
	dir := t.TempDir()
	sink := NewArtifactSink(d, dir, "run-123", zap.NewNop())

	sink.Capture(context.Background(), "Enter Email", nil)

	png := filepath.Join(dir, "run-123-enter-email.png")
	xml := filepath.Join(dir, "run-123-enter-email.xml")
	if data, err := os.ReadFile(png); err != nil || string(data) != "PNGDATA" {
		t.Errorf("screenshot file: data=%q err=%v", data, err)
	}
	if data, err := os.ReadFile(xml); err != nil || string(data) != "<hierarchy/>" {
		t.Errorf("page source file: data=%q err=%v", data, err)
	}
}
