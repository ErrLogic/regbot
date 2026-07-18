package appium

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Locator strategies accepted by the Appium UiAutomator2 driver.
const (
	ByID              = "id"
	ByAccessibilityID = "accessibility id"
	ByXPath           = "xpath"
	ByUIAutomator     = "-android uiautomator"
)

// w3cElementKey is the W3C WebDriver element-reference key.
const w3cElementKey = "element-6066-11e4-a52e-4f735466cecf"

// Sentinel errors returned by the client so callers can branch with errors.Is.
var (
	// ErrElementNotFound indicates the requested element does not exist.
	ErrElementNotFound = errors.New("appium: element not found")
	// ErrSessionExpired indicates the Appium session is no longer valid.
	ErrSessionExpired = errors.New("appium: session expired")
	// ErrTimeout indicates a wait exceeded its deadline.
	ErrTimeout = errors.New("appium: timeout")
)

// pollInterval is how often WaitForElement re-checks for an element.
const pollInterval = 250 * time.Millisecond

// Capabilities describes the Android session to create. Non-standard fields are
// sent to Appium with the "appium:" prefix.
type Capabilities struct {
	PlatformName      string
	AutomationName    string
	DeviceName        string
	UDID              string
	AppPackage        string
	AppActivity       string
	NewCommandTimeout time.Duration
	// NoReset keeps the app's existing state (logged-in session) instead of
	// resetting it. Use true for actions on an existing account (like, comment,
	// profile, post, live); leave false for registration, which needs the app
	// on its logged-out welcome screen.
	NoReset bool
}

// alwaysMatch renders the capabilities as a W3C alwaysMatch map. It includes a
// set of UiAutomator2 stability options that reduce mid-session instrumentation
// crashes on long flows (the driver stays resident instead of being torn down
// between commands, and the hidden-API policy error is ignored).
func (c Capabilities) alwaysMatch() map[string]any {
	m := map[string]any{"platformName": nonEmpty(c.PlatformName, "Android")}
	setIf(m, "appium:automationName", c.AutomationName)
	setIf(m, "appium:deviceName", c.DeviceName)
	setIf(m, "appium:udid", c.UDID)
	setIf(m, "appium:appPackage", c.AppPackage)
	setIf(m, "appium:appActivity", c.AppActivity)
	if c.NewCommandTimeout > 0 {
		m["appium:newCommandTimeout"] = int(c.NewCommandTimeout.Seconds())
	}

	// Stability options — reduce mid-session UiAutomator2 crashes without changing
	// app-launch/reset semantics.
	m["appium:disableWindowAnimation"] = true
	m["appium:ignoreHiddenApiPolicyError"] = true
	m["appium:uiautomator2ServerLaunchTimeout"] = 60000
	m["appium:uiautomator2ServerInstallTimeout"] = 60000
	m["appium:uiautomator2ServerReadTimeout"] = 60000
	m["appium:autoGrantPermissions"] = true

	// App-state control. Registration needs a clean, logged-out app (noReset
	// false, the default). Actions on an existing account set NoReset=true to
	// preserve the logged-in session. forceAppLaunch ensures the target app is
	// brought to the front on session start.
	m["appium:noReset"] = c.NoReset
	m["appium:forceAppLaunch"] = true

	return m
}

// Driver is a live Appium session.
type Driver struct {
	serverURL string
	sessionID string
	http      *http.Client
}

// Element is a reference to a located UI element within a Driver's session.
type Element struct {
	driver *Driver
	id     string
}

// NewDriver opens a new Appium session at serverURL with the given capabilities.
func NewDriver(ctx context.Context, serverURL string, caps Capabilities) (*Driver, error) {
	d := &Driver{
		serverURL: strings.TrimRight(serverURL, "/"),
		http:      &http.Client{Timeout: 60 * time.Second},
	}
	body := map[string]any{
		"capabilities": map[string]any{
			"alwaysMatch": caps.alwaysMatch(),
			"firstMatch":  []any{map[string]any{}},
		},
	}
	var out struct {
		SessionID string `json:"sessionId"`
	}
	if err := d.do(ctx, http.MethodPost, "/session", body, &out); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	if out.SessionID == "" {
		return nil, errors.New("appium: server returned empty session id")
	}
	d.sessionID = out.SessionID
	return d, nil
}

