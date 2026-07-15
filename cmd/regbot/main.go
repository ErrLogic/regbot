// Command regbot is the CLI entry point for RegBot, an educational tool that
// automates email-based account registration for Instagram and TikTok on an
// Android device using Appium and the on-device Gmail app for OTP retrieval.
//
// This binary is a thin adapter over the internal packages: it parses flags,
// loads configuration, builds the logger, and delegates to internal/core.
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
