package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"greenpause/internal/domain"
)

type ScheduleReminderCommand struct {
	TenantID       domain.TenantID
	UserID         domain.UserID
	DueAtUtc       time.Time
	Message        string
	IdempotencyKey string
	CorrelationID  string
}

type ScheduleReminderResult struct {
	ReminderID     domain.ReminderID
	AcceptedAtUnix int64
	Created        bool
}

type ScheduleReminderUseCase struct {
	repo     ReminderRepositoryPort
	index    ScheduleIndexPort
	audit    AuditLogPort
	clock    ClockPort
	idSource ReminderIDGeneratorPort
}

func NewScheduleReminderUseCase(
	repo ReminderRepositoryPort,
	index ScheduleIndexPort,
	audit AuditLogPort,
	clock ClockPort,
	idSource ReminderIDGeneratorPort,
) (*ScheduleReminderUseCase, error) {
	if repo == nil || index == nil || audit == nil || clock == nil || idSource == nil {
		return nil, errors.New("schedule use case dependencies must not be nil")
	}
	return &ScheduleReminderUseCase{repo: repo, index: index, audit: audit, clock: clock, idSource: idSource}, nil
}

func (uc *ScheduleReminderUseCase) Execute(ctx context.Context, cmd ScheduleReminderCommand) (ScheduleReminderResult, error) {
	now := uc.clock.Now().UTC()
	if cmd.DueAtUtc.IsZero() {
		return ScheduleReminderResult{}, fmt.Errorf("due timestamp missing")
	}

	reminder, err := domain.NewReminder(
		uc.idSource.NewReminderID(),
		cmd.TenantID,
		cmd.UserID,
		cmd.DueAtUtc,
		cmd.Message,
		cmd.IdempotencyKey,
		now,
	)
	if err != nil {
		return ScheduleReminderResult{}, err
	}

	stored, existing, err := uc.repo.SaveIfIdempotencyKeyAbsent(ctx, reminder)
	if err != nil {
		return ScheduleReminderResult{}, err
	}
	if !stored {
		return ScheduleReminderResult{
			ReminderID:     existing.ID,
			AcceptedAtUnix: now.Unix(),
			Created:        false,
		}, nil
	}

	if err := uc.index.Upsert(ctx, reminder.TenantID, reminder.DueAtUtc, reminder.ID); err != nil {
		return ScheduleReminderResult{}, err
	}
	if err := uc.audit.Append(ctx, AuditEvent{
		Type:          AuditEventTypeReminderCreated,
		TenantID:      reminder.TenantID,
		ReminderID:    reminder.ID,
		CorrelationID: cmd.CorrelationID,
		OccurredAtUtc: now,
	}); err != nil {
		return ScheduleReminderResult{}, err
	}

	return ScheduleReminderResult{
		ReminderID:     reminder.ID,
		AcceptedAtUnix: now.Unix(),
		Created:        true,
	}, nil
}
