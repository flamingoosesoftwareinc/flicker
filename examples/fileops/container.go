package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/flamingoosesoftwareinc/flicker"
	"github.com/flamingoosesoftwareinc/flicker/sqlite"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

type Config struct {
	WorkDir       string
	ArchiveDir    string
	PipePath      string
	CycleInterval time.Duration
	ServerAddr    string
	OTLPEndpoint  string
	LogLevel      slog.Level
}

func LoadConfig() *Config {
	return &Config{
		WorkDir:    getEnvOrDefault("WORK_DIR", "/data/work"),
		ArchiveDir: getEnvOrDefault("ARCHIVE_DIR", "/data/archive"),
		PipePath:   getEnvOrDefault("PIPE_PATH", "/data/events.pipe"),
		CycleInterval: parseDurationOrDefault(
			getEnvOrDefault("CYCLE_INTERVAL", "30s"),
			30*time.Second,
		),
		ServerAddr:   getEnvOrDefault("SERVER_ADDR", ":9001"),
		OTLPEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		LogLevel:     parseLogLevel(getEnvOrDefault("LOG_LEVEL", "info")),
	}
}

type container struct {
	config *Config

	state struct {
		logger         *slog.Logger
		metricsHandler http.Handler
		shutdownOTEL   func(context.Context) error
		store          *sqlite.Store
		engine         *flicker.Engine
		factory        *flicker.Factory[CycleRequest, CycleReport]
		mux            *http.ServeMux
		server         *http.Server
	}

	once struct {
		otel, logger, store, engine, mux, server sync.Once
	}
}

func newContainer(config *Config) *container {
	return &container{config: config}
}

func (c *container) OTEL() (http.Handler, error) {
	var err error
	c.once.otel.Do(func() {
		ctx := context.Background()

		res, resErr := resource.New(ctx,
			resource.WithAttributes(
				semconv.ServiceName("flicker-fileops"),
			),
		)
		if resErr != nil {
			err = resErr
			return
		}

		// Prometheus metrics exporter (always enabled).
		promExporter, promErr := prometheus.New()
		if promErr != nil {
			err = promErr
			return
		}
		mp := sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
			sdkmetric.WithReader(promExporter),
		)
		otel.SetMeterProvider(mp)
		c.state.metricsHandler = promhttp.Handler()

		shutdowns := []func(context.Context) error{mp.Shutdown}

		// OTLP trace exporter (conditional on endpoint).
		if c.config.OTLPEndpoint != "" {
			traceExp, traceErr := otlptracehttp.New(ctx,
				otlptracehttp.WithEndpoint(trimScheme(c.config.OTLPEndpoint)),
				otlptracehttp.WithInsecure(),
			)
			if traceErr != nil {
				err = traceErr
				return
			}
			tp := sdktrace.NewTracerProvider(
				sdktrace.WithResource(res),
				sdktrace.WithBatcher(traceExp),
			)
			otel.SetTracerProvider(tp)
			otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{},
				propagation.Baggage{},
			))
			shutdowns = append(shutdowns, tp.Shutdown)
		}

		c.state.shutdownOTEL = func(ctx context.Context) error {
			var errs []error
			for _, fn := range shutdowns {
				if e := fn(ctx); e != nil {
					errs = append(errs, e)
				}
			}
			if len(errs) > 0 {
				return fmt.Errorf("otel shutdown: %v", errs)
			}
			return nil
		}
	})
	return c.state.metricsHandler, err
}

func (c *container) Logger() *slog.Logger {
	c.once.logger.Do(func() {
		if _, err := c.OTEL(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to initialize otel: %v\n", err)
			os.Exit(1)
		}
		c.state.logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: c.config.LogLevel,
		}))
		slog.SetDefault(c.state.logger)
	})
	return c.state.logger
}

func (c *container) Store() *sqlite.Store {
	c.once.store.Do(func() {
		store, err := sqlite.NewStore(context.Background(), "file::memory:?cache=shared")
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to create store: %v\n", err)
			os.Exit(1)
		}
		c.state.store = store
	})
	return c.state.store
}

func (c *container) Engine() *flicker.Engine {
	c.once.engine.Do(func() {
		eng := flicker.NewEngine(c.Store(),
			flicker.WithWorkers(4),
			flicker.WithLogger(c.Logger()),
			flicker.WithDrainTimeout(10*time.Second),
		)
		c.state.factory = FileWranglerDef.Register(eng)
		c.state.engine = eng
	})
	return c.state.engine
}

func (c *container) Factory() *flicker.Factory[CycleRequest, CycleReport] {
	c.Engine() // ensure engine + factory are initialized
	return c.state.factory
}

func (c *container) Mux() *http.ServeMux {
	c.once.mux.Do(func() {
		mux := http.NewServeMux()

		mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"ok"}`))
		})

		if handler, err := c.OTEL(); err == nil && handler != nil {
			mux.Handle("GET /metrics", handler)
		}

		c.state.mux = mux
	})
	return c.state.mux
}

func (c *container) Server() *http.Server {
	c.once.server.Do(func() {
		c.state.server = &http.Server{
			Addr:         c.config.ServerAddr,
			Handler:      c.Mux(),
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		}
	})
	return c.state.server
}

func (c *container) Close(ctx context.Context) {
	if c.state.shutdownOTEL != nil {
		if err := c.state.shutdownOTEL(ctx); err != nil {
			c.state.logger.Error("failed to shutdown otel", "error", err)
		}
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func parseDurationOrDefault(val string, defaultVal time.Duration) time.Duration {
	d, err := time.ParseDuration(val)
	if err != nil {
		return defaultVal
	}
	return d
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func trimScheme(endpoint string) string {
	for _, prefix := range []string{"http://", "https://"} {
		if len(endpoint) > len(prefix) && endpoint[:len(prefix)] == prefix {
			return endpoint[len(prefix):]
		}
	}
	return endpoint
}
