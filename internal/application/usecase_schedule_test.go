package application_test

import (
	"context"
	"testing"
	"time"

	"greenpause/internal/application"
	"greenpause/internal/domain"
	"greenpause/internal/infrastructure/memory"
)

func TestScheduleReminderUseCase_IdempotentByTenantAndKey(t *testing.T) {
	now := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	repo := memory.NewReminderRepository()
	index := memory.NewScheduleIndex()
	audit := memory.NewAuditLog()
	clock := memory.FixedClock{Current: now}
	ids := memory.NewSequenceReminderIDGenerator()

	uc, err := application.NewScheduleReminderUseCase(repo, index, audit, clock, ids)
	if err != nil {
		t.Fatalf("new use case: %v", err)
	}

	cmd := application.ScheduleReminderCommand{
		TenantID:       domain.TenantID("tenant-a"),
		UserID:         domain.UserID("user-1"),
		DueAtUtc:       now.Add(2 * time.Minute),
		Message:        "Doctor appointment",
		IdempotencyKey: "idem-12345678",
		CorrelationID:  "corr-1",
	}

	first, err := uc.Execute(context.Background(), cmd)
	if err != nil {
		t.Fatalf("first execute: %v", err)
	}
	if !first.Created {
		t.Fatalf("first execute must create reminder")
	}

	second, err := uc.Execute(context.Background(), cmd)
	if err != nil {
		t.Fatalf("second execute: %v", err)
	}
	if second.Created {
		t.Fatalf("second execute must be idempotent")
	}
	if first.ReminderID != second.ReminderID {
		t.Fatalf("idempotent call must return same reminder id")
	}

	events := audit.Events()
	if len(events) != 1 {
		t.Fatalf("expected exactly one audit event, got %d", len(events))
	}
}

func TestScheduleReminderUseCase_RejectsDueAtInsideLeadTime(t *testing.T) {
	now := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	repo := memory.NewReminderRepository()
	index := memory.NewScheduleIndex()
	audit := memory.NewAuditLog()
	clock := memory.FixedClock{Current: now}
	ids := memory.NewSequenceReminderIDGenerator()

	uc, err := application.NewScheduleReminderUseCase(repo, index, audit, clock, ids)
	if err != nil {
		t.Fatalf("new use case: %v", err)
	}

	_, err = uc.Execute(context.Background(), application.ScheduleReminderCommand{
		TenantID:       domain.TenantID("tenant-a"),
		UserID:         domain.UserID("user-1"),
		DueAtUtc:       now.Add(20 * time.Second),
		Message:        "x",
		IdempotencyKey: "idem-12345678",
	})
	if err == nil {
		t.Fatalf("expected dueAt validation error")
	}
}
