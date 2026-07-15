# Product Requirements Document – RegBot (Email-Based Auto-Registration)

**Project Codename:** RegBot
**Version:** 2.0
**Author:** _‹Your Name›_
**Status:** Draft
**Date:** 2026-07-15

> ⚠️ **Educational use only.** Automated account creation violates the Terms of
> Service of Instagram and TikTok. This project exists to teach UI automation,
> cross-app orchestration, and clean Go architecture. See §7 before using it.

---

## 1. Executive Summary

RegBot is a command-line tool written in Go that automates the creation of a
single **Instagram** or **TikTok** account using an **email address** as the
unique identifier. The one-time verification code (OTP) is delivered to the
**Gmail app** already installed and signed in on the same Android device.

RegBot uses **Appium** (UiAutomator2) to drive the target app, then switches to
Gmail to read the code, and switches back to complete registration. No phone
number, SMS gateway, or external email API is required — the only prerequisite is
a device with a Google account signed into Gmail.

---

## 2. Goals & Non-Goals

### Goals

- Register **one** Instagram or TikTok account using a pre-existing email address
  (or one generated via Gmail `+alias` addressing).
- Extract the verification code **directly from the Gmail app** on-device.
- Perform reliable **cross-app automation**: switch to Gmail, find the latest
  verification email, parse the OTP, switch back to the target app.
