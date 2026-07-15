package main

import (
	"github.com/spf13/cobra"
)

// newRegisterCmd builds the `register` command group. The platform subcommands
// (`instagram`, `tiktok`) are added in a later phase; for now the group exists
// so the CLI surface and help output are in place.
func newRegisterCmd(_ *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a new account on a supported platform",
		Long:  "Register a new account on a supported platform (instagram, tiktok).",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// No platform selected: show help. Subcommands are wired in Phase 9.
			return cmd.Help()
		},
	}

	return cmd
}
