package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"github.com/Germatic/dinapay-routing/internal/app"
	"github.com/Germatic/dinapay-routing/internal/core"
	"net/http"
	"strings"
)

func New(router *app.Router, token string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { write(w, 200, map[string]string{"status": "up"}) })
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, _ *http.Request) { write(w, 200, map[string]string{"status": "ready"}) })
	mux.HandleFunc("POST /v1/routes/resolve", func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" || subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			write(w, 401, errorBody("unauthorized", errors.New("invalid service credentials"), false, r.Header.Get("X-Request-Id")))
			return
		}
		var in core.RouteRequest
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&in); err != nil {
			write(w, 400, errorBody("invalid_request", err, false, r.Header.Get("X-Request-Id")))
			return
		}
		out, err := router.Resolve(r.Context(), in)
		if err == nil {
			write(w, 200, out)
			return
		}
		var no app.NoRouteError
		if errors.As(err, &no) {
			write(w, 422, no.Decision)
			return
		}
		if errors.Is(err, app.ErrConflict) {
			write(w, 409, errorBody("conflict", err, false, in.RequestID))
			return
		}
		if errors.Is(err, app.ErrInvalid) {
			write(w, 400, errorBody("invalid_request", err, false, in.RequestID))
			return
		}
		write(w, 503, errorBody("unavailable", err, true, in.RequestID))
	})
	return mux
}
func errorBody(code string, err error, retryable bool, requestID string) map[string]any {
	return map[string]any{"code": code, "message": err.Error(), "retryable": retryable, "requestId": requestID}
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
