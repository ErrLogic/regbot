package job

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/ErrLogic/regbot/internal/db"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Executor runs a job against a real device. Implemented by automation.Service.
type Executor interface {
	Execute(ctx context.Context, j *db.Job, logFunc func(string, string, string)) error
}

// Worker picks jobs from the queue and executes them via an Executor.
type Worker struct {
	queue    *Queue
	streamer *LogStreamer
	store    *db.JobStore
	logStore *db.JobLogStore
	executor Executor
	poolSize int
}

// NewWorker creates a worker.
func NewWorker(
	rdb *redis.Client,
	store *db.JobStore,
	logStore *db.JobLogStore,
	executor Executor,
	poolSize int,
) *Worker {
	return &Worker{
		queue:    NewQueue(rdb),
		streamer: NewLogStreamer(rdb),
		store:    store,
		logStore: logStore,
		executor: executor,
		poolSize: poolSize,
	}
}

// Run starts the worker pool. Blocks until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	for i := 0; i < w.poolSize; i++ {
		go w.loop(ctx, i)
	}
	<-ctx.Done()
	return nil
}

func (w *Worker) loop(ctx context.Context, id int) {
	prefix := fmt.Sprintf("[worker-%d]", id)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		jobID, platform, err := w.queue.Dequeue(ctx)
		if err != nil {
			log.Printf("%s dequeue error: %v", prefix, err)
			time.Sleep(time.Second)
			continue
		}
		if jobID == "" {
			continue
		}

		w.execute(ctx, jobID, platform, prefix)
	}
}

func (w *Worker) execute(ctx context.Context, jobID, platform, prefix string) {
	uid, err := uuid.Parse(jobID)
	if err != nil {
		log.Printf("%s invalid job id %q", prefix, jobID)
		return
	}

	j, err := w.store.GetByID(ctx, uid)
	if err != nil || j == nil {
		log.Printf("%s job %s not found: %v", prefix, jobID, err)
		return
	}

	// Acquire device lock (5-minute TTL).
	locked, err := w.queue.DeviceLock(ctx, j.DeviceSerial, 5*time.Minute)
	if err != nil || !locked {
		log.Printf("%s device %s is busy, re-queuing", prefix, j.DeviceSerial)
		return
	}
	defer w.queue.DeviceUnlock(ctx, j.DeviceSerial)

	// Mark running.
	_ = w.store.UpdateStatus(ctx, uid, string(StatusRunning), nil, "")

	jl := NewLogger(w.logStore, w.streamer, uid)
	jl.Info("", fmt.Sprintf("Job started: %s/%s on %s", j.Platform, j.Type, j.DeviceSerial))
	w.publishEvent(ctx, jobID, "job.started", "{}")

	// Execute via the executor (automation.Service).
	logFunc := func(level, step, message string) {
		switch level {
		case "error":
			jl.Error(step, fmt.Errorf("%s", message))
		default:
			jl.Log(level, step, message)
		}
	}

	err = w.executor.Execute(ctx, j, logFunc)

	if err != nil {
		_ = w.store.UpdateStatus(ctx, uid, string(StatusFailed), nil, err.Error())
		jl.Error("", err)
		w.publishEvent(ctx, jobID, "job.failed", fmt.Sprintf(`{"error":"%s"}`, err.Error()))
		log.Printf("%s job %s failed: %v", prefix, jobID, err)
	} else {
		_ = w.store.UpdateStatus(ctx, uid, string(StatusCompleted), json.RawMessage(`{}`), "")
		jl.Info("", "Job completed successfully")
		w.publishEvent(ctx, jobID, "job.completed", "{}")
		log.Printf("%s job %s completed", prefix, jobID)
	}
}

func (w *Worker) publishEvent(ctx context.Context, jobID, event, data string) {
	w.streamer.rdb.Publish(ctx, "events:"+jobID, fmt.Sprintf(`{"event":"%s","data":%s}`, event, data))
}
