package db

import (
	"time"

	"github.com/google/uuid"
)

// User represents an API user for JWT authentication.
type User struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Device represents a connected ADB device.
type Device struct {
	ID             uuid.UUID  `json:"id"`
	Serial         string     `json:"serial"`
	Model          string     `json:"model"`
	State          string     `json:"state"` // offline, online, busy, unauthorized
	AndroidVersion string     `json:"android_version"`
	LastSeenAt     *time.Time `json:"last_seen_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// PlatformAccount represents a registered social media account.
type PlatformAccount struct {
	ID                uuid.UUID `json:"id"`
	Platform          string    `json:"platform"` // instagram, tiktok
	Email             string    `json:"email"`
	Username          string    `json:"username"`
	EncryptedPassword []byte    `json:"-"`
	EncryptionNonce   []byte    `json:"-"`
	Status            string    `json:"status"` // active, locked, disabled
	DeviceSerial      string    `json:"device_serial"`
	JobID             uuid.UUID `json:"job_id"`
	Metadata          []byte    `json:"metadata"` // JSONB
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Job represents an async automation task.
type Job struct {
	ID           uuid.UUID  `json:"id"`
	Type         string     `json:"type"` // register, like, comment, update_profile, create_post, watch_live
	Platform     string     `json:"platform"`
	Status       string     `json:"status"` // pending, running, completed, failed, cancelled
	Priority     int        `json:"priority"`
	Params       []byte     `json:"params"`           // JSONB
	Result       []byte     `json:"result,omitempty"` // JSONB
	ErrorMessage string     `json:"error_message,omitempty"`
	DeviceSerial string     `json:"device_serial"`
	AccountID    *uuid.UUID `json:"account_id,omitempty"`
	CreatedBy    *uuid.UUID `json:"created_by,omitempty"`
	RetryCount   int        `json:"retry_count"`
	MaxRetries   int        `json:"max_retries"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// JobLog represents a single log entry emitted during job execution.
type JobLog struct {
	ID        int64     `json:"id"`
	JobID     uuid.UUID `json:"job_id"`
	Level     string    `json:"level"` // debug, info, warn, error
	Step      string    `json:"step"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// Media represents an uploaded file stored as a BLOB.
type Media struct {
	ID         uuid.UUID  `json:"id"`
	Filename   string     `json:"filename"`
	MimeType   string     `json:"mime_type"`
	SizeBytes  int64      `json:"size_bytes"`
	Data       []byte     `json:"-"`
	UploadedBy *uuid.UUID `json:"uploaded_by,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}
