package application

import (
	"context"
	"errors"

	"greenpause/internal/domain"
)

type CancelReminderCommand struct {
	TenantID      domain.TenantID
	ReminderID    domain.ReminderID
	CorrelationID string
}

type CancelReminderUseCase struct {
	repo  ReminderRepositoryPort
	index ScheduleIndexPort
	audit AuditLogPort
	clock ClockPort
}

func NewCancelReminderUseCase(
	repo ReminderRepositoryPort,
	index ScheduleIndexPort,
	audit AuditLogPort,
	clock ClockPort,
) (*CancelReminderUseCase, error) {
	if repo == nil || index == nil || audit == nil || clock == nil {
		return nil, errors.New("cancel use case dependencies must not be nil")
	}
	return &CancelReminderUseCase{repo: repo, index: index, audit: audit, clock: clock}, nil
}

func (uc *CancelReminderUseCase) Execute(ctx context.Context, cmd CancelReminderCommand) error {
	now := uc.clock.Now().UTC()
	reminder, err := uc.repo.GetByID(ctx, cmd.TenantID, cmd.ReminderID)
	if err != nil {
		return err
	}
	if reminder == nil {
		return ErrReminderNotFound
	}
	if reminder.Status == domain.ReminderStatusCanceled {
		return nil
	}
	if err := reminder.Cancel(now); err != nil {
		return err
	}
	if err := uc.repo.Save(ctx, reminder); err != nil {
		return err
	}
	if err := uc.index.Remove(ctx, reminder.TenantID, reminder.ID); err != nil {
		return err
	}
	return uc.audit.Append(ctx, AuditEvent{
		Type:          AuditEventTypeReminderCanceled,
		TenantID:      reminder.TenantID,
		ReminderID:    reminder.ID,
		CorrelationID: cmd.CorrelationID,
		OccurredAtUtc: now,
	})
}
