package adb

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// fakeRunner records calls and returns canned output/errors.
type fakeRunner struct {
	out      string
	err      error
	lastName string
	lastArgs []string
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.lastName = name
	f.lastArgs = args
	return []byte(f.out), f.err
}

func newTestClient(r commandRunner, opts ...Option) *Client {
	c := New(opts...)
	c.runner = r
	return c
}

func TestCheckDeviceSingleAuthorised(t *testing.T) {
	r := &fakeRunner{out: "List of devices attached\nemulator-5554\tdevice\n"}
	if err := newTestClient(r).CheckDevice(context.Background()); err != nil {
		t.Fatalf("CheckDevice: %v", err)
	}
}

func TestCheckDeviceNone(t *testing.T) {
	r := &fakeRunner{out: "List of devices attached\n\n"}
	err := newTestClient(r).CheckDevice(context.Background())
	if !errors.Is(err, ErrNoDevice) {
		t.Fatalf("want ErrNoDevice, got %v", err)
	}
}

func TestCheckDeviceUnauthorised(t *testing.T) {
	r := &fakeRunner{out: "List of devices attached\nABC123\tunauthorized\n"}
	err := newTestClient(r).CheckDevice(context.Background())
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("want ErrUnauthorized, got %v", err)
	}
}

func TestCheckDeviceMultiple(t *testing.T) {
	r := &fakeRunner{out: "List of devices attached\nemulator-5554\tdevice\nemulator-5556\tdevice\n"}
	err := newTestClient(r).CheckDevice(context.Background())
	if !errors.Is(err, ErrMultipleDevices) {
		t.Fatalf("want ErrMultipleDevices, got %v", err)
	}
}

func TestCheckDeviceWithSerialSelectsDevice(t *testing.T) {
	// Two devices present; serial disambiguates and must pass.
	r := &fakeRunner{out: "List of devices attached\nemulator-5554\tdevice\nemulator-5556\tdevice\n"}
	c := newTestClient(r, WithSerial("emulator-5556"))
	if err := c.CheckDevice(context.Background()); err != nil {
		t.Fatalf("CheckDevice with serial: %v", err)
	}
	// The -s selector must be passed to adb.
	if len(c.runner.(*fakeRunner).lastArgs) < 3 || c.runner.(*fakeRunner).lastArgs[0] != "-s" {
		t.Errorf("expected -s serial in args, got %v", r.lastArgs)
	}
}

func TestCheckDeviceWithSerialMissing(t *testing.T) {
	r := &fakeRunner{out: "List of devices attached\nemulator-5554\tdevice\n"}
	c := newTestClient(r, WithSerial("nope"))
	if err := c.CheckDevice(context.Background()); !errors.Is(err, ErrNoDevice) {
		t.Fatalf("want ErrNoDevice for missing serial, got %v", err)
	}
}

func TestCheckDeviceRunnerError(t *testing.T) {
	r := &fakeRunner{err: errors.New("exec fail"), out: "boom"}
	if err := newTestClient(r).CheckDevice(context.Background()); err == nil {
		t.Fatal("expected error when runner fails")
	}
}

func TestIsInstalledTrue(t *testing.T) {
	r := &fakeRunner{out: "package:com.instagram.android\n"}
	ok, err := newTestClient(r).IsInstalled(context.Background(), "com.instagram.android")
	if err != nil || !ok {
		t.Fatalf("IsInstalled = %v, %v; want true, nil", ok, err)
	}
}

func TestIsInstalledFalseOnPrefixOnly(t *testing.T) {
	// pm does substring matching; a prefix hit must not count as installed.
	r := &fakeRunner{out: "package:com.instagram.android.beta\n"}
	ok, err := newTestClient(r).IsInstalled(context.Background(), "com.instagram.android")
	if err != nil {
		t.Fatalf("IsInstalled err: %v", err)
	}
	if ok {
		t.Fatal("prefix-only match should be false")
	}
}

func TestIsInstalledEmptyPkg(t *testing.T) {
	if _, err := newTestClient(&fakeRunner{}).IsInstalled(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty package")
	}
}

func TestInstallAPKSuccess(t *testing.T) {
	r := &fakeRunner{out: "Performing Streamed Install\nSuccess\n"}
	if err := newTestClient(r).InstallAPK(context.Background(), "/tmp/app.apk"); err != nil {
		t.Fatalf("InstallAPK: %v", err)
	}
}

func TestInstallAPKFailure(t *testing.T) {
	r := &fakeRunner{out: "Failure [INSTALL_FAILED_INVALID_APK]\n"}
	err := newTestClient(r).InstallAPK(context.Background(), "/tmp/app.apk")
	if err == nil || !strings.Contains(err.Error(), "INSTALL_FAILED_INVALID_APK") {
		t.Fatalf("expected failure with adb output, got %v", err)
	}
}

func TestParseDevicesSkipsDaemonLines(t *testing.T) {
	out := "* daemon not running; starting now at tcp:5037\n" +
		"* daemon started successfully\n" +
		"List of devices attached\n" +
		"emulator-5554\tdevice\n"
	entries := parseDevices(out)
	if len(entries) != 1 || entries[0].serial != "emulator-5554" || entries[0].state != "device" {
		t.Fatalf("parseDevices = %+v", entries)
	}
}
