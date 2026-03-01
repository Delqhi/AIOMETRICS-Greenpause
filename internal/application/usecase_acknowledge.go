package application

import (
	"context"
	"errors"

	"greenpause/internal/domain"
)

type AcknowledgeReminderCommand struct {
	TenantID      domain.TenantID
	ReminderID    domain.ReminderID
	CorrelationID string
}

type AcknowledgeReminderUseCase struct {
	repo  ReminderRepositoryPort
	audit AuditLogPort
	clock ClockPort
}

func NewAcknowledgeReminderUseCase(
	repo ReminderRepositoryPort,
	audit AuditLogPort,
	clock ClockPort,
) (*AcknowledgeReminderUseCase, error) {
	if repo == nil || audit == nil || clock == nil {
		return nil, errors.New("acknowledge use case dependencies must not be nil")
	}
	return &AcknowledgeReminderUseCase{repo: repo, audit: audit, clock: clock}, nil
}

func (uc *AcknowledgeReminderUseCase) Execute(ctx context.Context, cmd AcknowledgeReminderCommand) error {
	now := uc.clock.Now().UTC()
	reminder, err := uc.repo.GetByID(ctx, cmd.TenantID, cmd.ReminderID)
	if err != nil {
		return err
	}
	if reminder == nil {
		return ErrReminderNotFound
	}
	if reminder.Status == domain.ReminderStatusAcknowledged {
		return nil
	}
	if err := reminder.Acknowledge(now); err != nil {
		return err
	}
	if err := uc.repo.Save(ctx, reminder); err != nil {
		return err
	}
	return uc.audit.Append(ctx, AuditEvent{
		Type:          AuditEventTypeReminderAcknowledged,
		TenantID:      reminder.TenantID,
		ReminderID:    reminder.ID,
		CorrelationID: cmd.CorrelationID,
		OccurredAtUtc: now,
	})
}
