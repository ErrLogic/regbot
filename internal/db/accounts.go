package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AccountStore manages platform account persistence.
type AccountStore struct {
	pool *pgxpool.Pool
}

// NewAccountStore creates an AccountStore.
func NewAccountStore(pool *pgxpool.Pool) *AccountStore {
	return &AccountStore{pool: pool}
}

// Create inserts a new platform account.
func (s *AccountStore) Create(ctx context.Context, a *PlatformAccount) error {
	a.ID = uuid.New()
	a.CreatedAt = time.Now().UTC()
	a.UpdatedAt = a.CreatedAt

	_, err := s.pool.Exec(ctx,
		`INSERT INTO platform_accounts (id, platform, email, username, encrypted_password, encryption_nonce,
		 status, device_serial, job_id, metadata, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		a.ID, a.Platform, a.Email, a.Username, a.EncryptedPassword, a.EncryptionNonce,
		a.Status, a.DeviceSerial, a.JobID, a.Metadata, a.CreatedAt, a.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create account: %w", err)
	}
	return nil
}

// GetByID returns an account by ID.
func (s *AccountStore) GetByID(ctx context.Context, id uuid.UUID) (*PlatformAccount, error) {
	a := &PlatformAccount{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, platform, email, username, encrypted_password, encryption_nonce,
		 status, device_serial, job_id, metadata, created_at, updated_at
		 FROM platform_accounts WHERE id=$1`, id,
	).Scan(&a.ID, &a.Platform, &a.Email, &a.Username, &a.EncryptedPassword, &a.EncryptionNonce,
		&a.Status, &a.DeviceSerial, &a.JobID, &a.Metadata, &a.CreatedAt, &a.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}
	return a, nil
}

// List returns accounts, optionally filtered by platform and status.
func (s *AccountStore) List(ctx context.Context, platform, status string) ([]PlatformAccount, error) {
	query := `SELECT id, platform, email, username, encrypted_password, encryption_nonce,
		status, device_serial, job_id, metadata, created_at, updated_at
		FROM platform_accounts WHERE 1=1`
	args := []any{}
	argIdx := 1

	if platform != "" {
		query += fmt.Sprintf(" AND platform=$%d", argIdx)
		args = append(args, platform)
		argIdx++
	}
	if status != "" {
		query += fmt.Sprintf(" AND status=$%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()

	var accounts []PlatformAccount
	for rows.Next() {
		var a PlatformAccount
		if err := rows.Scan(&a.ID, &a.Platform, &a.Email, &a.Username, &a.EncryptedPassword,
			&a.EncryptionNonce, &a.Status, &a.DeviceSerial, &a.JobID, &a.Metadata,
			&a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		accounts = append(accounts, a)
	}
	if accounts == nil {
		accounts = []PlatformAccount{}
	}
	return accounts, nil
}

// Delete removes an account record.
func (s *AccountStore) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM platform_accounts WHERE id=$1`, id)
	return err
}
