package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserStore manages user persistence.
type UserStore struct {
	pool *pgxpool.Pool
}

// NewUserStore creates a UserStore.
func NewUserStore(pool *pgxpool.Pool) *UserStore {
	return &UserStore{pool: pool}
}

// Create inserts a new user.
func (s *UserStore) Create(ctx context.Context, u *User) error {
	u.ID = uuid.New()
	u.CreatedAt = time.Now().UTC()
	u.UpdatedAt = u.CreatedAt
	_, err := s.pool.Exec(ctx,
		`INSERT INTO users (id, username, password_hash, created_at, updated_at) VALUES ($1,$2,$3,$4,$5)`,
		u.ID, u.Username, u.PasswordHash, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// GetByUsername returns a user by username.
func (s *UserStore) GetByUsername(ctx context.Context, username string) (*User, error) {
	u := &User{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, username, password_hash, created_at, updated_at FROM users WHERE username=$1`,
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

// GetByID returns a user by ID.
func (s *UserStore) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	u := &User{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, username, password_hash, created_at, updated_at FROM users WHERE id=$1`,
		id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}
