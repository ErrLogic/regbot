// Package instagram implements Instagram platform automation actions.
package instagram

import (
	"context"
	"time"

	"github.com/ErrLogic/regbot/internal/appium"
	"github.com/ErrLogic/regbot/internal/flows"
	"github.com/ErrLogic/regbot/internal/job"
	"github.com/ErrLogic/regbot/internal/locators"
)

// CommentPost navigates to a post URL and posts a comment.
func CommentPost(
	ctx context.Context,
	driver *appium.Driver,
	loc locators.Map,
	params job.CommentParams,
	logFunc func(string, string, string),
) error {
	wait := 15 * time.Second
	probe := 2 * time.Second

	logFunc("info", "comment", "Starting comment flow for: "+params.PostURL)

	steps := []struct {
		name string
		run  func(context.Context) error
	}{
		{"navigate to post", func(ctx context.Context) error {
			if err := flows.TapByLocator(ctx, driver, loc, "search_button", wait); err != nil {
				return err
			}
			return flows.TypeByLocator(ctx, driver, loc, "search_field", params.PostURL, wait)
		}},
		{"open post", func(ctx context.Context) error {
			return flows.TapByLocator(ctx, driver, loc, "search_result_first", wait)
		}},
		{"tap comment field", func(ctx context.Context) error {
			return flows.TapByLocator(ctx, driver, loc, "comment_field", wait)
		}},
		{"type comment", func(ctx context.Context) error {
			return flows.TypeByLocator(ctx, driver, loc, "comment_input", params.Text, wait)
		}},
		{"submit comment", func(ctx context.Context) error {
			return flows.TapByLocator(ctx, driver, loc, "post_comment_button", wait)
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

	logFunc("info", "comment", "Comment posted successfully")
	return nil
}
