package edqlite

import (
	"math"
	"testing"
	"time"
)

func TestEvalCompleteness(t *testing.T) {
	eng := NewEngine("test-v1")
	rule := Rule{
		RuleID:    "r-comp-1",
		DatasetID: "ds1",
		RuleType:  RuleCompleteness,
		Field:     "email",
		Severity:  SeverityHigh,
		Threshold: 0.8,
	}
	data := Dataset{
		Record{"email": "a@b.com"},
		Record{"email": nil},
		Record{"email": "c@d.com"},
		Record{"name": "no-email-field"},
		Record{"email": "e@f.com"},
	}

	res := eng.evaluateRule(rule, data)

	// 3 pass, 2 fail => pass_rate = 0.6
	if res.Status != StatusFail {
		t.Errorf("expected fail, got %s", res.Status)
	}
	wantRate := 0.6
	if math.Abs(res.PassRate-wantRate) > 1e-9 {
		t.Errorf("pass_rate: want %f, got %f", wantRate, res.PassRate)
	}
	if res.Failed != 2 {
		t.Errorf("failed: want 2, got %d", res.Failed)
	}
	if len(res.SampleViolations) != 2 {
		t.Errorf("violation samples: want 2, got %d", len(res.SampleViolations))
	}
	if res.ObservedStats.NullRate == nil {
		t.Fatal("expected non-nil NullRate")
	}
	wantNull := 0.4
	if math.Abs(*res.ObservedStats.NullRate-wantNull) > 1e-9 {
		t.Errorf("NullRate: want %f, got %f", wantNull, *res.ObservedStats.NullRate)
	}
}

func TestEvalCompleteness_AllPass(t *testing.T) {
	eng := NewEngine("test-v1")
	rule := Rule{
		RuleID:    "r-comp-2",
		DatasetID: "ds1",
		RuleType:  RuleCompleteness,
		Field:     "name",
		Severity:  SeverityLow,
		Threshold: 1.0,
	}
	data := Dataset{
		Record{"name": "Alice"},
		Record{"name": "Bob"},
	}

	res := eng.evaluateRule(rule, data)

	if res.Status != StatusPass {
		t.Errorf("expected pass, got %s", res.Status)
	}
	if res.PassRate != 1.0 {
		t.Errorf("pass_rate: want 1.0, got %f", res.PassRate)
	}
	if res.Failed != 0 {
		t.Errorf("failed: want 0, got %d", res.Failed)
	}
	if len(res.SampleViolations) != 0 {
		t.Errorf("violation samples: want 0, got %d", len(res.SampleViolations))
	}
}

func TestEvalValidity(t *testing.T) {
	eng := NewEngine("test-v1")
	rule := Rule{
		RuleID:    "r-val-1",
		DatasetID: "ds1",
		RuleType:  RuleValidity,
		Field:     "status",
		Severity:  SeverityMedium,
		Threshold: 0.5,
		Params: RuleParams{
			AllowedValues: []string{"active", "inactive"},
		},
	}
	data := Dataset{
		Record{"status": "active"},
		Record{"status": "inactive"},
		Record{"status": "unknown"},
		Record{"status": nil},
	}

	res := eng.evaluateRule(rule, data)

	// 2 pass, 2 fail (unknown + nil) => pass_rate = 0.5
	if res.PassRate != 0.5 {
		t.Errorf("pass_rate: want 0.5, got %f", res.PassRate)
	}
	if res.Status != StatusPass {
		t.Errorf("expected pass (0.5 >= 0.5 threshold), got %s", res.Status)
	}
	if res.Failed != 2 {
		t.Errorf("failed: want 2, got %d", res.Failed)
	}
	if len(res.SampleViolations) != 2 {
		t.Errorf("violation samples: want 2, got %d", len(res.SampleViolations))
	}
}

