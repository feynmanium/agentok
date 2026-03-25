package edqlite

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestE2E_CreditBureauFeatures(t *testing.T) {
	// --- Build rule pack inline ---
	datasetID := "altdata.creditbureaufeatures"
	entity := "credit_bureau_record"
	owner := "data-quality-team"

	minScore := 300.0
	maxScore := 850.0
	minRows := 100
	maxRows := 10000

	rules := []Rule{
		{
			RuleID:    "CB-COMP-001",
			DatasetID: datasetID,
			RuleType:  RuleCompleteness,
			Entity:    entity,
			Field:     "credit_score",
			Severity:  SeverityCritical,
			Owner:     owner,
			Threshold: 0.995,
		},
		{
			RuleID:    "CB-VAL-001",
			DatasetID: datasetID,
			RuleType:  RuleValidity,
			Entity:    entity,
			Field:     "bureau",
			Severity:  SeverityHigh,
			Owner:     owner,
			Threshold: 0.99,
			Params: RuleParams{
				AllowedValues: []string{"experian", "equifax", "transunion"},
			},
		},
		{
			RuleID:    "CB-RNG-001",
			DatasetID: datasetID,
			RuleType:  RuleRange,
			Entity:    entity,
			Field:     "credit_score",
			Severity:  SeverityHigh,
			Owner:     owner,
			Threshold: 0.99,
			Params: RuleParams{
				Min: &minScore,
				Max: &maxScore,
			},
		},
		{
			RuleID:    "CB-UNQ-001",
			DatasetID: datasetID,
			RuleType:  RuleUniqueness,
			Entity:    entity,
			Field:     "customer_id",
			Severity:  SeverityCritical,
			Owner:     owner,
			Threshold: 1.0,
		},
		{
			RuleID:    "CB-VOL-001",
			DatasetID: datasetID,
			RuleType:  RuleVolume,
			Entity:    entity,
			Severity:  SeverityMedium,
			Owner:     owner,
			Threshold: 1.0,
			Params: RuleParams{
				MinRows: &minRows,
				MaxRows: &maxRows,
			},
		},
		{
			RuleID:    "CB-CMP-001",
			DatasetID: datasetID,
			RuleType:  RuleComparison,
			Entity:    entity,
			Field:     "credit_score",
			Severity:  SeverityHigh,
			Owner:     owner,
			Threshold: 0.95,
			Params: RuleParams{
				CompareField: "min_required_score",
				CompareOp:    CompareGreaterEqual,
			},
		},
	}

	// --- Generate dataset of 500 records with controlled violations ---
	bureaus := []string{"experian", "equifax", "transunion"}
	data := make(Dataset, 500)

	for i := 0; i < 500; i++ {
		score := 600.0 + float64(i%250)
		bureau := bureaus[i%3]
		customerID := fmt.Sprintf("CUST-%05d", i)
		minReqScore := 550.0

		data[i] = Record{
			"credit_score":       score,
			"bureau":             bureau,
			"customer_id":        customerID,
			"min_required_score": minReqScore,
			"ingestion_ts":       time.Now(),
		}
	}

	// Inject violations:
	// 1. Null credit_score (completeness violation) - 1 record
	data[10]["credit_score"] = nil

	// 2. Invalid bureau value (validity violation) - 2 records
	data[20]["bureau"] = "invalid_bureau"
	data[21]["bureau"] = "unknown"

	// 3. Out-of-range credit_score (range violation) - 2 records
	data[30]["credit_score"] = 100.0 // below 300
	data[31]["credit_score"] = 900.0 // above 850

	// 4. Duplicate customer_id (uniqueness violation) - 3 records
	data[40]["customer_id"] = "CUST-00000" // duplicate of record 0
	data[41]["customer_id"] = "CUST-00001" // duplicate of record 1
	data[42]["customer_id"] = "CUST-00002" // duplicate of record 2

	// 5. Comparison violation: credit_score < min_required_score - 5 records
	for i := 50; i < 55; i++ {
		data[i]["credit_score"] = 500.0
		data[i]["min_required_score"] = 600.0
	}

	// --- Run engine ---
	eng := NewEngine("1.0.0-test")
	result, err := eng.RunBatch(datasetID, rules, data)

	// --- Verify: RunResult is non-nil and no error ---
	if err != nil {
		t.Fatalf("RunBatch returned error: %v", err)
	}
	if result == nil {
		t.Fatal("RunBatch returned nil result")
	}

	// --- Verify header ---
	if result.Header.Mode != ModeBatch {
		t.Errorf("expected mode %q, got %q", ModeBatch, result.Header.Mode)
	}
	if result.Header.DatasetID != datasetID {
		t.Errorf("expected dataset_id %q, got %q", datasetID, result.Header.DatasetID)
	}
	if result.Header.RecordsChecked != 500 {
		t.Errorf("expected 500 records checked, got %d", result.Header.RecordsChecked)
	}
	if result.Header.EngineVersion != "1.0.0-test" {
		t.Errorf("expected engine version %q, got %q", "1.0.0-test", result.Header.EngineVersion)
	}
	if result.Header.RunID == "" {
		t.Error("expected non-empty run_id")
	}

	// --- Verify results count matches rules ---
	if len(result.Results) != len(rules) {
		t.Fatalf("expected %d results, got %d", len(rules), len(result.Results))
	}

	// --- Verify summary totals ---
	if result.Summary.TotalRules != len(rules) {
		t.Errorf("expected summary total_rules=%d, got %d", len(rules), result.Summary.TotalRules)
	}
	if result.Summary.Passed+result.Summary.Failed+result.Summary.Errors+result.Summary.Skipped != result.Summary.TotalRules {
		t.Errorf("summary counts don't add up: passed=%d failed=%d errors=%d skipped=%d total=%d",
			result.Summary.Passed, result.Summary.Failed, result.Summary.Errors, result.Summary.Skipped, result.Summary.TotalRules)
	}

	// --- Verify JSON serialization ---
	jsonBytes, err := result.JSON()
	if err != nil {
		t.Fatalf("JSON serialization failed: %v", err)
	}
	if len(jsonBytes) == 0 {
		t.Fatal("JSON output is empty")
	}

	// Verify it round-trips as valid JSON.
	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("JSON round-trip failed: %v", err)
	}
	if _, ok := parsed["header"]; !ok {
		t.Error("parsed JSON missing 'header' key")
	}
	if _, ok := parsed["results"]; !ok {
		t.Error("parsed JSON missing 'results' key")
	}
	if _, ok := parsed["summary"]; !ok {
		t.Error("parsed JSON missing 'summary' key")
	}

	// --- Verify specific rule outcomes ---
	resultByID := make(map[string]RuleResult, len(result.Results))
	for _, r := range result.Results {
		resultByID[r.RuleID] = r
	}

	// Completeness: 1 null out of 500 => pass_rate = 499/500 = 0.998 >= 0.995 => pass
	compResult := resultByID["CB-COMP-001"]
	if compResult.Status != StatusPass {
		t.Errorf("CB-COMP-001: expected pass, got %s (pass_rate=%.4f)", compResult.Status, compResult.PassRate)
	}
	if compResult.Failed != 1 {
		t.Errorf("CB-COMP-001: expected 1 failure, got %d", compResult.Failed)
	}

	// Validity: 2 invalid bureaus out of 500 => pass_rate = 498/500 = 0.996 >= 0.99 => pass
	valResult := resultByID["CB-VAL-001"]
	if valResult.Status != StatusPass {
		t.Errorf("CB-VAL-001: expected pass, got %s (pass_rate=%.4f)", valResult.Status, valResult.PassRate)
	}
	if valResult.Failed != 2 {
		t.Errorf("CB-VAL-001: expected 2 failures, got %d", valResult.Failed)
	}

	// Range: 1 null (non-numeric) + 2 out-of-range = 3 failures out of 500.
	// But record 10 has nil credit_score (counted as non-numeric failure),
	// and records 30, 31 are out-of-range.
	// Also the 5 records (50-54) have credit_score=500 which is in range [300,850].
	// So 3 failures => pass_rate = 497/500 = 0.994 >= 0.99 => pass
	rngResult := resultByID["CB-RNG-001"]
	if rngResult.Status != StatusPass {
		t.Errorf("CB-RNG-001: expected pass, got %s (pass_rate=%.4f)", rngResult.Status, rngResult.PassRate)
	}
	if rngResult.Failed != 3 {
		t.Errorf("CB-RNG-001: expected 3 failures, got %d", rngResult.Failed)
	}

	// Uniqueness: 3 duplicates out of 500 => pass_rate = 497/500 = 0.994 < 1.0 => fail
	unqResult := resultByID["CB-UNQ-001"]
	if unqResult.Status != StatusFail {
		t.Errorf("CB-UNQ-001: expected fail, got %s (pass_rate=%.4f)", unqResult.Status, unqResult.PassRate)
	}
	if unqResult.Failed != 3 {
		t.Errorf("CB-UNQ-001: expected 3 failures, got %d", unqResult.Failed)
	}

	// Volume: 500 rows, min=100 max=10000 => pass
	volResult := resultByID["CB-VOL-001"]
	if volResult.Status != StatusPass {
		t.Errorf("CB-VOL-001: expected pass, got %s", volResult.Status)
	}

	// Comparison: credit_score >= min_required_score.
	// 5 records (50-54) have score=500 < min_required=600 => fail.
	// Record 30 has score=100 < min_required=550 => also fails.
	// Record 10 has nil credit_score => skipped by comparison (not checked).
	// So ~494 checked, 6 failed => pass_rate ~ 0.9878 >= 0.95 => pass
	cmpResult := resultByID["CB-CMP-001"]
	if cmpResult.Status != StatusPass {
		t.Errorf("CB-CMP-001: expected pass, got %s (pass_rate=%.4f)", cmpResult.Status, cmpResult.PassRate)
	}
	if cmpResult.Failed != 6 {
		t.Errorf("CB-CMP-001: expected 6 failures, got %d", cmpResult.Failed)
	}

	// --- Verify overall summary counts ---
	// Expected: 5 pass (completeness, validity, range, volume, comparison), 1 fail (uniqueness)
	if result.Summary.Passed != 5 {
		t.Errorf("expected 5 passed rules, got %d", result.Summary.Passed)
	}
	if result.Summary.Failed != 1 {
		t.Errorf("expected 1 failed rule, got %d", result.Summary.Failed)
	}

	t.Logf("E2E test completed: %d rules evaluated, %d passed, %d failed, latency=%.1fms",
		result.Summary.TotalRules, result.Summary.Passed, result.Summary.Failed, result.Summary.LatencyMs)
}
