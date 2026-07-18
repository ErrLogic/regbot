package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DeviceStore manages device persistence.
type DeviceStore struct {
	pool *pgxpool.Pool
}

// NewDeviceStore creates a DeviceStore.
func NewDeviceStore(pool *pgxpool.Pool) *DeviceStore {
	return &DeviceStore{pool: pool}
}

// Upsert inserts or updates a device record by serial.
func (s *DeviceStore) Upsert(ctx context.Context, d *Device) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	now := time.Now().UTC()
	d.UpdatedAt = now
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}

	_, err := s.pool.Exec(ctx,
		`INSERT INTO devices (id, serial, model, state, android_version, last_seen_at, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 ON CONFLICT (serial) DO UPDATE SET
		   model=EXCLUDED.model, state=EXCLUDED.state,
		   android_version=EXCLUDED.android_version, last_seen_at=EXCLUDED.last_seen_at,
		   updated_at=EXCLUDED.updated_at`,
		d.ID, d.Serial, d.Model, d.State, d.AndroidVersion, d.LastSeenAt, d.CreatedAt, d.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert device: %w", err)
	}
	return nil
}

// GetBySerial returns a device by its serial number.
func (s *DeviceStore) GetBySerial(ctx context.Context, serial string) (*Device, error) {
	d := &Device{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, serial, model, state, android_version, last_seen_at, created_at, updated_at
		 FROM devices WHERE serial=$1`, serial,
	).Scan(&d.ID, &d.Serial, &d.Model, &d.State, &d.AndroidVersion, &d.LastSeenAt, &d.CreatedAt, &d.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}
	return d, nil
}

// List returns all devices, optionally filtered by state.
func (s *DeviceStore) List(ctx context.Context, state string) ([]Device, error) {
	var rows pgx.Rows
	var err error
	if state == "" {
		rows, err = s.pool.Query(ctx,
			`SELECT id, serial, model, state, android_version, last_seen_at, created_at, updated_at
			 FROM devices ORDER BY last_seen_at DESC NULLS LAST`)
	} else {
		rows, err = s.pool.Query(ctx,
			`SELECT id, serial, model, state, android_version, last_seen_at, created_at, updated_at
			 FROM devices WHERE state=$1 ORDER BY last_seen_at DESC NULLS LAST`, state)
	}
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.ID, &d.Serial, &d.Model, &d.State, &d.AndroidVersion, &d.LastSeenAt, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}
		devices = append(devices, d)
	}
	if devices == nil {
		devices = []Device{}
	}
	return devices, nil
}

// Delete removes a device record.
func (s *DeviceStore) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM devices WHERE id=$1`, id)
	return err
}
