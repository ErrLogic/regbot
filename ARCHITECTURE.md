# Architecture Document – RegBot (Email-Based)

**Project Codename:** RegBot
**Version:** 2.0
**Status:** Draft
**Last updated:** 2026-07-15

> ⚠️ **Educational use only.** Automated account creation violates the Terms of
> Service of Instagram and TikTok. This document describes a reference design for
> learning UI automation, cross-app orchestration, and clean Go architecture. See
> [`PRD.md`](./PRD.md) §7 for the full legal/ethical notice.

---

## 1. High-Level Overview

RegBot uses **Appium** (UiAutomator2 driver) for UI automation on a single
Android device. It orchestrates **two apps** on that device:

1. The **target app** – Instagram or TikTok – where the account is created.
2. **Gmail** – where the one-time verification code (OTP) is delivered.

All UI interactions go through a small, purpose-built Appium HTTP client
(`internal/appium`). **ADB** is used only for pre-flight checks (device present,
app installed) and optional APK installation — never for UI actions.

A single Appium session is reused for the whole run. App switching is done with
Appium's `mobile: activateApp` / `LaunchApp`, so no second session is required.

```text
┌───────────────────────────────────────────┐
│                CLI (cobra)                  │
│   regbot register <platform> --email ...    │
└───────────────────────┬─────────────────────┘
                        │
┌───────────────────────▼─────────────────────┐
│              Core Orchestrator               │
│   - Config load & validation                 │
│   - Appium session lifecycle                 │
│   - Flow selection & delegation              │
│   - Result / artifact handling               │
└───────────────────────┬─────────────────────┘
                        │  PlatformFlow interface
┌───────────────────────▼─────────────────────┐
│                Platform Flows                │
│   - InstagramFlow                            │
│   - TikTokFlow                               │
│   (each is a sequence of named Steps and     │
│    calls OTPProvider for the code)           │
└───────────┬─────────────────────┬────────────┘
            │                     │  OTPProvider interface
            │       ┌─────────────▼────────────┐
            │       │      GmailAppProvider      │
            │       │  reads OTP from Gmail via  │
            │       │  the shared Appium driver  │
            │       └─────────────┬────────────┘
            │                     │
┌───────────▼─────────────────────▼────────────┐
│            Appium Client (HTTP)               │
│   Driver · Element · Waits · Screenshot        │
│   (WebDriver / Appium JSON protocol)          │
└───────────────────────┬─────────────────────┘
                        │
┌───────────────────────▼─────────────────────┐
│          Appium Server → Android Device       │
│              (ADB for pre-flight)             │
└───────────────────────────────────────────────┘
```

---

## 2. Component Breakdown

### 2.1 CLI (`cmd/regbot`)

- Built with `cobra`. Root command `regbot`, subcommand group `register`.
- Subcommands: `register instagram`, `register tiktok`.
- Global flags:
  - `--email` – target email address (required unless supplied in config).
  - `--config` – path to YAML config (default `./config.yaml`).
  - `--log-level` – `debug|info|warn|error` (default `info`).
  - `--dry-run` – validate config and locators, connect to Appium, but do not
    submit the final registration.
- Responsibilities: parse flags, load config, validate inputs, build the logger,
  call `core.Register(ctx, ...)`, then render the result and set the exit code.
- Contains **no automation logic** — it is a thin adapter over the core.

### 2.2 Core Orchestrator (`internal/core`)

- Primary type: `RegistrationService`.
- Entry point:

  ```go
  // Register drives a full registration for the given platform and returns the
  // created account, or an error describing the failing step.
  func (s *RegistrationService) Register(
      ctx context.Context,
      platform Platform, // "instagram" | "tiktok"
      email string,
      cfg config.Config,
  ) (Account, error)
  ```

- Responsibilities:
  1. Build Appium capabilities from `cfg` and open **one** driver session.
  2. Load locator maps for the target platform **and** Gmail.
  3. Construct the `GmailAppProvider` bound to the same driver.
  4. Select and construct the correct `PlatformFlow`, injecting the OTP provider.
  5. Run `flow.Register(...)`, streaming step progress to the logger.
  6. On error, capture a screenshot + page source into the artifacts dir.
  7. Always tear down the session (`defer driver.Quit()`).
  8. Return the `Account` or a wrapped error.

