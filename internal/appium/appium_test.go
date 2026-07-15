package appium

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// recordedRequest captures what the client sent for later assertions.
type recordedRequest struct {
	method string
	path   string
	body   map[string]any
}

// newTestServer returns an httptest server whose handler is provided by fn, plus
// a pointer to the most recent recorded request.
func newTestServer(t *testing.T, fn func(w http.ResponseWriter, rec recordedRequest)) (*httptest.Server, *recordedRequest) {
	t.Helper()
	last := &recordedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{}
		if raw, _ := io.ReadAll(r.Body); len(raw) > 0 {
			_ = json.Unmarshal(raw, &body)
		}
		*last = recordedRequest{method: r.Method, path: r.URL.Path, body: body}
		w.Header().Set("Content-Type", "application/json")
		fn(w, *last)
	}))
	t.Cleanup(srv.Close)
	return srv, last
}

// newSessionDriver returns a Driver with a fixed session id, bypassing creation,
// pointed at srv.
func newSessionDriver(srv *httptest.Server) *Driver {
	return &Driver{serverURL: srv.URL, sessionID: "sess-1", http: srv.Client()}
}

func TestNewDriverCreatesSession(t *testing.T) {
	srv, last := newTestServer(t, func(w http.ResponseWriter, _ recordedRequest) {
		_, _ = io.WriteString(w, `{"value":{"sessionId":"sess-1","capabilities":{}}}`)
	})

	d, err := NewDriver(context.Background(), srv.URL, Capabilities{
		PlatformName:      "Android",
		AutomationName:    "UiAutomator2",
		DeviceName:        "emulator-5554",
		NewCommandTimeout: 120 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewDriver: %v", err)
	}
	if d.sessionID != "sess-1" {
		t.Errorf("sessionID = %q", d.sessionID)
	}
	if last.method != http.MethodPost || last.path != "/session" {
		t.Errorf("request = %s %s", last.method, last.path)
	}
	caps := last.body["capabilities"].(map[string]any)
	always := caps["alwaysMatch"].(map[string]any)
	if always["platformName"] != "Android" {
		t.Errorf("platformName = %v", always["platformName"])
	}
	if always["appium:automationName"] != "UiAutomator2" {
		t.Errorf("automationName = %v", always["appium:automationName"])
	}
}

func TestFindElementAndClick(t *testing.T) {
	srv, last := newTestServer(t, func(w http.ResponseWriter, rec recordedRequest) {
		switch rec.path {
		case "/session/sess-1/element":
			_, _ = io.WriteString(w, `{"value":{"`+w3cElementKey+`":"elem-9"}}`)
		default: // click
			_, _ = io.WriteString(w, `{"value":null}`)
		}
	})
	d := newSessionDriver(srv)

	el, err := d.FindElement(context.Background(), ByID, "com.app:id/email")
	if err != nil {
		t.Fatalf("FindElement: %v", err)
	}
	if last.body["using"] != "id" || last.body["value"] != "com.app:id/email" {
		t.Errorf("find body = %v", last.body)
	}

	if err := el.Click(context.Background()); err != nil {
		t.Fatalf("Click: %v", err)
	}
	if last.path != "/session/sess-1/element/elem-9/click" {
		t.Errorf("click path = %s", last.path)
	}
}

func TestGetText(t *testing.T) {
	srv, _ := newTestServer(t, func(w http.ResponseWriter, _ recordedRequest) {
		_, _ = io.WriteString(w, `{"value":"Confirm Your Email"}`)
	})
	d := newSessionDriver(srv)
	el := &Element{driver: d, id: "elem-1"}

	text, err := el.GetText(context.Background())
	if err != nil {
		t.Fatalf("GetText: %v", err)
	}
	if text != "Confirm Your Email" {
		t.Errorf("text = %q", text)
	}
}

func TestFindElementNotFound(t *testing.T) {
	srv, _ := newTestServer(t, func(w http.ResponseWriter, _ recordedRequest) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"value":{"error":"no such element","message":"could not find email"}}`)
	})
	d := newSessionDriver(srv)

	_, err := d.FindElement(context.Background(), ByID, "missing")
	if !errors.Is(err, ErrElementNotFound) {
		t.Fatalf("want ErrElementNotFound, got %v", err)
	}
}

func TestSessionExpiredMapping(t *testing.T) {
	srv, _ := newTestServer(t, func(w http.ResponseWriter, _ recordedRequest) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"value":{"error":"invalid session id","message":"gone"}}`)
	})
	d := newSessionDriver(srv)

	_, err := d.PageSource(context.Background())
	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("want ErrSessionExpired, got %v", err)
	}
}

