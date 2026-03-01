package application

import (
	"time"

	"greenpause/internal/domain"
)

type AuditEventType string

const (
	AuditEventTypeReminderCreated      AuditEventType = "ReminderCreated"
	AuditEventTypeReminderCanceled     AuditEventType = "ReminderCanceled"
	AuditEventTypeReminderAcknowledged AuditEventType = "ReminderAcknowledged"
)

type AuditEvent struct {
	Type          AuditEventType
	TenantID      domain.TenantID
	ReminderID    domain.ReminderID
	CorrelationID string
	OccurredAtUtc time.Time
}