func TestEvalRange(t *testing.T) {
	eng := NewEngine("test-v1")
	rule := Rule{
		RuleID:    "r-range-1",
		DatasetID: "ds1",
		RuleType:  RuleRange,
		Field:     "score",
		Severity:  SeverityHigh,
		Threshold: 0.5,
		Params: RuleParams{
			Min: floatPtr(0),
			Max: floatPtr(100),
		},
	}
	data := Dataset{
		Record{"score": 50.0},
		Record{"score": 0.0},
		Record{"score": 100.0},
		Record{"score": -1.0},
		Record{"score": 101.0},
	}

	res := eng.evaluateRule(rule, data)

	// 3 pass, 2 fail => pass_rate = 0.6
	if res.Failed != 2 {
		t.Errorf("failed: want 2, got %d", res.Failed)
	}
	wantRate := 0.6
	if math.Abs(res.PassRate-wantRate) > 1e-9 {
		t.Errorf("pass_rate: want %f, got %f", wantRate, res.PassRate)
	}
	if res.Status != StatusPass {
		t.Errorf("expected pass (0.6 >= 0.5), got %s", res.Status)
	}

	// Check observed stats.
	if res.ObservedStats.Min == nil || *res.ObservedStats.Min != -1.0 {
		t.Errorf("observed min: want -1.0, got %v", res.ObservedStats.Min)
	}
	if res.ObservedStats.Max == nil || *res.ObservedStats.Max != 101.0 {
		t.Errorf("observed max: want 101.0, got %v", res.ObservedStats.Max)
	}
	wantMean := 50.0
	if res.ObservedStats.Mean == nil || math.Abs(*res.ObservedStats.Mean-wantMean) > 1e-9 {
		t.Errorf("observed mean: want %f, got %v", wantMean, res.ObservedStats.Mean)
	}
}

func TestEvalRange_MinOnly(t *testing.T) {
	eng := NewEngine("test-v1")
	rule := Rule{
		RuleID:    "r-range-min",
		DatasetID: "ds1",
		RuleType:  RuleRange,
		Field:     "age",
		Severity:  SeverityMedium,
		Threshold: 1.0,
		Params: RuleParams{
			Min: floatPtr(18),
		},
	}
	data := Dataset{
		Record{"age": 20.0},
		Record{"age": 18.0},
		Record{"age": 17.0},
	}

	res := eng.evaluateRule(rule, data)

	if res.Failed != 1 {
		t.Errorf("failed: want 1, got %d", res.Failed)
	}
	if res.Status != StatusFail {
		t.Errorf("expected fail, got %s", res.Status)
	}
	// Ensure no max-related violation reason appears for the value 20.
	for _, v := range res.SampleViolations {
		if v.Value == 20.0 {
			t.Errorf("unexpected violation for value 20 with min-only rule")
		}
	}
}

func TestEvalUniqueness(t *testing.T) {
	eng := NewEngine("test-v1")
	rule := Rule{
		RuleID:    "r-uniq-1",
		DatasetID: "ds1",
		RuleType:  RuleUniqueness,
		Field:     "id",
		Severity:  SeverityCritical,
		Threshold: 1.0,
	}
	data := Dataset{
		Record{"id": "a"},
		Record{"id": "b"},
		Record{"id": "a"},
		Record{"id": "c"},
		Record{"id": "b"},
	}

	res := eng.evaluateRule(rule, data)

	// 2 duplicates (second "a" + second "b") => failed=2
	if res.Failed != 2 {
		t.Errorf("failed: want 2, got %d", res.Failed)
	}
	if res.Status != StatusFail {
		t.Errorf("expected fail, got %s", res.Status)
	}
	if res.ObservedStats.UniqueN == nil || *res.ObservedStats.UniqueN != 3 {
		t.Errorf("unique_count: want 3, got %v", res.ObservedStats.UniqueN)
	}
}

func TestEvalUniqueness_AllUnique(t *testing.T) {
	eng := NewEngine("test-v1")
	rule := Rule{
		RuleID:    "r-uniq-2",
		DatasetID: "ds1",
		RuleType:  RuleUniqueness,
		Field:     "id",
		Severity:  SeverityLow,
		Threshold: 1.0,
	}
	data := Dataset{
		Record{"id": "x"},
		Record{"id": "y"},
		Record{"id": "z"},
	}

	res := eng.evaluateRule(rule, data)

	if res.Failed != 0 {
		t.Errorf("failed: want 0, got %d", res.Failed)
	}
	if res.Status != StatusPass {
		t.Errorf("expected pass, got %s", res.Status)
	}
	if res.PassRate != 1.0 {
		t.Errorf("pass_rate: want 1.0, got %f", res.PassRate)
	}
	if res.ObservedStats.UniqueN == nil || *res.ObservedStats.UniqueN != 3 {
		t.Errorf("unique_count: want 3, got %v", res.ObservedStats.UniqueN)
	}
}

