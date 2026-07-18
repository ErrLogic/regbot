package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// JobLogStore manages job log persistence.
type JobLogStore struct {
	pool *pgxpool.Pool
}

// NewJobLogStore creates a JobLogStore.
func NewJobLogStore(pool *pgxpool.Pool) *JobLogStore {
	return &JobLogStore{pool: pool}
}

// Create inserts a new job log entry.
func (s *JobLogStore) Create(ctx context.Context, jl *JobLog) error {
	jl.CreatedAt = time.Now().UTC()
	err := s.pool.QueryRow(ctx,
		`INSERT INTO job_logs (job_id, level, step, message, created_at) VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		jl.JobID, jl.Level, jl.Step, jl.Message, jl.CreatedAt,
	).Scan(&jl.ID)
	if err != nil {
		return fmt.Errorf("create job log: %w", err)
	}
	return nil
}

// ListByJob returns log entries for a job, paginated.
func (s *JobLogStore) ListByJob(ctx context.Context, jobID uuid.UUID, limit, offset int) ([]JobLog, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, job_id, level, step, message, created_at FROM job_logs
		 WHERE job_id=$1 ORDER BY created_at ASC LIMIT $2 OFFSET $3`,
		jobID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list job logs: %w", err)
	}
	defer rows.Close()

	var logs []JobLog
	for rows.Next() {
		var jl JobLog
		if err := rows.Scan(&jl.ID, &jl.JobID, &jl.Level, &jl.Step, &jl.Message, &jl.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan job log: %w", err)
		}
		logs = append(logs, jl)
	}
	if logs == nil {
		logs = []JobLog{}
	}
	return logs, nil
}
