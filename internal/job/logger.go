package job

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// LogEntry is a single log event emitted during job execution.
type LogEntry struct {
	JobID     string    `json:"job_id"`
	Level     string    `json:"level"` // debug, info, warn, error
	Step      string    `json:"step"`  // current automation step
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// LogStreamer fans out log entries to connected SSE clients via Redis Pub/Sub.
type LogStreamer struct {
	rdb  *redis.Client
	mu   sync.RWMutex
	subs map[string]map[string]chan LogEntry // jobID -> subID -> chan
}

// NewLogStreamer creates a log streamer.
func NewLogStreamer(rdb *redis.Client) *LogStreamer {
	return &LogStreamer{
		rdb:  rdb,
		subs: make(map[string]map[string]chan LogEntry),
	}
}

// Publish sends a log entry to the Redis channel for the job.
func (s *LogStreamer) Publish(ctx context.Context, entry LogEntry) {
	entry.Timestamp = time.Now().UTC()
	data, _ := json.Marshal(entry)
	s.rdb.Publish(ctx, "logs:"+entry.JobID, string(data))
}

// Subscribe returns a channel that receives log entries for a job.
// Call the returned unsubscribe function to clean up.
func (s *LogStreamer) Subscribe(ctx context.Context, jobID string) (<-chan LogEntry, func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan LogEntry, 64)
	subID := randomID(8)

	if s.subs[jobID] == nil {
		s.subs[jobID] = make(map[string]chan LogEntry)
	}
	s.subs[jobID][subID] = ch

	// Subscribe to Redis channel.
	pubsub := s.rdb.Subscribe(ctx, "logs:"+jobID)
	go func() {
		defer func() { _ = pubsub.Close() }()
		for msg := range pubsub.Channel() {
			var entry LogEntry
			if err := json.Unmarshal([]byte(msg.Payload), &entry); err != nil {
				continue
			}
			s.mu.RLock()
			for _, sub := range s.subs[jobID] {
				select {
				case sub <- entry:
				default:
					// drop if subscriber is slow
				}
			}
			s.mu.RUnlock()
		}
	}()

	unsub := func() {
		s.mu.Lock()
		delete(s.subs[jobID], subID)
		if len(s.subs[jobID]) == 0 {
			delete(s.subs, jobID)
		}
		s.mu.Unlock()
		close(ch)
	}

	return ch, unsub
}

func randomID(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}
