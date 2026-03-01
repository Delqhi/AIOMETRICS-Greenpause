package memory

import (
	"context"
	"sync"

	"greenpause/internal/application"
)

type AuditLog struct {
	mu     sync.RWMutex
	events []application.AuditEvent
}

func NewAuditLog() *AuditLog {
	return &AuditLog{}
}

func (a *AuditLog) Append(_ context.Context, event application.AuditEvent) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.events = append(a.events, event)
	return nil
}

func (a *AuditLog) Events() []application.AuditEvent {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]application.AuditEvent, len(a.events))
	copy(out, a.events)
	return out
}