// FindElement locates a single element by strategy and selector.
func (d *Driver) FindElement(ctx context.Context, by, selector string) (*Element, error) {
	body := map[string]string{"using": by, "value": selector}
	var out map[string]string
	if err := d.do(ctx, http.MethodPost, d.path("/element"), body, &out); err != nil {
		return nil, fmt.Errorf("find element %s=%q: %w", by, selector, err)
	}
	id := out[w3cElementKey]
	if id == "" {
		return nil, fmt.Errorf("find element %s=%q: %w", by, selector, ErrElementNotFound)
	}
	return &Element{driver: d, id: id}, nil
}

// WaitForElement polls for an element until it is found or timeout elapses,
// honouring ctx cancellation.
func (d *Driver) WaitForElement(ctx context.Context, by, selector string, timeout time.Duration) (*Element, error) {
	deadline := time.Now().Add(timeout)
	for {
		el, err := d.FindElement(ctx, by, selector)
		if err == nil {
			return el, nil
		}
		if !errors.Is(err, ErrElementNotFound) {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("wait for element %s=%q: %w", by, selector, ErrTimeout)
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for element %s=%q: %w", by, selector, ctx.Err())
		case <-time.After(pollInterval):
		}
	}
}

// Click taps the element.
func (e *Element) Click(ctx context.Context) error {
	if err := e.driver.do(ctx, http.MethodPost, e.path("/click"), map[string]any{}, nil); err != nil {
		return fmt.Errorf("click element: %w", err)
	}
	return nil
}

// SendKeys types text into the element.
func (e *Element) SendKeys(ctx context.Context, text string) error {
	body := map[string]any{"text": text, "value": strings.Split(text, "")}
	if err := e.driver.do(ctx, http.MethodPost, e.path("/value"), body, nil); err != nil {
		return fmt.Errorf("send keys: %w", err)
	}
	return nil
}

// GetText returns the element's visible text.
func (e *Element) GetText(ctx context.Context) (string, error) {
	var text string
	if err := e.driver.do(ctx, http.MethodGet, e.path("/text"), nil, &text); err != nil {
		return "", fmt.Errorf("get text: %w", err)
	}
	return text, nil
}

// Swipe performs a pointer gesture from (x1,y1) to (x2,y2). steps controls the
// gesture duration (higher is slower).
func (d *Driver) Swipe(ctx context.Context, x1, y1, x2, y2, steps int) error {
	durationMS := steps * 5
	if durationMS <= 0 {
		durationMS = 200
	}
	body := map[string]any{
		"actions": []any{
			map[string]any{
				"type": "pointer", "id": "finger1",
				"parameters": map[string]any{"pointerType": "touch"},
				"actions": []any{
					map[string]any{"type": "pointerMove", "duration": 0, "x": x1, "y": y1},
					map[string]any{"type": "pointerDown", "button": 0},
					map[string]any{"type": "pointerMove", "duration": durationMS, "x": x2, "y": y2},
					map[string]any{"type": "pointerUp", "button": 0},
				},
			},
		},
	}
	if err := d.do(ctx, http.MethodPost, d.path("/actions"), body, nil); err != nil {
		return fmt.Errorf("swipe: %w", err)
	}
	return nil
}

// Tap performs a touch tap at screen coordinates (x, y) using the W3C actions API.
func (d *Driver) Tap(ctx context.Context, x, y int) error {
	body := map[string]any{
		"actions": []any{
			map[string]any{
				"type": "pointer", "id": "finger1",
				"parameters": map[string]any{"pointerType": "touch"},
				"actions": []any{
					map[string]any{"type": "pointerMove", "duration": 0, "x": x, "y": y},
					map[string]any{"type": "pointerDown", "button": 0},
					map[string]any{"type": "pause", "duration": 80},
					map[string]any{"type": "pointerUp", "button": 0},
				},
			},
		},
	}
	if err := d.do(ctx, http.MethodPost, d.path("/actions"), body, nil); err != nil {
		return fmt.Errorf("tap: %w", err)
	}
	return nil
}

// PressBack sends the Android back key (keycode 4).
func (d *Driver) PressBack(ctx context.Context) error {
	body := map[string]any{"keycode": 4}
	if err := d.do(ctx, http.MethodPost, d.path("/appium/device/press_keycode"), body, nil); err != nil {
		return fmt.Errorf("press back: %w", err)
	}
	return nil
}

// LaunchApp brings the given package to the foreground (mobile: activateApp),
// preserving its existing task where possible.
func (d *Driver) LaunchApp(ctx context.Context, pkg string) error {
	body := map[string]any{"script": "mobile: activateApp", "args": []any{map[string]any{"appId": pkg}}}
	if err := d.do(ctx, http.MethodPost, d.path("/execute/sync"), body, nil); err != nil {
		return fmt.Errorf("launch app %q: %w", pkg, err)
	}
	return nil
}

