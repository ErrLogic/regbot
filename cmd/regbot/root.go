package main

import (
	"github.com/spf13/cobra"
)

// globalFlags holds the persistent flags shared by every subcommand.
type globalFlags struct {
	configPath string
	logLevel   string
	dryRun     bool
}

// newRootCmd builds the root `regbot` command and wires the persistent flags
// and the `register` command group. Subcommands are added in later phases.
func newRootCmd() *cobra.Command {
	flags := &globalFlags{}

	root := &cobra.Command{
		Use:   "regbot",
		Short: "RegBot automates email-based registration for Instagram and TikTok",
		Long: "RegBot is an educational CLI that automates email-based account\n" +
			"registration for Instagram and TikTok on an Android device. It drives\n" +
			"the target app via Appium and reads the verification code directly from\n" +
			"the on-device Gmail app.\n\n" +
			"Educational use only: automated account creation violates the Terms of\n" +
			"Service of Instagram and TikTok. See PRD.md §7.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	pf := root.PersistentFlags()
	pf.StringVar(&flags.configPath, "config", "config.yaml", "path to the YAML config file")
	pf.StringVar(&flags.logLevel, "log-level", "info", "log level: debug|info|warn|error")
	pf.BoolVar(&flags.dryRun, "dry-run", false, "validate and connect, but do not submit the final registration")

	root.AddCommand(newRegisterCmd(flags))

	return root
}
