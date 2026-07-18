package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MediaStore manages uploaded media blob persistence.
type MediaStore struct {
	pool *pgxpool.Pool
}

// NewMediaStore creates a MediaStore.
func NewMediaStore(pool *pgxpool.Pool) *MediaStore {
	return &MediaStore{pool: pool}
}

// Create inserts a new media record with the blob data.
func (s *MediaStore) Create(ctx context.Context, m *Media) error {
	m.ID = uuid.New()
	m.CreatedAt = time.Now().UTC()

	_, err := s.pool.Exec(ctx,
		`INSERT INTO media (id, filename, mime_type, size_bytes, data, uploaded_by, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		m.ID, m.Filename, m.MimeType, m.SizeBytes, m.Data, m.UploadedBy, m.CreatedAt)
	if err != nil {
		return fmt.Errorf("create media: %w", err)
	}
	return nil
}

// GetByID returns media metadata and blob by ID.
func (s *MediaStore) GetByID(ctx context.Context, id uuid.UUID) (*Media, error) {
	m := &Media{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, filename, mime_type, size_bytes, data, uploaded_by, created_at
		 FROM media WHERE id=$1`, id,
	).Scan(&m.ID, &m.Filename, &m.MimeType, &m.SizeBytes, &m.Data, &m.UploadedBy, &m.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get media: %w", err)
	}
	return m, nil
}

// GetMetadata returns media metadata without the blob data.
func (s *MediaStore) GetMetadata(ctx context.Context, id uuid.UUID) (*Media, error) {
	m := &Media{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, filename, mime_type, size_bytes, uploaded_by, created_at
		 FROM media WHERE id=$1`, id,
	).Scan(&m.ID, &m.Filename, &m.MimeType, &m.SizeBytes, &m.UploadedBy, &m.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get media metadata: %w", err)
	}
	return m, nil
}

// List returns media metadata, paginated.
func (s *MediaStore) List(ctx context.Context, limit, offset int) ([]Media, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, filename, mime_type, size_bytes, uploaded_by, created_at
		 FROM media ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list media: %w", err)
	}
	defer rows.Close()

	var media []Media
	for rows.Next() {
		var m Media
		if err := rows.Scan(&m.ID, &m.Filename, &m.MimeType, &m.SizeBytes, &m.UploadedBy, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan media: %w", err)
		}
		media = append(media, m)
	}
	if media == nil {
		media = []Media{}
	}
	return media, nil
}

// Delete removes a media record.
func (s *MediaStore) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM media WHERE id=$1`, id)
	return err
}
