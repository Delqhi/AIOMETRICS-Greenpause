package postgres

import (
	"context"
	"database/sql"

	"greenpause/internal/application"
)

type AuditLog struct {
	db *sql.DB
}

func NewAuditLog(db *sql.DB) *AuditLog {
	return &AuditLog{db: db}
}

func (a *AuditLog) Append(ctx context.Context, event application.AuditEvent) error {
	const query = `
INSERT INTO audit_event_records (tenant_id, reminder_id, event_type, correlation_id, occurred_at_utc)
VALUES ($1, $2, $3, $4, $5)
`
	_, err := a.db.ExecContext(ctx, query,
		event.TenantID,
		event.ReminderID,
		event.Type,
		event.CorrelationID,
		event.OccurredAtUtc,
	)
	return err
}
