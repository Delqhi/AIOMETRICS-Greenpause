package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"greenpause/internal/domain"
)

type ReminderRepository struct {
	db *sql.DB
}

func NewReminderRepository(db *sql.DB) *ReminderRepository {
	return &ReminderRepository{db: db}
}

func (r *ReminderRepository) SaveIfIdempotencyKeyAbsent(ctx context.Context, reminder *domain.Reminder) (bool, *domain.Reminder, error) {
	const query = `
INSERT INTO reminder_records (
  tenant_id, reminder_id, user_id, due_at_utc, message,
  status, idempotency_key, created_at_utc, triggered_at_utc,
  canceled_at_utc, acknowledged_at_utc, version
) VALUES (
  $1, $2, $3, $4, $5,
  $6, $7, $8, $9,
  $10, $11, $12
)
ON CONFLICT (tenant_id, idempotency_key) DO NOTHING
RETURNING
  tenant_id, reminder_id, user_id, due_at_utc, message,
  status, idempotency_key, created_at_utc, triggered_at_utc,
  canceled_at_utc, acknowledged_at_utc, version
`

	_, err := scanReminder(r.db.QueryRowContext(ctx, query,
		reminder.TenantID,
		reminder.ID,
		reminder.UserID,
		reminder.DueAtUtc,
		reminder.Message,
		reminder.Status,
		reminder.IdempotencyKey,
		reminder.CreatedAtUtc,
		reminder.TriggeredAtUtc,
		reminder.CanceledAtUtc,
		reminder.AcknowledgedAt,
		reminder.Version,
	))
	if err == nil {
		return true, nil, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, nil, err
	}

	const existingQuery = `
SELECT
  tenant_id, reminder_id, user_id, due_at_utc, message,
  status, idempotency_key, created_at_utc, triggered_at_utc,
  canceled_at_utc, acknowledged_at_utc, version
FROM reminder_records
WHERE tenant_id = $1 AND idempotency_key = $2
`
	existing, err := scanReminder(r.db.QueryRowContext(ctx, existingQuery, reminder.TenantID, reminder.IdempotencyKey))
	if err != nil {
		return false, nil, err
	}
	return false, existing, nil
}

func (r *ReminderRepository) GetByID(ctx context.Context, tenantID domain.TenantID, reminderID domain.ReminderID) (*domain.Reminder, error) {
	const query = `
SELECT
  tenant_id, reminder_id, user_id, due_at_utc, message,
  status, idempotency_key, created_at_utc, triggered_at_utc,
  canceled_at_utc, acknowledged_at_utc, version
FROM reminder_records
WHERE tenant_id = $1 AND reminder_id = $2
`
	reminder, err := scanReminder(r.db.QueryRowContext(ctx, query, tenantID, reminderID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return reminder, nil
}

func (r *ReminderRepository) Save(ctx context.Context, reminder *domain.Reminder) error {
	const query = `
UPDATE reminder_records
SET
  user_id = $3,
  due_at_utc = $4,
  message = $5,
  status = $6,
  created_at_utc = $7,
  triggered_at_utc = $8,
  canceled_at_utc = $9,
  acknowledged_at_utc = $10,
  version = $11
WHERE tenant_id = $1 AND reminder_id = $2
`
	_, err := r.db.ExecContext(ctx, query,
		reminder.TenantID,
		reminder.ID,
		reminder.UserID,
		reminder.DueAtUtc,
		reminder.Message,
		reminder.Status,
		reminder.CreatedAtUtc,
		reminder.TriggeredAtUtc,
		reminder.CanceledAtUtc,
		reminder.AcknowledgedAt,
		reminder.Version,
	)
	return err
}

type scanRow interface {
	Scan(dest ...any) error
}

func scanReminder(row scanRow) (*domain.Reminder, error) {
	var (
		tenantID       string
		reminderID     string
		userID         string
		dueAt          sql.NullTime
		message        string
		statusRaw      string
		idempotencyKey string
		createdAt      sql.NullTime
		triggeredAt    sql.NullTime
		canceledAt     sql.NullTime
		acknowledgedAt sql.NullTime
		version        int64
	)

	if err := row.Scan(
		&tenantID,
		&reminderID,
		&userID,
		&dueAt,
		&message,
		&statusRaw,
		&idempotencyKey,
		&createdAt,
		&triggeredAt,
		&canceledAt,
		&acknowledgedAt,
		&version,
	); err != nil {
		return nil, err
	}

	if !dueAt.Valid || !createdAt.Valid {
		return nil, fmt.Errorf("required timestamp missing")
	}

	status, err := parseReminderStatus(statusRaw)
	if err != nil {
		return nil, err
	}

	reminder := &domain.Reminder{
		ID:             domain.ReminderID(reminderID),
		TenantID:       domain.TenantID(tenantID),
		UserID:         domain.UserID(userID),
		DueAtUtc:       dueAt.Time.UTC(),
		Message:        message,
		Status:         status,
		IdempotencyKey: idempotencyKey,
		CreatedAtUtc:   createdAt.Time.UTC(),
		Version:        version,
	}

	if triggeredAt.Valid {
		t := triggeredAt.Time.UTC()
		reminder.TriggeredAtUtc = &t
	}
	if canceledAt.Valid {
		t := canceledAt.Time.UTC()
		reminder.CanceledAtUtc = &t
	}
	if acknowledgedAt.Valid {
		t := acknowledgedAt.Time.UTC()
		reminder.AcknowledgedAt = &t
	}

	return reminder, nil
}

func parseReminderStatus(raw string) (domain.ReminderStatus, error) {
	status := domain.ReminderStatus(raw)
	switch status {
	case domain.ReminderStatusScheduled,
		domain.ReminderStatusTriggered,
		domain.ReminderStatusAcknowledged,
		domain.ReminderStatusCanceled:
		return status, nil
	default:
		return "", fmt.Errorf("unsupported reminder status: %s", raw)
	}
}
