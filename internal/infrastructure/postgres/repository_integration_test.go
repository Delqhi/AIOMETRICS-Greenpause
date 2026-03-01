package postgres

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"greenpause/internal/application"
	"greenpause/internal/domain"
)

func TestReminderRepository_Integration_SaveGetAndIdempotency(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if strings.TrimSpace(dsn) == "" {
		t.Skip("TEST_POSTGRES_DSN not set")
	}
	driverName := os.Getenv("TEST_POSTGRES_DRIVER")
	if strings.TrimSpace(driverName) == "" {
		driverName = "postgres"
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		if strings.Contains(err.Error(), "unknown driver") {
			t.Skip("no postgres driver registered in test binary")
		}
		t.Fatalf("ping db: %v", err)
	}

	if err := EnsureSchema(ctx, db); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	if _, err := db.ExecContext(ctx, `TRUNCATE TABLE audit_event_records, reminder_records`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	repo := NewReminderRepository(db)
	now := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)

	first, err := domain.NewReminder(
		domain.ReminderID("rem-1"),
		domain.TenantID("tenant-a"),
		domain.UserID("user-1"),
		now.Add(2*time.Minute),
		"Reminder 1",
		"idem-12345678",
		now,
	)
	if err != nil {
		t.Fatalf("new first reminder: %v", err)
	}
	stored, existing, err := repo.SaveIfIdempotencyKeyAbsent(ctx, first)
	if err != nil {
		t.Fatalf("save first: %v", err)
	}
	if !stored || existing != nil {
		t.Fatalf("first write must create")
	}

	second, err := domain.NewReminder(
		domain.ReminderID("rem-2"),
		domain.TenantID("tenant-a"),
		domain.UserID("user-2"),
		now.Add(3*time.Minute),
		"Reminder 2",
		"idem-12345678",
		now,
	)
	if err != nil {
		t.Fatalf("new second reminder: %v", err)
	}
	stored, existing, err = repo.SaveIfIdempotencyKeyAbsent(ctx, second)
	if err != nil {
		t.Fatalf("save second: %v", err)
	}
	if stored {
		t.Fatalf("same tenant + key must not create new record")
	}
	if existing == nil || existing.ID != first.ID {
		t.Fatalf("expected existing reminder to be returned")
	}

	loaded, err := repo.GetByID(ctx, first.TenantID, first.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if loaded == nil {
		t.Fatalf("expected reminder")
	}
	if loaded.Status != domain.ReminderStatusScheduled {
		t.Fatalf("expected scheduled status, got %s", loaded.Status)
	}

	if err := loaded.Cancel(now.Add(90 * time.Second)); err != nil {
		t.Fatalf("cancel reminder: %v", err)
	}
	if err := repo.Save(ctx, loaded); err != nil {
		t.Fatalf("save update: %v", err)
	}
	updated, err := repo.GetByID(ctx, first.TenantID, first.ID)
	if err != nil {
		t.Fatalf("reload reminder: %v", err)
	}
	if updated.Status != domain.ReminderStatusCanceled {
		t.Fatalf("expected canceled status, got %s", updated.Status)
	}
}

func TestAuditLog_Integration_Append(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if strings.TrimSpace(dsn) == "" {
		t.Skip("TEST_POSTGRES_DSN not set")
	}
	driverName := os.Getenv("TEST_POSTGRES_DRIVER")
	if strings.TrimSpace(driverName) == "" {
		driverName = "postgres"
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		if strings.Contains(err.Error(), "unknown driver") {
			t.Skip("no postgres driver registered in test binary")
		}
		t.Fatalf("ping db: %v", err)
	}

	if err := EnsureSchema(ctx, db); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	if _, err := db.ExecContext(ctx, `TRUNCATE TABLE audit_event_records`); err != nil {
		t.Fatalf("truncate audit table: %v", err)
	}

	audit := NewAuditLog(db)
	err = audit.Append(ctx, application.AuditEvent{
		Type:          application.AuditEventTypeReminderCreated,
		TenantID:      domain.TenantID("tenant-a"),
		ReminderID:    domain.ReminderID("rem-1"),
		CorrelationID: "corr-1",
		OccurredAtUtc: time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("append audit event: %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_event_records`).Scan(&count); err != nil {
		t.Fatalf("count audit events: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 audit event, got %d", count)
	}
}

func TestReminderRepository_Integration_UnknownStatusFailsScan(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if strings.TrimSpace(dsn) == "" {
		t.Skip("TEST_POSTGRES_DSN not set")
	}
	driverName := os.Getenv("TEST_POSTGRES_DRIVER")
	if strings.TrimSpace(driverName) == "" {
		driverName = "postgres"
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		if strings.Contains(err.Error(), "unknown driver") {
			t.Skip("no postgres driver registered in test binary")
		}
		t.Fatalf("ping db: %v", err)
	}

	if err := EnsureSchema(ctx, db); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	if _, err := db.ExecContext(ctx, `TRUNCATE TABLE reminder_records`); err != nil {
		t.Fatalf("truncate reminder table: %v", err)
	}

	_, err = db.ExecContext(ctx, `
INSERT INTO reminder_records (
  tenant_id, reminder_id, user_id, due_at_utc, message,
  status, idempotency_key, created_at_utc, version
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
`, "tenant-a", "rem-err", "user-1", time.Now().UTC().Add(time.Minute), "x", "Unsupported", "idem-12345678", time.Now().UTC(), int64(1))
	if err != nil {
		t.Fatalf("insert malformed row: %v", err)
	}

	repo := NewReminderRepository(db)
	_, err = repo.GetByID(ctx, domain.TenantID("tenant-a"), domain.ReminderID("rem-err"))
	if err == nil {
		t.Fatalf("expected scan failure for unknown status")
	}
	if errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("unexpected no rows")
	}
}
