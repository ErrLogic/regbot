package instagram

import (
	"context"
	"time"

	"github.com/ErrLogic/regbot/internal/appium"
	"github.com/ErrLogic/regbot/internal/flows"
	"github.com/ErrLogic/regbot/internal/job"
	"github.com/ErrLogic/regbot/internal/locators"
)

// LikePost navigates to a post URL and taps the like button.
func LikePost(
	ctx context.Context,
	driver *appium.Driver,
	loc locators.Map,
	params job.LikeParams,
	logFunc func(string, string, string),
) error {
	wait := 15 * time.Second
	probe := 2 * time.Second

	logFunc("info", "like", "Starting like post flow for: "+params.PostURL)

	steps := []struct {
		name string
		run  func(context.Context) error
	}{
		{"navigate to post", func(ctx context.Context) error {
			// Use search or intent-based navigation.
			// First, try to open search.
			if err := flows.TapByLocator(ctx, driver, loc, "search_button", wait); err != nil {
				return err
			}
			// Type the post URL in the search field.
			if err := flows.TypeByLocator(ctx, driver, loc, "search_field", params.PostURL, wait); err != nil {
				return err
			}
			return nil
		}},
		{"open post", func(ctx context.Context) error {
			// Tap the first search result.
			return flows.TapByLocator(ctx, driver, loc, "search_result_first", wait)
		}},
		{"tap like", func(ctx context.Context) error {
			return flows.TapByLocator(ctx, driver, loc, "like_button", wait)
		}},
		{"dismiss overlays", func(ctx context.Context) error {
			flows.DismissIfPresent(ctx, driver, loc, "skip_button", probe)
			return nil
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

	logFunc("info", "like", "Like post completed")
	return nil
}
