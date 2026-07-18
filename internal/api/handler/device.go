package handler

import (
	"net/http"

	"github.com/ErrLogic/regbot/internal/device"
	"github.com/ErrLogic/regbot/internal/httputil"
)

// DeviceHandler handles device listing and selection.
type DeviceHandler struct {
	mgr *device.Manager
}

// NewDeviceHandler creates a device handler.
func NewDeviceHandler(mgr *device.Manager) *DeviceHandler {
	return &DeviceHandler{mgr: mgr}
}

// List handles GET /api/v1/devices.
func (h *DeviceHandler) List(w http.ResponseWriter, r *http.Request) {
	devices, err := h.mgr.ListDevices(r.Context())
	if err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "failed to scan devices: "+err.Error())
		return
	}
	httputil.JSON(w, http.StatusOK, devices)
}

// Refresh handles POST /api/v1/devices/refresh.
func (h *DeviceHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	devices, err := h.mgr.ListDevices(r.Context())
	if err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "failed to scan devices: "+err.Error())
		return
	}
	httputil.JSON(w, http.StatusOK, devices)
}
