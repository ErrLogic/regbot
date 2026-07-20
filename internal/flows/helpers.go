package flows

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ErrLogic/regbot/internal/appium"
	"github.com/ErrLogic/regbot/internal/locators"
)

// tapByLocator resolves the named element and taps it.
func tapByLocator(ctx context.Context, driver *appium.Driver, loc locators.Map, name string, wait time.Duration) error {
	el, err := loc.Resolve(ctx, driver, name, wait)
	if err != nil {
		return fmt.Errorf("tap %q: %w", name, err)
	}
	if err := el.Click(ctx); err != nil {
		return fmt.Errorf("tap %q: %w", name, err)
	}
	return nil
}

// TapByLocator resolves the named element and taps it (exported).
func TapByLocator(ctx context.Context, driver *appium.Driver, loc locators.Map, name string, wait time.Duration) error {
	return tapByLocator(ctx, driver, loc, name, wait)
}

// typeByLocator resolves the named element, clears any existing text, and types
// the new text into it.
func typeByLocator(ctx context.Context, driver *appium.Driver, loc locators.Map, name, text string, wait time.Duration) error {
	el, err := loc.Resolve(ctx, driver, name, wait)
	if err != nil {
		return fmt.Errorf("type into %q: %w", name, err)
	}
	// Clear any pre-filled value first so SendKeys doesn't append.
	_ = el.Clear(ctx)
	if err := el.SendKeys(ctx, text); err != nil {
		return fmt.Errorf("type into %q: %w", name, err)
	}
	return nil
}

// TypeByLocator resolves the named element and types text into it (exported).
func TypeByLocator(ctx context.Context, driver *appium.Driver, loc locators.Map, name, text string, wait time.Duration) error {
	return typeByLocator(ctx, driver, loc, name, text, wait)
}

// dismissIfPresent taps the named element if it is present within wait, and
// reports whether it did. A not-found result is not an error (the interstitial
// simply was not shown).
func dismissIfPresent(ctx context.Context, driver *appium.Driver, loc locators.Map, name string, wait time.Duration) bool {
	el, err := loc.Resolve(ctx, driver, name, wait)
	if err != nil {
		return false
	}
	if err := el.Click(ctx); err != nil {
		return false
	}
	return true
}

// DismissIfPresent taps the named element if it is visible within wait (exported).
func DismissIfPresent(ctx context.Context, driver *appium.Driver, loc locators.Map, name string, wait time.Duration) bool {
	return dismissIfPresent(ctx, driver, loc, name, wait)
}

// isPresent reports whether the named element is visible within wait. It treats
// not-found/timeout as "absent" and any other error as absent too (callers use
// this for best-effort branching).
func isPresent(ctx context.Context, driver *appium.Driver, loc locators.Map, name string, wait time.Duration) bool {
	_, err := loc.Resolve(ctx, driver, name, wait)
	if err == nil {
		return true
	}
	if errors.Is(err, appium.ErrElementNotFound) || errors.Is(err, appium.ErrTimeout) {
		return false
	}
	return false
}

// IsPresent reports whether the named element is visible within wait (exported).
func IsPresent(ctx context.Context, driver *appium.Driver, loc locators.Map, name string, wait time.Duration) bool {
	return isPresent(ctx, driver, loc, name, wait)
}
