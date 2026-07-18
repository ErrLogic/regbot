package job

import (
	"context"
	"fmt"
	"time"

	"github.com/ErrLogic/regbot/internal/db"
	"github.com/google/uuid"
)

// Logger is a convenience wrapper that writes log entries to both the database
// and the log streamer for real-time delivery.
type Logger struct {
	store    *db.JobLogStore
	streamer *LogStreamer
	jobID    uuid.UUID
}

// NewLogger creates a job-scoped logger.
func NewLogger(store *db.JobLogStore, streamer *LogStreamer, jobID uuid.UUID) *Logger {
	return &Logger{store: store, streamer: streamer, jobID: jobID}
}

// Log writes a log entry to the database and publishes it for SSE streaming.
func (l *Logger) Log(level, step, message string) {
	entry := LogEntry{
		JobID:     l.jobID.String(),
		Level:     level,
		Step:      step,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}

	// Persist to DB (fire and forget).
	go func() {
		_ = l.store.Create(context.Background(), &db.JobLog{
			JobID:   l.jobID,
			Level:   level,
			Step:    step,
			Message: message,
		})
	}()

	// Publish for real-time streaming.
	l.streamer.Publish(context.Background(), entry)
}

// Info logs at info level.
func (l *Logger) Info(step, message string) {
	l.Log("info", step, message)
}

// Warn logs at warn level.
func (l *Logger) Warn(step, message string) {
	l.Log("warn", step, message)
}

// Error logs at error level.
func (l *Logger) Error(step string, err error) {
	l.Log("error", step, fmt.Sprintf("%v", err))
}

// Debug logs at debug level.
func (l *Logger) Debug(step, message string) {
	l.Log("debug", step, message)
}
