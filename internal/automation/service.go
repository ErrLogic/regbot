// Package automation orchestrates platform-specific actions for background job execution.
package automation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ErrLogic/regbot/internal/adb"
	"github.com/ErrLogic/regbot/internal/appium"
	"github.com/ErrLogic/regbot/internal/config"
	"github.com/ErrLogic/regbot/internal/db"
	"github.com/ErrLogic/regbot/internal/job"
	"github.com/ErrLogic/regbot/internal/locators"
	"github.com/ErrLogic/regbot/internal/platform/instagram"
	"github.com/ErrLogic/regbot/internal/platform/tiktok"
	"github.com/ErrLogic/regbot/internal/session"
)

// LogFunc is called by automation steps to report progress.
type LogFunc func(level, step, message string)

// Service dispatches jobs to the correct platform action.
type Service struct {
	cfg      config.Config
	sessions *session.Pool
}

// NewService creates an automation service.
func NewService(cfg config.Config, sessions *session.Pool) *Service {
	return &Service{cfg: cfg, sessions: sessions}
}

// Execute runs a job against the appropriate platform action.
// Implements job.Executor.
func (s *Service) Execute(
	ctx context.Context,
	j *db.Job,
	logFunc func(string, string, string),
) error {
	logFunc("info", "", fmt.Sprintf("Executing %s/%s", j.Platform, j.Type))

	// Resolve device caps.
	caps := appium.Capabilities{
		PlatformName:      s.cfg.Device.PlatformName,
		AutomationName:    s.cfg.Device.AutomationName,
		DeviceName:        s.cfg.Device.DeviceName,
		UDID:              j.DeviceSerial,
		NewCommandTimeout: s.cfg.Appium.NewCommandTimeout,
		// Registration needs a clean, logged-out app; all other actions run
		// against the existing logged-in session.
		NoReset: j.Type != string(job.TypeRegister),
	}

	targetPkg, err := s.targetPackage(j.Platform)
	if err != nil {
		return err
	}
	caps.AppPackage = targetPkg
	if j.Platform == "instagram" {
		caps.AppActivity = s.cfg.Apps.InstagramActivity
	}

	// Ensure the device is reachable (recovers wireless/TLS idle drops) before
	// opening a session.
	adbClient := adb.New(adb.WithSerial(j.DeviceSerial))
	if err := adbClient.EnsureConnected(ctx); err != nil {
		logFunc("warn", "", fmt.Sprintf("device reconnect check: %v", err))
	}

	// Registration needs the target app on its logged-out welcome screen. Clear
	// its data first so it always starts fresh, regardless of prior state.
	if j.Type == string(job.TypeRegister) {
		logFunc("info", "", "clearing "+targetPkg+" data for a fresh registration")
		if err := adbClient.ClearApp(ctx, targetPkg); err != nil {
			logFunc("warn", "", fmt.Sprintf("clear app data: %v", err))
		}
	}

	// Acquire Appium session.
	driver, err := s.sessions.Acquire(ctx, j.DeviceSerial, caps)
	if err != nil {
		return fmt.Errorf("acquire session: %w", err)
	}

	// Load locators.
	targetLoc, err := locators.Load(s.cfg.Paths.LocatorsDir, j.Platform)
	if err != nil {
		return fmt.Errorf("load locators: %w", err)
	}

	// Parse params.
	parsed, err := job.ParseParams(job.Type(j.Type), json.RawMessage(j.Params))
	if err != nil {
		return fmt.Errorf("parse params: %w", err)
	}

	// Dispatch to platform.
	switch j.Platform {
	case "instagram":
		return s.execInstagram(ctx, driver, targetLoc, j.Type, parsed, logFunc)
	case "tiktok":
		return s.execTikTok(ctx, driver, targetLoc, j.Type, parsed, logFunc)
	default:
		return fmt.Errorf("unsupported platform: %s", j.Platform)
	}
}

func (s *Service) execInstagram(
	ctx context.Context, driver *appium.Driver, loc locators.Map,
	jobType string, params any, logFunc LogFunc,
) error {
	switch jobType {
	case "register":
		p := params.(job.RegisterParams)
		return instagram.Register(ctx, driver, loc, s.cfg, p, logFunc)
	case "like":
		p := params.(job.LikeParams)
		return instagram.LikePost(ctx, driver, loc, p, logFunc)
	case "comment":
		p := params.(job.CommentParams)
		return instagram.CommentPost(ctx, driver, loc, p, logFunc)
	case "update_profile":
		p := params.(job.UpdateProfileParams)
		return instagram.UpdateProfile(ctx, driver, loc, p, logFunc)
	case "create_post":
		p := params.(job.CreatePostParams)
		return instagram.CreatePost(ctx, driver, loc, s.cfg, p, logFunc)
	case "watch_live":
		p := params.(job.WatchLiveParams)
		return instagram.WatchLive(ctx, driver, loc, p, logFunc)
	default:
		return fmt.Errorf("unsupported instagram job type: %s", jobType)
	}
}

func (s *Service) execTikTok(
	ctx context.Context, driver *appium.Driver, loc locators.Map,
	jobType string, params any, logFunc LogFunc,
) error {
	switch jobType {
	case "register":
		p := params.(job.RegisterParams)
		return tiktok.Register(ctx, driver, loc, s.cfg, p, logFunc)
	case "like":
		p := params.(job.LikeParams)
		return tiktok.LikePost(ctx, driver, loc, p, logFunc)
	case "comment":
		p := params.(job.CommentParams)
		return tiktok.CommentPost(ctx, driver, loc, p, logFunc)
	case "update_profile":
		p := params.(job.UpdateProfileParams)
		return tiktok.UpdateProfile(ctx, driver, loc, p, logFunc)
	case "create_post":
		p := params.(job.CreatePostParams)
		return tiktok.CreatePost(ctx, driver, loc, s.cfg, p, logFunc)
	case "watch_live":
		p := params.(job.WatchLiveParams)
		return tiktok.WatchLive(ctx, driver, loc, p, logFunc)
	default:
		return fmt.Errorf("unsupported tiktok job type: %s", jobType)
	}
}

func (s *Service) targetPackage(platform string) (string, error) {
	switch platform {
	case "instagram":
		return s.cfg.Apps.InstagramPackage, nil
	case "tiktok":
		return s.cfg.Apps.TikTokPackage, nil
	default:
		return "", fmt.Errorf("unknown platform: %s", platform)
	}
}