func TestWaitForElementTimesOut(t *testing.T) {
	srv, _ := newTestServer(t, func(w http.ResponseWriter, _ recordedRequest) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"value":{"error":"no such element","message":"nope"}}`)
	})
	d := newSessionDriver(srv)

	_, err := d.WaitForElement(context.Background(), ByID, "missing", 300*time.Millisecond)
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("want ErrTimeout, got %v", err)
	}
}

func TestWaitForElementRespectsContext(t *testing.T) {
	srv, _ := newTestServer(t, func(w http.ResponseWriter, _ recordedRequest) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"value":{"error":"no such element","message":"nope"}}`)
	})
	d := newSessionDriver(srv)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := d.WaitForElement(ctx, ByID, "missing", 5*time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}

func TestScreenshotDecodesBase64(t *testing.T) {
	want := []byte{0x89, 0x50, 0x4e, 0x47}
	enc := base64.StdEncoding.EncodeToString(want)
	srv, _ := newTestServer(t, func(w http.ResponseWriter, _ recordedRequest) {
		_, _ = io.WriteString(w, `{"value":"`+enc+`"}`)
	})
	d := newSessionDriver(srv)

	got, err := d.Screenshot(context.Background())
	if err != nil {
		t.Fatalf("Screenshot: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("screenshot bytes = %v", got)
	}
}

func TestLaunchAppSendsActivateApp(t *testing.T) {
	srv, last := newTestServer(t, func(w http.ResponseWriter, _ recordedRequest) {
		_, _ = io.WriteString(w, `{"value":null}`)
	})
	d := newSessionDriver(srv)

	if err := d.LaunchApp(context.Background(), "com.google.android.gm"); err != nil {
		t.Fatalf("LaunchApp: %v", err)
	}
	if last.path != "/session/sess-1/execute/sync" {
		t.Errorf("path = %s", last.path)
	}
	if last.body["script"] != "mobile: activateApp" {
		t.Errorf("script = %v", last.body["script"])
	}
}

func TestClipboardRoundTripEncoding(t *testing.T) {
	srv, last := newTestServer(t, func(w http.ResponseWriter, rec recordedRequest) {
		if rec.path == "/session/sess-1/appium/device/get_clipboard" {
			enc := base64.StdEncoding.EncodeToString([]byte("123456"))
			_, _ = io.WriteString(w, `{"value":"`+enc+`"}`)
			return
		}
		_, _ = io.WriteString(w, `{"value":null}`)
	})
	d := newSessionDriver(srv)

	if err := d.SetClipboard(context.Background(), "123456"); err != nil {
		t.Fatalf("SetClipboard: %v", err)
	}
	if last.body["content"] != base64.StdEncoding.EncodeToString([]byte("123456")) {
		t.Errorf("set clipboard content not base64: %v", last.body["content"])
	}

	got, err := d.GetClipboard(context.Background())
	if err != nil {
		t.Fatalf("GetClipboard: %v", err)
	}
	if got != "123456" {
		t.Errorf("clipboard = %q", got)
	}
}

func TestQuitDeletesSession(t *testing.T) {
	srv, last := newTestServer(t, func(w http.ResponseWriter, _ recordedRequest) {
		_, _ = io.WriteString(w, `{"value":null}`)
	})
	d := newSessionDriver(srv)

	if err := d.Quit(context.Background()); err != nil {
		t.Fatalf("Quit: %v", err)
	}
	if last.method != http.MethodDelete || last.path != "/session/sess-1" {
		t.Errorf("quit request = %s %s", last.method, last.path)
	}
	if d.sessionID != "" {
		t.Errorf("session id not cleared: %q", d.sessionID)
	}
}
