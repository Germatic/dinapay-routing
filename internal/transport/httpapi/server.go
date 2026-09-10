package httpapi

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Germatic/dinapay-routing/internal/app"
	"github.com/Germatic/dinapay-routing/internal/core"
	"github.com/Germatic/dinapay-routing/internal/observability"
)

func New(router *app.Router, token string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { write(w, 200, map[string]string{"status": "up"}) })
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, _ *http.Request) { write(w, 200, map[string]string{"status": "ready"}) })
	mux.Handle("GET /metrics", internalOnly(observability.Handler()))
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
	return observe(mux)
}
func internalOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Forwarded-For") != "" || r.Header.Get("X-Real-IP") != "" {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func observe(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		requestID := r.Header.Get("X-Request-Id")
		if !validRequestID(requestID) {
			requestID = randomHex(16)
		}
		traceparent := r.Header.Get("traceparent")
		if !validTraceparent(traceparent) {
			traceparent = fmt.Sprintf("00-%s-%s-01", randomHex(16), randomHex(8))
		}
		w.Header().Set("X-Request-Id", requestID)
		w.Header().Set("traceparent", traceparent)
		capture := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		ctx := core.WithObservability(r.Context(), requestID, traceparent)
		request := r.WithContext(ctx)
		next.ServeHTTP(capture, request)
		observability.ObserveHTTP(r.Method, request.Pattern, capture.status, time.Since(started))
		if r.URL.Path != "/health" && r.URL.Path != "/ready" && r.URL.Path != "/metrics" {
			slog.Info("http request", "method", r.Method, "path", r.URL.Path, "status", capture.status, "duration_ms", time.Since(started).Milliseconds(), "request_id", requestID, "traceparent", traceparent)
		}
	})
}
func randomHex(size int) string {
	raw := make([]byte, size)
	_, _ = rand.Read(raw)
	return fmt.Sprintf("%x", raw)
}
func validRequestID(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, c := range value {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '-' && c != '_' && c != '.' {
			return false
		}
	}
	return true
}
func validTraceparent(value string) bool {
	if len(value) != 55 || value[2] != '-' || value[35] != '-' || value[52] != '-' {
		return false
	}
	for i, c := range value {
		if i == 2 || i == 35 || i == 52 {
			continue
		}
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return value[3:35] != strings.Repeat("0", 32) && value[36:52] != strings.Repeat("0", 16)
}
func errorBody(code string, err error, retryable bool, requestID string) map[string]any {
	return map[string]any{"code": code, "message": err.Error(), "retryable": retryable, "requestId": requestID}
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
