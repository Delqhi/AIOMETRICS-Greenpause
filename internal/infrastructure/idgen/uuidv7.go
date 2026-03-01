package idgen

import (
	"fmt"
	"time"

	"greenpause/internal/domain"
	"greenpause/pkg/timeutil"
)

type UUIDv7ReminderIDGenerator struct{}

func NewUUIDv7ReminderIDGenerator() UUIDv7ReminderIDGenerator {
	return UUIDv7ReminderIDGenerator{}
}

func (g UUIDv7ReminderIDGenerator) NewReminderID() domain.ReminderID {
	id, err := timeutil.NewUUIDv7String(time.Now().UTC())
	if err != nil {
		panic(fmt.Errorf("uuidv7 generation failed: %w", err))
	}
	return domain.ReminderID(id)
}
