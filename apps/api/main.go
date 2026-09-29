package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"trafficos/apps/api/internal/application"
	"trafficos/apps/api/internal/realtime"
	"trafficos/apps/api/internal/storage"
	"trafficos/apps/api/internal/transport"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	var store storage.Store
	var deps *storage.Dependencies
	hub := realtime.New()
	if env("STORAGE_MODE", "postgres") == "memory" {
		store = storage.NewMemory()
		slog.Warn("Explicit memory mode: PostgreSQL, Redis and NATS disabled; audit is session-only")
	} else {
		startup, done := context.WithTimeout(ctx, 15*time.Second)
		p, err := storage.Open(startup, env("DATABASE_URL", "postgres://trafficos:trafficos@localhost:5432/trafficos?sslmode=disable"))
		if err != nil {
			slog.Error("postgres startup failed", "error", err)
			os.Exit(1)
		}
		defer p.Pool.Close()
		if err = p.Migrate(startup); err != nil {
			slog.Error("migration failed", "error", err)
			os.Exit(1)
		}
		done()
		store = p
		if len(os.Args) > 1 && os.Args[1] == "migrate" {
			slog.Info("migrations applied")
			return
		}
		deps, err = storage.Connect(env("REDIS_URL", "redis://localhost:6379/0"), env("NATS_URL", "nats://localhost:4222"), hub)
		if err != nil {
			slog.Error("dependencies startup failed", "error", err)
			os.Exit(1)
		}
		defer deps.Close()
		go deps.Relay(ctx, p)
	}
	service := application.New(store, hub)
	service.NativeBus = deps != nil
	if deps != nil {
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					snapshot := service.Snapshot()
					cacheCtx, done := context.WithTimeout(ctx, time.Second)
					raw, _ := jsonBytes(snapshot)
					if err := deps.Redis.Set(cacheCtx, "trafficos:current", raw, 10*time.Second).Err(); err != nil {
						slog.Warn("snapshot cache write failed", "error", err)
					}
					done()
				}
			}
		}()
	}
	server := transport.New(service, deps, strings.Split(env("ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000"), ","), env("SIMULATOR_TOKEN", "local-simulator-change-me"))
	httpServer := &http.Server{Addr: env("API_ADDR", ":8080"), Handler: server.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 * 1024}
	go func() {
		<-ctx.Done()
		shutdown, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		httpServer.Shutdown(shutdown)
	}()
	slog.Info("TrafficOS API listening", "address", httpServer.Addr, "storage", store.Mode())
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("HTTP server failed", "error", err)
		os.Exit(1)
	}
}
