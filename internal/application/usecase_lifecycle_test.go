package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"greenpause/internal/application"
	"greenpause/internal/domain"
	"greenpause/internal/infrastructure/memory"
)

func TestCancelReminderUseCase_RemovesScheduleIndexAndWritesAudit(t *testing.T) {
	now := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	repo := memory.NewReminderRepository()
	index := memory.NewScheduleIndex()
	audit := memory.NewAuditLog()
	clock := memory.FixedClock{Current: now}
	ids := memory.NewSequenceReminderIDGenerator()

	scheduleUC, err := application.NewScheduleReminderUseCase(repo, index, audit, clock, ids)
	if err != nil {
		t.Fatalf("new schedule use case: %v", err)
	}
	cancelUC, err := application.NewCancelReminderUseCase(repo, index, audit, clock)
	if err != nil {
		t.Fatalf("new cancel use case: %v", err)
	}

	created, err := scheduleUC.Execute(context.Background(), application.ScheduleReminderCommand{
		TenantID:       domain.TenantID("tenant-a"),
		UserID:         domain.UserID("user-1"),
		DueAtUtc:       now.Add(2 * time.Minute),
		Message:        "Appointment",
		IdempotencyKey: "idem-12345678",
		CorrelationID:  "corr-create",
	})
	if err != nil {
		t.Fatalf("schedule execute: %v", err)
	}

	if err := cancelUC.Execute(context.Background(), application.CancelReminderCommand{
		TenantID:      domain.TenantID("tenant-a"),
		ReminderID:    created.ReminderID,
		CorrelationID: "corr-cancel",
	}); err != nil {
		t.Fatalf("cancel execute: %v", err)
	}

	if _, exists := index.DueAt(domain.TenantID("tenant-a"), created.ReminderID); exists {
		t.Fatalf("schedule index must remove canceled reminder")
	}

	loaded, err := repo.GetByID(context.Background(), domain.TenantID("tenant-a"), created.ReminderID)
	if err != nil {
		t.Fatalf("get reminder: %v", err)
	}
	if loaded.Status != domain.ReminderStatusCanceled {
		t.Fatalf("expected canceled status, got %s", loaded.Status)
	}

	events := audit.Events()
	if len(events) != 2 {
		t.Fatalf("expected create + cancel events, got %d", len(events))
	}
}

func TestAcknowledgeReminderUseCase_OnlyTriggeredAllowed(t *testing.T) {
	now := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	repo := memory.NewReminderRepository()
	audit := memory.NewAuditLog()
	clock := memory.FixedClock{Current: now}
	ackUC, err := application.NewAcknowledgeReminderUseCase(repo, audit, clock)
	if err != nil {
		t.Fatalf("new acknowledge use case: %v", err)
	}

	reminder, err := domain.NewReminder(
		domain.ReminderID("rem-1"),
		domain.TenantID("tenant-a"),
		domain.UserID("user-1"),
		now.Add(2*time.Minute),
		"Message",
		"idem-12345678",
		now,
	)
	if err != nil {
		t.Fatalf("new reminder: %v", err)
	}
	stored, _, err := repo.SaveIfIdempotencyKeyAbsent(context.Background(), reminder)
	if err != nil || !stored {
		t.Fatalf("seed reminder failed: stored=%v err=%v", stored, err)
	}

	err = ackUC.Execute(context.Background(), application.AcknowledgeReminderCommand{
		TenantID:      reminder.TenantID,
		ReminderID:    reminder.ID,
		CorrelationID: "corr-ack",
	})
	if !errors.Is(err, domain.ErrInvalidStatusTransition) {
		t.Fatalf("expected invalid transition, got %v", err)
	}

	if err := reminder.Trigger(now.Add(1 * time.Minute)); err != nil {
		t.Fatalf("trigger reminder: %v", err)
	}
	if err := repo.Save(context.Background(), reminder); err != nil {
		t.Fatalf("persist triggered reminder: %v", err)
	}

	err = ackUC.Execute(context.Background(), application.AcknowledgeReminderCommand{
		TenantID:      reminder.TenantID,
		ReminderID:    reminder.ID,
		CorrelationID: "corr-ack-2",
	})
	if err != nil {
		t.Fatalf("acknowledge execute: %v", err)
	}

	updated, err := repo.GetByID(context.Background(), reminder.TenantID, reminder.ID)
	if err != nil {
		t.Fatalf("reload reminder: %v", err)
	}
	if updated.Status != domain.ReminderStatusAcknowledged {
		t.Fatalf("expected acknowledged status, got %s", updated.Status)
	}
}

func TestGetReminderUseCase_ReturnsReminderView(t *testing.T) {
	now := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	repo := memory.NewReminderRepository()

	reminder, err := domain.NewReminder(
		domain.ReminderID("rem-1"),
		domain.TenantID("tenant-a"),
		domain.UserID("user-1"),
		now.Add(3*time.Minute),
		"Message",
		"idem-12345678",
		now,
	)
	if err != nil {
		t.Fatalf("new reminder: %v", err)
	}
	stored, _, err := repo.SaveIfIdempotencyKeyAbsent(context.Background(), reminder)
	if err != nil || !stored {
		t.Fatalf("seed reminder failed: stored=%v err=%v", stored, err)
	}

	getUC, err := application.NewGetReminderUseCase(repo)
	if err != nil {
		t.Fatalf("new get use case: %v", err)
	}

	view, err := getUC.Execute(context.Background(), application.GetReminderQuery{
		TenantID:   domain.TenantID("tenant-a"),
		ReminderID: domain.ReminderID("rem-1"),
	})
	if err != nil {
		t.Fatalf("get execute: %v", err)
	}
	if view.ReminderID != reminder.ID {
		t.Fatalf("unexpected reminder id: got %s want %s", view.ReminderID, reminder.ID)
	}
}