- Owns the **session lifecycle** and the **artifact directory** for a run.

### 2.3 Platform Flows (`internal/flows`)

A flow is a declarative **sequence of named steps**. A tiny step engine runs them
in order, logs each step, applies per-step retry/backoff, and captures an
artifact on failure.

```go
// PlatformFlow registers a single account on one platform.
type PlatformFlow interface {
    Register(
        ctx context.Context,
        driver *appium.Driver,
        otp otp.OTPProvider,
        email string,
        locators locators.Map,
    ) (Account, error)
}

// Step is one named unit of work within a flow.
type Step struct {
    Name string
    Run  func(ctx context.Context) error
}
```

- `InstagramFlow` and `TikTokFlow` each implement `PlatformFlow`.
- Each declares its own `[]Step`; the shared runner (`runSteps`) executes them.
- OTP retrieval is a step that calls `otp.GetCode(ctx, email, timeout)` at the
  right moment (after the app has sent the email).
- Flows never talk to Gmail directly and never construct an OTP provider — the
  provider is injected by the core, keeping flows testable with a mock.

### 2.4 OTP Provider – GmailApp (`internal/otp/gmailapp`)

```go
// OTPProvider retrieves a one-time verification code for targetEmail.
type OTPProvider interface {
    GetCode(ctx context.Context, targetEmail string, timeout time.Duration) (string, error)
}
```

`GmailAppProvider` holds a reference to the **shared** Appium driver (it switches
apps rather than opening a second session) plus the Gmail locator map and config
(sender allow-list, refresh strategy, code regex).

**Algorithm (`GetCode`):**

1. Record the currently-foregrounded app package (to restore later).
2. `driver.LaunchApp("com.google.android.gm")` and wait for the inbox to render.
3. Normalise Gmail state: if a conversation is open, press Back until the inbox
   list is visible (resilient to "already open" states).
4. Pull-to-refresh (swipe down from near the top of the list).
5. Poll on an interval (default every 5 s, up to `timeout`): scan visible list
   items for a sender matching the configured allow-list (`instagram`, `tiktok`,
   `no-reply@…`) **and** a recent timestamp.
6. Tap the matching email to open it.
7. Read the body via `GetText()` on the message text view (or WebView), and apply
   the configured code regex (default `\d{6}`) to extract the OTP.
8. Restore the target app: `driver.LaunchApp(previousPackage)`.
9. Return the code, or an error (with screenshot) on timeout / no match.

- Gmail locators are loaded from `locators/gmail.json`.
- The provider must be **idempotent and resilient**: safe to call again if the
  first attempt timed out, and tolerant of Gmail opening into a thread, a
  promotions tab, or a "select account" prompt.

### 2.5 Appium Client (`internal/appium`)

A minimal HTTP client implementing the parts of the WebDriver / Appium JSON
protocol that the flows need. We deliberately avoid third-party Go Appium
libraries to keep the surface small and understandable.

- Core types: `Driver`, `Element`.
- Methods:
  - `NewDriver(ctx, serverURL string, caps Capabilities) (*Driver, error)`
  - `(*Driver) FindElement(ctx, by, selector) (*Element, error)`
  - `(*Driver) WaitForElement(ctx, by, selector, timeout) (*Element, error)`
  - `(*Element) Click(ctx) error`
  - `(*Element) SendKeys(ctx, text string) error`
  - `(*Element) GetText(ctx) (string, error)`
  - `(*Driver) Swipe(ctx, x1, y1, x2, y2, steps int) error`
  - `(*Driver) LaunchApp(ctx, pkg string) error` (wraps `mobile: activateApp`)
  - `(*Driver) PressBack(ctx) error`
  - `(*Driver) Screenshot(ctx) ([]byte, error)`
  - `(*Driver) PageSource(ctx) (string, error)`
  - `(*Driver) SetClipboard/GetClipboard(ctx, ...)` (used for pasting the OTP)
  - `(*Driver) Quit(ctx) error`
- Supported locator strategies (`by`): `id` (resource-id), `accessibility id`
  (content-desc), `xpath`, `-android uiautomator`.
