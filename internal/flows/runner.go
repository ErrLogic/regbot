package flows

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Step is one named unit of work within a flow.
type Step struct {
	Name string
	Run  func(ctx context.Context) error
}

// FailureSink is notified when a step exhausts its retries, so it can capture
// diagnostics (screenshot, page source) for the failing step.
type FailureSink interface {
	Capture(ctx context.Context, step string, cause error)
}

// RetryPolicy controls per-step retries.
type RetryPolicy struct {
	// Attempts is the maximum number of tries per step (minimum 1).
	Attempts int
	// Backoff is the delay between attempts.
	Backoff time.Duration
}

// attempts returns the effective attempt count (at least 1).
func (p RetryPolicy) attempts() int {
	if p.Attempts < 1 {
		return 1
	}
	return p.Attempts
}

// runSteps executes steps in order, logging each step's start, outcome, and
// duration. A failing step is retried per policy; if it still fails, the sink is
// notified and the error is returned wrapped with the step name.
func runSteps(ctx context.Context, logger *zap.Logger, sink FailureSink, policy RetryPolicy, steps ...Step) error {
	for _, step := range steps {
		if err := runStep(ctx, logger, sink, policy, step); err != nil {
			return err
		}
	}
	return nil
}

// runStep runs a single step with retries.
func runStep(ctx context.Context, logger *zap.Logger, sink FailureSink, policy RetryPolicy, step Step) error {
	start := time.Now()
	logger.Info("step start", zap.String("step", step.Name))

	var err error
retry:
	for attempt := 1; attempt <= policy.attempts(); attempt++ {
		if ctx.Err() != nil {
			err = ctx.Err()
			break
		}
		if err = step.Run(ctx); err == nil {
			logger.Info("step done",
				zap.String("step", step.Name),
				zap.Int("attempt", attempt),
				zap.Duration("elapsed", time.Since(start)))
			return nil
		}
		logger.Warn("step attempt failed",
			zap.String("step", step.Name),
			zap.Int("attempt", attempt),
			zap.Error(err))

		if attempt < policy.attempts() {
			select {
			case <-ctx.Done():
				err = ctx.Err()
				break retry
			case <-time.After(policy.Backoff):
			}
		}
	}

	if sink != nil {
		sink.Capture(ctx, step.Name, err)
	}
	return fmt.Errorf("step %q: %w", step.Name, err)
}
