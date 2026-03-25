package edqlite

import "fmt"

// RuleType enumerates the core data-quality check categories.
type RuleType string

const (
	RuleCompleteness RuleType = "completeness"
	RuleValidity     RuleType = "validity"
	RuleRange        RuleType = "range"
	RuleUniqueness   RuleType = "uniqueness"
	RuleTimeliness   RuleType = "timeliness"
	RuleVolume       RuleType = "volume"
	RuleComparison   RuleType = "comparison"
)

// Severity indicates the impact level of a rule failure.
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// ComparisonOp defines comparison operators for comparison rules.
type ComparisonOp string

const (
	CompareEqual        ComparisonOp = "eq"
	CompareNotEqual     ComparisonOp = "ne"
	CompareGreaterThan  ComparisonOp = "gt"
	CompareLessThan     ComparisonOp = "lt"
	CompareGreaterEqual ComparisonOp = "gte"
	CompareLessEqual    ComparisonOp = "lte"
)

// Rule is the canonical rule specification per Section 4 of the EDQ Lite spec.
type Rule struct {
	RuleID    string   `json:"rule_id" yaml:"rule_id"`
	DatasetID string   `json:"dataset_id" yaml:"dataset_id"`
	RuleType  RuleType `json:"rule_type" yaml:"rule_type"`
	Entity    string   `json:"entity" yaml:"entity"`
	Field     string   `json:"field" yaml:"field"`
	Severity  Severity `json:"severity" yaml:"severity"`
	Owner     string   `json:"owner" yaml:"owner"`
	Tags      []string `json:"tags,omitempty" yaml:"tags,omitempty"`

	// Threshold is the pass threshold (e.g., 0.995 means 99.5% must pass).
	Threshold float64 `json:"threshold" yaml:"threshold"`

	// Params holds rule-type-specific parameters.
	Params RuleParams `json:"params" yaml:"params"`

	// Execution hints
	BatchOnly          bool   `json:"batch_only,omitempty" yaml:"batch_only,omitempty"`
	RTEnabled          bool   `json:"rt_enabled,omitempty" yaml:"rt_enabled,omitempty"`
	ComputeProfileHint string `json:"compute_profile_hint,omitempty" yaml:"compute_profile_hint,omitempty"`

	// WaiverPolicy describes conditions under which failures can be waived.
	WaiverPolicy string `json:"waiver_policy,omitempty" yaml:"waiver_policy,omitempty"`
}

// RuleParams contains type-specific parameters for each rule type.
type RuleParams struct {
	// Validity: allowed values
	AllowedValues []string `json:"allowed_values,omitempty" yaml:"allowed_values,omitempty"`

	// Range: min/max bounds
	Min *float64 `json:"min,omitempty" yaml:"min,omitempty"`
	Max *float64 `json:"max,omitempty" yaml:"max,omitempty"`

	// Timeliness: max latency
	MaxLatencySec int `json:"max_latency_sec,omitempty" yaml:"max_latency_sec,omitempty"`

	// Volume: expected row count bands
	MinRows *int `json:"min_rows,omitempty" yaml:"min_rows,omitempty"`
	MaxRows *int `json:"max_rows,omitempty" yaml:"max_rows,omitempty"`

	// Comparison: cross-field or cross-record checks
	CompareField string       `json:"compare_field,omitempty" yaml:"compare_field,omitempty"`
	CompareOp    ComparisonOp `json:"compare_op,omitempty" yaml:"compare_op,omitempty"`
	CompareValue any          `json:"compare_value,omitempty" yaml:"compare_value,omitempty"`

	// TimestampField for timeliness checks (defaults to Field if empty)
	TimestampField string `json:"timestamp_field,omitempty" yaml:"timestamp_field,omitempty"`
}

// Validate checks that a rule has required fields and consistent parameters.
func (r *Rule) Validate() error {
	if r.RuleID == "" {
		return fmt.Errorf("rule: rule_id is required")
	}
	if r.DatasetID == "" {
		return fmt.Errorf("rule %s: dataset_id is required", r.RuleID)
	}
	if r.RuleType == "" {
		return fmt.Errorf("rule %s: rule_type is required", r.RuleID)
	}

	switch r.RuleType {
	case RuleCompleteness:
		if r.Field == "" {
			return fmt.Errorf("rule %s: field is required for completeness", r.RuleID)
		}
	case RuleValidity:
		if r.Field == "" {
			return fmt.Errorf("rule %s: field is required for validity", r.RuleID)
		}
		if len(r.Params.AllowedValues) == 0 {
			return fmt.Errorf("rule %s: allowed_values required for validity", r.RuleID)
		}
	case RuleRange:
		if r.Field == "" {
			return fmt.Errorf("rule %s: field is required for range", r.RuleID)
		}
		if r.Params.Min == nil && r.Params.Max == nil {
			return fmt.Errorf("rule %s: min and/or max required for range", r.RuleID)
		}
	case RuleUniqueness:
		if r.Field == "" {
			return fmt.Errorf("rule %s: field is required for uniqueness", r.RuleID)
		}
	case RuleTimeliness:
		if r.Params.MaxLatencySec <= 0 {
			return fmt.Errorf("rule %s: max_latency_sec must be > 0 for timeliness", r.RuleID)
		}
	case RuleVolume:
		if r.Params.MinRows == nil && r.Params.MaxRows == nil {
			return fmt.Errorf("rule %s: min_rows and/or max_rows required for volume", r.RuleID)
		}
	case RuleComparison:
		if r.Field == "" {
			return fmt.Errorf("rule %s: field is required for comparison", r.RuleID)
		}
		if r.Params.CompareOp == "" {
			return fmt.Errorf("rule %s: compare_op is required for comparison", r.RuleID)
		}
		if r.Params.CompareField == "" && r.Params.CompareValue == nil {
			return fmt.Errorf("rule %s: compare_field or compare_value required for comparison", r.RuleID)
		}
	default:
		return fmt.Errorf("rule %s: unknown rule_type %q", r.RuleID, r.RuleType)
	}
	return nil
}
