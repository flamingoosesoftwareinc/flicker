package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/flamingoosesoftwareinc/flicker"
)

func main() {
	config := LoadConfig()
	c := newContainer(config)

	logger := c.Logger()
	logger.Info("starting flicker fileops example",
		"work_dir", config.WorkDir,
		"archive_dir", config.ArchiveDir,
		"cycle_interval", config.CycleInterval,
	)

	// Ensure work/archive directories exist.
	os.MkdirAll(config.WorkDir, 0o755)
	os.MkdirAll(config.ArchiveDir, 0o755)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Start engine in background.
	engine := c.Engine()
	go func() {
		if err := engine.Start(ctx); err != nil {
			logger.Error("engine stopped", "error", err)
		}
	}()

	// Start HTTP server (healthz + metrics) in background.
	srv := c.Server()
	go func() {
		logger.Info("http server starting", "addr", config.ServerAddr)
		if err := srv.ListenAndServe(); err != nil {
			logger.Info("http server stopped", "error", err)
		}
	}()

	// Start event bridge (named pipe → SendEvent).
	startEventBridge(ctx, config.PipePath, engine, logger)

	// Ticker loop — submit a new workflow each cycle.
	factory := c.Factory()
	ticker := time.NewTicker(config.CycleInterval)
	defer ticker.Stop()

	cycleNum := 0
	for {
		cycleNum++
		submitCycle(ctx, factory, config, cycleNum, logger)

		select {
		case <-ctx.Done():
			logger.Info("shutting down")
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
			srv.Shutdown(shutdownCtx)
			c.Close(shutdownCtx)
			shutdownCancel()
			return
		case <-ticker.C:
		}
	}
}

func submitCycle(
	ctx context.Context,
	factory *flicker.Factory[CycleRequest, CycleReport],
	config *Config,
	cycleNum int,
	logger *slog.Logger,
) {
	instance, err := factory.Submit(ctx, CycleRequest{
		WorkDir:    config.WorkDir,
		ArchiveDir: config.ArchiveDir,
		CycleNum:   cycleNum,
	})
	if err != nil {
		logger.Error("failed to submit cycle", "cycle_num", cycleNum, "error", err)
		return
	}

	logger.Info("cycle submitted",
		"cycle_num", cycleNum,
		"workflow_id", instance.ID(),
	)

	// Poll for result in background (non-blocking).
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		timeout := time.After(2 * time.Minute)

		for {
			select {
			case <-timeout:
				logger.Warn("cycle timed out waiting for result", "cycle_num", cycleNum)
				return
			case <-ticker.C:
				result, err := instance.Result(ctx)
				if err != nil {
					continue
				}
				switch result.Status {
				case "completed":
					data, _ := json.Marshal(result.Response)
					logger.Info("cycle completed",
						"cycle_num", cycleNum,
						"report", string(data),
					)
					return
				case "failed":
					logger.Error("cycle failed",
						"cycle_num", cycleNum,
						"error", result.Error,
					)
					return
				}
			}
		}
	}()
}

func init() {
	// Print startup banner.
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║  flicker fileops example             ║")
	fmt.Println("║  exercises every workflow API         ║")
	fmt.Println("╚══════════════════════════════════════╝")
}