func TestEvalTimeliness(t *testing.T) {
	eng := NewEngine("test-v1")
	rule := Rule{
		RuleID:    "r-time-1",
		DatasetID: "ds1",
		RuleType:  RuleTimeliness,
		Field:     "updated_at",
		Severity:  SeverityHigh,
		Threshold: 0.5,
		Params: RuleParams{
			MaxLatencySec: 60,
		},
	}
	now := time.Now()
	data := Dataset{
		Record{"updated_at": now.Add(-10 * time.Second)},  // recent, pass
		Record{"updated_at": now.Add(-30 * time.Second)},  // recent, pass
		Record{"updated_at": now.Add(-120 * time.Second)}, // stale, fail
		Record{"updated_at": now.Add(-300 * time.Second)}, // stale, fail
	}

	res := eng.evaluateRule(rule, data)

	if res.Failed != 2 {
		t.Errorf("failed: want 2, got %d", res.Failed)
	}
	wantRate := 0.5
	if math.Abs(res.PassRate-wantRate) > 1e-9 {
		t.Errorf("pass_rate: want %f, got %f", wantRate, res.PassRate)
	}
	if res.Status != StatusPass {
		t.Errorf("expected pass (0.5 >= 0.5), got %s", res.Status)
	}
}

func TestEvalVolume_Pass(t *testing.T) {
	eng := NewEngine("test-v1")
	rule := Rule{
		RuleID:    "r-vol-1",
		DatasetID: "ds1",
		RuleType:  RuleVolume,
		Severity:  SeverityMedium,
		Threshold: 1.0,
		Params: RuleParams{
			MinRows: intPtr(2),
			MaxRows: intPtr(10),
		},
	}
	data := Dataset{
		Record{"a": 1},
		Record{"a": 2},
		Record{"a": 3},
	}

	res := eng.evaluateRule(rule, data)

	if res.Status != StatusPass {
		t.Errorf("expected pass, got %s", res.Status)
	}
	if res.PassRate != 1.0 {
		t.Errorf("pass_rate: want 1.0, got %f", res.PassRate)
	}
}

func TestEvalVolume_TooFew(t *testing.T) {
	eng := NewEngine("test-v1")
	rule := Rule{
		RuleID:    "r-vol-low",
		DatasetID: "ds1",
		RuleType:  RuleVolume,
		Severity:  SeverityHigh,
		Threshold: 1.0,
		Params: RuleParams{
			MinRows: intPtr(5),
		},
	}
	data := Dataset{
		Record{"a": 1},
		Record{"a": 2},
	}

	res := eng.evaluateRule(rule, data)

	if res.Status != StatusFail {
		t.Errorf("expected fail, got %s", res.Status)
	}
	if res.Failed != 1 {
		t.Errorf("failed: want 1, got %d", res.Failed)
	}
	if len(res.SampleViolations) == 0 {
		t.Fatal("expected at least one violation")
	}
	if res.SampleViolations[0].Value != 2 {
		t.Errorf("violation value: want 2 (row count), got %v", res.SampleViolations[0].Value)
	}
}

func TestEvalVolume_TooMany(t *testing.T) {
	eng := NewEngine("test-v1")
	rule := Rule{
		RuleID:    "r-vol-high",
		DatasetID: "ds1",
		RuleType:  RuleVolume,
		Severity:  SeverityHigh,
		Threshold: 1.0,
		Params: RuleParams{
			MaxRows: intPtr(2),
		},
	}
	data := Dataset{
		Record{"a": 1},
		Record{"a": 2},
		Record{"a": 3},
		Record{"a": 4},
	}

	res := eng.evaluateRule(rule, data)

	if res.Status != StatusFail {
		t.Errorf("expected fail, got %s", res.Status)
	}
	if res.Failed != 1 {
		t.Errorf("failed: want 1, got %d", res.Failed)
	}
	if len(res.SampleViolations) == 0 {
		t.Fatal("expected at least one violation")
	}
	if res.SampleViolations[0].Value != 4 {
		t.Errorf("violation value: want 4 (row count), got %v", res.SampleViolations[0].Value)
	}
}

