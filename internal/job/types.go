// Package job implements the background job queue, worker pool, and log streaming
// for RegBot automation tasks.
package job

import (
	"encoding/json"
	"fmt"
)

// Type enumerates automation job types.
type Type string

// Supported job types.
const (
	TypeRegister      Type = "register"
	TypeLike          Type = "like"
	TypeComment       Type = "comment"
	TypeUpdateProfile Type = "update_profile"
	TypeCreatePost    Type = "create_post"
	TypeWatchLive     Type = "watch_live"
)

// Status enumerates job lifecycle states.
type Status string

// Job statuses.
const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

// RegisterParams is the input for a registration job.
type RegisterParams struct {
	Platform string `json:"platform"`
	Email    string `json:"email"`
	DryRun   bool   `json:"dry_run"`
	// UseSSO selects Google single-sign-on registration (TikTok) instead of
	// email + OTP.
	UseSSO bool `json:"use_sso"`
}

// LikeParams is the input for a like-post job.
type LikeParams struct {
	PostURL string `json:"post_url"`
}

// CommentParams is the input for a comment job.
type CommentParams struct {
	PostURL string `json:"post_url"`
	Text    string `json:"text"`
}

// UpdateProfileParams is the input for a profile update job.
type UpdateProfileParams struct {
	Bio         string `json:"bio,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
}

// CreatePostParams is the input for a create-post job.
type CreatePostParams struct {
	Caption  string   `json:"caption"`
	MediaIDs []string `json:"media_ids"`
}

// WatchLiveParams is the input for a watch-live job.
type WatchLiveParams struct {
	LiveURL         string `json:"live_url"`
	DurationSeconds int    `json:"duration_seconds"`
}

// RegisterResult is the output of a successful registration job.
type RegisterResult struct {
	AccountID string `json:"account_id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

// ParseParams unmarshals JSON params into the appropriate struct based on job type.
func ParseParams(jobType Type, data json.RawMessage) (any, error) {
	switch jobType {
	case TypeRegister:
		var p RegisterParams
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return p, nil
	case TypeLike:
		var p LikeParams
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return p, nil
	case TypeComment:
		var p CommentParams
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return p, nil
	case TypeUpdateProfile:
		var p UpdateProfileParams
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return p, nil
	case TypeCreatePost:
		var p CreatePostParams
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return p, nil
	case TypeWatchLive:
		var p WatchLiveParams
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return p, nil
	default:
		return nil, fmt.Errorf("unknown job type: %s", jobType)
	}
}
