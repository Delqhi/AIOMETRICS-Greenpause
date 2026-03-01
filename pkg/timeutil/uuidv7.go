package timeutil

import (
	"crypto/rand"
	"fmt"
	"time"
)

func NewUUIDv7String(now time.Time) (string, error) {
	tsMillis := uint64(now.UTC().UnixMilli())

	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}

	b[0] = byte(tsMillis >> 40)
	b[1] = byte(tsMillis >> 32)
	b[2] = byte(tsMillis >> 24)
	b[3] = byte(tsMillis >> 16)
	b[4] = byte(tsMillis >> 8)
	b[5] = byte(tsMillis)

	b[6] = (b[6] & 0x0F) | 0x70
	b[8] = (b[8] & 0x3F) | 0x80

	return fmt.Sprintf(
		"%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		b[0], b[1], b[2], b[3],
		b[4], b[5],
		b[6], b[7],
		b[8], b[9],
		b[10], b[11], b[12], b[13], b[14], b[15],
	), nil
}
