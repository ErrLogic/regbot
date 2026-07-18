package handler

import (
	"net/http"

	"github.com/ErrLogic/regbot/internal/httputil"
)

// Health handles GET /api/v1/health (liveness probe).
func Health(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