// Screenshot returns the current screen as PNG bytes.
func (d *Driver) Screenshot(ctx context.Context) ([]byte, error) {
	var b64 string
	if err := d.do(ctx, http.MethodGet, d.path("/screenshot"), nil, &b64); err != nil {
		return nil, fmt.Errorf("screenshot: %w", err)
	}
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("screenshot: decode: %w", err)
	}
	return data, nil
}

// PageSource returns the current UI hierarchy as XML.
func (d *Driver) PageSource(ctx context.Context) (string, error) {
	var src string
	if err := d.do(ctx, http.MethodGet, d.path("/source"), nil, &src); err != nil {
		return "", fmt.Errorf("page source: %w", err)
	}
	return src, nil
}

// SetClipboard sets the device clipboard to text (plaintext).
func (d *Driver) SetClipboard(ctx context.Context, text string) error {
	body := map[string]any{
		"contentType": "plaintext",
		"content":     base64.StdEncoding.EncodeToString([]byte(text)),
	}
	if err := d.do(ctx, http.MethodPost, d.path("/appium/device/set_clipboard"), body, nil); err != nil {
		return fmt.Errorf("set clipboard: %w", err)
	}
	return nil
}

// GetClipboard returns the device clipboard contents (plaintext).
func (d *Driver) GetClipboard(ctx context.Context) (string, error) {
	body := map[string]any{"contentType": "plaintext"}
	var b64 string
	if err := d.do(ctx, http.MethodPost, d.path("/appium/device/get_clipboard"), body, &b64); err != nil {
		return "", fmt.Errorf("get clipboard: %w", err)
	}
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", fmt.Errorf("get clipboard: decode: %w", err)
	}
	return string(data), nil
}

// Quit ends the Appium session.
func (d *Driver) Quit(ctx context.Context) error {
	if d.sessionID == "" {
		return nil
	}
	if err := d.do(ctx, http.MethodDelete, "/session/"+d.sessionID, nil, nil); err != nil {
		return fmt.Errorf("quit: %w", err)
	}
	d.sessionID = ""
	return nil
}

// path builds a session-scoped endpoint path.
func (d *Driver) path(suffix string) string {
	return "/session/" + d.sessionID + suffix
}

// path builds an element-scoped endpoint path.
func (e *Element) path(suffix string) string {
	return e.driver.path("/element/" + e.id + suffix)
}

// apiError is the Appium/W3C error payload found in the "value" of a non-2xx
// response.
type apiError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// do sends a JSON request and decodes the "value" of the response into out
// (which may be nil). Non-2xx responses are mapped to sentinel errors.
func (d *Driver) do(ctx context.Context, method, path string, in, out any) error {
	var reqBody io.Reader
	if in != nil {
		raw, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, d.serverURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := d.http.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var envelope struct {
		Value json.RawMessage `json:"value"`
	}
	// A body is not guaranteed on every response; ignore decode on empty.
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return fmt.Errorf("decode response (http %d): %w", resp.StatusCode, err)
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return mapError(resp.StatusCode, envelope.Value)
	}

	if out == nil {
		return nil
	}
	if len(envelope.Value) == 0 || string(envelope.Value) == "null" {
		return nil
	}
	// For session creation the sessionId is a field within value, so decoding
	// value directly into out (a struct with a sessionId tag) works.
	if err := json.Unmarshal(envelope.Value, out); err != nil {
		return fmt.Errorf("decode value: %w", err)
	}
	return nil
}

// mapError converts an error-response value into a sentinel-wrapped error.
func mapError(status int, value json.RawMessage) error {
	var ae apiError
	_ = json.Unmarshal(value, &ae)
	switch ae.Error {
	case "no such element", "stale element reference":
		return fmt.Errorf("%w: %s", ErrElementNotFound, ae.Message)
	case "invalid session id":
		return fmt.Errorf("%w: %s", ErrSessionExpired, ae.Message)
	default:
		msg := ae.Message
		if ae.Error != "" {
			msg = ae.Error + ": " + msg
		}
		return fmt.Errorf("appium: http %d: %s", status, strings.TrimSuffix(msg, ": "))
	}
}

// nonEmpty returns v if non-empty, otherwise fallback.
func nonEmpty(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

// setIf sets key to v in m only when v is non-empty.
func setIf(m map[string]any, key, v string) {
	if v != "" {
		m[key] = v
	}
}
