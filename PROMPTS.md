# RegBot – Phased Implementation Prompts

Prompts for building RegBot with Claude Code, one phase at a time (make sure you stop after one phase and mark this docs on the phase as done).
Each phase is self-contained, states its dependencies, and ends with a
**Definition of Done** i need verify before moving on. Run them in order.

> Source of truth: [`PRD.md`](./PRD.md) (requirements), [`ARCHITECTURE.md`](./ARCHITECTURE.md)
> (design), [`CLAUDE.md`](./CLAUDE.md) (conventions). Every prompt below assumes
> those three documents are in context.
>
> ⚠️ **Educational use only** — see [`PRD.md`](./PRD.md) §7.

## How to use these prompts

1. Ensure you read all `PRD.md`, `ARCHITECTURE.md`, and `CLAUDE.md` in the repo.
2. Run one phase prompt at a time.
3. After each phase: run `go build ./...`, `go vet ./...`, `go test ./...`, and
   `golangci-lint run`. Fix before continuing.
4. Commit at the end of each green phase.

**Global conventions (apply to every phase):** Go 1.22 or newer; `context.Context` on all
I/O; wrap errors as `fmt.Errorf("step %q: %w", name, err)`; exported symbols get
doc comments; structured logging via `zap`; no third-party Appium library; never
hard-code locators; never log the generated password; keep `cmd/regbot` thin.

---

## Phase 0 — Project Scaffolding ✅ DONE

**Depends on:** nothing.
**Status:** Completed 2026-07-16. `go build ./...`, `go vet ./...`, and `gofmt -l` clean; `./regbot --help` lists the `register` group.

```text
Scaffold the RegBot Go module per ARCHITECTURE.md and CLAUDE.md. Do NOT implement
business logic yet — only structure, stubs, and tooling.

Tasks:
1. Initialise `go.mod` (module `github.com/ErrLogic/regbot`, Go 1.22 or newer) and add deps:
   cobra, viper, zap.
2. Create the directory tree from CLAUDE.md: cmd/regbot; internal/{adb,appium,
   config,core,flows,locators,otp,otp/gmailapp}; locators/.
3. Add a `main.go` in cmd/regbot with a cobra root command `regbot` and an empty
   `register` command group (subcommands added later). Wire `--config`,
   `--log-level`, and `--dry-run` global flags.
4. Add a `Makefile` with targets: build, test, itest (integration tag), lint, vet.
5. Add `.golangci.yml` (enable govet, errcheck, staticcheck, revive, gofmt).
6. Add a `.gitignore` (binary `regbot`, `artifacts/`, `*.log`, local config).
7. Add placeholder locator files locators/{instagram,tiktok,gmail}.json with the
   schema from ARCHITECTURE.md §2.7 and a "version" + empty "elements" object.

Definition of Done: `go build ./...` succeeds; `./regbot --help` shows the
`register` group; `golangci-lint run` is clean.
```

---

## Phase 1 — Configuration & Logging ✅ DONE

**Depends on:** Phase 0.
**Status:** Completed 2026-07-16. `internal/config` implements `Load` (viper + REGBOT_ env overrides + defaults), `Validate` (field-named errors), and `NewLogger`/`Redacted`; sample `config.yaml` added; table-driven tests pass; build/vet/lint clean.

```text
Implement internal/config and a shared logger, following FR-8 (PRD.md) and the
config schema in ARCHITECTURE.md §2.7.

Tasks:
1. Define a `Config` struct mirroring the annotated config.yaml (appium, device,
   apps, email, otp, account, timeouts, paths, logging). Use typed durations.
2. Load via viper: YAML file (from --config, default ./config.yaml) + env
   overrides (prefix REGBOT_). 
3. Implement `Load(path string) (Config, error)` and `(Config) Validate() error`.
   Validate: appium.server_url set and parseable; exactly one of email.address /
   email.base_address provided; regex compiles; timeouts > 0; paths non-empty.
4. Generate a sample `config.yaml` at repo root matching ARCHITECTURE.md §2.7.
5. Build a zap logger constructor: JSON to stderr + to logging.file, level from
   config/flag. Provide a redacting field helper so passwords are never logged.
6. Unit tests (table-driven) for Load + Validate covering: missing server_url,
   both/neither email fields, bad regex, zero timeout.

Definition of Done: tests pass; a malformed config produces a precise error
naming the offending field; running with a valid config logs a startup line.
```

