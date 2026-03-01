package postgres

import (
	"context"
	"database/sql"
)

func EnsureSchema(ctx context.Context, db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS reminder_records (
  tenant_id TEXT NOT NULL,
  reminder_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  due_at_utc TIMESTAMPTZ NOT NULL,
  message TEXT NOT NULL,
  status TEXT NOT NULL,
  idempotency_key TEXT NOT NULL,
  created_at_utc TIMESTAMPTZ NOT NULL,
  triggered_at_utc TIMESTAMPTZ NULL,
  canceled_at_utc TIMESTAMPTZ NULL,
  acknowledged_at_utc TIMESTAMPTZ NULL,
  version BIGINT NOT NULL,
  PRIMARY KEY (tenant_id, reminder_id),
  UNIQUE (tenant_id, idempotency_key)
);

CREATE TABLE IF NOT EXISTS audit_event_records (
  tenant_id TEXT NOT NULL,
  reminder_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  correlation_id TEXT NOT NULL,
  occurred_at_utc TIMESTAMPTZ NOT NULL
);
`
	_, err := db.ExecContext(ctx, schema)
	return err
}
