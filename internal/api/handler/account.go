// Package handler implements the REST API HTTP handlers.
package handler

import (
	"net/http"

	"github.com/ErrLogic/regbot/internal/db"
	"github.com/ErrLogic/regbot/internal/httputil"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// AccountHandler handles platform account listing and management.
type AccountHandler struct {
	store *db.AccountStore
}

// NewAccountHandler creates an account handler.
func NewAccountHandler(store *db.AccountStore) *AccountHandler {
	return &AccountHandler{store: store}
}

// List handles GET /api/v1/accounts.
func (h *AccountHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	platform := q.Get("platform")
	status := q.Get("status")

	accounts, err := h.store.List(r.Context(), platform, status)
	if err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "failed to list accounts: "+err.Error())
		return
	}
	httputil.JSON(w, http.StatusOK, accounts)
}

// Get handles GET /api/v1/accounts/{id}.
func (h *AccountHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "invalid account id")
		return
	}
	acct, err := h.store.GetByID(r.Context(), id)
	if err != nil || acct == nil {
		httputil.JSONError(w, http.StatusNotFound, "account not found")
		return
	}
	httputil.JSON(w, http.StatusOK, acct)
}

// Delete handles DELETE /api/v1/accounts/{id}.
func (h *AccountHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "invalid account id")
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "failed to delete account: "+err.Error())
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
