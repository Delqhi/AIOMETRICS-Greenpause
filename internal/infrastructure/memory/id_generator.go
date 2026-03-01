package memory

import (
	"fmt"
	"sync/atomic"

	"greenpause/internal/domain"
)

type SequenceReminderIDGenerator struct {
	n atomic.Uint64
}

func NewSequenceReminderIDGenerator() *SequenceReminderIDGenerator {
	return &SequenceReminderIDGenerator{}
}

func (g *SequenceReminderIDGenerator) NewReminderID() domain.ReminderID {
	n := g.n.Add(1)
	return domain.ReminderID(fmt.Sprintf("rem-%020d", n))
}
