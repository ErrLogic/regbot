// Package api implements the REST API router and server for RegBot.
package api

import (
	"github.com/ErrLogic/regbot/internal/api/handler"
	"github.com/ErrLogic/regbot/internal/api/middleware"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

// NewRouter builds the chi router with all routes and middleware.
func NewRouter(
	jwtSecret string,
	authH *handler.AuthHandler,
	deviceH *handler.DeviceHandler,
	jobH *handler.JobHandler,
	accountH *handler.AccountHandler,
	mediaH *handler.MediaHandler,
) chi.Router {
	r := chi.NewRouter()

	// Global middleware. RealIP is intentionally omitted (spoofable behind an
	// untrusted proxy); RemoteAddr is used as-is.
	r.Use(chiMiddleware.RequestID)
	r.Use(middleware.Logging)
	r.Use(middleware.CORS)
	r.Use(chiMiddleware.Recoverer)

	// Public routes (no auth).
	r.Group(func(r chi.Router) {
		r.Get("/api/v1/health", handler.Health)
		r.Post("/api/v1/auth/login", authH.Login)
		r.Post("/api/v1/auth/register", authH.Register)
	})

	// Protected routes (JWT required).
	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTAuth(jwtSecret))

		// Devices.
		r.Get("/api/v1/devices", deviceH.List)
		r.Post("/api/v1/devices/refresh", deviceH.Refresh)

		// Jobs.
		r.Get("/api/v1/jobs", jobH.List)
		r.Get("/api/v1/jobs/{id}", jobH.Get)
		r.Get("/api/v1/jobs/{id}/logs", jobH.GetLogs)
		r.Get("/api/v1/jobs/{id}/stream", jobH.Stream)
		r.Post("/api/v1/jobs/{id}/cancel", jobH.Cancel)

		r.Post("/api/v1/jobs/register", jobH.CreateRegister)
		r.Post("/api/v1/jobs/like", jobH.CreateLike)
		r.Post("/api/v1/jobs/comment", jobH.CreateComment)
		r.Post("/api/v1/jobs/update-profile", jobH.CreateUpdateProfile)
		r.Post("/api/v1/jobs/create-post", jobH.CreatePost)
		r.Post("/api/v1/jobs/watch-live", jobH.CreateWatchLive)

		// Accounts.
		r.Get("/api/v1/accounts", accountH.List)
		r.Get("/api/v1/accounts/{id}", accountH.Get)
		r.Delete("/api/v1/accounts/{id}", accountH.Delete)

		// Media.
		r.Post("/api/v1/media/upload", mediaH.Upload)
		r.Get("/api/v1/media", mediaH.List)
		r.Get("/api/v1/media/{id}", mediaH.Download)
		r.Delete("/api/v1/media/{id}", mediaH.Delete)
	})

	return r
}