---

## Phase 2 — ADB Pre-flight Helpers ✅ DONE

**Depends on:** Phase 1.
**Status:** Completed 2026-07-16. `internal/adb` implements `CheckDevice`, `IsInstalled`, `InstallAPK` over a `commandRunner` seam, configurable adb path + serial, and sentinel errors (`ErrNoDevice`/`ErrUnauthorized`/`ErrMultipleDevices`); 13 tests pass with a mocked runner; build/vet/lint clean.

```text
Implement internal/adb per ARCHITECTURE.md §2.6. Shell out to `adb`; do not use it
for any UI action.

Tasks:
1. `CheckDevice(ctx) error` — parse `adb devices`; error if zero, multiple, or
   unauthorised devices (clear message for each case).
2. `IsInstalled(ctx, pkg string) (bool, error)` — via `adb shell pm list packages`.
3. `InstallAPK(ctx, apkPath string) error` — `adb install -r`, wrap output on
   failure.
4. Make the adb binary path and optional device serial configurable (accept them
   as function params or a small Options struct; core will pass from Config).
5. Unit tests: inject a fake command runner (interface seam) and assert parsing of
   representative `adb devices` / `pm list packages` outputs.

Definition of Done: tests pass with a mocked runner; no real device required to
test; errors clearly distinguish "no device" vs "unauthorised" vs "multiple".
```

---

## Phase 3 — Appium HTTP Client

**Depends on:** Phase 1.

```text
Implement internal/appium — a minimal WebDriver/Appium JSON client (no external
Appium library), per ARCHITECTURE.md §2.5 and the method list in CLAUDE.md.

Tasks:
1. Types `Driver` and `Element`; a `Capabilities` type built from Config.
2. `NewDriver(ctx, serverURL, caps)` — POST /session, store session id; set a
   sane http.Client timeout.
3. Element location: `FindElement`, `WaitForElement(...,timeout)` (poll until
   visible or timeout, honouring ctx). Support `by`: id, accessibility id, xpath,
   -android uiautomator.
4. Actions: Element.Click / SendKeys / GetText; Driver.Swipe (W3C actions or
   `mobile: swipeGesture`); PressBack; LaunchApp (mobile: activateApp);
   Screenshot; PageSource; SetClipboard/GetClipboard; Quit.
5. Sentinel errors ErrElementNotFound, ErrSessionExpired, ErrTimeout; map Appium
   error responses to these.
6. Unit tests with httptest.Server: assert request bodies/paths and decode canned
   responses for create-session, find-element, click, get-text, and an error case.

Definition of Done: tests pass without a real Appium server; every method takes
ctx and returns wrapped errors; no third-party Appium dependency was added.
```

---

## Phase 4 — Locator Schema, Loader & Validation

**Depends on:** Phase 1.

```text
Implement internal/locators per ARCHITECTURE.md §2.7 and FR-6 (PRD.md).

Tasks:
1. Define `Selector{ By, Selector string }`, `Map` = name -> ordered []Selector,
   and a file model with "version" + "elements".
2. `Load(dir, app string) (Map, error)` — read locators/<app>.json, parse, and
   fail with a precise message on malformed JSON or an unknown `by` value.
3. `(Map) Require(names ...string) error` — verify all needed element names exist;
   error lists any missing names and the file.
4. Add a helper `Resolve(driver, name)` pattern (or document it) that tries each
   candidate selector in order (first match wins) via WaitForElement.
5. Flesh out locators/instagram.json, tiktok.json, gmail.json with best-known
   element names for each flow (create-account, email field, code field, next
   button; Gmail: inbox list item, sender, subject, body). Mark uncertain
   selectors with a TODO comment field.
6. Unit tests: valid load, missing file, bad JSON, unknown `by`, Require failure.

Definition of Done: tests pass; Require() names the exact missing element; every
element used by later flows exists in the JSON (even if selector is a TODO).
```

---

