package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/ErrLogic/regbot/internal/config"
	"github.com/ErrLogic/regbot/internal/flows"
	"github.com/ErrLogic/regbot/internal/locators"
	"github.com/ErrLogic/regbot/internal/otp/gmailapp"
)

// locatorCheck pairs an app with the element names a flow requires.
type locatorCheck struct {
	app      string
	required []string
}

// locatorChecks is the set of files and required names to verify.
var locatorChecks = []locatorCheck{
	{"instagram", flows.InstagramLocatorNames},
	{"tiktok", flows.TikTokLocatorNames},
	{"gmail", gmailapp.RequiredLocators},
}

// newLocatorsCmd builds the `locators` command group and its `verify`
// subcommand.
func newLocatorsCmd(gf *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "locators",
		Short: "Locator maintenance commands",
		RunE:  func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "verify",
		Short: "Load and validate all locator files, reporting drift and TODOs",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runLocatorsVerify(gf)
		},
	})
	return cmd
}

// runLocatorsVerify loads config to find the locators dir, then verifies.
func runLocatorsVerify(gf *globalFlags) error {
	cfg, err := config.Load(gf.configPath)
	if err != nil {
		return usageErrorf("load config: %w", err)
	}
	if problems := verifyLocators(os.Stdout, cfg.Paths.LocatorsDir); problems > 0 {
		return usageErrorf("%d locator file(s) have problems", problems)
	}
	return nil
}

// verifyLocators loads each locator file, checks the required element names are
// present, and reports a summary (version, element count, TODO count) to out.
// It returns the number of files with problems.
func verifyLocators(out io.Writer, dir string) int {
	problems := 0
	for _, c := range locatorChecks {
		m, err := locators.Load(dir, c.app)
		if err != nil {
			_, _ = fmt.Fprintf(out, "%-10s FAIL  %v\n", c.app, err)
			problems++
			continue
		}
		if err := m.Require(c.required...); err != nil {
			_, _ = fmt.Fprintf(out, "%-10s FAIL  %v\n", c.app, err)
			problems++
			continue
		}
		_, _ = fmt.Fprintf(out, "%-10s OK    version=%s elements=%d todo=%d\n",
			c.app, m.Version, len(m.Elements), countTODO(m))
	}
	return problems
}

// countTODO counts selectors still marked with a TODO note.
func countTODO(m locators.Map) int {
	n := 0
	for _, sels := range m.Elements {
		for _, s := range sels {
			if s.TODO != "" {
				n++
			}
		}
	}
	return n
}
