package tiktok

import (
	"context"
	"time"

	"github.com/ErrLogic/regbot/internal/appium"
	"github.com/ErrLogic/regbot/internal/flows"
	"github.com/ErrLogic/regbot/internal/job"
	"github.com/ErrLogic/regbot/internal/locators"
)

// UpdateProfile updates the TikTok profile bio, display name, or avatar.
func UpdateProfile(
	ctx context.Context,
	driver *appium.Driver,
	loc locators.Map,
	params job.UpdateProfileParams,
	logFunc func(string, string, string),
) error {
	wait := 15 * time.Second

	logFunc("info", "profile", "Starting TikTok profile update")

	steps := []struct {
		name string
		run  func(context.Context) error
	}{
		{"navigate to profile", func(ctx context.Context) error {
			return flows.TapByLocator(ctx, driver, loc, "profile_tab", wait)
		}},
		{"tap edit profile", func(ctx context.Context) error {
			return flows.TapByLocator(ctx, driver, loc, "edit_profile_button", wait)
		}},
	}

	if params.DisplayName != "" {
		steps = append(steps, struct {
			name string
			run  func(context.Context) error
		}{
			"set display name",
			func(ctx context.Context) error {
				return flows.TypeByLocator(ctx, driver, loc, "nickname_field", params.DisplayName, wait)
			},
		})
	}
	if params.Bio != "" {
		steps = append(steps, struct {
			name string
			run  func(context.Context) error
		}{
			"set bio",
			func(ctx context.Context) error {
				return flows.TypeByLocator(ctx, driver, loc, "bio_field", params.Bio, wait)
			},
		})
	}

	steps = append(steps, struct {
		name string
		run  func(context.Context) error
	}{
		"save profile",
		func(ctx context.Context) error {
			return flows.TapByLocator(ctx, driver, loc, "save_button", wait)
		},
	})

	for _, step := range steps {
		logFunc("info", step.name, "running")
		if err := step.run(ctx); err != nil {
			logFunc("error", step.name, err.Error())
			return err
		}
		logFunc("info", step.name, "done")
	}

	logFunc("info", "profile", "TikTok profile updated")
	return nil
}
