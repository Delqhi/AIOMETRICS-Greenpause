package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	minLeadTimeSeconds      = 30
	minIdempotencyKeyLength = 8
	maxIdempotencyKeyLength = 128
	maxMessageBytes         = 1024
)

type TenantID string

type UserID string

type ReminderID string

type ReminderStatus string

const (
	ReminderStatusScheduled    ReminderStatus = "Scheduled"
	ReminderStatusTriggered    ReminderStatus = "Triggered"
	ReminderStatusAcknowledged ReminderStatus = "Acknowledged"
	ReminderStatusCanceled     ReminderStatus = "Canceled"
)

var (
	ErrInvalidReminderID       = errors.New("invalid reminder id")
	ErrInvalidTenantID         = errors.New("invalid tenant id")
	ErrInvalidUserID           = errors.New("invalid user id")
	ErrInvalidDueAt            = errors.New("invalid due timestamp")
	ErrInvalidMessage          = errors.New("invalid message")
	ErrInvalidIdempotencyKey   = errors.New("invalid idempotency key")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
)

type Reminder struct {
	ID             ReminderID
	TenantID       TenantID
	UserID         UserID
	DueAtUtc       time.Time
	Message        string
	Status         ReminderStatus
	IdempotencyKey string
	CreatedAtUtc   time.Time
	TriggeredAtUtc *time.Time
	CanceledAtUtc  *time.Time
	AcknowledgedAt *time.Time
	Version        int64
}

func NewReminder(
	id ReminderID,
	tenantID TenantID,
	userID UserID,
	dueAt time.Time,
	message string,
	idempotencyKey string,
	now time.Time,
) (*Reminder, error) {
	if strings.TrimSpace(string(id)) == "" {
		return nil, ErrInvalidReminderID
	}
	if strings.TrimSpace(string(tenantID)) == "" {
		return nil, ErrInvalidTenantID
	}
	if strings.TrimSpace(string(userID)) == "" {
		return nil, ErrInvalidUserID
	}

	normalizedNow := now.UTC()
	normalizedDueAt := dueAt.UTC()
	if normalizedDueAt.Before(normalizedNow.Add(minLeadTimeSeconds * time.Second)) {
		return nil, ErrInvalidDueAt
	}

	if strings.TrimSpace(message) == "" {
		return nil, ErrInvalidMessage
	}
	if !utf8.ValidString(message) {
		return nil, ErrInvalidMessage
	}
	if len([]byte(message)) > maxMessageBytes {
		return nil, ErrInvalidMessage
	}

	trimmedKey := strings.TrimSpace(idempotencyKey)
	if len(trimmedKey) < minIdempotencyKeyLength || len(trimmedKey) > maxIdempotencyKeyLength {
		return nil, ErrInvalidIdempotencyKey
	}

	return &Reminder{
		ID:             id,
		TenantID:       tenantID,
		UserID:         userID,
		DueAtUtc:       normalizedDueAt,
		Message:        message,
		Status:         ReminderStatusScheduled,
		IdempotencyKey: trimmedKey,
		CreatedAtUtc:   normalizedNow,
		Version:        1,
	}, nil
}

func (r *Reminder) Trigger(at time.Time) error {
	if r.Status != ReminderStatusScheduled {
		return ErrInvalidStatusTransition
	}
	normalized := at.UTC()
	r.Status = ReminderStatusTriggered
	r.TriggeredAtUtc = &normalized
	r.Version++
	return nil
}

func (r *Reminder) Cancel(at time.Time) error {
	if r.Status != ReminderStatusScheduled {
		return ErrInvalidStatusTransition
	}
	normalized := at.UTC()
	r.Status = ReminderStatusCanceled
	r.CanceledAtUtc = &normalized
	r.Version++
	return nil
}

func (r *Reminder) Acknowledge(at time.Time) error {
	if r.Status != ReminderStatusTriggered {
		return ErrInvalidStatusTransition
	}
	normalized := at.UTC()
	r.Status = ReminderStatusAcknowledged
	r.AcknowledgedAt = &normalized
	r.Version++
	return nil
}

func (r *Reminder) Clone() *Reminder {
	if r == nil {
		return nil
	}

	copyReminder := *r
	if r.TriggeredAtUtc != nil {
		triggered := *r.TriggeredAtUtc
		copyReminder.TriggeredAtUtc = &triggered
	}
	if r.CanceledAtUtc != nil {
		canceled := *r.CanceledAtUtc
		copyReminder.CanceledAtUtc = &canceled
	}
	if r.AcknowledgedAt != nil {
		ack := *r.AcknowledgedAt
		copyReminder.AcknowledgedAt = &ack
	}
	return &copyReminder
}
