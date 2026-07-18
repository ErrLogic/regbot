package handler

import (
	"net/http"
	"time"

	"github.com/ErrLogic/regbot/internal/crypto"
	"github.com/ErrLogic/regbot/internal/db"
	"github.com/ErrLogic/regbot/internal/httputil"
	"github.com/golang-jwt/jwt/v5"
)

// AuthHandler handles login and user registration.
type AuthHandler struct {
	users     *db.UserStore
	jwtSecret string
}

// NewAuthHandler creates an auth handler.
func NewAuthHandler(users *db.UserStore, jwtSecret string) *AuthHandler {
	return &AuthHandler{users: users, jwtSecret: jwtSecret}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type tokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

// Login handles POST /api/v1/auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httputil.DecodeBody(r, &req); err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Username == "" || req.Password == "" {
		httputil.JSONError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	user, err := h.users.GetByUsername(r.Context(), req.Username)
	if err != nil || user == nil {
		httputil.JSONError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := crypto.CheckPassword(user.PasswordHash, req.Password); err != nil {
		httputil.JSONError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID.String(),
		"iat": time.Now().Unix(),
		"exp": expiresAt.Unix(),
	})

	signed, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	httputil.JSON(w, http.StatusOK, tokenResponse{
		Token:     signed,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	})
}

// Register handles POST /api/v1/auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := httputil.DecodeBody(r, &req); err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Username == "" || req.Password == "" {
		httputil.JSONError(w, http.StatusBadRequest, "username and password are required")
		return
	}
	if len(req.Password) < 6 {
		httputil.JSONError(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user := &db.User{Username: req.Username, PasswordHash: hash}
	if err := h.users.Create(r.Context(), user); err != nil {
		httputil.JSONError(w, http.StatusConflict, "username already exists")
		return
	}

	httputil.JSON(w, http.StatusCreated, map[string]string{"id": user.ID.String(), "username": user.Username})
}
