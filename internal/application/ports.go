package application

import (
	"context"
	"time"

	"greenpause/internal/domain"
)

type ReminderRepositoryPort interface {
	SaveIfIdempotencyKeyAbsent(ctx context.Context, reminder *domain.Reminder) (stored bool, existing *domain.Reminder, err error)
	GetByID(ctx context.Context, tenantID domain.TenantID, reminderID domain.ReminderID) (*domain.Reminder, error)
	Save(ctx context.Context, reminder *domain.Reminder) error
}

type ScheduleIndexPort interface {
	Upsert(ctx context.Context, tenantID domain.TenantID, dueAtUtc time.Time, reminderID domain.ReminderID) error
	Remove(ctx context.Context, tenantID domain.TenantID, reminderID domain.ReminderID) error
}

type AuditLogPort interface {
	Append(ctx context.Context, event AuditEvent) error
}

type ClockPort interface {
	Now() time.Time
}

type ReminderIDGeneratorPort interface {
	NewReminderID() domain.ReminderID
}
