package locators

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ErrLogic/regbot/internal/appium"
)

// validBy is the set of accepted locator strategies.
var validBy = map[string]bool{
	appium.ByID:              true,
	appium.ByAccessibilityID: true,
	appium.ByXPath:           true,
	appium.ByUIAutomator:     true,
	"class name":             true,
}

// Selector is a single candidate for locating an element. TODO carries an
// optional note when the selector is a best guess pending verification.
type Selector struct {
	By       string `json:"by"`
	Selector string `json:"selector"`
	TODO     string `json:"todo,omitempty"`
}

// Map holds the locators for one app: a logical element name maps to an ordered
// list of candidate selectors (first match wins). Source is the file it was
// loaded from, used in error messages.
type Map struct {
	Version  string
	Source   string
	Elements map[string][]Selector
}

// file is the on-disk JSON model.
type file struct {
	Version  string                `json:"version"`
	Elements map[string][]Selector `json:"elements"`
}

// Load reads and validates locators/<app>.json from dir. It fails with a precise
// message on a missing file, malformed JSON, an unknown `by` strategy, or an
// element with no/empty selectors.
func Load(dir, app string) (Map, error) {
	path := filepath.Join(dir, app+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return Map{}, fmt.Errorf("read locators %q: %w", path, err)
	}

	var f file
	if err := json.Unmarshal(raw, &f); err != nil {
		return Map{}, fmt.Errorf("parse locators %q: %w", path, err)
	}

	for name, sels := range f.Elements {
		if len(sels) == 0 {
			return Map{}, fmt.Errorf("locators %q: element %q has no selectors", path, name)
		}
		for i, s := range sels {
			if strings.TrimSpace(s.Selector) == "" {
				return Map{}, fmt.Errorf("locators %q: element %q selector[%d] has empty value", path, name, i)
			}
			if !validBy[s.By] {
				return Map{}, fmt.Errorf("locators %q: element %q selector[%d] has unknown by %q", path, name, i, s.By)
			}
		}
	}

	if f.Elements == nil {
		f.Elements = map[string][]Selector{}
	}
	return Map{Version: f.Version, Source: path, Elements: f.Elements}, nil
}

// Require verifies that every named element exists, returning an error listing
// the missing names and the source file.
func (m Map) Require(names ...string) error {
	var missing []string
	for _, n := range names {
		if len(m.Elements[n]) == 0 {
			missing = append(missing, n)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("locators %s: missing elements: %s", m.Source, strings.Join(missing, ", "))
	}
	return nil
}

// Candidates returns the ordered selectors for name, and whether it is defined.
func (m Map) Candidates(name string) ([]Selector, bool) {
	sels, ok := m.Elements[name]
	return sels, ok && len(sels) > 0
}

// Resolve locates the element named by name, trying each candidate selector in
// order (first match wins). The timeout is divided across the candidates so the
// total wait is bounded regardless of how many candidates are configured.
func (m Map) Resolve(ctx context.Context, driver *appium.Driver, name string, timeout time.Duration) (*appium.Element, error) {
	cands, ok := m.Candidates(name)
	if !ok {
		return nil, fmt.Errorf("locators %s: unknown element %q", m.Source, name)
	}

	perCandidate := timeout / time.Duration(len(cands))
	if perCandidate <= 0 {
		perCandidate = timeout
	}

	var lastErr error
	for _, s := range cands {
		el, err := driver.WaitForElement(ctx, s.By, s.Selector, perCandidate)
		if err == nil {
			return el, nil
		}
		if ctx.Err() != nil {
			return nil, fmt.Errorf("resolve %q: %w", name, ctx.Err())
		}
		lastErr = err
	}
	return nil, fmt.Errorf("resolve %q: no candidate matched: %w", name, lastErr)
}
