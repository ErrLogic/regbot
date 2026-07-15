# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

## Project Overview

RegBot is a Go CLI tool that automates **email-based registration** for Instagram
and TikTok on an Android device. It uses **Appium** (UiAutomator2) for UI
automation and reads the verification code directly from the **Gmail app** on the
same device — no external email API or phone number is needed.

> ⚠️ **Educational use only.** Automated account creation violates the Terms of
> Service of Instagram and TikTok. See [`PRD.md`](./PRD.md) §7. Do not use this
> to create accounts you are not authorised to automate.

Companion docs:
- [`PRD.md`](./PRD.md) – product requirements, functional specs, milestones.
- [`ARCHITECTURE.md`](./ARCHITECTURE.md) – components, interfaces, data flow.
- [`PROMPTS.md`](./PROMPTS.md) – phased, copy-pasteable implementation prompts.

## Build & Run

- **Go 1.22 or newer+**
- Build: `go build -o regbot ./cmd/regbot`
- Run: `./regbot register instagram --email myuser@gmail.com --config config.yaml`
- Dry run (validate + connect, no final submit): add `--dry-run`
- Test: `go test ./...`
- Integration tests (needs a real device): `go test -tags=integration ./...`
- Lint: `golangci-lint run`

**Prerequisites at runtime:** an Appium server with the UiAutomator2 driver, a
connected Android device (or emulator) with Instagram/TikTok and Gmail installed,
and Gmail signed into the account that will receive the codes.

## Directory Structure

```text
├── cmd/
│   └── regbot/            # CLI entry (cobra); thin adapter over core
├── internal/
│   ├── adb/               # Minimal ADB wrappers (device check, APK install)
│   ├── appium/            # Custom Appium HTTP client (no external package)
│   ├── config/            # Config loading & validation (viper)
│   ├── core/              # Orchestrator, session lifecycle, artifacts
│   ├── flows/             # Platform flows (instagram, tiktok) + step engine
│   ├── locators/          # Locator JSON loader & validation
│   └── otp/               # OTPProvider interface
│       └── gmailapp/      # Gmail-app OTP provider (default)
├── locators/              # JSON locator files per app
│   ├── instagram.json
│   ├── tiktok.json
│   └── gmail.json
├── config.yaml
├── go.mod
├── PRD.md
├── ARCHITECTURE.md
└── PROMPTS.md
```

## Key Libraries

- `github.com/spf13/cobra` – CLI
- `github.com/spf13/viper` – configuration
- `go.uber.org/zap` – structured logging
- Standard `net/http` for the Appium client (no third-party Appium library)

## Appium Client Philosophy

We intentionally avoid third-party Go Appium libraries. `internal/appium`
implements only what the flows need. Every method takes `context.Context`:

- `NewDriver(ctx, serverURL, caps) (*Driver, error)`
- `(*Driver) FindElement(ctx, by, selector) (*Element, error)`
- `(*Driver) WaitForElement(ctx, by, selector, timeout) (*Element, error)`
- `(*Element) Click(ctx) error`, `SendKeys(ctx, text) error`, `GetText(ctx) (string, error)`
- `(*Driver) Swipe(ctx, x1, y1, x2, y2, steps) error`
- `(*Driver) LaunchApp(ctx, pkg) error` (wraps `mobile: activateApp`)
- `(*Driver) PressBack(ctx) error`
- `(*Driver) Screenshot(ctx) ([]byte, error)` / `PageSource(ctx) (string, error)`
- `(*Driver) SetClipboard` / `GetClipboard` (for pasting the OTP)
- `(*Driver) Quit(ctx) error`

Supported `by` strategies: `id`, `accessibility id`, `xpath`,
`-android uiautomator`. Sentinel errors: `ErrElementNotFound`,
`ErrSessionExpired`, `ErrTimeout`.

## Flow Implementation Rules

- Every platform flow implements `flows.PlatformFlow`.
- A flow is a sequence of **named steps**; a shared step runner executes them in
  order, logs each step, applies per-step retry/backoff, and writes an artifact
  (screenshot + page source) on failure.
- OTP retrieval is always delegated to an `OTPProvider` **injected by the core** —
  flows never construct a provider or talk to Gmail directly (keeps them testable
  with a mock).
- The **GmailAppProvider** is the default provider. It reuses the same Appium
  driver, switching to Gmail and back via `LaunchApp`. No second session.
- Prefer `mobile: activateApp` over relaunch so returning from Gmail preserves
  the in-progress registration screen.
- Locators for Gmail (email list item, sender, body) must be defined in
  `locators/gmail.json`. Never hard-code selectors.

## Gmail OTP Provider Design

When `GetCode(ctx, targetEmail, timeout)` is called, the provider:

1. Records the current foreground app, then `driver.LaunchApp("com.google.android.gm")`.
2. Normalises state: backs out of any open conversation to the inbox list.
3. Performs a pull-to-refresh (swipe down from near the top).
4. Polls (every `poll_interval`, up to `timeout`) for an email whose sender
   matches the configured allow-list (`instagram`, `tiktok`, `no-reply`, …) and
   whose timestamp is recent.
5. Taps the email and reads the body via `GetText()`.
6. Extracts the code with the configured regex (default `\d{6}`), i.e.
   `regexp.MustCompile(`\d{6}`)`.
7. Restores the previous app (`driver.LaunchApp(previousPackage)`).
8. Returns the code, or an error with a screenshot on timeout/no-match.

The provider must be resilient to Gmail already being open in a conversation, a
non-primary tab, or an account picker, and safe to retry after a timeout.

## Important Coding Conventions

- All exported functions, types, and interfaces have doc comments.
- Use `context.Context` everywhere for cancellation/timeout; honour cancellation
  in every wait/poll loop.
- Wrap errors with the failing step: `fmt.Errorf("step %q: %w", stepName, err)`.
- Use sentinel errors + `errors.Is/As` at boundaries; don't string-match errors.
- Log with `zap` (structured). **Only the final credentials go to stdout**, as a
  single JSON object. The generated password must never appear in logs or
  artifacts.
- Never hard-code locators; always load them from the JSON files.
- Do not commit real email addresses, credentials, or API keys.
- Keep `cmd/regbot` a thin adapter; automation logic lives in `internal/*`.
- `flows` depends on the `otp` **interface**, never on `gmailapp` directly.

## Quick Start for Claude

1. Skim [`ARCHITECTURE.md`](./ARCHITECTURE.md) for the component map and interfaces.
2. Read `internal/core/orchestrator.go` – session creation and flow invocation.
3. Read `internal/flows/instagram.go` – the full registration step sequence.
4. Read `internal/otp/gmailapp/gmailapp.go` – Gmail reading logic.
5. Locator files under `locators/` show exactly how elements are identified.

When implementing from scratch, follow the phased prompts in
[`PROMPTS.md`](./PROMPTS.md) in order — each phase is self-contained and builds on
the previous one.