## Phase 5 — OTPProvider Interface & GmailAppProvider

**Depends on:** Phases 3, 4.

```text
Define the OTP interface and implement the Gmail-app provider per ARCHITECTURE.md
§2.4 and the algorithm in CLAUDE.md "Gmail OTP Provider Design".

Tasks:
1. internal/otp: `OTPProvider` interface with
   `GetCode(ctx, targetEmail string, timeout time.Duration) (string, error)`.
2. internal/otp/gmailapp: `GmailAppProvider` holding the shared *appium.Driver,
   the Gmail locator Map, and OTP config (sender_allowlist, code_regex,
   poll_interval). Constructor `New(driver, locators, cfg)`.
3. Implement GetCode exactly per the documented steps: record current app; launch
   Gmail; normalise state (PressBack out of any open thread to the inbox);
   pull-to-refresh (Swipe); poll every poll_interval up to timeout for a list item
   whose sender matches the allow-list and timestamp is recent; open it; GetText
   on the body; extract with the compiled regex; restore the previous app.
4. On timeout/no-match, return an error carrying a screenshot (write to artifacts
   via a passed-in sink or return bytes for the caller to persist) and context.
5. Factor the regex extraction + sender matching into pure functions and unit-test
   them (sample email bodies with/without a 6-digit code; sender variants).
6. Add a `mockOTPProvider` in a test helper for use by flow tests later.

Definition of Done: extraction/matching unit tests pass; GetCode compiles against
the Phase 3 driver; the provider never opens a second Appium session.
```

---

## Phase 6 — Flow & Step Engine

**Depends on:** Phases 3, 4, 5.

```text
Implement the flow abstractions and the shared step runner per ARCHITECTURE.md
§2.3 and CLAUDE.md "Flow Implementation Rules".

Tasks:
1. internal/flows: `Account` struct (Platform, Email, Username, Password,
   CreatedAt, Status); `Platform` type; `PlatformFlow` interface with
   `Register(ctx, driver, otp, email, locators) (Account, error)`.
2. `Step{ Name string; Run func(ctx) error }` and `runSteps(ctx, logger, artifacts,
   steps ...Step) error`: run in order, log start/finish/duration per step, retry
   a failed step up to configured attempts with backoff, and on final failure
   capture screenshot + page source to the artifacts dir named by run-id+step,
   wrapping the error with the step name.
3. A credentials helper: generate compliant password (config length, char
   classes), username (prefix+random) with a "taken -> regenerate" retry hook,
   and an adult birthday. Never log the password.
4. Small helpers for common UI patterns: tapByLocator(name), typeByLocator(name,
   text), dismissIfPresent(name) for interstitials.
5. Unit tests for runSteps (order, retry, artifact-on-failure via a fake sink) and
   the credential generator (policy compliance, uniqueness of username suffix).

Definition of Done: tests pass; runSteps produces an artifact on failure; the
credential generator output satisfies the configured policy and is never logged.
```

---

## Phase 7 — Instagram Flow

**Depends on:** Phase 6.

```text
Implement internal/flows/instagram.go per FR-3 (PRD.md). Use only locator names
from locators/instagram.json (and gmail.json via the injected OTP provider).

Tasks:
1. `InstagramFlow` implements PlatformFlow. Build its []Step:
   launch/create-account; choose email; enter email; tap Next; wait for
   "Confirm Your Email"; call otp.GetCode; enter code; set full name; set
   generated password; accept/confirm generated username (handle "username
   taken" via regenerate); set adult birthday; dismiss optional screens;
   finalise (skip final submit when cfg dry-run).
2. Use the tap/type/dismiss helpers from Phase 6; every element via locator name.
3. Return a populated Account on success; wrap step errors with names.
4. Table-driven unit test with a fake driver seam + mockOTPProvider asserting the
   step sequence and that GetCode is called after "tap Next".

Definition of Done: unit test passes; no selector is hard-coded; --dry-run path
stops before final submission; flow compiles against real driver + provider.
```

---

## Phase 8 — TikTok Flow

**Depends on:** Phase 6 (mirror Phase 7).

