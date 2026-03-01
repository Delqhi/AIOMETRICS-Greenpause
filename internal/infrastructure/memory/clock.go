package memory

import "time"

type FixedClock struct {
	Current time.Time
}

func (c FixedClock) Now() time.Time {
	return c.Current
}
