package adb

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Sentinel errors returned by CheckDevice so callers can distinguish the failure
// modes with errors.Is.
var (
	// ErrNoDevice indicates no device is connected.
	ErrNoDevice = errors.New("adb: no device connected")
	// ErrMultipleDevices indicates more than one authorised device is connected
	// and no serial was configured to disambiguate.
	ErrMultipleDevices = errors.New("adb: multiple devices connected")
	// ErrUnauthorized indicates a device is connected but not authorised for adb.
	ErrUnauthorized = errors.New("adb: device unauthorised")
)

// commandRunner abstracts process execution so the client can be unit-tested
// without a real adb binary or device.
type commandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// execRunner runs commands via os/exec, capturing combined stdout+stderr.
type execRunner struct{}

// Run executes name with args and returns the combined output.
func (execRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// Client wraps a small set of ADB operations used for pre-flight checks and
// optional APK installation. It is never used to drive the UI.
type Client struct {
	adbPath string
	serial  string
	runner  commandRunner
}

// Option configures a Client.
type Option func(*Client)

// WithADBPath overrides the adb binary path (default "adb").
func WithADBPath(path string) Option {
	return func(c *Client) {
		if path != "" {
			c.adbPath = path
		}
	}
}

// WithSerial pins operations to a specific device serial (adb -s <serial>).
func WithSerial(serial string) Option {
	return func(c *Client) { c.serial = serial }
}

// New constructs a Client using the real exec-based runner.
func New(opts ...Option) *Client {
	c := &Client{adbPath: "adb", runner: execRunner{}}
	for _, o := range opts {
		o(c)
	}
	return c
}

// args prepends the -s <serial> selector when a serial is configured.
func (c *Client) args(extra ...string) []string {
	if c.serial != "" {
		return append([]string{"-s", c.serial}, extra...)
	}
	return extra
}

// deviceEntry is one parsed row of `adb devices` output.
type deviceEntry struct {
	serial string
	state  string
}

// CheckDevice ensures a usable device is connected. If a serial is configured,
// it verifies that device is present and authorised; otherwise it requires
// exactly one authorised device. It returns a wrapped ErrNoDevice,
// ErrUnauthorized, or ErrMultipleDevices on failure.
func (c *Client) CheckDevice(ctx context.Context) error {
	out, err := c.runner.Run(ctx, c.adbPath, c.args("devices")...)
	if err != nil {
		return fmt.Errorf("adb devices: %w: %s", err, strings.TrimSpace(string(out)))
	}
	entries := parseDevices(string(out))

	if c.serial != "" {
		for _, e := range entries {
			if e.serial != c.serial {
				continue
			}
			switch e.state {
			case "device":
				return nil
			case "unauthorized":
				return fmt.Errorf("%w: %s", ErrUnauthorized, c.serial)
			default:
				return fmt.Errorf("%w: %s is %q", ErrNoDevice, c.serial, e.state)
			}
		}
		return fmt.Errorf("%w: %s", ErrNoDevice, c.serial)
	}

	var authorised, unauthorised int
	for _, e := range entries {
		switch e.state {
		case "device":
			authorised++
		case "unauthorized":
			unauthorised++
		}
	}
	switch {
	case authorised == 1:
		return nil
	case authorised > 1:
		return fmt.Errorf("%w: %d authorised", ErrMultipleDevices, authorised)
	case unauthorised > 0:
		return fmt.Errorf("%w: %d present but not authorised", ErrUnauthorized, unauthorised)
	default:
		return ErrNoDevice
	}
}

// parseDevices parses `adb devices` output into entries, skipping the header
// line and any blank or daemon-status lines.
func parseDevices(out string) []deviceEntry {
	var entries []deviceEntry
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" || strings.HasPrefix(line, "List of devices") || strings.HasPrefix(line, "*") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		entries = append(entries, deviceEntry{serial: fields[0], state: fields[1]})
	}
	return entries
}

// IsInstalled reports whether the given package is installed on the device.
func (c *Client) IsInstalled(ctx context.Context, pkg string) (bool, error) {
	if pkg == "" {
		return false, errors.New("adb: empty package name")
	}
	out, err := c.runner.Run(ctx, c.adbPath, c.args("shell", "pm", "list", "packages", pkg)...)
	if err != nil {
		return false, fmt.Errorf("adb pm list packages %q: %w: %s", pkg, err, strings.TrimSpace(string(out)))
	}
	// `pm list packages <pkg>` does a substring match, so confirm an exact line.
	want := "package:" + pkg
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(strings.TrimRight(line, "\r")) == want {
			return true, nil
		}
	}
	return false, nil
}

// InstallAPK installs (or reinstalls, -r) the APK at apkPath onto the device.
func (c *Client) InstallAPK(ctx context.Context, apkPath string) error {
	if apkPath == "" {
		return errors.New("adb: empty apk path")
	}
	out, err := c.runner.Run(ctx, c.adbPath, c.args("install", "-r", apkPath)...)
	text := strings.TrimSpace(string(out))
	if err != nil {
		return fmt.Errorf("adb install %q: %w: %s", apkPath, err, text)
	}
	if !strings.Contains(text, "Success") {
		return fmt.Errorf("adb install %q failed: %s", apkPath, text)
	}
	return nil
}