- Sentinel errors: `ErrElementNotFound`, `ErrSessionExpired`, `ErrTimeout`.
- Every method takes `context.Context` for cancellation and timeout.
- Transport: a single `http.Client` with a sane timeout; requests are JSON.

### 2.6 ADB Helpers (`internal/adb`)

- `CheckDevice(ctx) error` – ensure exactly one authorised device is connected
  (wraps `adb devices`), returns a clear error if none/multiple/unauthorised.
- `IsInstalled(ctx, pkg string) (bool, error)` – `adb shell pm list packages`.
- `InstallAPK(ctx, apkPath string) error` – install a target app if missing.
- Used **only** for pre-flight and setup, never for UI actions.

### 2.7 Configuration & Locators (`internal/config`, `locators/`)

Config is loaded with `viper` (YAML file + env overrides) and validated before a
session is opened. See [PRD.md §4 FR-8](./PRD.md) for the requirement.

```yaml
# config.yaml
appium:
  server_url: "http://127.0.0.1:4723"
  new_command_timeout: 120s

device:
  platform_name: "Android"
  device_name: "emulator-5554"
  automation_name: "UiAutomator2"
  udid: ""                       # optional; pins a specific device

apps:
  instagram_package: "com.instagram.android"
  instagram_activity: "com.instagram.mainactivity.MainActivity"
  tiktok_package: "com.zhiliaoapp.musically"
  gmail_package: "com.google.android.gm"

email:
  address: ""                    # explicit address, OR:
  base_address: "myaccount@gmail.com"  # for +alias generation
  alias_tag_prefix: "reg"        # produces myaccount+reg<rand>@gmail.com

otp:
  sender_allowlist: ["instagram", "tiktok", "no-reply"]
  code_regex: "\\d{6}"
  wait_timeout: 60s
  poll_interval: 5s

account:
  password_length: 16
  username_prefix: "user"

timeouts:
  element_wait: 15s
  step_retry: 2                  # retries per step

paths:
  locators_dir: "./locators"
  artifacts_dir: "./artifacts"   # screenshots, page source, results

logging:
  level: "info"
  file: "./regbot.log"
```

**Locator files** live under `locators/` — one per app:

- `locators/instagram.json`
- `locators/tiktok.json`
- `locators/gmail.json`

Each file maps a logical element name to one or more ordered selector candidates
(first match wins), enabling fallback across app versions:

```json
{
  "version": "instagram-v300",
  "elements": {
    "create_new_account": [
      { "by": "id", "selector": "com.instagram.android:id/sign_up_with_email" },
      { "by": "accessibility id", "selector": "Create new account" },
      { "by": "-android uiautomator", "selector": "new UiSelector().textContains(\"Create new account\")" }
    ],
    "email_field": [
      { "by": "id", "selector": "com.instagram.android:id/email_field" }
    ],
    "confirmation_code_field": [
      { "by": "id", "selector": "com.instagram.android:id/confirmation_field" }
    ]
  }
}
```

The loader validates that every element name a flow depends on is present, and
fails fast with a clear message otherwise.

---

## 3. Data Flow (Registration with Gmail OTP)

Sequence for the OTP-bearing portion of a flow:

```text
Flow                 Target App        OTPProvider          Gmail
 │  enter email  ────────►│                                   │
 │  tap "Next"   ────────►│  (server emails the code) ─ ─ ─ ─►│
 │                        │                                   │
 │  GetCode(ctx,email) ──────────────►│                       │
 │                        │            │  activateApp(gmail) ─►│
 │                        │            │  pull-to-refresh   ──►│
 │                        │            │  poll for sender   ──►│
 │                        │            │  open email        ──►│
 │                        │            │  GetText + regex   ◄──│
 │                        │            │  activateApp(app)  ──►│ (target app)
 │  code ◄──────────────────────────── │                       │
 │  paste code into field►│                                   │
 │  continue     ────────►│                                   │
```

1. Flow enters the "email" step and types the address.
2. Flow taps "Next"; the app's backend sends the verification email.
3. Flow calls `otpProvider.GetCode(ctx, email, timeout)`.
4. `GmailAppProvider` foregrounds Gmail, refreshes, finds and opens the email,
   extracts the code via regex, then re-foregrounds the target app.
