// Package automation orchestrates platform-specific actions for background job execution.
package automation

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ErrLogic/regbot/internal/adb"
	"github.com/ErrLogic/regbot/internal/appium"
	"github.com/ErrLogic/regbot/internal/config"
	"github.com/ErrLogic/regbot/internal/crypto"
	"github.com/ErrLogic/regbot/internal/db"
	"github.com/ErrLogic/regbot/internal/flows"
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
	accounts *db.AccountStore
}

// NewService creates an automation service.
func NewService(cfg config.Config, sessions *session.Pool, accounts *db.AccountStore) *Service {
	return &Service{cfg: cfg, sessions: sessions, accounts: accounts}
}

// Execute runs a job against the appropriate platform action.
func (s *Service) Execute(
	ctx context.Context,
	j *db.Job,
	logFunc func(string, string, string),
) error {
	logFunc("info", "", fmt.Sprintf("Executing %s/%s", j.Platform, j.Type))

	caps := appium.Capabilities{
		PlatformName:      s.cfg.Device.PlatformName,
		AutomationName:    s.cfg.Device.AutomationName,
		DeviceName:        s.cfg.Device.DeviceName,
		UDID:              j.DeviceSerial,
		NewCommandTimeout: s.cfg.Appium.NewCommandTimeout,
		NoReset:           j.Type != string(job.TypeRegister),
	}

	targetPkg, err := s.targetPackage(j.Platform)
	if err != nil {
		return err
	}
	caps.AppPackage = targetPkg
	if j.Platform == "instagram" {
		caps.AppActivity = s.cfg.Apps.InstagramActivity
	}

	adbClient := adb.New(adb.WithSerial(j.DeviceSerial))
	if err := adbClient.EnsureConnected(ctx); err != nil {
		logFunc("warn", "", fmt.Sprintf("device reconnect check: %v", err))
	}

	if j.Type == string(job.TypeRegister) {
		s.sessions.Release(ctx, j.DeviceSerial)
		logFunc("info", "", "clearing "+targetPkg+" data for a fresh registration")
		if err := adbClient.ClearApp(ctx, targetPkg); err != nil {
			logFunc("warn", "", fmt.Sprintf("clear app data: %v", err))
		}
		time.Sleep(2 * time.Second)
	}

	driver, err := s.sessions.Acquire(ctx, j.DeviceSerial, caps)
	if err != nil {
		return fmt.Errorf("acquire session: %w", err)
	}

	targetLoc, err := locators.Load(s.cfg.Paths.LocatorsDir, j.Platform)
	if err != nil {
		return fmt.Errorf("load locators: %w", err)
	}

	parsed, err := job.ParseParams(job.Type(j.Type), json.RawMessage(j.Params))
	if err != nil {
		return fmt.Errorf("parse params: %w", err)
	}

	switch j.Platform {
	case "instagram":
		return s.execIG(ctx, driver, targetLoc, j, parsed, logFunc)
	case "tiktok":
		return s.execTT(ctx, driver, targetLoc, j, parsed, logFunc)
	default:
		return fmt.Errorf("unsupported platform: %s", j.Platform)
	}
}

func (s *Service) execIG(ctx context.Context, driver *appium.Driver, loc locators.Map, j *db.Job, params any, logFunc LogFunc) error {
	// Register returns account info that must be persisted.
	if j.Type == "register" {
		p := params.(job.RegisterParams)
		acct, err := instagram.Register(ctx, driver, loc, s.cfg, p, logFunc)
		if err != nil {
			return err
		}
		s.saveAccount(ctx, acct, j)
		return nil
	}
	switch j.Type {
	case "like":
		return instagram.LikePost(ctx, driver, loc, params.(job.LikeParams), logFunc)
	case "comment":
		return instagram.CommentPost(ctx, driver, loc, params.(job.CommentParams), logFunc)
	case "update_profile":
		return instagram.UpdateProfile(ctx, driver, loc, params.(job.UpdateProfileParams), logFunc)
	case "create_post":
		return instagram.CreatePost(ctx, driver, loc, s.cfg, params.(job.CreatePostParams), logFunc)
	case "watch_live":
		return instagram.WatchLive(ctx, driver, loc, params.(job.WatchLiveParams), logFunc)
	default:
		return fmt.Errorf("unsupported instagram job type: %s", j.Type)
	}
}

func (s *Service) execTT(ctx context.Context, driver *appium.Driver, loc locators.Map, j *db.Job, params any, logFunc LogFunc) error {
	if j.Type == "register" {
		p := params.(job.RegisterParams)
		acct, err := tiktok.Register(ctx, driver, loc, s.cfg, p, logFunc)
		if err != nil {
			return err
		}
		s.saveAccount(ctx, acct, j)
		return nil
	}
	switch j.Type {
	case "like":
		return tiktok.LikePost(ctx, driver, loc, params.(job.LikeParams), logFunc)
	case "comment":
		return tiktok.CommentPost(ctx, driver, loc, params.(job.CommentParams), logFunc)
	case "update_profile":
		return tiktok.UpdateProfile(ctx, driver, loc, params.(job.UpdateProfileParams), logFunc)
	case "create_post":
		return tiktok.CreatePost(ctx, driver, loc, s.cfg, params.(job.CreatePostParams), logFunc)
	case "watch_live":
		return tiktok.WatchLive(ctx, driver, loc, params.(job.WatchLiveParams), logFunc)
	default:
		return fmt.Errorf("unsupported tiktok job type: %s", j.Type)
	}
}

func (s *Service) saveAccount(ctx context.Context, acct *flows.Account, j *db.Job) {
	if acct == nil || s.accounts == nil || acct.Password == "" {
		return
	}
	key := sha256.Sum256([]byte(s.cfg.Server.JWTSecret))
	encrypted, err := crypto.Encrypt([]byte(acct.Password), key[:])
	if err != nil {
		return
	}
	pa := &db.PlatformAccount{
		Platform:          string(acct.Platform),
		Email:             acct.Email,
		Username:          acct.Username,
		EncryptedPassword: encrypted,
		Status:            "active",
		JobID:             j.ID,
		DeviceSerial:      j.DeviceSerial,
	}
	_ = s.accounts.Create(ctx, pa)
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
