package memory_test

import (
	"context"
	"testing"
	"time"

	"greenpause/internal/domain"
	"greenpause/internal/infrastructure/memory"
)

func TestReminderRepository_Contract_SaveAndGetByID(t *testing.T) {
	repo := memory.NewReminderRepository()
	now := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)

	reminder, err := domain.NewReminder(
		domain.ReminderID("rem-1"),
		domain.TenantID("tenant-a"),
		domain.UserID("user-1"),
		now.Add(5*time.Minute),
		"Reminder text",
		"idem-12345678",
		now,
	)
	if err != nil {
		t.Fatalf("new reminder: %v", err)
	}

	stored, existing, err := repo.SaveIfIdempotencyKeyAbsent(context.Background(), reminder)
	if err != nil {
		t.Fatalf("save absent: %v", err)
	}
	if !stored || existing != nil {
		t.Fatalf("first save must store reminder")
	}

	loaded, err := repo.GetByID(context.Background(), reminder.TenantID, reminder.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if loaded == nil {
		t.Fatalf("expected reminder to exist")
	}
	if loaded.ID != reminder.ID {
		t.Fatalf("unexpected id: got %s want %s", loaded.ID, reminder.ID)
	}

	loaded.Message = "mutated"
	reloaded, err := repo.GetByID(context.Background(), reminder.TenantID, reminder.ID)
	if err != nil {
		t.Fatalf("reloaded get by id: %v", err)
	}
	if reloaded.Message != "Reminder text" {
		t.Fatalf("repository must protect internal state via cloning")
	}
}

func TestReminderRepository_Contract_IdempotencyKeyUniquenessPerTenant(t *testing.T) {
	repo := memory.NewReminderRepository()
	now := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)

	first, err := domain.NewReminder(
		domain.ReminderID("rem-1"),
		domain.TenantID("tenant-a"),
		domain.UserID("user-1"),
		now.Add(5*time.Minute),
		"Reminder 1",
		"idem-12345678",
		now,
	)
	if err != nil {
		t.Fatalf("new first reminder: %v", err)
	}
	second, err := domain.NewReminder(
		domain.ReminderID("rem-2"),
		domain.TenantID("tenant-a"),
		domain.UserID("user-2"),
		now.Add(6*time.Minute),
		"Reminder 2",
		"idem-12345678",
		now,
	)
	if err != nil {
		t.Fatalf("new second reminder: %v", err)
	}
	thirdOtherTenant, err := domain.NewReminder(
		domain.ReminderID("rem-3"),
		domain.TenantID("tenant-b"),
		domain.UserID("user-3"),
		now.Add(7*time.Minute),
		"Reminder 3",
		"idem-12345678",
		now,
	)
	if err != nil {
		t.Fatalf("new third reminder: %v", err)
	}

	stored, existing, err := repo.SaveIfIdempotencyKeyAbsent(context.Background(), first)
	if err != nil {
		t.Fatalf("save first: %v", err)
	}
	if !stored || existing != nil {
		t.Fatalf("first write must create")
	}

	stored, existing, err = repo.SaveIfIdempotencyKeyAbsent(context.Background(), second)
	if err != nil {
		t.Fatalf("save second: %v", err)
	}
	if stored {
		t.Fatalf("same tenant + key must not create new reminder")
	}
	if existing == nil || existing.ID != first.ID {
		t.Fatalf("must return existing reminder")
	}

	stored, existing, err = repo.SaveIfIdempotencyKeyAbsent(context.Background(), thirdOtherTenant)
	if err != nil {
		t.Fatalf("save third: %v", err)
	}
	if !stored || existing != nil {
		t.Fatalf("other tenant may reuse same idempotency key")
	}
}
