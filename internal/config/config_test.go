package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const validYAML = `
appium:
  server_url: "http://127.0.0.1:4723"
  new_command_timeout: 90s
device:
  platform_name: "Android"
  device_name: "emulator-5554"
  automation_name: "UiAutomator2"
apps:
  instagram_package: "com.instagram.android"
  gmail_package: "com.google.android.gm"
email:
  address: "tester@gmail.com"
otp:
  sender_allowlist: ["instagram", "tiktok"]
  code_regex: "\\d{6}"
  wait_timeout: 45s
  poll_interval: 3s
account:
  password_length: 16
  username_prefix: "user"
timeouts:
  element_wait: 10s
  step_retry: 2
paths:
  locators_dir: "./locators"
  artifacts_dir: "./artifacts"
logging:
  level: "info"
  file: ""
`

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestLoadValid(t *testing.T) {
	path := writeConfig(t, validYAML)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.Appium.ServerURL; got != "http://127.0.0.1:4723" {
		t.Errorf("ServerURL = %q", got)
	}
	if got := cfg.Appium.NewCommandTimeout; got != 90*time.Second {
		t.Errorf("NewCommandTimeout = %v, want 90s", got)
	}
	if got := cfg.OTP.WaitTimeout; got != 45*time.Second {
		t.Errorf("WaitTimeout = %v, want 45s", got)
	}
	if got := cfg.OTP.PollInterval; got != 3*time.Second {
		t.Errorf("PollInterval = %v, want 3s", got)
	}
	if got := cfg.OTP.SenderAllowlist; len(got) != 2 || got[0] != "instagram" {
		t.Errorf("SenderAllowlist = %v", got)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate on valid config: %v", err)
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml")); err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestLoadDefaultsApplied(t *testing.T) {
	// A near-empty file should still receive defaults for unset keys.
	path := writeConfig(t, "email:\n  address: \"tester@gmail.com\"\n")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.OTP.CodeRegex != `\d{6}` {
		t.Errorf("default CodeRegex = %q", cfg.OTP.CodeRegex)
	}
	if cfg.Account.PasswordLength != 16 {
		t.Errorf("default PasswordLength = %d", cfg.Account.PasswordLength)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate with defaults: %v", err)
	}
}

func TestLoadEnvOverride(t *testing.T) {
	t.Setenv("REGBOT_APPIUM_SERVER_URL", "http://10.0.0.5:4444")
	path := writeConfig(t, validYAML)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.Appium.ServerURL; got != "http://10.0.0.5:4444" {
		t.Errorf("env override not applied: ServerURL = %q", got)
	}
}

func baseValidConfig() Config {
	cfg, _ := Load(writeConfigNoT(validYAML))
	return cfg
}

// writeConfigNoT is a helper for baseValidConfig used outside a *testing.T; it
// writes to a temp file and ignores cleanup (the OS temp dir is reclaimed).
func writeConfigNoT(body string) string {
	f, _ := os.CreateTemp("", "regbot-*.yaml")
	_, _ = f.WriteString(body)
	_ = f.Close()
	return f.Name()
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{
			name:    "valid",
			mutate:  func(*Config) {},
			wantErr: "",
		},
		{
			name:    "missing server url",
			mutate:  func(c *Config) { c.Appium.ServerURL = "" },
			wantErr: "appium.server_url",
		},
		{
			name:    "malformed server url",
			mutate:  func(c *Config) { c.Appium.ServerURL = "not-a-url" },
			wantErr: "appium.server_url",
		},
		{
			name:    "zero command timeout",
			mutate:  func(c *Config) { c.Appium.NewCommandTimeout = 0 },
			wantErr: "appium.new_command_timeout",
		},
		{
			name:    "both email fields set",
			mutate:  func(c *Config) { c.Email.BaseAddress = "base@gmail.com" },
			wantErr: "email",
		},
		{
			name: "neither email field set",
			mutate: func(c *Config) {
				c.Email.Address = ""
				c.Email.BaseAddress = ""
			},
			wantErr: "email",
		},
		{
			name:    "bad regex",
			mutate:  func(c *Config) { c.OTP.CodeRegex = "(" },
			wantErr: "otp.code_regex",
		},
		{
			name:    "zero otp wait timeout",
			mutate:  func(c *Config) { c.OTP.WaitTimeout = 0 },
			wantErr: "otp.wait_timeout",
		},
		{
			name:    "zero poll interval",
			mutate:  func(c *Config) { c.OTP.PollInterval = 0 },
			wantErr: "otp.poll_interval",
		},
		{
			name:    "zero password length",
			mutate:  func(c *Config) { c.Account.PasswordLength = 0 },
			wantErr: "account.password_length",
		},
		{
			name:    "zero element wait",
			mutate:  func(c *Config) { c.Timeouts.ElementWait = 0 },
			wantErr: "timeouts.element_wait",
		},
		{
			name:    "empty locators dir",
			mutate:  func(c *Config) { c.Paths.LocatorsDir = "" },
			wantErr: "paths.locators_dir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := baseValidConfig()
			tt.mutate(&cfg)
			err := cfg.Validate()
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tt.wantErr != "" && err == nil:
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			case tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr):
				t.Fatalf("error %q does not mention %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestNewLoggerAndRedacted(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "regbot.log")
	logger, err := NewLogger(LoggingConfig{Level: "info", File: logFile})
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	logger.Info("startup", Redacted("password"))
	_ = logger.Sync()

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	out := string(data)
	if !strings.Contains(out, "startup") {
		t.Errorf("log file missing startup line: %s", out)
	}
	if strings.Contains(out, "\"password\":\"[REDACTED]\"") == false {
		t.Errorf("redacted field not masked in log: %s", out)
	}
}

func TestNewLoggerBadLevel(t *testing.T) {
	if _, err := NewLogger(LoggingConfig{Level: "loud"}); err == nil {
		t.Fatal("expected error for invalid log level")
	}
}
