package flicker

import (
	"errors"
	"fmt"
	"time"
)

// SuspendError signals that a workflow should be suspended until the given time.
// Return this from a workflow step to pause execution. The engine will set
// status=suspended and retry_after=ResumeAt without incrementing attempts.
type SuspendError struct {
	ResumeAt time.Time
}

func (e *SuspendError) Error() string {
	return fmt.Sprintf("workflow suspended until %s", e.ResumeAt.Format(time.RFC3339))
}

// IsSuspend checks whether an error is a SuspendError using errors.As.
func IsSuspend(err error) (*SuspendError, bool) {
	var se *SuspendError
	if errors.As(err, &se) {
		return se, true
	}

	return nil, false
}
