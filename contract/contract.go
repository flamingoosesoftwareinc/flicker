package contract

// Contract describes a workflow's public surface: its name, version,
// request/response types, steps, and providers. Extracted via static
// analysis of Go source code — no runtime dependency on flicker.
type Contract struct {
	Name         string     `json:"name"`
	Version      string     `json:"version"`
	RequestType  TypeShape  `json:"request_type"`
	ResponseType TypeShape  `json:"response_type"`
	Steps        []Step     `json:"steps"`
	Providers    []Provider `json:"providers,omitempty"`
	Errors       []string   `json:"errors,omitempty"`
}

// TypeShape describes a Go type's structure recursively.
type TypeShape struct {
	Name   string       `json:"name"`
	Pkg    string       `json:"pkg,omitempty"`
	Kind   string       `json:"kind"`
	Fields []FieldShape `json:"fields,omitempty"`
	Elem   *TypeShape   `json:"elem,omitempty"`
	Key    *TypeShape   `json:"key,omitempty"`
}

// FieldShape describes a single struct field.
type FieldShape struct {
	Name    string    `json:"name"`
	Type    TypeShape `json:"type"`
	JSONTag string    `json:"json_tag,omitempty"`
}

// StepKind classifies how a step name was resolved.
type StepKind string

const (
	StepKindLiteral  StepKind = "literal"
	StepKindConstant StepKind = "constant"
	StepKindProvider StepKind = "provider"
	StepKindDynamic  StepKind = "dynamic"
)

// Step describes a single workflow step extracted from the Execute method.
type Step struct {
	Name     string     `json:"name"`
	Kind     StepKind   `json:"kind"`
	Type     *TypeShape `json:"type,omitempty"`
	StepType string     `json:"step_type"`
	Branches []Branch   `json:"branches,omitempty"`
}

// Branch describes a named parallel branch containing nested steps.
type Branch struct {
	Name  string `json:"name"`
	Steps []Step `json:"steps"`
}

// Provider describes a durable provider registered in a workflow.
type Provider struct {
	Prefix string    `json:"prefix"`
	Type   TypeShape `json:"type"`
}
