# RegBot

RegBot is a Go CLI that automates **email-based registration** for Instagram and
TikTok on an Android device. It drives the target app with **Appium**
(UiAutomator2) and reads the verification code directly from the on-device
**Gmail app** — no phone number, SMS gateway, or external email API required.

> ⚠️ **Educational use only.** Automated account creation violates the Terms of
> Service of Instagram and TikTok. This project exists to study mobile UI
> automation, cross-app orchestration, and clean Go architecture. See
> [Legal & Ethical Notice](#legal--ethical-notice) and [`PRD.md`](./PRD.md) §7.

## How it works

1. Open one Appium session against the target app.
2. Drive the registration screens (create account → email → request code).
3. Switch to Gmail, find the latest verification email from an allow-listed
   sender, and extract the code with a regex.
4. Switch back, enter the code, set generated credentials, and finish (or stop
   before the final submit with `--dry-run`).
5. Print the created account as JSON to stdout.

See [`ARCHITECTURE.md`](./ARCHITECTURE.md) for the component design and
[`PRD.md`](./PRD.md) for requirements.

## Prerequisites

- **Go 1.22+** (to build).
- **Appium server** with the **UiAutomator2** driver, reachable at the configured
  `appium.server_url` (default `http://127.0.0.1:4723`).
- **Android device or emulator** (Android 10+, unrooted) connected via ADB.
- The **target app** (Instagram or TikTok) and the **Gmail app** installed on the
  device, with Gmail signed into an account that receives the verification email.
- **`adb`** on your `PATH` (used only for pre-flight checks).

## Build

```bash
go build -o regbot ./cmd/regbot
# or
make build
```

## Configure

Copy the sample and edit it (keep real addresses out of version control — use a
`*.local.yaml`, which is gitignored):

```bash
cp config.yaml config.local.yaml
```

Environment overrides use the `REGBOT_` prefix with underscores for nested keys,
e.g. `REGBOT_APPIUM_SERVER_URL=http://127.0.0.1:4723`.

## Usage

```bash
# Instagram, explicit email
./regbot register instagram --email myuser@gmail.com --config config.local.yaml

# TikTok, generate a +alias from base_address in config
./regbot register tiktok --config config.local.yaml

# Validate + connect but stop before the final submit
./regbot register instagram --email myuser@gmail.com --dry-run

# Verify locator files load and required elements are present
./regbot locators verify
```

On success the account is printed to **stdout** as a single JSON object:

```json
{
  "platform": "instagram",
  "email": "myuser@gmail.com",
  "username": "user_8f3a21bd",
  "password": "‹generated›",
  "created_at": "2026-07-16T10:22:31Z",
  "status": "success"
}
```

Structured logs go to **stderr** and the log file; the generated password appears
**only** on stdout, never in logs or artifacts.

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--config` | Path to the YAML config file | `config.yaml` |
| `--email` | Target email (overrides config) | — |
| `--log-level` | `debug` \| `info` \| `warn` \| `error` | `info` |
| `--dry-run` | Validate + connect, but do not submit | `false` |

### Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success (credentials printed) |
| `1` | Configuration/validation error (before any UI action) |
| `2` | Automation failure (a step failed; artifacts written) |
| `3` | Verification code not received within the timeout |
| `130` | Interrupted (Ctrl-C / context cancelled) |

## Configuration reference

The full annotated schema lives in [`config.yaml`](./config.yaml) and
[`ARCHITECTURE.md`](./ARCHITECTURE.md) §2.7. Key sections:

- `appium` — server URL and session/command timeout.
- `device` — platform, device name, automation name, optional UDID.
- `apps` — package/activity ids for Instagram, TikTok, and Gmail.
- `email` — a fixed `address`, **or** a `base_address` (+ `alias_tag_prefix`) for
  `user+reg<rand>@gmail.com` alias generation. Exactly one of the two is required.
- `otp` — `sender_allowlist`, `code_regex` (default `\d{6}`), `wait_timeout`,
  `poll_interval`.
- `account` — `password_length`, `username_prefix`.
- `timeouts` — `element_wait`, `step_retry`, `step_backoff`.
- `paths` — `locators_dir`, `artifacts_dir`.
- `logging` — `level`, `file`.

## Locators

UI selectors live in versioned JSON files under [`locators/`](./locators) — one
per app. Each logical element maps to an **ordered list of candidate selectors**
(first match wins), so you can add fallbacks across app versions without touching
Go code. Run `./regbot locators verify` to confirm all files load and every
required element is present. The shipped selectors are **placeholders** marked
with `todo` notes and must be verified against your app versions on a device.

## Artifacts

On any step failure, RegBot writes to `paths.artifacts_dir`:

- `‹run-id›-‹step›.png` — screenshot,
- `‹run-id›-‹step›.xml` — page source,
- `‹run-id›-result.json` — run summary and error (never contains the password).

## Testing

```bash
go test ./...                       # unit tests (no device needed)
go test -tags=integration ./...     # on-device smoke test (see below)
golangci-lint run                   # lint
make test | make lint               # via Makefile
```

The integration test (`internal/core/integration_test.go`, build tag
`integration`) performs a real `--dry-run` Instagram registration and requires a
running Appium server, a connected device with the apps installed, and Gmail
signed in. Point it at a config with `REGBOT_CONFIG=/path/to/config.local.yaml`.

## Troubleshooting

| Symptom | Likely cause / fix |
|---------|--------------------|
| `preflight: adb: no device connected` | No device/emulator attached, or `adb` not on PATH. Run `adb devices`. |
| `preflight: adb: device unauthorised` | Accept the USB-debugging prompt on the device. |
| `preflight: required app "…" is not installed` | Install the target app / Gmail on the device. |
| `open appium session: …` | Appium server not running or wrong `server_url`. |
| `resolve "…": no candidate matched` | Locators drifted; update `locators/‹app›.json` and re-run `locators verify`. |
| Exit `3` / `verification code not found` | Email not received in time, or `sender_allowlist`/`code_regex` mismatch. Increase `otp.wait_timeout`. |
| Exit `1` | Config invalid; the error names the offending field. |

## Legal & Ethical Notice

Automated account creation violates the Terms of Service of Instagram and TikTok,
and may violate local laws or platform anti-fraud provisions. RegBot is provided
strictly for **educational study** of mobile UI automation and software
architecture. Do not use it to create accounts on services you are not authorised
to automate, to evade bans, to impersonate, or to conduct spam or fraud. Use
test-only accounts and comply with all applicable terms and laws. You are solely
responsible for how you use this software.