- Support a **configurable email address** (fixed, or generated from a base
  address using Gmail's `username+tag@gmail.com` aliasing).
- Provide **idempotent, resumable** flows built from discrete named steps.
- Be **extensible** to other OTP sources (IMAP, Outlook app) behind one interface.
- Emit **structured logs** and **machine-readable output** for the created account.

### Non-Goals

- Phone-number or SMS-based registration.
- Bulk / mass account creation (single account per invocation by design).
- CAPTCHA or challenge solving (assumed absent; fail clearly if present).
- Content posting, profile setup beyond required fields, or account warm-up.
- In-process orchestration across multiple devices.

---

## 3. Personas & User Stories

| Persona   | Need |
|-----------|------|
| Developer | Create a throwaway test account for a mobile-automation experiment. |
| Tester    | Update UI selectors without touching Go code. |
| Operator  | See per-step logs, including when Gmail is opened and the OTP is read. |
| Maintainer| Add a new OTP source by implementing one interface. |

1. **As a developer,** I run
   `regbot register instagram --email myalias+insta2026@gmail.com` and get a new
   account with its credentials printed as JSON.
2. **As a tester,** I want email locators and Gmail navigation externalised in
   JSON, so I can fix them without recompiling.
3. **As an operator,** I need a structured log of each step — including the Gmail
   switch and OTP extraction — plus a screenshot on any failure.
4. **As a maintainer,** I can add Outlook or IMAP support by implementing the
   `OTPProvider` interface without modifying the flows.

---

## 4. Functional Requirements

### FR-1: Device & Session Setup

- Connect to a running Appium server (URL from config); fail clearly if it is
  unreachable.
- Pre-flight via ADB: exactly one authorised device connected; target app and
  Gmail installed.
- Create **one** Android driver session for the target app and reuse it for the
  whole run (Gmail is reached by app-switching, not a second session).
- **Acceptance:** given a connected device and running Appium, a session opens
  within 30 s or the tool exits non-zero with a diagnostic.

### FR-2: Email Configuration

- The user supplies the email address via `--email` or `config.yaml`.
- Optionally generate a unique address from a base using Gmail aliasing:
  `base+‹prefix›‹random›@gmail.com`.
- Validate the address is well-formed before starting.
- **Acceptance:** an invalid or missing address aborts before any UI action.

### FR-3: Instagram Registration Flow

1. Launch Instagram → tap **Create New Account**.
2. Choose **Email** (if not default) → enter address → tap **Next**.
3. Wait for the **Confirm Your Email** screen.
4. **Switch to Gmail** (see FR-5) and retrieve the 6-digit code.
5. Switch back to Instagram.
6. Enter the OTP and proceed.
7. Set full name, generate a compliant password, generate/confirm a username,
   set birthday (choose an adult date), skipping optional screens where possible.
8. Complete registration and output the credentials.
- **Acceptance:** on a clean device, completes end-to-end and prints a JSON
  credential object; on any failed step, writes an artifact and exits non-zero.

### FR-4: TikTok Registration Flow

1. Launch TikTok → **Sign up** → **Use phone or email** → **Email**.
2. Set birthday (adult date) → enter email → tap **Next** / **Send code**.
3. **Switch to Gmail** (FR-5), retrieve code, switch back, enter code.
4. Set a compliant password; set/confirm nickname; skip interests/contacts.
5. Complete registration and output credentials.
- **Acceptance:** same as FR-3.

### FR-5: Gmail OTP Provider

- Implements the `OTPProvider` interface.
- Using the shared Appium driver, it:
  1. Foregrounds Gmail (`com.google.android.gm`), remembering the prior app.
  2. Normalises state (backs out of any open conversation to the inbox).
  3. Pull-to-refreshes the inbox.
  4. Polls for an email whose **sender** matches the configured allow-list
     (`instagram`, `tiktok`, `no-reply`, …) and whose timestamp is recent.
  5. Opens it and extracts the code from the body via a configurable regex
     (default `\d{6}`).
  6. Restores the previous app.
- **Configurable:** sender allow-list, code regex, `wait_timeout` (default 60 s),
  `poll_interval` (default 5 s).
- **Resilience:** tolerant of Gmail opening into a thread, a non-primary tab, or
  an account picker; safe to retry after a timeout.
- **Acceptance:** given a matching email in the inbox, returns the code within
  the timeout; otherwise returns an error carrying a screenshot and step name.

### FR-6: Locator Abstraction

- All UI selectors (Instagram, TikTok, Gmail) live in versioned JSON files under
  `locators/`.
- Each logical element maps to an **ordered list of selector candidates**
  (first match wins) to allow fallback across app versions (e.g. try
  `resource-id`, then `content-desc`, then a UiAutomator text match).
- The loader validates that every element a flow needs is present, failing fast
  otherwise.
- **Acceptance:** a missing/typo'd element name aborts at startup with the name
  and file identified.

### FR-7: Logging & Output

- Structured JSON logs to **stderr** and to a log file; per-step timing and
  status; explicit log lines for the Gmail switch and OTP extraction.
- The final account credentials print to **stdout** as a single JSON object.
- The password must **never** appear in logs or artifacts.
- **Output schema (stdout):**

  ```json
  {
    "platform": "instagram",
    "email": "myalias+insta2026@gmail.com",
    "username": "user_8f3a21",
    "password": "‹generated›",
    "created_at": "2026-07-15T10:22:31Z",
    "status": "success"
  }
  ```

### FR-8: Configuration

- `config.yaml` (loaded via `viper`, env-overridable) includes:
  - Appium server URL and session/command timeouts.
  - Device capabilities (platform, device name, automation name, optional UDID).
  - App packages/activities for Instagram, TikTok, and Gmail.
  - Email address or base address + alias settings.
  - OTP settings (sender allow-list, regex, timeouts, poll interval).
  - Account generation settings (password length, username prefix).
  - Locator directory and artifacts directory paths.
  - Logging level and file path.
- See [`ARCHITECTURE.md` §2.7](./ARCHITECTURE.md) for the full annotated schema.

### FR-9: Credential Generation (blind-spot fill)

- **Password:** random, `password_length` chars (default 16), satisfying common
  policy (upper, lower, digit, symbol); never logged.
- **Username:** derived from a prefix + random suffix; on "username taken",
  regenerate and retry a bounded number of times.
- **Full name / nickname:** generated (e.g. from a small word list) unless
  supplied via config.
- **Birthday:** a fixed or random **adult** date to pass age gates.

### FR-10: Run Lifecycle & Exit Codes (blind-spot fill)

- Each run has a `run-id` (timestamp + short random) used to name artifacts.
- Exit codes:
  - `0` – success (credentials printed).
  - `1` – configuration/validation error (before any UI action).
  - `2` – automation failure (a step failed; artifact written).
  - `3` – OTP not received within timeout.
  - `130` – interrupted (Ctrl-C / context cancelled).
- `--dry-run` validates everything and opens a session but stops before the final
  submission, exiting `0`.

---

## 5. Non-Functional Requirements

- **Reliability:** ≥ 90 % success on clean devices with stable internet.
- **Performance:** ≤ 2 minutes per registration, including Gmail polling.
- **Cross-app stability:** gracefully handles Gmail in any prior state.
- **Observability:** every failure yields a screenshot, page source, and a
  result JSON in the artifacts directory.
- **Maintainability:** adding an OTP source or platform touches only one package.
- **Portability:** runs anywhere Go builds; only Appium + ADB are external deps.

---

## 6. Constraints & Assumptions

- Device: Android 10+, unrooted, with Gmail installed and signed into the account
  that will receive the verification emails.
- The Google account can receive external email.
- Instagram/TikTok do not present a CAPTCHA on email registration at time of
  writing; if they do, the run fails clearly (challenge solving is a non-goal).
- Gmail UI is close to stock Material design (heavily-customised builds may need
  updated locators).
- Appium server (with the UiAutomator2 driver) is installed and reachable.

---

## 7. Legal & Ethical Notice

**Automated account creation violates the Terms of Service of Instagram and
TikTok, and may violate local laws or platform anti-fraud provisions.** RegBot is
provided strictly for **educational study** of mobile UI automation and software
architecture. Do not use it to create accounts on services you are not
authorised to automate, to evade bans, to impersonate, or to conduct spam or
fraud. You are solely responsible for how you use this software. Use test-only
accounts and comply with all applicable terms and laws.

---

## 8. Milestones

| Phase | Deliverable |
|-------|-------------|
| P0 | Repo scaffold, `go.mod`, CI, CLAUDE.md conventions. |
| P1 | Config loading + validation, structured logging. |
| P2 | ADB pre-flight helpers. |
| P3 | Appium HTTP client with unit tests. |
| P4 | Locator schema, loader, and validation. |
| P5 | `OTPProvider` interface + `GmailAppProvider`. |
| P6 | Flow/step engine + `PlatformFlow` interface. |
| P7 | Instagram flow. |
| P8 | TikTok flow. |
| P9 | Core orchestrator + CLI wiring + artifacts. |
| P10 | Tests (unit + opt-in integration) and hardening. |

Detailed, copy-pasteable build prompts for each phase are in
[`PROMPTS.md`](./PROMPTS.md).
