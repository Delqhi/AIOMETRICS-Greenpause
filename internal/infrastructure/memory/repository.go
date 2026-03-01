package memory

import (
	"context"
	"sync"

	"greenpause/internal/domain"
)

type ReminderRepository struct {
	mu             sync.RWMutex
	byID           map[string]*domain.Reminder
	idempotencyKey map[string]domain.ReminderID
}

func NewReminderRepository() *ReminderRepository {
	return &ReminderRepository{
		byID:           make(map[string]*domain.Reminder),
		idempotencyKey: make(map[string]domain.ReminderID),
	}
}

func (r *ReminderRepository) SaveIfIdempotencyKeyAbsent(_ context.Context, reminder *domain.Reminder) (bool, *domain.Reminder, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	idempotencyMapKey := buildIdempotencyMapKey(reminder.TenantID, reminder.IdempotencyKey)
	if existingID, exists := r.idempotencyKey[idempotencyMapKey]; exists {
		stored := r.byID[buildReminderMapKey(reminder.TenantID, existingID)]
		return false, stored.Clone(), nil
	}

	clone := reminder.Clone()
	r.byID[buildReminderMapKey(clone.TenantID, clone.ID)] = clone
	r.idempotencyKey[idempotencyMapKey] = clone.ID
	return true, nil, nil
}

func (r *ReminderRepository) GetByID(_ context.Context, tenantID domain.TenantID, reminderID domain.ReminderID) (*domain.Reminder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stored := r.byID[buildReminderMapKey(tenantID, reminderID)]
	if stored == nil {
		return nil, nil
	}
	return stored.Clone(), nil
}

func (r *ReminderRepository) Save(_ context.Context, reminder *domain.Reminder) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	clone := reminder.Clone()
	r.byID[buildReminderMapKey(clone.TenantID, clone.ID)] = clone
	return nil
}

func buildReminderMapKey(tenantID domain.TenantID, reminderID domain.ReminderID) string {
	return string(tenantID) + "::" + string(reminderID)
}

func buildIdempotencyMapKey(tenantID domain.TenantID, key string) string {
	return string(tenantID) + "::" + key
}
