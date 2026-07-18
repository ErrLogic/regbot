package tiktok

import (
	"context"
	"time"

	"github.com/ErrLogic/regbot/internal/appium"
	"github.com/ErrLogic/regbot/internal/flows"
	"github.com/ErrLogic/regbot/internal/job"
	"github.com/ErrLogic/regbot/internal/locators"
)

// LikePost navigates to a TikTok post URL and taps the like button.
func LikePost(
	ctx context.Context,
	driver *appium.Driver,
	loc locators.Map,
	params job.LikeParams,
	logFunc func(string, string, string),
) error {
	wait := 15 * time.Second

	logFunc("info", "like", "Starting TikTok like flow for: "+params.PostURL)

	steps := []struct {
		name string
		run  func(context.Context) error
	}{
		{"navigate to discover", func(ctx context.Context) error {
			return flows.TapByLocator(ctx, driver, loc, "discover_tab", wait)
		}},
		{"search post", func(ctx context.Context) error {
			return flows.TypeByLocator(ctx, driver, loc, "search_field", params.PostURL, wait)
		}},
		{"open result", func(ctx context.Context) error {
			return flows.TapByLocator(ctx, driver, loc, "search_result_first", wait)
		}},
		{"tap like", func(ctx context.Context) error {
			return flows.TapByLocator(ctx, driver, loc, "like_button", wait)
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

	logFunc("info", "like", "TikTok like completed")
	return nil
}
