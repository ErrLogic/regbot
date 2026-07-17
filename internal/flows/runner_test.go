package flows

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

type fakeSink struct {
	steps  []string
	causes []error
}

func (f *fakeSink) Capture(_ context.Context, step string, cause error) {
	f.steps = append(f.steps, step)
	f.causes = append(f.causes, cause)
}

func nopLogger() *zap.Logger { return zap.NewNop() }

func TestRunStepsOrder(t *testing.T) {
	var order []string
	mk := func(name string) Step {
		return Step{Name: name, Run: func(context.Context) error {
			order = append(order, name)
			return nil
		}}
	}
	sink := &fakeSink{}
	err := runSteps(context.Background(), nopLogger(), sink, RetryPolicy{Attempts: 1},
		mk("a"), mk("b"), mk("c"))
	if err != nil {
		t.Fatalf("runSteps: %v", err)
	}
	if strings.Join(order, ",") != "a,b,c" {
		t.Errorf("order = %v", order)
	}
	if len(sink.steps) != 0 {
		t.Errorf("sink should not be called on success: %v", sink.steps)
	}
}

func TestRunStepsRetrySucceeds(t *testing.T) {
	calls := 0
	step := Step{Name: "flaky", Run: func(context.Context) error {
		calls++
		if calls < 2 {
			return errors.New("transient")
		}
		return nil
	}}
	sink := &fakeSink{}
	err := runSteps(context.Background(), nopLogger(), sink,
		RetryPolicy{Attempts: 3, Backoff: time.Millisecond}, step)
	if err != nil {
		t.Fatalf("runSteps: %v", err)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
	if len(sink.steps) != 0 {
		t.Errorf("sink should not be called when a retry succeeds")
	}
}

func TestRunStepsFailureCapturesArtifact(t *testing.T) {
	cause := errors.New("kaboom")
	step := Step{Name: "boom", Run: func(context.Context) error { return cause }}
	sink := &fakeSink{}

	err := runSteps(context.Background(), nopLogger(), sink,
		RetryPolicy{Attempts: 2, Backoff: time.Millisecond}, step)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `step "boom"`) {
		t.Errorf("error should name the step: %v", err)
	}
	if !errors.Is(err, cause) {
		t.Errorf("error should wrap the cause: %v", err)
	}
	if len(sink.steps) != 1 || sink.steps[0] != "boom" {
		t.Errorf("sink steps = %v", sink.steps)
	}
}

func TestRunStepsContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ran := false
	step := Step{Name: "never", Run: func(context.Context) error {
		ran = true
		return nil
	}}
	sink := &fakeSink{}
	err := runSteps(ctx, nopLogger(), sink, RetryPolicy{Attempts: 3}, step)
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
	if ran {
		t.Error("step should not run under cancelled context")
	}
}
