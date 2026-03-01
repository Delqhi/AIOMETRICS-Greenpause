package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"greenpause/internal/application"
	"greenpause/internal/infrastructure/httpapi"
	"greenpause/internal/infrastructure/idgen"
	"greenpause/internal/infrastructure/memory"
	"greenpause/internal/infrastructure/postgres"
	"greenpause/internal/infrastructure/redis"
	"greenpause/internal/infrastructure/system"
)

func main() {
	addr := ":" + envOrDefault("APP_PORT", "8080")

	repo, audit, index, cleanup, err := buildAdapters()
	if err != nil {
		log.Fatalf("build adapters: %v", err)
	}
	defer cleanup()

	clock := system.Clock{}
	idSource := idgen.NewUUIDv7ReminderIDGenerator()

	scheduleUC, err := application.NewScheduleReminderUseCase(repo, index, audit, clock, idSource)
	if err != nil {
		log.Fatalf("init schedule use case: %v", err)
	}
	getUC, err := application.NewGetReminderUseCase(repo)
	if err != nil {
		log.Fatalf("init get use case: %v", err)
	}
	cancelUC, err := application.NewCancelReminderUseCase(repo, index, audit, clock)
	if err != nil {
		log.Fatalf("init cancel use case: %v", err)
	}
	ackUC, err := application.NewAcknowledgeReminderUseCase(repo, audit, clock)
	if err != nil {
		log.Fatalf("init acknowledge use case: %v", err)
	}

	server, err := httpapi.NewServer(scheduleUC, getUC, cancelUC, ackUC)
	if err != nil {
		log.Fatalf("init http server: %v", err)
	}

	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, server.Handler()); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

func buildAdapters() (
	application.ReminderRepositoryPort,
	application.AuditLogPort,
	application.ScheduleIndexPort,
	func(),
	error,
) {
	storageBackend := strings.ToLower(strings.TrimSpace(envOrDefault("STORAGE_BACKEND", "memory")))
	scheduleBackend := strings.ToLower(strings.TrimSpace(envOrDefault("SCHEDULE_BACKEND", "memory")))

	var (
		repo    application.ReminderRepositoryPort
		audit   application.AuditLogPort
		index   application.ScheduleIndexPort
		cleanup = func() {}
	)

	switch storageBackend {
	case "memory":
		repo = memory.NewReminderRepository()
		audit = memory.NewAuditLog()
	case "postgres":
		driverName := envOrDefault("DATABASE_DRIVER", "postgres")
		dsn := strings.TrimSpace(os.Getenv("DATABASE_DSN"))
		if dsn == "" {
			return nil, nil, nil, cleanup, fmt.Errorf("DATABASE_DSN required for postgres backend")
		}

		db, err := sql.Open(driverName, dsn)
		if err != nil {
			return nil, nil, nil, cleanup, err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			_ = db.Close()
			return nil, nil, nil, cleanup, fmt.Errorf("ping postgres: %w", err)
		}
		if err := postgres.EnsureSchema(ctx, db); err != nil {
			_ = db.Close()
			return nil, nil, nil, cleanup, fmt.Errorf("ensure schema: %w", err)
		}

		repo = postgres.NewReminderRepository(db)
		audit = postgres.NewAuditLog(db)
		cleanup = func() {
			_ = db.Close()
		}
	default:
		return nil, nil, nil, cleanup, fmt.Errorf("unsupported STORAGE_BACKEND: %s", storageBackend)
	}

	switch scheduleBackend {
	case "memory":
		index = memory.NewScheduleIndex()
	case "redis":
		addr := strings.TrimSpace(os.Getenv("REDIS_ADDR"))
		if addr == "" {
			return nil, nil, nil, cleanup, fmt.Errorf("REDIS_ADDR required for redis schedule backend")
		}
		index = redis.NewScheduleIndex(addr)
	default:
		return nil, nil, nil, cleanup, fmt.Errorf("unsupported SCHEDULE_BACKEND: %s", scheduleBackend)
	}

	return repo, audit, index, cleanup, nil
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