5. Flow sets the OTP into the confirmation field (via `SendKeys`, or clipboard
   paste if the field rejects synthetic keystrokes) and proceeds.

---

## 4. Cross-Cutting Concerns

### 4.1 Concurrency Model

- One run = one device = one Appium session = one goroutine driving the flow.
- Bulk creation is out of scope; parallelism is achieved by running multiple
  `regbot` processes against multiple devices/servers (a non-goal to orchestrate
  in-process). No shared mutable state between runs.

### 4.2 Wait & Retry Strategy

- **Element waits:** `WaitForElement` polls until visible or `element_wait`.
- **Step retries:** the step runner retries a failed step up to `step_retry`
  times with a short backoff, re-capturing state each attempt.
- **OTP polling:** bounded by `otp.wait_timeout`, polling every `poll_interval`.
- All waits honour `ctx` cancellation so a `--timeout` or Ctrl-C aborts promptly.

### 4.3 App-Switching Mechanics

- Prefer `mobile: activateApp` (brings an existing task to the foreground without
  a cold restart) over relaunching, to preserve the in-progress registration
  screen when returning from Gmail.
- Record the foreground package before switching so the provider can restore it.

### 4.4 Clipboard Handling

- Some confirmation fields split the OTP into per-digit boxes or block paste.
  Strategy: try `SendKeys` first; if verification fails, fall back to
  `SetClipboard` + long-press paste, or per-digit key events.

### 4.5 Error Handling & Artifacts

- Every error is wrapped with the failing step: `fmt.Errorf("step %q: %w", ...)`.
- On any step failure the runner writes to `paths.artifacts_dir`:
  - `‹run-id›-‹step›.png` – screenshot,
  - `‹run-id›-‹step›.xml` – page source,
  - `‹run-id›-result.json` – partial result + error.
- Recovery: for known transient screens (interstitials, "not now" dialogs) flows
  include dismissal steps; on unexpected screens the step fails and is retried.

### 4.6 Security & Secrets

- Generated passwords and the final credentials are the only sensitive outputs.
- Credentials go to **stdout only** as JSON; logs (stderr/file) must never print
  the password. Redact secrets in any error/screenshot path where feasible.
- No real email addresses, tokens, or credentials are committed to the repo.

---

## 5. Package Dependency Graph

```text
cmd/regbot
   └── internal/core
         ├── internal/config
         ├── internal/adb
         ├── internal/appium
         ├── internal/flows
         │      └── internal/appium
         │      └── internal/otp        (interface only)
         ├── internal/otp
         │      └── internal/otp/gmailapp
         │             └── internal/appium
         └── internal/locators
```

- `flows` depends on the `otp` **interface**, not on `gmailapp`, so flows stay
  decoupled from Gmail and unit-testable with a mock provider.
- `appium` depends on nothing internal (leaf package).

---

## 6. Testing Strategy

- **Unit tests:**
  - `appium` – mock the Appium server with an `httptest.Server`; assert requests
    and decode canned responses.
  - `flows` – inject a fake `*appium.Driver` (or an interface seam) and a mock
    `OTPProvider`; assert the step sequence and error wrapping.
  - `config` / `locators` – table-driven validation tests (missing fields,
    malformed selectors, absent element names).
  - `otp/gmailapp` – test the regex extraction and sender matching in isolation.
- **Integration tests** (tagged `//go:build integration`, opt-in):
  - Require a real/emulated device with Gmail signed in and a pre-sent test
    email; drive against test-only accounts.
- **Contract:** locator JSON files are validated in CI against a schema.

---

## 7. Open Questions / Future Work

- **Alias generation:** confirm Instagram/TikTok accept `+tag` Gmail aliases at
  registration time; fall back to explicit addresses if not.
- **Additional OTP providers:** an `imap` provider (`internal/otp/imap`) and an
  Outlook-app provider behind the same `OTPProvider` interface.
- **Challenge handling:** email registration may present a CAPTCHA or "suspicious
  activity" challenge; currently an explicit non-goal — flows should fail clearly
  rather than attempt to solve it.
- **Locator drift:** app updates change resource-ids frequently; consider a
  locator health-check command and versioned locator sets keyed by app version.
