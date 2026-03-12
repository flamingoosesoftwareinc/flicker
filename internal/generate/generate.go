package generate

import (
	"time"

	"github.com/google/uuid"
)

// ID returns a new UUID v4 string.
func ID() string {
	return uuid.New().String()
}

// Now returns the current UTC time.
func Now() time.Time {
	return time.Now().UTC()
}
