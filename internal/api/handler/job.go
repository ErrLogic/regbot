package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/ErrLogic/regbot/internal/api/middleware"
	"github.com/ErrLogic/regbot/internal/db"
	"github.com/ErrLogic/regbot/internal/httputil"
	"github.com/ErrLogic/regbot/internal/job"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// JobHandler handles job creation, listing, and monitoring.
type JobHandler struct {
	store    *db.JobStore
	logStore *db.JobLogStore
	accounts *db.AccountStore
	queue    *job.Queue
	streamer *job.LogStreamer
}

// NewJobHandler creates a job handler.
func NewJobHandler(
	store *db.JobStore,
	logStore *db.JobLogStore,
	accounts *db.AccountStore,
	rdb *redis.Client,
) *JobHandler {
	return &JobHandler{
		store:    store,
		logStore: logStore,
		accounts: accounts,
		queue:    job.NewQueue(rdb),
		streamer: job.NewLogStreamer(rdb),
	}
}

// List handles GET /api/v1/jobs.
func (h *JobHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := db.JobFilter{
		Status:   q.Get("status"),
		Type:     q.Get("type"),
		Platform: q.Get("platform"),
		Limit:    50,
		Offset:   0,
	}
	if v, err := strconv.Atoi(q.Get("per_page")); err == nil && v > 0 {
		filter.Limit = v
	}
	if v, err := strconv.Atoi(q.Get("page")); err == nil && v > 1 {
		filter.Offset = (v - 1) * filter.Limit
	}

	jobs, err := h.store.List(r.Context(), filter)
	if err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "failed to list jobs: "+err.Error())
		return
	}
	httputil.JSON(w, http.StatusOK, jobs)
}

// Get handles GET /api/v1/jobs/{id}.
func (h *JobHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "invalid job id")
		return
	}
	job, err := h.store.GetByID(r.Context(), id)
	if err != nil || job == nil {
		httputil.JSONError(w, http.StatusNotFound, "job not found")
		return
	}
	httputil.JSON(w, http.StatusOK, job)
}

// GetLogs handles GET /api/v1/jobs/{id}/logs.
func (h *JobHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "invalid job id")
		return
	}
	limit := 100
	offset := 0
	if v, err := strconv.Atoi(r.URL.Query().Get("per_page")); err == nil && v > 0 {
		limit = v
	}
	logs, err := h.logStore.ListByJob(r.Context(), id, limit, offset)
	if err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "failed to get logs: "+err.Error())
		return
	}
	httputil.JSON(w, http.StatusOK, logs)
}

// Stream handles GET /api/v1/jobs/{id}/stream (SSE).
func (h *JobHandler) Stream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		httputil.JSONError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	ch, unsub := h.streamer.Subscribe(r.Context(), id)
	defer unsub()

	for {
		select {
		case <-r.Context().Done():
			return
		case entry, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(entry)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// Cancel handles POST /api/v1/jobs/{id}/cancel.
func (h *JobHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "invalid job id")
		return
	}
	if err := h.store.UpdateStatus(r.Context(), id, string(job.StatusCancelled), nil, ""); err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "failed to cancel job: "+err.Error())
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// CreateRegister handles POST /api/v1/jobs/register.
func (h *JobHandler) CreateRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Platform     string `json:"platform"`
		DeviceSerial string `json:"device_serial"`
		Email        string `json:"email"`
		DryRun       bool   `json:"dry_run"`
		UseSSO       bool   `json:"use_sso"`
	}
	if err := httputil.DecodeBody(r, &req); err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Platform == "" {
		httputil.JSONError(w, http.StatusBadRequest, "platform is required")
		return
	}
	// Email is required for the email+OTP path but not for Google SSO.
	if req.Email == "" && !req.UseSSO {
		httputil.JSONError(w, http.StatusBadRequest, "email is required (or set use_sso)")
		return
	}
	params := job.RegisterParams{
		Platform: req.Platform,
		Email:    req.Email,
		DryRun:   req.DryRun,
		UseSSO:   req.UseSSO,
	}
	h.createJob(w, r, string(job.TypeRegister), req.Platform, req.DeviceSerial, params)
}

