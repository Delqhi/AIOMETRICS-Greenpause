package memory

import (
	"context"
	"sync"
	"time"

	"greenpause/internal/domain"
)

type ScheduleIndex struct {
	mu    sync.RWMutex
	items map[string]time.Time
}

func NewScheduleIndex() *ScheduleIndex {
	return &ScheduleIndex{items: make(map[string]time.Time)}
}

func (s *ScheduleIndex) Upsert(_ context.Context, tenantID domain.TenantID, dueAtUtc time.Time, reminderID domain.ReminderID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[buildReminderMapKey(tenantID, reminderID)] = dueAtUtc.UTC()
	return nil
}

func (s *ScheduleIndex) Remove(_ context.Context, tenantID domain.TenantID, reminderID domain.ReminderID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, buildReminderMapKey(tenantID, reminderID))
	return nil
}

func (s *ScheduleIndex) DueAt(tenantID domain.TenantID, reminderID domain.ReminderID) (time.Time, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	dueAt, ok := s.items[buildReminderMapKey(tenantID, reminderID)]
	return dueAt, ok
}
