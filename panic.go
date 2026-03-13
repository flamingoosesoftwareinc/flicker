package flicker

import (
	"fmt"
	"runtime"
)

// panicToError calls fn and converts any panic into an error with a stack trace.
func panicToError(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			err = fmt.Errorf("panic recovered: %v\n\n%s", r, buf[:n])
		}
	}()

	return fn()
}
