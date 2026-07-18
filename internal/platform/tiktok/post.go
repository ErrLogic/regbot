package tiktok

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

// CreatePost creates a new TikTok post with uploaded media and a caption.
func CreatePost(
	ctx context.Context,
	driver *appium.Driver,
	loc locators.Map,
	cfg config.Config,
	params job.CreatePostParams,
	logFunc func(string, string, string),
) error {
	wait := 15 * time.Second

	logFunc("info", "create_post", fmt.Sprintf("Creating TikTok post with %d media files", len(params.MediaIDs)))

	steps := []struct {
		name string
		run  func(context.Context) error
	}{
		{"tap create button", func(ctx context.Context) error {
			return flows.TapByLocator(ctx, driver, loc, "create_post_button", wait)
		}},
		{"select media", func(ctx context.Context) error {
			return flows.TapByLocator(ctx, driver, loc, "upload_media_button", wait)
		}},
		{"pick from gallery", func(ctx context.Context) error {
			return flows.TapByLocator(ctx, driver, loc, "gallery_first_item", wait)
		}},
		{"tap next", func(ctx context.Context) error {
			return flows.TapByLocator(ctx, driver, loc, "next_button", wait)
		}},
		{"enter caption", func(ctx context.Context) error {
			if params.Caption == "" {
				return nil
			}
			return flows.TypeByLocator(ctx, driver, loc, "caption_field", params.Caption, wait)
		}},
		{"tap post", func(ctx context.Context) error {
			return flows.TapByLocator(ctx, driver, loc, "post_button", wait)
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

	logFunc("info", "create_post", "TikTok post published")
	return nil
}
