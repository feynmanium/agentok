package edqlite

import (
	"testing"
)

func TestValidate_Completeness_OK(t *testing.T) {
	r := Rule{
		RuleID:    "r1",
		DatasetID: "ds1",
		RuleType:  RuleCompleteness,
		Field:     "name",
		Threshold: 1.0,
	}
	if err := r.Validate(); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidate_MissingRuleID(t *testing.T) {
	r := Rule{
		DatasetID: "ds1",
		RuleType:  RuleCompleteness,
		Field:     "name",
	}
	err := r.Validate()
	if err == nil {
		t.Fatal("expected error for missing rule_id")
	}
}

func TestValidate_MissingDatasetID(t *testing.T) {
	r := Rule{
		RuleID:   "r1",
		RuleType: RuleCompleteness,
		Field:    "name",
	}
	err := r.Validate()
	if err == nil {
		t.Fatal("expected error for missing dataset_id")
	}
}

func TestValidate_Validity_NoAllowedValues(t *testing.T) {
	r := Rule{
		RuleID:    "r1",
		DatasetID: "ds1",
		RuleType:  RuleValidity,
		Field:     "status",
	}
	err := r.Validate()
	if err == nil {
		t.Fatal("expected error for validity without allowed_values")
	}
}

func TestValidate_Range_NoBounds(t *testing.T) {
	r := Rule{
		RuleID:    "r1",
		DatasetID: "ds1",
		RuleType:  RuleRange,
		Field:     "score",
	}
	err := r.Validate()
	if err == nil {
		t.Fatal("expected error for range without min or max")
	}
}

func TestValidate_Timeliness_NoLatency(t *testing.T) {
	r := Rule{
		RuleID:    "r1",
		DatasetID: "ds1",
		RuleType:  RuleTimeliness,
		Field:     "ts",
		Params: RuleParams{
			MaxLatencySec: 0,
		},
	}
	err := r.Validate()
	if err == nil {
		t.Fatal("expected error for timeliness with 0 max_latency_sec")
	}
}

func TestValidate_Volume_NoBounds(t *testing.T) {
	r := Rule{
		RuleID:    "r1",
		DatasetID: "ds1",
		RuleType:  RuleVolume,
	}
	err := r.Validate()
	if err == nil {
		t.Fatal("expected error for volume without min_rows or max_rows")
	}
}

func TestValidate_Comparison_NoOp(t *testing.T) {
	r := Rule{
		RuleID:    "r1",
		DatasetID: "ds1",
		RuleType:  RuleComparison,
		Field:     "a",
		Params: RuleParams{
			CompareField: "b",
		},
	}
	err := r.Validate()
	if err == nil {
		t.Fatal("expected error for comparison without compare_op")
	}
}

func TestValidate_Comparison_NoFieldOrValue(t *testing.T) {
	r := Rule{
		RuleID:    "r1",
		DatasetID: "ds1",
		RuleType:  RuleComparison,
		Field:     "a",
		Params: RuleParams{
			CompareOp: CompareEqual,
			// Neither CompareField nor CompareValue set.
		},
	}
	err := r.Validate()
	if err == nil {
		t.Fatal("expected error for comparison without compare_field or compare_value")
	}
}

func TestValidate_UnknownType(t *testing.T) {
	r := Rule{
		RuleID:    "r1",
		DatasetID: "ds1",
		RuleType:  RuleType("magic"),
		Field:     "x",
	}
	err := r.Validate()
	if err == nil {
		t.Fatal("expected error for unknown rule_type")
	}
}
