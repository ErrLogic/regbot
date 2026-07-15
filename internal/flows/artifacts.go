package flows

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"github.com/ErrLogic/regbot/internal/appium"
)

// ArtifactSink captures a screenshot and page source for a failing step and
// writes them to a directory, named by run id and step. It implements
// FailureSink.
type ArtifactSink struct {
	driver *appium.Driver
	dir    string
	runID  string
	logger *zap.Logger
}

// NewArtifactSink builds an ArtifactSink writing under dir with filenames
// prefixed by runID.
func NewArtifactSink(driver *appium.Driver, dir, runID string, logger *zap.Logger) *ArtifactSink {
	return &ArtifactSink{driver: driver, dir: dir, runID: runID, logger: logger}
}

// Capture writes a screenshot (.png) and page source (.xml) for the failing
// step. Errors capturing artifacts are logged, not returned.
func (a *ArtifactSink) Capture(ctx context.Context, step string, cause error) {
	if err := os.MkdirAll(a.dir, 0o750); err != nil {
		a.logger.Warn("create artifacts dir", zap.String("dir", a.dir), zap.Error(err))
		return
	}
	base := filepath.Join(a.dir, a.runID+"-"+sanitize(step))

	if png, err := a.driver.Screenshot(ctx); err == nil {
		if werr := os.WriteFile(base+".png", png, 0o600); werr != nil {
			a.logger.Warn("write screenshot", zap.Error(werr))
		}
	} else {
		a.logger.Warn("capture screenshot", zap.Error(err))
	}

	if src, err := a.driver.PageSource(ctx); err == nil {
		if werr := os.WriteFile(base+".xml", []byte(src), 0o600); werr != nil {
			a.logger.Warn("write page source", zap.Error(werr))
		}
	} else {
		a.logger.Warn("capture page source", zap.Error(err))
	}

	a.logger.Warn("captured failure artifacts",
		zap.String("step", step),
		zap.String("prefix", base),
		zap.Error(cause))
}

// sanitize makes a step name safe for use in a filename.
func sanitize(s string) string {
	r := strings.NewReplacer(" ", "-", "/", "-", "\\", "-", ":", "-")
	return strings.ToLower(r.Replace(strings.TrimSpace(s)))
}
