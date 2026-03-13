package flicker

import (
	"context"
	"time"

	"github.com/flamingoosesoftwareinc/flicker/semconv"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/flamingoosesoftwareinc/flicker"

// noopSpan is a span that does nothing. Used when telemetry is nil.
var noopSpan = trace.SpanFromContext(context.Background())

// telemetry holds the tracer, meter, instruments, and semantic conventions.
// All methods are safe to call on a nil receiver.
type telemetry struct {
	tracer        trace.Tracer
	conv          *semconv.Conventions
	cacheHitSpans bool

	// Attribute keys (derived from conventions).
	attrWorkflowID      attribute.Key
	attrWorkflowType    attribute.Key
	attrWorkflowVersion attribute.Key
	attrWorkflowStatus  attribute.Key
	attrWorkflowAttempt attribute.Key
	attrStepName        attribute.Key
	attrStepCacheHit    attribute.Key

	// Instruments.
	workflowSubmitted metric.Int64Counter
	workflowCompleted metric.Int64Counter
	workflowDuration  metric.Float64Histogram
	workflowActive    metric.Int64UpDownCounter
	workflowSuspended metric.Int64UpDownCounter
	stepExecuted      metric.Int64Counter
	stepDuration      metric.Float64Histogram
}

func newTelemetry(
	tp trace.TracerProvider,
	mp metric.MeterProvider,
	conv *semconv.Conventions,
	cacheHitSpans bool,
) *telemetry {
	if tp == nil {
		tp = otel.GetTracerProvider()
	}
	if mp == nil {
		mp = otel.GetMeterProvider()
	}
	if conv == nil {
		conv = semconv.Default()
	}

	tracer := tp.Tracer(instrumentationName)
	meter := mp.Meter(instrumentationName)

	t := &telemetry{
		tracer:        tracer,
		conv:          conv,
		cacheHitSpans: cacheHitSpans,

		attrWorkflowID:      attribute.Key(conv.WorkflowID),
		attrWorkflowType:    attribute.Key(conv.WorkflowType),
		attrWorkflowVersion: attribute.Key(conv.WorkflowVersion),
		attrWorkflowStatus:  attribute.Key(conv.WorkflowStatus),
		attrWorkflowAttempt: attribute.Key(conv.WorkflowAttempt),
		attrStepName:        attribute.Key(conv.StepName),
		attrStepCacheHit:    attribute.Key(conv.StepCacheHit),
	}

	// Create instruments. Per OTel convention, creation errors are non-fatal
	// — instruments fall back to noop on error.
	t.workflowSubmitted, _ = meter.Int64Counter(conv.MetricWorkflowSubmitted,
		metric.WithDescription("Number of workflows submitted"),
		metric.WithUnit("{workflow}"),
	)
	t.workflowCompleted, _ = meter.Int64Counter(conv.MetricWorkflowCompleted,
		metric.WithDescription("Number of workflows reaching terminal state"),
		metric.WithUnit("{workflow}"),
	)
	t.workflowDuration, _ = meter.Float64Histogram(conv.MetricWorkflowDuration,
		metric.WithDescription("Duration of a single workflow execution attempt"),
		metric.WithUnit("s"),
	)
	t.workflowActive, _ = meter.Int64UpDownCounter(conv.MetricWorkflowActive,
		metric.WithDescription("Number of currently executing workflows"),
		metric.WithUnit("{workflow}"),
	)
	t.workflowSuspended, _ = meter.Int64UpDownCounter(conv.MetricWorkflowSuspended,
		metric.WithDescription("Number of currently suspended workflows"),
		metric.WithUnit("{workflow}"),
	)
	t.stepExecuted, _ = meter.Int64Counter(conv.MetricStepExecuted,
		metric.WithDescription("Number of steps executed"),
		metric.WithUnit("{step}"),
	)
	t.stepDuration, _ = meter.Float64Histogram(conv.MetricStepDuration,
		metric.WithDescription("Duration of step execution (cache misses only)"),
		metric.WithUnit("s"),
	)

	return t
}

// --- Span helpers ---

func (t *telemetry) startWorkflowSpan(
	ctx context.Context,
	record *WorkflowRecord,
) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, t.conv.SpanWorkflowExecute,
		trace.WithAttributes(
			t.attrWorkflowID.String(record.ID),
			t.attrWorkflowType.String(record.Type),
			t.attrWorkflowVersion.String(record.Version),
			t.attrWorkflowAttempt.Int(record.Attempts),
		),
	)
}

func (t *telemetry) startStepSpan(
	ctx context.Context,
	stepName string,
	cacheHit bool,
) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, t.conv.SpanStep,
		trace.WithAttributes(
			t.attrStepName.String(stepName),
			t.attrStepCacheHit.Bool(cacheHit),
		),
	)
}

func (t *telemetry) startBranchSpan(
	ctx context.Context,
	branchName string,
) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, t.conv.SpanParallelBranch,
		trace.WithAttributes(
			t.attrStepName.String(branchName),
		),
	)
}

func (t *telemetry) endSpanWithError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}

// --- Metric helpers ---

func (t *telemetry) wfTypeAttrs(wfType, version string) metric.MeasurementOption {
	return metric.WithAttributes(
		t.attrWorkflowType.String(wfType),
		t.attrWorkflowVersion.String(version),
	)
}

func (t *telemetry) recordSubmitted(ctx context.Context, wfType, version string) {
	t.workflowSubmitted.Add(ctx, 1, t.wfTypeAttrs(wfType, version))
}

func (t *telemetry) recordCompleted(ctx context.Context, wfType, version string, status Status) {
	t.workflowCompleted.Add(ctx, 1,
		metric.WithAttributes(
			t.attrWorkflowType.String(wfType),
			t.attrWorkflowVersion.String(version),
			t.attrWorkflowStatus.String(string(status)),
		),
	)
}

func (t *telemetry) recordDuration(ctx context.Context, wfType, version string, d time.Duration) {
	t.workflowDuration.Record(ctx, d.Seconds(), t.wfTypeAttrs(wfType, version))
}

func (t *telemetry) adjustActive(ctx context.Context, delta int64, wfType, version string) {
	t.workflowActive.Add(ctx, delta, t.wfTypeAttrs(wfType, version))
}

func (t *telemetry) adjustSuspended(ctx context.Context, delta int64) {
	t.workflowSuspended.Add(ctx, delta)
}

func (t *telemetry) recordStepExecuted(ctx context.Context, stepName string, cacheHit bool) {
	t.stepExecuted.Add(ctx, 1,
		metric.WithAttributes(
			t.attrStepName.String(stepName),
			t.attrStepCacheHit.Bool(cacheHit),
		),
	)
}

func (t *telemetry) recordStepDuration(ctx context.Context, stepName string, d time.Duration) {
	t.stepDuration.Record(ctx, d.Seconds(),
		metric.WithAttributes(
			t.attrStepName.String(stepName),
		),
	)
}
