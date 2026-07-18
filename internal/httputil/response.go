// Package httputil provides shared HTTP response helpers for API handlers.
package httputil

import (
	"encoding/json"
	"net/http"
)

// Envelope is the standard JSON response wrapper.
type Envelope struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
	Meta  any    `json:"meta,omitempty"`
}

// MetaPage holds pagination metadata.
type MetaPage struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

// JSON responds with a JSON envelope.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{Data: data})
}

// JSONError responds with a JSON error envelope.
func JSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{Error: msg})
}

// JSONPage responds with a paginated JSON envelope.
func JSONPage(w http.ResponseWriter, status int, data any, meta MetaPage) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{Data: data, Meta: meta})
}

// DecodeBody decodes a JSON request body into v.
func DecodeBody(r *http.Request, v any) error {
	defer func() { _ = r.Body.Close() }()
	return json.NewDecoder(r.Body).Decode(v)
}
