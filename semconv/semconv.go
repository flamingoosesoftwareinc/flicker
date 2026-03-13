package semconv

// Option configures semantic conventions.
type Option func(*config)

type config struct {
	prefix string
}

// WithPrefix sets the namespace prefix for all attribute keys and metric names.
// Default is "flicker".
func WithPrefix(prefix string) Option {
	return func(c *config) {
		c.prefix = prefix
	}
}

// Conventions holds semantic convention names for Flicker telemetry.
// All names follow OpenTelemetry's {namespace}.{entity}.{attribute} pattern.
type Conventions struct {
	// Attribute keys.
	WorkflowID      string
	WorkflowType    string
	WorkflowVersion string
	WorkflowStatus  string
	WorkflowAttempt string
	StepName        string
	StepCacheHit    string

	// Span names.
	SpanWorkflowExecute string
	SpanStep            string
	SpanParallelBranch  string

	// Metric names.
	MetricWorkflowSubmitted string
	MetricWorkflowCompleted string
	MetricWorkflowDuration  string
	MetricWorkflowActive    string
	MetricWorkflowSuspended string
	MetricStepExecuted      string
	MetricStepDuration      string
}

// New creates Conventions with the given options.
func New(opts ...Option) *Conventions {
	cfg := &config{prefix: "flicker"}
	for _, opt := range opts {
		opt(cfg)
	}

	p := cfg.prefix

	return &Conventions{
		WorkflowID:      p + ".workflow.id",
		WorkflowType:    p + ".workflow.type",
		WorkflowVersion: p + ".workflow.version",
		WorkflowStatus:  p + ".workflow.status",
		WorkflowAttempt: p + ".workflow.attempt",
		StepName:        p + ".step.name",
		StepCacheHit:    p + ".step.cache_hit",

		SpanWorkflowExecute: p + ".workflow.execute",
		SpanStep:            p + ".workflow.step",
		SpanParallelBranch:  p + ".workflow.branch",

		MetricWorkflowSubmitted: p + ".workflow.submitted",
		MetricWorkflowCompleted: p + ".workflow.completed",
		MetricWorkflowDuration:  p + ".workflow.duration",
		MetricWorkflowActive:    p + ".workflow.active",
		MetricWorkflowSuspended: p + ".workflow.suspended",
		MetricStepExecuted:      p + ".step.executed",
		MetricStepDuration:      p + ".step.duration",
	}
}

// Default returns conventions with the default "flicker" prefix.
func Default() *Conventions {
	return New()
}
