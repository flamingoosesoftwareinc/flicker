package main

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"syscall"

	"github.com/flamingoosesoftwareinc/flicker"
)

// startEventBridge creates a named pipe and reads from it in a loop. Each
// line written to the pipe is treated as a cycle number — the bridge sends
// a cleanup event to the engine for that cycle.
//
// Usage from inside the container:
//
//	echo "3" > /data/events.pipe
//
// This delivers a CleanupSignal to the workflow waiting on "cleanup:3".
func startEventBridge(
	ctx context.Context,
	pipePath string,
	engine *flicker.Engine,
	logger *slog.Logger,
) {
	// Create FIFO if it doesn't exist.
	_ = os.Remove(pipePath)
	if err := syscall.Mkfifo(pipePath, 0o666); err != nil {
		logger.Error("failed to create named pipe", "path", pipePath, "error", err)
		return
	}

	logger.Info("event bridge started", "pipe", pipePath)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Open blocks until a writer connects.
			f, err := os.Open(pipePath)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				logger.Error("failed to open pipe", "error", err)
				continue
			}

			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" {
					continue
				}

				correlationKey := fmt.Sprintf("cleanup:%s", line)
				payload := CleanupSignal{Reason: fmt.Sprintf("manual trigger for cycle %s", line)}

				if err := engine.SendEvent(ctx, correlationKey, payload); err != nil {
					logger.Warn("event delivery failed",
						"correlation_key", correlationKey,
						"error", err,
					)
				} else {
					logger.Info("event delivered",
						"correlation_key", correlationKey,
					)
				}
			}

			f.Close()
		}
	}()
}
