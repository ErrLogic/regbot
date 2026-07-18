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

// scriptedRunner returns a different canned output per call, cycling through
// steps. It records the command sequence for assertions.
type scriptedRunner struct {
	steps []struct {
		out string
		err error
	}
	calls [][]string
	i     int
}

func (s *scriptedRunner) Run(_ context.Context, _ string, args ...string) ([]byte, error) {
	s.calls = append(s.calls, args)
	if s.i >= len(s.steps) {
		return []byte(""), nil
	}
	step := s.steps[s.i]
	s.i++
	return []byte(step.out), step.err
}

const tlsSerial = "adb-04b2d3470504-cKtVBV._adb-tls-connect._tcp"

func TestIsNetworkSerial(t *testing.T) {
	cases := map[string]bool{
		tlsSerial:          true,
		"192.168.1.5:5555": true,
		"emulator-5554":    false,
		"ABC123DEF":        false,
	}
	for serial, want := range cases {
		if got := isNetworkSerial(serial); got != want {
			t.Errorf("isNetworkSerial(%q) = %v, want %v", serial, got, want)
		}
	}
}

func TestConnectNoOpForUSB(t *testing.T) {
	r := &scriptedRunner{}
	c := newTestClient(r, WithSerial("emulator-5554"))
	if err := c.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if len(r.calls) != 0 {
		t.Errorf("expected no adb calls for USB serial, got %v", r.calls)
	}
}

func TestConnectNetworkSerial(t *testing.T) {
	r := &scriptedRunner{steps: []struct {
		out string
		err error
	}{
		{out: "connected to " + tlsSerial},
	}}
	c := newTestClient(r, WithSerial(tlsSerial))
	if err := c.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if len(r.calls) != 1 || r.calls[0][0] != "connect" {
		t.Errorf("expected 'adb connect', got %v", r.calls)
	}
}

func TestConnectFailure(t *testing.T) {
	r := &scriptedRunner{steps: []struct {
		out string
		err error
	}{
		{out: "failed to connect to " + tlsSerial},
	}}
	c := newTestClient(r, WithSerial(tlsSerial))
	if err := c.Connect(context.Background()); err == nil {
		t.Fatal("expected error when adb connect reports failure")
	}
}

func TestEnsureConnectedAlreadyPresent(t *testing.T) {
	// First CheckDevice succeeds — no reconnect needed.
	r := &scriptedRunner{steps: []struct {
		out string
		err error
	}{
		{out: "List of devices attached\n" + tlsSerial + "\tdevice\n"},
	}}
	c := newTestClient(r, WithSerial(tlsSerial))
	if err := c.EnsureConnected(context.Background()); err != nil {
		t.Fatalf("EnsureConnected: %v", err)
	}
	if len(r.calls) != 1 {
		t.Errorf("expected 1 call (devices), got %d: %v", len(r.calls), r.calls)
	}
}

func TestEnsureConnectedReconnects(t *testing.T) {
	// 1st devices: not present. connect: ok. 2nd devices: present.
	r := &scriptedRunner{steps: []struct {
		out string
		err error
	}{
		{out: "List of devices attached\n\n"},
		{out: "connected to " + tlsSerial},
		{out: "List of devices attached\n" + tlsSerial + "\tdevice\n"},
	}}
	c := newTestClient(r, WithSerial(tlsSerial))
	if err := c.EnsureConnected(context.Background()); err != nil {
		t.Fatalf("EnsureConnected: %v", err)
	}
	if len(r.calls) != 3 {
		t.Fatalf("expected 3 calls (devices, connect, devices), got %d: %v", len(r.calls), r.calls)
	}
	if r.calls[1][0] != "connect" {
		t.Errorf("second call should be connect, got %v", r.calls[1])
	}
}

func TestEnsureConnectedUSBFailsThrough(t *testing.T) {
	// USB serial, not present: no reconnect attempt, returns the check error.
	r := &scriptedRunner{steps: []struct {
		out string
		err error
	}{
		{out: "List of devices attached\n\n"},
		{out: "List of devices attached\n\n"},
	}}
	c := newTestClient(r, WithSerial("emulator-5554"))
	if err := c.EnsureConnected(context.Background()); !errors.Is(err, ErrNoDevice) {
		t.Fatalf("want ErrNoDevice for absent USB device, got %v", err)
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
