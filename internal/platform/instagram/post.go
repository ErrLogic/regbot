package instagram

import (
	"context"
	"fmt"
	"time"

	"github.com/ErrLogic/regbot/internal/appium"
	"github.com/ErrLogic/regbot/internal/config"
	"github.com/ErrLogic/regbot/internal/flows"
	"github.com/ErrLogic/regbot/internal/job"
	"github.com/ErrLogic/regbot/internal/locators"
)

// CreatePost creates a new Instagram post with uploaded media and a caption.
func CreatePost(
	ctx context.Context,
	driver *appium.Driver,
	loc locators.Map,
	cfg config.Config,
	params job.CreatePostParams,
	logFunc func(string, string, string),
) error {
	wait := 15 * time.Second
	probe := 2 * time.Second

	logFunc("info", "create_post", fmt.Sprintf("Creating post with %d media files", len(params.MediaIDs)))

	steps := []struct {
		name string
		run  func(context.Context) error
	}{
		{"tap create button", func(ctx context.Context) error {
			return flows.TapByLocator(ctx, driver, loc, "create_post_button", wait)
		}},
		{"select media", func(ctx context.Context) error {
			// Select the first media item from the gallery.
			// Media should already be pushed to the device by the worker.
			return flows.TapByLocator(ctx, driver, loc, "gallery_first_item", wait)
		}},
		{"tap next after media", func(ctx context.Context) error {
			flows.DismissIfPresent(ctx, driver, loc, "skip_button", probe)
			return flows.TapByLocator(ctx, driver, loc, "next_button", wait)
		}},
		{"enter caption", func(ctx context.Context) error {
			if params.Caption == "" {
				return nil
			}
			return flows.TypeByLocator(ctx, driver, loc, "caption_field", params.Caption, wait)
		}},
		{"tap share", func(ctx context.Context) error {
			return flows.TapByLocator(ctx, driver, loc, "share_button", wait)
		}},
		{"wait for publish", func(ctx context.Context) error {
			time.Sleep(3 * time.Second)
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

	logFunc("info", "create_post", "Post published successfully")
	return nil
}
