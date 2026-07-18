package device

import "testing"

func TestMapADBState(t *testing.T) {
	cases := map[string]string{
		"device":         "online",
		"unauthorized":   "unauthorized",
		"offline":        "offline",
		"no permissions": "offline",
		"recovery":       "offline",
		"":               "offline",
	}
	for adbState, want := range cases {
		if got := mapADBState(adbState); got != want {
			t.Errorf("mapADBState(%q) = %q, want %q", adbState, got, want)
		}
	}
}

func TestParseDevices(t *testing.T) {
	out := "List of devices attached\n" +
		"emulator-5554\tdevice\n" +
		"192.168.1.13:38349\tdevice\n" +
		"ABC123\tunauthorized\n" +
		"* daemon started successfully *\n"
	entries := parseDevices(out)
	if len(entries) != 3 {
		t.Fatalf("expected 3 device entries, got %d: %+v", len(entries), entries)
	}
	if entries[0].serial != "emulator-5554" || entries[0].state != "device" {
		t.Errorf("entry[0] = %+v", entries[0])
	}
	if entries[1].serial != "192.168.1.13:38349" || entries[1].state != "device" {
		t.Errorf("entry[1] = %+v", entries[1])
	}
	if entries[2].state != "unauthorized" {
		t.Errorf("entry[2] = %+v", entries[2])
	}
}
