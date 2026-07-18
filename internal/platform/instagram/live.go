package instagram

import (
	"context"
	"time"

	"github.com/ErrLogic/regbot/internal/appium"
	"github.com/ErrLogic/regbot/internal/flows"
	"github.com/ErrLogic/regbot/internal/job"
	"github.com/ErrLogic/regbot/internal/locators"
)

// WatchLive navigates to a live stream URL and watches it for the specified duration.
func WatchLive(
	ctx context.Context,
	driver *appium.Driver,
	loc locators.Map,
	params job.WatchLiveParams,
	logFunc func(string, string, string),
) error {
	wait := 15 * time.Second

	logFunc("info", "watch_live", "Navigating to live stream: "+params.LiveURL)

	steps := []struct {
		name string
		run  func(context.Context) error
	}{
		{"navigate to live", func(ctx context.Context) error {
			if err := flows.TapByLocator(ctx, driver, loc, "search_button", wait); err != nil {
				return err
			}
			return flows.TypeByLocator(ctx, driver, loc, "search_field", params.LiveURL, wait)
		}},
		{"tap live result", func(ctx context.Context) error {
			return flows.TapByLocator(ctx, driver, loc, "search_result_live", wait)
		}},
	}

	for _, step := range steps {
		logFunc("info", step.name, "running")
		if err := step.run(ctx); err != nil {
			logFunc("error", step.name, err.Error())
			return err
		}
		logFunc("info", step.name, "done")
	}

	// Watch for the specified duration.
	duration := time.Duration(params.DurationSeconds) * time.Second
	if duration <= 0 {
		duration = 60 * time.Second
	}
	logFunc("info", "watch_live", "Watching live stream...")
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
	}

	logFunc("info", "watch_live", "Live stream watch completed")
	return nil
}
