package edqlite

import "testing"

func floatPtr(f float64) *float64 { return &f }
func intPtr(i int) *int           { return &i }

func TestComparison_CrossField_Equal(t *testing.T) {
	engine := NewEngine("test")
	rule := Rule{
		RuleID:    "cmp-eq-1",
		DatasetID: "ds1",
		RuleType:  RuleComparison,
		Field:     "field_a",
		Severity:  SeverityHigh,
		Threshold: 1.0,
		Params: RuleParams{
			CompareField: "field_b",
			CompareOp:    CompareEqual,
		},
	}

	data := Dataset{
		Record{"field_a": 10, "field_b": 10},
		Record{"field_a": 20, "field_b": 20},
		Record{"field_a": 30, "field_b": 30},
	}

	result, err := engine.RunBatch("ds1", []Rule{rule}, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rr := result.Results[0]
	if rr.Status != StatusPass {
		t.Errorf("expected pass, got %s", rr.Status)
	}
	if rr.Failed != 0 {
		t.Errorf("expected 0 failures, got %d", rr.Failed)
	}
	if rr.Checked != 3 {
		t.Errorf("expected 3 checked, got %d", rr.Checked)
	}
}

func TestComparison_CrossField_NotEqual(t *testing.T) {
	engine := NewEngine("test")
	rule := Rule{
		RuleID:    "cmp-ne-1",
		DatasetID: "ds1",
		RuleType:  RuleComparison,
		Field:     "field_a",
		Severity:  SeverityMedium,
		Threshold: 1.0,
		Params: RuleParams{
			CompareField: "field_b",
			CompareOp:    CompareNotEqual,
		},
	}

	data := Dataset{
		Record{"field_a": 1, "field_b": 2},
		Record{"field_a": 3, "field_b": 4},
		Record{"field_a": 5, "field_b": 5}, // equal → failure
	}

	result, err := engine.RunBatch("ds1", []Rule{rule}, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rr := result.Results[0]
	if rr.Failed != 1 {
		t.Errorf("expected 1 failure, got %d", rr.Failed)
	}
	if rr.Status != StatusFail {
		t.Errorf("expected fail, got %s", rr.Status)
	}
}

func TestComparison_CrossValue_GreaterThan(t *testing.T) {
	engine := NewEngine("test")
	rule := Rule{
		RuleID:    "cmp-gt-1",
		DatasetID: "ds1",
		RuleType:  RuleComparison,
		Field:     "score",
		Severity:  SeverityLow,
		Threshold: 1.0,
		Params: RuleParams{
			CompareOp:    CompareGreaterThan,
			CompareValue: 50.0,
		},
	}

	data := Dataset{
		Record{"score": 60.0},
		Record{"score": 70.0},
		Record{"score": 80.0},
	}

	result, err := engine.RunBatch("ds1", []Rule{rule}, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rr := result.Results[0]
	if rr.Status != StatusPass {
		t.Errorf("expected pass, got %s", rr.Status)
	}
	if rr.Failed != 0 {
		t.Errorf("expected 0 failures, got %d", rr.Failed)
	}

	// Now add a value that does not satisfy > 50
	data = append(data, Record{"score": 40.0})
	result, err = engine.RunBatch("ds1", []Rule{rule}, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rr = result.Results[0]
	if rr.Failed != 1 {
		t.Errorf("expected 1 failure, got %d", rr.Failed)
	}
}

func TestComparison_CrossValue_LessThan(t *testing.T) {
	engine := NewEngine("test")
	rule := Rule{
		RuleID:    "cmp-lt-1",
		DatasetID: "ds1",
		RuleType:  RuleComparison,
		Field:     "latency",
		Severity:  SeverityHigh,
		Threshold: 1.0,
		Params: RuleParams{
			CompareOp:    CompareLessThan,
			CompareValue: 100.0,
		},
	}

	data := Dataset{
		Record{"latency": 50.0},
		Record{"latency": 99.0},
		Record{"latency": 100.0}, // not < 100, failure
	}

	result, err := engine.RunBatch("ds1", []Rule{rule}, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rr := result.Results[0]
	if rr.Failed != 1 {
		t.Errorf("expected 1 failure, got %d", rr.Failed)
	}
	if rr.Status != StatusFail {
		t.Errorf("expected fail, got %s", rr.Status)
	}
}

func TestComparison_NullPrimaryField(t *testing.T) {
	engine := NewEngine("test")
	rule := Rule{
		RuleID:    "cmp-null-primary",
		DatasetID: "ds1",
		RuleType:  RuleComparison,
		Field:     "field_a",
		Severity:  SeverityMedium,
		Threshold: 1.0,
		Params: RuleParams{
			CompareField: "field_b",
			CompareOp:    CompareEqual,
		},
	}

	data := Dataset{
		Record{"field_a": nil, "field_b": 10},   // null primary → skipped
		Record{"field_b": 20},                     // missing primary → skipped
		Record{"field_a": 30, "field_b": 30},      // valid
	}

	result, err := engine.RunBatch("ds1", []Rule{rule}, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rr := result.Results[0]
	// Only 1 record should be checked (the third one).
	if rr.Checked != 1 {
		t.Errorf("expected 1 checked, got %d", rr.Checked)
	}
	if rr.Failed != 0 {
		t.Errorf("expected 0 failures, got %d", rr.Failed)
	}
	if rr.Status != StatusPass {
		t.Errorf("expected pass, got %s", rr.Status)
	}
}

func TestComparison_NullCompareField(t *testing.T) {
	engine := NewEngine("test")
	rule := Rule{
		RuleID:    "cmp-null-compare",
		DatasetID: "ds1",
		RuleType:  RuleComparison,
		Field:     "field_a",
		Severity:  SeverityHigh,
		Threshold: 1.0,
		Params: RuleParams{
			CompareField: "field_b",
			CompareOp:    CompareEqual,
		},
	}

	data := Dataset{
		Record{"field_a": 10, "field_b": nil},  // null compare → failure
		Record{"field_a": 20},                    // missing compare → failure
		Record{"field_a": 30, "field_b": 30},     // valid
	}

	result, err := engine.RunBatch("ds1", []Rule{rule}, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rr := result.Results[0]
	if rr.Checked != 3 {
		t.Errorf("expected 3 checked, got %d", rr.Checked)
	}
	if rr.Failed != 2 {
		t.Errorf("expected 2 failures, got %d", rr.Failed)
	}
	if rr.Status != StatusFail {
		t.Errorf("expected fail, got %s", rr.Status)
	}
	// Verify violation reason mentions null compare field.
	if len(rr.SampleViolations) < 1 {
		t.Fatal("expected at least 1 sample violation")
	}
	v := rr.SampleViolations[0]
	if v.Field != "field_a" {
		t.Errorf("expected violation field 'field_a', got %q", v.Field)
	}
}

func TestComparison_StringComparison(t *testing.T) {
	engine := NewEngine("test")
	rule := Rule{
		RuleID:    "cmp-str-1",
		DatasetID: "ds1",
		RuleType:  RuleComparison,
		Field:     "name",
		Severity:  SeverityLow,
		Threshold: 1.0,
		Params: RuleParams{
			CompareOp:    CompareGreaterThan,
			CompareValue: "banana",
		},
	}

	data := Dataset{
		Record{"name": "cherry"},  // "cherry" > "banana" → pass
		Record{"name": "apple"},   // "apple" > "banana" → false, failure
		Record{"name": "date"},    // "date" > "banana" → pass
	}

	result, err := engine.RunBatch("ds1", []Rule{rule}, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rr := result.Results[0]
	if rr.Failed != 1 {
		t.Errorf("expected 1 failure, got %d", rr.Failed)
	}
	if rr.Checked != 3 {
		t.Errorf("expected 3 checked, got %d", rr.Checked)
	}
}

func TestCompareValues(t *testing.T) {
	tests := []struct {
		name string
		a, b any
		op   ComparisonOp
		want bool
	}{
		// Numeric eq
		{"num_eq_true", 10.0, 10.0, CompareEqual, true},
		{"num_eq_false", 10.0, 20.0, CompareEqual, false},
		// Numeric ne
		{"num_ne_true", 10.0, 20.0, CompareNotEqual, true},
		{"num_ne_false", 10.0, 10.0, CompareNotEqual, false},
		// Numeric gt
		{"num_gt_true", 20.0, 10.0, CompareGreaterThan, true},
		{"num_gt_false", 10.0, 20.0, CompareGreaterThan, false},
		{"num_gt_equal", 10.0, 10.0, CompareGreaterThan, false},
		// Numeric lt
		{"num_lt_true", 10.0, 20.0, CompareLessThan, true},
		{"num_lt_false", 20.0, 10.0, CompareLessThan, false},
		// Numeric gte
		{"num_gte_true_gt", 20.0, 10.0, CompareGreaterEqual, true},
		{"num_gte_true_eq", 10.0, 10.0, CompareGreaterEqual, true},
		{"num_gte_false", 5.0, 10.0, CompareGreaterEqual, false},
		// Numeric lte
		{"num_lte_true_lt", 10.0, 20.0, CompareLessEqual, true},
		{"num_lte_true_eq", 10.0, 10.0, CompareLessEqual, true},
		{"num_lte_false", 20.0, 10.0, CompareLessEqual, false},
		// String eq
		{"str_eq_true", "hello", "hello", CompareEqual, true},
		{"str_eq_false", "hello", "world", CompareEqual, false},
		// String ne
		{"str_ne_true", "hello", "world", CompareNotEqual, true},
		{"str_ne_false", "hello", "hello", CompareNotEqual, false},
		// String gt
		{"str_gt_true", "banana", "apple", CompareGreaterThan, true},
		{"str_gt_false", "apple", "banana", CompareGreaterThan, false},
		// String lt
		{"str_lt_true", "apple", "banana", CompareLessThan, true},
		{"str_lt_false", "banana", "apple", CompareLessThan, false},
		// String gte
		{"str_gte_true", "banana", "banana", CompareGreaterEqual, true},
		{"str_gte_false", "apple", "banana", CompareGreaterEqual, false},
		// String lte
		{"str_lte_true", "apple", "banana", CompareLessEqual, true},
		{"str_lte_false", "banana", "apple", CompareLessEqual, false},
		// Mixed int types (uses numeric path)
		{"int_eq", 42, 42, CompareEqual, true},
		{"int_gt", int64(100), int32(50), CompareGreaterThan, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := compareValues(tc.a, tc.b, tc.op)
			if got != tc.want {
				t.Errorf("compareValues(%v, %v, %q) = %v, want %v",
					tc.a, tc.b, tc.op, got, tc.want)
			}
		})
	}
}
