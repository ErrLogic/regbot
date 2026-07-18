package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/ErrLogic/regbot/internal/adb"
	"github.com/ErrLogic/regbot/internal/config"
	"github.com/ErrLogic/regbot/internal/core"
	"github.com/ErrLogic/regbot/internal/flows"
)

// registerFlags holds flags scoped to the register command group.
type registerFlags struct {
	email string
	sso   bool
}

// newRegisterCmd builds the `register` command group and its platform
// subcommands.
func newRegisterCmd(gf *globalFlags) *cobra.Command {
	rf := &registerFlags{}
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a new account on a supported platform",
		Long:  "Register a new account on a supported platform (instagram, tiktok).",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.PersistentFlags().StringVar(&rf.email, "email", "", "target email address (overrides config)")
	cmd.PersistentFlags().BoolVar(&rf.sso, "sso", false, "register via Google single-sign-on (TikTok only; skips email OTP)")

	cmd.AddCommand(newPlatformCmd(gf, rf, flows.PlatformInstagram))
	cmd.AddCommand(newPlatformCmd(gf, rf, flows.PlatformTikTok))
	return cmd
}

// newPlatformCmd builds a `register <platform>` subcommand.
func newPlatformCmd(gf *globalFlags, rf *registerFlags, platform flows.Platform) *cobra.Command {
	return &cobra.Command{
		Use:   string(platform),
		Short: fmt.Sprintf("Register a new %s account", platform),
		RunE: func(_ *cobra.Command, _ []string) error {
			return runRegister(gf, rf, platform)
		},
	}
}

// runRegister loads config, wires dependencies, runs the flow, and prints the
// account as JSON to stdout on success.
func runRegister(gf *globalFlags, rf *registerFlags, platform flows.Platform) error {
	cfg, err := config.Load(gf.configPath)
	if err != nil {
		return usageErrorf("load config: %w", err)
	}
	if gf.logLevel != "" {
		cfg.Logging.Level = gf.logLevel
	}
	if rf.sso {
		cfg.Account.UseGoogleSSO = true
	}
	if err := cfg.Validate(); err != nil {
		return usageErrorf("invalid config: %w", err)
	}

	logger, err := config.NewLogger(cfg.Logging)
	if err != nil {
		return usageErrorf("build logger: %w", err)
	}
	defer func() { _ = logger.Sync() }()

	email := rf.email
	if email == "" && !cfg.Account.UseGoogleSSO {
		// SSO uses the on-device Google account, so an email is not required.
		email, err = core.ResolveEmail(cfg.Email)
		if err != nil {
			return usageErrorf("resolve email: %w", err)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	svc := core.NewService(logger, adb.New(adb.WithSerial(cfg.Device.UDID)))
	acct, err := svc.Register(ctx, platform, email, cfg, gf.dryRun)
	if err != nil {
		return err
	}

	// Credentials are the only thing printed to stdout, as a single JSON object.
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(acct); err != nil {
		logger.Error("encode account", zap.Error(err))
		return err
	}
	return nil
}