```text
Implement internal/flows/tiktok.go per FR-4 (PRD.md), mirroring the Instagram flow
but with TikTok's screen order: Sign up -> use phone/email -> Email -> set adult
birthday -> enter email -> Send code -> GetCode -> enter code -> set password ->
set/confirm nickname -> skip interests/contacts -> finalise.

Tasks:
1. `TikTokFlow` implements PlatformFlow with its own []Step; reuse Phase 6 helpers
   and credential generator; all selectors from locators/tiktok.json.
2. Handle the birthday-before-email ordering and the interests/contacts skips.
3. Unit test the step sequence with a fake driver + mockOTPProvider.

Definition of Done: unit test passes; selectors externalised; dry-run honoured.
```

---

## Phase 9 — Core Orchestrator & CLI Wiring

**Depends on:** Phases 2, 3, 5, 7, 8.

```text
Implement internal/core and finish cmd/regbot per ARCHITECTURE.md §2.1–2.2 and
FR-1/FR-7/FR-10 (PRD.md).

Tasks:
1. internal/core: `RegistrationService` with `Register(ctx, platform, email, cfg)
   (flows.Account, error)`:
   - ADB pre-flight (CheckDevice, IsInstalled for target app + Gmail).
   - Build capabilities; open ONE appium.Driver; `defer driver.Quit`.
   - Load target + gmail locator maps; Require the names each flow needs.
   - Construct GmailAppProvider(driver, gmailLocators, cfg.OTP).
   - Select InstagramFlow/TikTokFlow; run flow.Register with the provider.
   - Assign a run-id; on error persist artifacts + a result.json.
2. Email resolution: use email.address, else generate base+alias per FR-2/FR-9.
3. cmd/regbot: implement `register instagram` and `register tiktok` subcommands;
   parse flags, Load+Validate config, build logger, call core.Register, print the
   Account as JSON to stdout on success (FR-7 schema), and map outcomes to the
   exit codes in FR-10 (0/1/2/3/130). Honour --dry-run and ctx cancellation
   (SIGINT).
4. Wire structured logging of each step; ensure the password never reaches logs.
5. Integration-tagged smoke test (build tag `integration`) documenting the manual
   device prerequisites; skipped by default.

Definition of Done: `./regbot register instagram --email ... --dry-run` runs the
full path except final submit and exits 0; a forced failure writes artifacts and
exits 2; stdout on success is a single valid JSON object matching FR-7.
```

---

## Phase 10 — Testing, Hardening & Docs

**Depends on:** all previous phases.

```text
Harden and document per PRD §5 (NFRs) and ARCHITECTURE.md §6.

Tasks:
1. Raise unit coverage on appium, config, locators, otp, flows; add tests for
   context cancellation mid-wait and mid-poll.
2. Add a `locators verify` maintenance subcommand: load all locator files, compile
   selectors, and (optionally, with a device) check presence — to catch drift.
3. Add retry/backoff config plumbing end-to-end; verify OTP timeout maps to exit 3.
4. Ensure artifacts (screenshot, page source, result.json) are written on every
   failure path and never contain the password.
5. Write a README.md: prerequisites (Appium+UiAutomator2, device, Gmail signed
   in), setup, usage, config reference, troubleshooting, and the §7 legal notice.
6. Final pass: `go vet`, `golangci-lint run`, `go test ./...` all clean.

Definition of Done: full test suite green; README complete; a dry-run demonstrates
the end-to-end path; no secret ever appears in logs or artifacts.
```

---

## Cross-phase checklists

**Before committing any phase**
- [ ] `go build ./...` and `go vet ./...` clean
- [ ] `go test ./...` green
- [ ] `golangci-lint run` clean
- [ ] New exported symbols have doc comments
- [ ] No hard-coded locators; no secrets/emails committed
- [ ] Errors wrapped with step/context; ctx honoured in waits

**Definition of Done for the whole project**
- [ ] `register instagram` and `register tiktok` complete on a clean device
- [ ] OTP read from the Gmail app and entered automatically
- [ ] Credentials printed as JSON (FR-7); password absent from logs/artifacts
- [ ] Failures produce screenshot + page source + result.json
- [ ] Exit codes per FR-10; `--dry-run` supported
- [ ] Adding a new OTP source or platform touches only one package
