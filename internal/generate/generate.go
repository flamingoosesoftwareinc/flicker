package generate

import (
	"fmt"
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

// Fake provides deterministic ID and time generation for tests.
type Fake struct {
	counter int
	prefix  string
	tick    time.Time
}

// NewFake creates a Fake with a given ID prefix and base time.
// IDs are sequential: "prefix-001", "prefix-002", etc.
// Time advances by 1 second per call to Now().
func NewFake(prefix string, baseTime time.Time) *Fake {
	return &Fake{
		prefix: prefix,
		tick:   baseTime,
	}
}

// NewID returns the next sequential ID.
func (f *Fake) NewID() string {
	f.counter++

	return fmt.Sprintf("%s-%03d", f.prefix, f.counter)
}

// Now returns the current fake time, then advances by 1 second.
func (f *Fake) Now() time.Time {
	t := f.tick
	f.tick = f.tick.Add(time.Second)

	return t
}