// CreateLike handles POST /api/v1/jobs/like.
func (h *JobHandler) CreateLike(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Platform     string `json:"platform"`
		DeviceSerial string `json:"device_serial"`
		PostURL      string `json:"post_url"`
	}
	if err := httputil.DecodeBody(r, &req); err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	params := job.LikeParams{PostURL: req.PostURL}
	h.createJob(w, r, string(job.TypeLike), req.Platform, req.DeviceSerial, params)
}

// CreateComment handles POST /api/v1/jobs/comment.
func (h *JobHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Platform     string `json:"platform"`
		DeviceSerial string `json:"device_serial"`
		PostURL      string `json:"post_url"`
		Text         string `json:"text"`
	}
	if err := httputil.DecodeBody(r, &req); err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	params := job.CommentParams{PostURL: req.PostURL, Text: req.Text}
	h.createJob(w, r, string(job.TypeComment), req.Platform, req.DeviceSerial, params)
}

// CreateUpdateProfile handles POST /api/v1/jobs/update-profile.
func (h *JobHandler) CreateUpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Platform     string `json:"platform"`
		DeviceSerial string `json:"device_serial"`
		Bio          string `json:"bio"`
		DisplayName  string `json:"display_name"`
		AvatarURL    string `json:"avatar_url"`
	}
	if err := httputil.DecodeBody(r, &req); err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	params := job.UpdateProfileParams{
		Bio: req.Bio, DisplayName: req.DisplayName, AvatarURL: req.AvatarURL,
	}
	h.createJob(w, r, string(job.TypeUpdateProfile), req.Platform, req.DeviceSerial, params)
}

// CreatePost handles POST /api/v1/jobs/create-post.
func (h *JobHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Platform     string   `json:"platform"`
		DeviceSerial string   `json:"device_serial"`
		Caption      string   `json:"caption"`
		MediaIDs     []string `json:"media_ids"`
	}
	if err := httputil.DecodeBody(r, &req); err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	params := job.CreatePostParams{Caption: req.Caption, MediaIDs: req.MediaIDs}
	h.createJob(w, r, string(job.TypeCreatePost), req.Platform, req.DeviceSerial, params)
}

// CreateWatchLive handles POST /api/v1/jobs/watch-live.
func (h *JobHandler) CreateWatchLive(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Platform        string `json:"platform"`
		DeviceSerial    string `json:"device_serial"`
		LiveURL         string `json:"live_url"`
		DurationSeconds int    `json:"duration_seconds"`
	}
	if err := httputil.DecodeBody(r, &req); err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	params := job.WatchLiveParams{LiveURL: req.LiveURL, DurationSeconds: req.DurationSeconds}
	h.createJob(w, r, string(job.TypeWatchLive), req.Platform, req.DeviceSerial, params)
}

// createJob is the shared job creation logic. deviceSerial names the target
// device (stored on the job row); params is the type-specific parameter struct
// persisted as JSONB.
func (h *JobHandler) createJob(w http.ResponseWriter, r *http.Request, jobType, platform, deviceSerial string, params any) {
	paramsJSON, _ := json.Marshal(params)

	userID := middleware.UserIDFromContext(r.Context())
	var createdBy *uuid.UUID
	if uid, err := uuid.Parse(userID); err == nil {
		createdBy = &uid
	}

	j := &db.Job{
		Type:         jobType,
		Platform:     platform,
		Status:       string(job.StatusPending),
		Params:       paramsJSON,
		DeviceSerial: deviceSerial,
		CreatedBy:    createdBy,
		MaxRetries:   3,
	}

	if err := h.store.Create(r.Context(), j); err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "failed to create job: "+err.Error())
		return
	}

	// Enqueue in Redis.
	if err := h.queue.Enqueue(r.Context(), platform, j.ID.String()); err != nil {
		log.Printf("failed to enqueue job %s: %v", j.ID, err)
	}

	httputil.JSON(w, http.StatusCreated, map[string]any{
		"id":     j.ID,
		"status": j.Status,
	})
}
