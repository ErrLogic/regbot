package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// JobStore manages job persistence.
type JobStore struct {
	pool *pgxpool.Pool
}

// NewJobStore creates a JobStore.
func NewJobStore(pool *pgxpool.Pool) *JobStore {
	return &JobStore{pool: pool}
}

// Create inserts a new job.
func (s *JobStore) Create(ctx context.Context, j *Job) error {
	j.ID = uuid.New()
	j.CreatedAt = time.Now().UTC()
	j.UpdatedAt = j.CreatedAt

	_, err := s.pool.Exec(ctx,
		`INSERT INTO jobs (id, type, platform, status, priority, params, result, error_message,
		 device_serial, account_id, created_by, retry_count, max_retries, started_at, completed_at, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		j.ID, j.Type, j.Platform, j.Status, j.Priority, j.Params, j.Result, j.ErrorMessage,
		j.DeviceSerial, j.AccountID, j.CreatedBy, j.RetryCount, j.MaxRetries,
		j.StartedAt, j.CompletedAt, j.CreatedAt, j.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create job: %w", err)
	}
	return nil
}

// GetByID returns a job by ID.
func (s *JobStore) GetByID(ctx context.Context, id uuid.UUID) (*Job, error) {
	j := &Job{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, type, platform, status, priority, params, result, error_message,
		 device_serial, account_id, created_by, retry_count, max_retries, started_at, completed_at, created_at, updated_at
		 FROM jobs WHERE id=$1`, id,
	).Scan(&j.ID, &j.Type, &j.Platform, &j.Status, &j.Priority, &j.Params, &j.Result, &j.ErrorMessage,
		&j.DeviceSerial, &j.AccountID, &j.CreatedBy, &j.RetryCount, &j.MaxRetries,
		&j.StartedAt, &j.CompletedAt, &j.CreatedAt, &j.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	return j, nil
}

// JobFilter holds optional filters for listing jobs.
type JobFilter struct {
	Status   string
	Type     string
	Platform string
	Limit    int
	Offset   int
}

// List returns jobs matching the filter.
func (s *JobStore) List(ctx context.Context, f JobFilter) ([]Job, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}

	query := `SELECT id, type, platform, status, priority, params, result, error_message,
		device_serial, account_id, created_by, retry_count, max_retries, started_at, completed_at, created_at, updated_at
		FROM jobs WHERE 1=1`
	args := []any{}
	argIdx := 1

	if f.Status != "" {
		query += fmt.Sprintf(" AND status=$%d", argIdx)
		args = append(args, f.Status)
		argIdx++
	}
	if f.Type != "" {
		query += fmt.Sprintf(" AND type=$%d", argIdx)
		args = append(args, f.Type)
		argIdx++
	}
	if f.Platform != "" {
		query += fmt.Sprintf(" AND platform=$%d", argIdx)
		args = append(args, f.Platform)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, f.Limit, f.Offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.ID, &j.Type, &j.Platform, &j.Status, &j.Priority, &j.Params, &j.Result,
			&j.ErrorMessage, &j.DeviceSerial, &j.AccountID, &j.CreatedBy, &j.RetryCount, &j.MaxRetries,
			&j.StartedAt, &j.CompletedAt, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}
		jobs = append(jobs, j)
	}
	if jobs == nil {
		jobs = []Job{}
	}
	return jobs, nil
}

// UpdateStatus sets the job status and optionally result/error.
func (s *JobStore) UpdateStatus(ctx context.Context, id uuid.UUID, status string, result []byte, errMsg string) error {
	now := time.Now().UTC()
	var startedAt, completedAt *time.Time
	if status == "running" {
		startedAt = &now
	}
	if status == "completed" || status == "failed" || status == "cancelled" {
		completedAt = &now
	}

	_, err := s.pool.Exec(ctx,
		`UPDATE jobs SET status=$1, result=$2, error_message=$3, started_at=COALESCE($4, started_at),
		 completed_at=$5, updated_at=$6 WHERE id=$7`,
		status, result, errMsg, startedAt, completedAt, now, id)
	if err != nil {
		return fmt.Errorf("update job status: %w", err)
	}
	return nil
}

// GetNextPending returns the next pending job for a platform (ordered by priority, then age).
func (s *JobStore) GetNextPending(ctx context.Context, platform string) (*Job, error) {
	j := &Job{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, type, platform, status, priority, params, result, error_message,
		 device_serial, account_id, created_by, retry_count, max_retries, started_at, completed_at, created_at, updated_at
		 FROM jobs WHERE status='pending' AND platform=$1
		 ORDER BY priority DESC, created_at ASC LIMIT 1`, platform,
	).Scan(&j.ID, &j.Type, &j.Platform, &j.Status, &j.Priority, &j.Params, &j.Result, &j.ErrorMessage,
		&j.DeviceSerial, &j.AccountID, &j.CreatedBy, &j.RetryCount, &j.MaxRetries,
		&j.StartedAt, &j.CompletedAt, &j.CreatedAt, &j.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get next pending: %w", err)
	}
	return j, nil
}
