package job

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisAddr returns the local Redis address, allowing override via REGBOT_REDIS_ADDR.
func redisAddr() string {
	if a := os.Getenv("REGBOT_REDIS_ADDR"); a != "" {
		return a
	}
	return "localhost:6379"
}

// newTestRedis connects to a local Redis or skips the test if unavailable.
func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr(), DB: 15}) // DB 15 = test scratch
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not available at %s: %v", redisAddr(), err)
	}
	// Clean the test DB before use.
	rdb.FlushDB(ctx)
	t.Cleanup(func() {
		rdb.FlushDB(context.Background())
		_ = rdb.Close()
	})
	return rdb
}

// TestEnqueueDequeueRoundtrip verifies a job can be enqueued and read back. This
// guards against the XREADGROUP argument-ordering bug (keys first, then IDs).
func TestEnqueueDequeueRoundtrip(t *testing.T) {
	rdb := newTestRedis(t)
	q := NewQueue(rdb)
	ctx := context.Background()

	if err := q.Enqueue(ctx, "tiktok", "job-123"); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	jobID, platform, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Dequeue: %v", err)
	}
	if jobID != "job-123" {
		t.Errorf("jobID = %q, want job-123", jobID)
	}
	if platform != "tiktok" {
		t.Errorf("platform = %q, want tiktok", platform)
	}
}

// TestDequeueEmptyReturnsNoError verifies a dequeue with no pending jobs blocks
// briefly then returns cleanly (no error, empty id) rather than failing.
func TestDequeueEmptyReturnsNoError(t *testing.T) {
	rdb := newTestRedis(t)
	q := NewQueue(rdb)

	jobID, _, err := q.Dequeue(context.Background())
	if err != nil {
		t.Fatalf("Dequeue on empty: %v", err)
	}
	if jobID != "" {
		t.Errorf("expected empty jobID on idle dequeue, got %q", jobID)
	}
}

// TestDeviceLock verifies the SETNX-based device lock is exclusive.
func TestDeviceLock(t *testing.T) {
	rdb := newTestRedis(t)
	q := NewQueue(rdb)
	ctx := context.Background()

	ok, err := q.DeviceLock(ctx, "emulator-5554", time.Minute)
	if err != nil || !ok {
		t.Fatalf("first lock should succeed: ok=%v err=%v", ok, err)
	}

	// A second queue (different consumer) must not acquire the same lock.
	q2 := NewQueue(rdb)
	ok2, err := q2.DeviceLock(ctx, "emulator-5554", time.Minute)
	if err != nil {
		t.Fatalf("second lock err: %v", err)
	}
	if ok2 {
		t.Error("second lock should fail while the first is held")
	}

	// After unlock, the lock is acquirable again.
	q.DeviceUnlock(ctx, "emulator-5554")
	ok3, _ := q2.DeviceLock(ctx, "emulator-5554", time.Minute)
	if !ok3 {
		t.Error("lock should be acquirable after unlock")
	}
}
