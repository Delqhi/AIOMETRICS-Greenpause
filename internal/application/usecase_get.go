package application

import (
	"context"
	"errors"
	"time"

	"greenpause/internal/domain"
)

type GetReminderQuery struct {
	TenantID   domain.TenantID
	ReminderID domain.ReminderID
}

type ReminderView struct {
	ReminderID     domain.ReminderID
	TenantID       domain.TenantID
	UserID         domain.UserID
	DueAtUtc       time.Time
	ReminderStatus domain.ReminderStatus
	Message        string
	CreatedAtUtc   time.Time
	TriggeredAtUtc *time.Time
	CanceledAtUtc  *time.Time
	AcknowledgedAt *time.Time
}

type GetReminderUseCase struct {
	repo ReminderRepositoryPort
}

func NewGetReminderUseCase(repo ReminderRepositoryPort) (*GetReminderUseCase, error) {
	if repo == nil {
		return nil, errors.New("get use case dependencies must not be nil")
	}
	return &GetReminderUseCase{repo: repo}, nil
}

func (uc *GetReminderUseCase) Execute(ctx context.Context, query GetReminderQuery) (ReminderView, error) {
	reminder, err := uc.repo.GetByID(ctx, query.TenantID, query.ReminderID)
	if err != nil {
		return ReminderView{}, err
	}
	if reminder == nil {
		return ReminderView{}, ErrReminderNotFound
	}

	return ReminderView{
		ReminderID:     reminder.ID,
		TenantID:       reminder.TenantID,
		UserID:         reminder.UserID,
		DueAtUtc:       reminder.DueAtUtc,
		ReminderStatus: reminder.Status,
		Message:        reminder.Message,
		CreatedAtUtc:   reminder.CreatedAtUtc,
		TriggeredAtUtc: reminder.TriggeredAtUtc,
		CanceledAtUtc:  reminder.CanceledAtUtc,
		AcknowledgedAt: reminder.AcknowledgedAt,
	}, nil
}
