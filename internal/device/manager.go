// Package device manages ADB device discovery, selection, and health monitoring.
package device

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ErrLogic/regbot/internal/adb"
	"github.com/ErrLogic/regbot/internal/db"
)

// Manager discovers and tracks Android devices via ADB.
type Manager struct {
	adb    *adb.Client
	store  *db.DeviceStore
	mu     sync.RWMutex
	cached []db.Device
}

// NewManager creates a device manager.
func NewManager(adbClient *adb.Client, store *db.DeviceStore) *Manager {
	return &Manager{adb: adbClient, store: store}
}

// ListDevices returns the list of connected ADB devices. It performs a fresh
// scan via `adb devices` and persists results to the database.
func (m *Manager) ListDevices(ctx context.Context) ([]db.Device, error) {
	out, err := m.adb.Devices(ctx)
	if err != nil {
		return nil, fmt.Errorf("device: scan: %w", err)
	}

	entries := parseDevices(out)
	now := time.Now().UTC()

	devices := make([]db.Device, 0, len(entries))
	for _, e := range entries {
		d := db.Device{
			Serial:     e.serial,
			State:      mapADBState(e.state),
			LastSeenAt: &now,
		}
		// Try to get model info if the device is authorised ("device" state).
		if e.state == "device" {
			if model, err := m.adb.GetProp(ctx, e.serial, "ro.product.model"); err == nil {
				d.Model = model
			}
			if av, err := m.adb.GetProp(ctx, e.serial, "ro.build.version.release"); err == nil {
				d.AndroidVersion = av
			}
		}
		// Try to load existing ID from DB.
		if existing, _ := m.store.GetBySerial(ctx, e.serial); existing != nil {
			d.ID = existing.ID
			d.CreatedAt = existing.CreatedAt
		}
		if err := m.store.Upsert(ctx, &d); err != nil {
			return nil, fmt.Errorf("device: persist %s: %w", e.serial, err)
		}
		devices = append(devices, d)
	}

	m.mu.Lock()
	m.cached = devices
	m.mu.Unlock()

	return devices, nil
}

// GetCached returns the last fetched device list (no new scan).
func (m *Manager) GetCached() []db.Device {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]db.Device, len(m.cached))
	copy(out, m.cached)
	return out
}

// mapADBState translates a raw `adb devices` state into the DB's allowed set
// (offline, online, busy, unauthorized). ADB reports "device" for an authorised,
// ready device, which maps to "online".
func mapADBState(adbState string) string {
	switch adbState {
	case "device":
		return "online"
	case "unauthorized":
		return "unauthorized"
	case "offline":
		return "offline"
	default:
		// "no permissions", "recovery", "sideload", "bootloader", etc.
		return "offline"
	}
}

// deviceEntry is one parsed row of `adb devices` output.
type deviceEntry struct {
	serial string
	state  string
}

// parseDevices parses `adb devices` output.
func parseDevices(out string) []deviceEntry {
	var entries []deviceEntry
	for _, line := range splitLines(out) {
		fields := splitFields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[0] == "List" || fields[0] == "*" {
			continue
		}
		entries = append(entries, deviceEntry{serial: fields[0], state: fields[1]})
	}
	return entries
}

// splitLines splits s by newlines, trimming whitespace and carriage returns.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := trimSpace(s[start:i])
			if line != "" {
				lines = append(lines, line)
			}
			start = i + 1
		}
	}
	if start < len(s) {
		line := trimSpace(s[start:])
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// splitFields splits s by whitespace, returning at most 3 fields.
func splitFields(s string) []string {
	var fields []string
	start := -1
	for i := 0; i < len(s) && len(fields) < 3; i++ {
		if s[i] == ' ' || s[i] == '\t' || s[i] == '\r' {
			if start >= 0 {
				fields = append(fields, s[start:i])
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}
	if start >= 0 && len(fields) < 3 {
		fields = append(fields, s[start:])
	}
	return fields
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
