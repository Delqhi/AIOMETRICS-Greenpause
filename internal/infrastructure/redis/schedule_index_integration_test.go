package redis

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"greenpause/internal/domain"
)

func TestScheduleIndex_Integration_UpsertAndRemove(t *testing.T) {
	addr := os.Getenv("TEST_REDIS_ADDR")
	if strings.TrimSpace(addr) == "" {
		t.Skip("TEST_REDIS_ADDR not set")
	}

	index := NewScheduleIndex(addr)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tenantID := domain.TenantID("tenant-it")
	reminderID := domain.ReminderID("rem-it-1")
	dueAt := time.Now().UTC().Add(3 * time.Minute).Round(time.Millisecond)

	if err := index.Upsert(ctx, tenantID, dueAt, reminderID); err != nil {
		t.Fatalf("upsert score: %v", err)
	}

	score, err := index.Score(ctx, tenantID, reminderID)
	if err != nil {
		t.Fatalf("read score: %v", err)
	}
	if score == nil {
		t.Fatalf("expected score to exist")
	}
	if int64(*score) != dueAt.UnixMilli() {
		t.Fatalf("unexpected score: got %d want %d", int64(*score), dueAt.UnixMilli())
	}

	if err := index.Remove(ctx, tenantID, reminderID); err != nil {
		t.Fatalf("remove score: %v", err)
	}

	score, err = index.Score(ctx, tenantID, reminderID)
	if err != nil {
		t.Fatalf("read score after remove: %v", err)
	}
	if score != nil {
		t.Fatalf("expected score to be removed")
	}
}
