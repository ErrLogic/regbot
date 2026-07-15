package locators

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ErrLogic/regbot/internal/appium"
)

func writeLocator(t *testing.T, app, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, app+".json"), []byte(body), 0o600); err != nil {
		t.Fatalf("write locator: %v", err)
	}
	return dir
}

const validLocatorJSON = `{
  "version": "test-v1",
  "elements": {
    "email_field": [
      { "by": "id", "selector": "com.app:id/email" },
      { "by": "-android uiautomator", "selector": "new UiSelector().className(\"android.widget.EditText\")" }
    ],
    "next_button": [
      { "by": "accessibility id", "selector": "Next" }
    ]
  }
}`

func TestLoadValid(t *testing.T) {
	dir := writeLocator(t, "instagram", validLocatorJSON)
	m, err := Load(dir, "instagram")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if m.Version != "test-v1" {
		t.Errorf("version = %q", m.Version)
	}
	cands, ok := m.Candidates("email_field")
	if !ok || len(cands) != 2 {
		t.Fatalf("email_field candidates = %v, %v", cands, ok)
	}
	if cands[0].By != "id" || cands[0].Selector != "com.app:id/email" {
		t.Errorf("first candidate = %+v", cands[0])
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(t.TempDir(), "nope"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadBadJSON(t *testing.T) {
	dir := writeLocator(t, "instagram", `{ "version": "x", "elements": { `)
	if _, err := Load(dir, "instagram"); err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestLoadUnknownBy(t *testing.T) {
	dir := writeLocator(t, "instagram", `{
  "version": "x",
  "elements": { "e": [ { "by": "magic", "selector": "foo" } ] }
}`)
	_, err := Load(dir, "instagram")
	if err == nil || !strings.Contains(err.Error(), "unknown by") {
		t.Fatalf("expected unknown-by error, got %v", err)
	}
}

func TestLoadEmptySelectors(t *testing.T) {
	dir := writeLocator(t, "instagram", `{
  "version": "x",
  "elements": { "e": [] }
}`)
	if _, err := Load(dir, "instagram"); err == nil {
		t.Fatal("expected error for element with no selectors")
	}
}

func TestRequire(t *testing.T) {
	dir := writeLocator(t, "instagram", validLocatorJSON)
	m, err := Load(dir, "instagram")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if err := m.Require("email_field", "next_button"); err != nil {
		t.Errorf("Require present: %v", err)
	}

	err = m.Require("email_field", "missing_one", "also_missing")
	if err == nil {
		t.Fatal("expected Require error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "missing_one") || !strings.Contains(msg, "also_missing") {
		t.Errorf("error should name missing elements: %s", msg)
	}
	if !strings.Contains(msg, "instagram.json") {
		t.Errorf("error should name the source file: %s", msg)
	}
}

// newElementServer returns an appium.Driver whose element lookups succeed only
// when the request body contains hitSubstring (a distinctive fragment of the
// selector, chosen to avoid JSON-escaping ambiguity).
func newElementServer(t *testing.T, hitSubstring string) *appium.Driver {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/session":
			_, _ = io.WriteString(w, `{"value":{"sessionId":"s1","capabilities":{}}}`)
		case strings.HasSuffix(r.URL.Path, "/element"):
			raw, _ := io.ReadAll(r.Body)
			if strings.Contains(string(raw), hitSubstring) {
				_, _ = io.WriteString(w, `{"value":{"element-6066-11e4-a52e-4f735466cecf":"el-1"}}`)
				return
			}
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"value":{"error":"no such element","message":"nope"}}`)
		default:
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

func TestResolveFirstCandidateHit(t *testing.T) {
	dir := writeLocator(t, "instagram", validLocatorJSON)
	m, _ := Load(dir, "instagram")
	d := newElementServer(t, "id/email")

	el, err := m.Resolve(context.Background(), d, "email_field", time.Second)
	if err != nil || el == nil {
		t.Fatalf("Resolve first candidate: %v", err)
	}
}

func TestResolveFallsBackToSecondCandidate(t *testing.T) {
	dir := writeLocator(t, "instagram", validLocatorJSON)
	m, _ := Load(dir, "instagram")
	// Only the second candidate's selector matches.
	d := newElementServer(t, "EditText")

	el, err := m.Resolve(context.Background(), d, "email_field", 600*time.Millisecond)
	if err != nil || el == nil {
		t.Fatalf("Resolve fallback: %v", err)
	}
}

func TestResolveUnknownElement(t *testing.T) {
	dir := writeLocator(t, "instagram", validLocatorJSON)
	m, _ := Load(dir, "instagram")
	d := newElementServer(t, "anything")

	if _, err := m.Resolve(context.Background(), d, "does_not_exist", time.Second); err == nil {
		t.Fatal("expected error for unknown element")
	}
}

// TestShippedLocatorFilesValid ensures the JSON files committed to the repo load
// cleanly (guards against typos/invalid strategies in the placeholder files).
func TestShippedLocatorFilesValid(t *testing.T) {
	dir := filepath.Join("..", "..", "locators")
	for _, app := range []string{"instagram", "tiktok", "gmail"} {
		if _, err := Load(dir, app); err != nil {
			t.Errorf("shipped %s.json: %v", app, err)
		}
	}
}
