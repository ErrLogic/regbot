package handler

import (
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ErrLogic/regbot/internal/api/middleware"
	"github.com/ErrLogic/regbot/internal/db"
	"github.com/ErrLogic/regbot/internal/httputil"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const maxUploadSize = 100 << 20 // 100 MB

// MediaHandler handles media upload and download.
type MediaHandler struct {
	store *db.MediaStore
}

// NewMediaHandler creates a media handler.
func NewMediaHandler(store *db.MediaStore) *MediaHandler {
	return &MediaHandler{store: store}
}

// Upload handles POST /api/v1/media/upload.
func (h *MediaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "file too large or invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(file)
	if err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "failed to read file")
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		ext := strings.ToLower(filepath.Ext(header.Filename))
		switch ext {
		case ".jpg", ".jpeg":
			mimeType = "image/jpeg"
		case ".png":
			mimeType = "image/png"
		case ".mp4":
			mimeType = "video/mp4"
		case ".mov":
			mimeType = "video/quicktime"
		default:
			mimeType = "application/octet-stream"
		}
	}

	userID := middleware.UserIDFromContext(r.Context())
	var uploadedBy *uuid.UUID
	if uid, err := uuid.Parse(userID); err == nil {
		uploadedBy = &uid
	}

	media := &db.Media{
		Filename:   header.Filename,
		MimeType:   mimeType,
		SizeBytes:  int64(len(data)),
		Data:       data,
		UploadedBy: uploadedBy,
	}

	if err := h.store.Create(r.Context(), media); err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "failed to store media: "+err.Error())
		return
	}

	httputil.JSON(w, http.StatusCreated, map[string]any{
		"id":        media.ID,
		"filename":  media.Filename,
		"mime_type": media.MimeType,
		"size":      media.SizeBytes,
	})
}

// Download handles GET /api/v1/media/{id}.
func (h *MediaHandler) Download(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "invalid media id")
		return
	}
	m, err := h.store.GetByID(r.Context(), id)
	if err != nil || m == nil {
		httputil.JSONError(w, http.StatusNotFound, "media not found")
		return
	}
	w.Header().Set("Content-Type", m.MimeType)
	w.Header().Set("Content-Disposition", "inline; filename=\""+m.Filename+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(m.Data)
}

// Delete handles DELETE /api/v1/media/{id}.
func (h *MediaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "invalid media id")
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "failed to delete media: "+err.Error())
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// List handles GET /api/v1/media.
func (h *MediaHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0
	media, err := h.store.List(r.Context(), limit, offset)
	if err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "failed to list media: "+err.Error())
		return
	}
	httputil.JSON(w, http.StatusOK, media)
}
