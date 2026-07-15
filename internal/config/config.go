package config

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config is the root configuration for a RegBot run. It mirrors the annotated
// config.yaml documented in ARCHITECTURE.md §2.7.
type Config struct {
	Appium   AppiumConfig   `mapstructure:"appium"`
	Device   DeviceConfig   `mapstructure:"device"`
	Apps     AppsConfig     `mapstructure:"apps"`
	Email    EmailConfig    `mapstructure:"email"`
	OTP      OTPConfig      `mapstructure:"otp"`
	Account  AccountConfig  `mapstructure:"account"`
	Timeouts TimeoutsConfig `mapstructure:"timeouts"`
	Paths    PathsConfig    `mapstructure:"paths"`
	Logging  LoggingConfig  `mapstructure:"logging"`
}

// AppiumConfig holds the Appium server endpoint and session timeout.
type AppiumConfig struct {
	ServerURL         string        `mapstructure:"server_url"`
	NewCommandTimeout time.Duration `mapstructure:"new_command_timeout"`
}

// DeviceConfig holds the Android device capabilities.
type DeviceConfig struct {
	PlatformName   string `mapstructure:"platform_name"`
	DeviceName     string `mapstructure:"device_name"`
	AutomationName string `mapstructure:"automation_name"`
	UDID           string `mapstructure:"udid"`
}

// AppsConfig holds the package/activity identifiers for the target apps and Gmail.
type AppsConfig struct {
	InstagramPackage  string `mapstructure:"instagram_package"`
	InstagramActivity string `mapstructure:"instagram_activity"`
	TikTokPackage     string `mapstructure:"tiktok_package"`
	GmailPackage      string `mapstructure:"gmail_package"`
}

// EmailConfig selects the target email address. Exactly one of Address or
// BaseAddress must be provided; BaseAddress enables +alias generation.
type EmailConfig struct {
	Address        string `mapstructure:"address"`
	BaseAddress    string `mapstructure:"base_address"`
	AliasTagPrefix string `mapstructure:"alias_tag_prefix"`
}

// OTPConfig configures how verification codes are located and parsed.
type OTPConfig struct {
	SenderAllowlist []string      `mapstructure:"sender_allowlist"`
	CodeRegex       string        `mapstructure:"code_regex"`
	WaitTimeout     time.Duration `mapstructure:"wait_timeout"`
	PollInterval    time.Duration `mapstructure:"poll_interval"`
}

// AccountConfig configures generated-credential parameters.
type AccountConfig struct {
	PasswordLength int    `mapstructure:"password_length"`
	UsernamePrefix string `mapstructure:"username_prefix"`
}

// TimeoutsConfig holds UI wait and retry settings.
type TimeoutsConfig struct {
	ElementWait time.Duration `mapstructure:"element_wait"`
	StepRetry   int           `mapstructure:"step_retry"`
}

// PathsConfig holds filesystem locations used by a run.
type PathsConfig struct {
	LocatorsDir  string `mapstructure:"locators_dir"`
	ArtifactsDir string `mapstructure:"artifacts_dir"`
}

// LoggingConfig configures the structured logger.
type LoggingConfig struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

// Load reads configuration from the YAML file at path, applying environment
// overrides prefixed with REGBOT_ (nested keys use underscores, e.g.
// REGBOT_APPIUM_SERVER_URL). It does not validate the result; call Validate.
func Load(path string) (Config, error) {
	v := viper.New()
	setDefaults(v)

	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	v.SetEnvPrefix("REGBOT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %q: %w", path, err)
	}
	return cfg, nil
}

// setDefaults registers default values so that every key is known to viper
// (which is required for environment overrides to take effect on Unmarshal).
// Email address fields default to empty so they never conflict with a value set
// in the file.
func setDefaults(v *viper.Viper) {
	v.SetDefault("appium.server_url", "http://127.0.0.1:4723")
	v.SetDefault("appium.new_command_timeout", 120*time.Second)

	v.SetDefault("device.platform_name", "Android")
	v.SetDefault("device.device_name", "emulator-5554")
	v.SetDefault("device.automation_name", "UiAutomator2")
	v.SetDefault("device.udid", "")

	v.SetDefault("apps.instagram_package", "com.instagram.android")
	v.SetDefault("apps.instagram_activity", "com.instagram.mainactivity.MainActivity")
	v.SetDefault("apps.tiktok_package", "com.zhiliaoapp.musically")
	v.SetDefault("apps.gmail_package", "com.google.android.gm")

	v.SetDefault("email.address", "")
	v.SetDefault("email.base_address", "")
	v.SetDefault("email.alias_tag_prefix", "reg")

	v.SetDefault("otp.sender_allowlist", []string{"instagram", "tiktok", "no-reply"})
	v.SetDefault("otp.code_regex", `\d{6}`)
	v.SetDefault("otp.wait_timeout", 60*time.Second)
	v.SetDefault("otp.poll_interval", 5*time.Second)

	v.SetDefault("account.password_length", 16)
	v.SetDefault("account.username_prefix", "user")

	v.SetDefault("timeouts.element_wait", 15*time.Second)
	v.SetDefault("timeouts.step_retry", 2)

	v.SetDefault("paths.locators_dir", "./locators")
	v.SetDefault("paths.artifacts_dir", "./artifacts")

	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.file", "./regbot.log")
}

// Validate checks the configuration for internal consistency, returning an error
// whose message names the first offending field.
func (c Config) Validate() error {
	if c.Appium.ServerURL == "" {
		return errors.New("appium.server_url: must be set")
	}
	if u, err := url.Parse(c.Appium.ServerURL); err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("appium.server_url: %q is not a valid URL", c.Appium.ServerURL)
	}
	if c.Appium.NewCommandTimeout <= 0 {
		return errors.New("appium.new_command_timeout: must be > 0")
	}

	hasAddress := c.Email.Address != ""
	hasBase := c.Email.BaseAddress != ""
	if hasAddress == hasBase {
		return errors.New("email: exactly one of email.address or email.base_address must be set")
	}

	if c.OTP.CodeRegex == "" {
		return errors.New("otp.code_regex: must be set")
	}
	if _, err := regexp.Compile(c.OTP.CodeRegex); err != nil {
		return fmt.Errorf("otp.code_regex: invalid regexp: %w", err)
	}
	if c.OTP.WaitTimeout <= 0 {
		return errors.New("otp.wait_timeout: must be > 0")
	}
	if c.OTP.PollInterval <= 0 {
		return errors.New("otp.poll_interval: must be > 0")
	}

	if c.Account.PasswordLength <= 0 {
		return errors.New("account.password_length: must be > 0")
	}

	if c.Timeouts.ElementWait <= 0 {
		return errors.New("timeouts.element_wait: must be > 0")
	}
	if c.Timeouts.StepRetry < 0 {
		return errors.New("timeouts.step_retry: must be >= 0")
	}

	if c.Paths.LocatorsDir == "" {
		return errors.New("paths.locators_dir: must be set")
	}
	if c.Paths.ArtifactsDir == "" {
		return errors.New("paths.artifacts_dir: must be set")
	}

	return nil
}