func TestRunBatch(t *testing.T) {
	eng := NewEngine("v0.1.0")
	rules := []Rule{
		{
			RuleID:    "r1",
			DatasetID: "ds1",
			RuleType:  RuleCompleteness,
			Field:     "name",
			Severity:  SeverityHigh,
			Threshold: 1.0,
		},
		{
			RuleID:    "r2",
			DatasetID: "ds1",
			RuleType:  RuleVolume,
			Severity:  SeverityLow,
			Threshold: 1.0,
			Params: RuleParams{
				MinRows: intPtr(1),
				MaxRows: intPtr(100),
			},
		},
	}
	data := Dataset{
		Record{"name": "Alice"},
		Record{"name": "Bob"},
	}

	result, err := eng.RunBatch("ds1", rules, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Header checks.
	if result.Header.EngineVersion != "v0.1.0" {
		t.Errorf("engine_version: want v0.1.0, got %s", result.Header.EngineVersion)
	}
	if result.Header.DatasetID != "ds1" {
		t.Errorf("dataset_id: want ds1, got %s", result.Header.DatasetID)
	}
	if result.Header.Mode != ModeBatch {
		t.Errorf("mode: want batch, got %s", result.Header.Mode)
	}
	if result.Header.RecordsChecked != 2 {
		t.Errorf("records_checked: want 2, got %d", result.Header.RecordsChecked)
	}
	if result.Header.RunID == "" {
		t.Error("run_id should not be empty")
	}

	// Results checks.
	if len(result.Results) != 2 {
		t.Fatalf("results count: want 2, got %d", len(result.Results))
	}
	if result.Results[0].RuleID != "r1" {
		t.Errorf("first result rule_id: want r1, got %s", result.Results[0].RuleID)
	}
	if result.Results[1].RuleID != "r2" {
		t.Errorf("second result rule_id: want r2, got %s", result.Results[1].RuleID)
	}

	// Summary checks.
	if result.Summary.TotalRules != 2 {
		t.Errorf("total_rules: want 2, got %d", result.Summary.TotalRules)
	}
	if result.Summary.Passed != 2 {
		t.Errorf("passed: want 2, got %d", result.Summary.Passed)
	}
	if result.Summary.Failed != 0 {
		t.Errorf("failed: want 0, got %d", result.Summary.Failed)
	}
	if result.Summary.OverallPass != 1.0 {
		t.Errorf("overall_pass_rate: want 1.0, got %f", result.Summary.OverallPass)
	}
}

func TestRunBatch_ValidationError(t *testing.T) {
	eng := NewEngine("v0.1.0")
	rules := []Rule{
		{
			// Missing RuleID — should fail validation.
			DatasetID: "ds1",
			RuleType:  RuleCompleteness,
			Field:     "name",
			Threshold: 1.0,
		},
	}
	data := Dataset{Record{"name": "x"}}

	_, err := eng.RunBatch("ds1", rules, data)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestEvaluateRecord(t *testing.T) {
	eng := NewEngine("v0.1.0")
	rules := []Rule{
		{
			RuleID:    "rt-1",
			DatasetID: "ds1",
			RuleType:  RuleCompleteness,
			Field:     "name",
			Severity:  SeverityHigh,
			Threshold: 1.0,
			RTEnabled: true,
		},
		{
			RuleID:    "batch-only-1",
			DatasetID: "ds1",
			RuleType:  RuleCompleteness,
			Field:     "name",
			Severity:  SeverityLow,
			Threshold: 1.0,
			RTEnabled: false,
		},
		{
			RuleID:    "rt-2",
			DatasetID: "ds1",
			RuleType:  RuleValidity,
			Field:     "status",
			Severity:  SeverityMedium,
			Threshold: 1.0,
			RTEnabled: true,
			Params: RuleParams{
				AllowedValues: []string{"active", "inactive"},
			},
		},
	}
	rec := Record{"name": "Alice", "status": "active"}

	result, err := eng.EvaluateRecord(rules, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only RT-enabled rules should be evaluated.
	if len(result.Results) != 2 {
		t.Fatalf("results count: want 2 (RT only), got %d", len(result.Results))
	}

	ruleIDs := map[string]bool{}
	for _, r := range result.Results {
		ruleIDs[r.RuleID] = true
	}
	if !ruleIDs["rt-1"] || !ruleIDs["rt-2"] {
		t.Errorf("expected rt-1 and rt-2 in results, got %v", ruleIDs)
	}
	if ruleIDs["batch-only-1"] {
		t.Error("batch-only rule should not appear in RT results")
	}

	// Mode should be rt.
	if result.Header.Mode != ModeNearRT {
		t.Errorf("mode: want rt, got %s", result.Header.Mode)
	}
	if result.Header.RecordsChecked != 1 {
		t.Errorf("records_checked: want 1, got %d", result.Header.RecordsChecked)
	}
}

func TestThresholdBoundary(t *testing.T) {
	eng := NewEngine("test-v1")
	rule := Rule{
		RuleID:    "r-boundary",
		DatasetID: "ds1",
		RuleType:  RuleCompleteness,
		Field:     "email",
		Severity:  SeverityMedium,
		Threshold: 0.5,
	}
	// 1 pass, 1 fail => pass_rate = 0.5, which equals threshold exactly.
	data := Dataset{
		Record{"email": "a@b.com"},
		Record{"email": nil},
	}

	res := eng.evaluateRule(rule, data)

	if res.PassRate != 0.5 {
		t.Errorf("pass_rate: want 0.5, got %f", res.PassRate)
	}
	// pass_rate >= threshold should be pass.
	if res.Status != StatusPass {
		t.Errorf("expected pass when pass_rate equals threshold, got %s", res.Status)
	}
}
