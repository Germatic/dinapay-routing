package main

import (
	"context"
	"encoding/json"
	"github.com/Germatic/dinapay-routing/internal/adapters/controlplane"
	"github.com/Germatic/dinapay-routing/internal/adapters/postgres"
	"github.com/Germatic/dinapay-routing/internal/adapters/zen"
	"github.com/Germatic/dinapay-routing/internal/app"
	"github.com/Germatic/dinapay-routing/internal/core"
	"github.com/Germatic/dinapay-routing/internal/transport/httpapi"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func main() {
	var routes []core.Registration
	if os.Getenv("CONTROL_PLANE_URL") == "" {
		if err := json.Unmarshal([]byte(os.Getenv("ROUTES_JSON")), &routes); err != nil || len(routes) == 0 {
			slog.Error("ROUTES_JSON is invalid or empty", "error", err)
			os.Exit(1)
		}
	}
	db, err := pgxpool.New(context.Background(), required("DB_URL"))
	if err != nil {
		slog.Error("database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	store := postgres.New(db)
	if err := store.Migrate(context.Background()); err != nil {
		slog.Error("migration", "error", err)
		os.Exit(1)
	}
	rules := zen.New(required("ZEN_URL"), os.Getenv("ZEN_PROJECT"), os.Getenv("ZEN_DECISION"), os.Getenv("ZEN_ACCESS_TOKEN"))
	var router *app.Router
	if controlPlaneURL := os.Getenv("CONTROL_PLANE_URL"); controlPlaneURL != "" {
		source := controlplane.New(controlPlaneURL, required("CONTROL_PLANE_TOKEN"), env("ENVIRONMENT", "sandbox"), env("API_VERSION", "v2"))
		refreshCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err = source.Refresh(refreshCtx)
		cancel()
		if err != nil {
			slog.Error("initial control-plane snapshot", "error", err)
			os.Exit(1)
		}
		go source.Run(context.Background(), 5*time.Second, func(err error) { slog.Error("refresh control-plane snapshot", "error", err) })
		router = app.NewWithRegistrations(rules, store, source, required("ZEN_POLICY_VERSION"))
	} else {
		router = app.New(rules, store, routes, required("ZEN_POLICY_VERSION"))
	}
	server := &http.Server{Addr: ":" + env("PORT", "8091"), Handler: httpapi.New(router, required("SERVICE_TOKEN")), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	slog.Info("dinapay-routing starting", "addr", server.Addr, "routes", len(routes))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server", "error", err)
		os.Exit(1)
	}
}
func required(k string) string {
	v := os.Getenv(k)
	if v == "" {
		slog.Error("required environment variable missing", "key", k)
		os.Exit(1)
	}
	return v
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
