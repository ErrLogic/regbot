//go:build tools

// This file pins runtime dependencies that are declared for the project but not
// yet imported by any package (they are wired up in later phases). The `tools`
// build tag keeps this file out of normal builds while ensuring `go mod tidy`
// retains the modules. Remove entries here once the packages are imported for
// real.
package tools

import (
	_ "github.com/spf13/viper"
	_ "go.uber.org/zap"
)
