package edqlite

import (
	"encoding/json"
	"testing"
	"time"
)

func TestComputeSummary(t *testing.T) {
	results := []RuleResult{
		{RuleID: "R001", Status: StatusPass},
		{RuleID: "R002", Status: StatusFail},
		{RuleID: "R003", Status: StatusPass},
		{RuleID: "R004", Status: StatusError},
		{RuleID: "R005", Status: StatusSkip},
	}

	s := computeSummary(results, 150*time.Millisecond)

	if s.TotalRules != 5 {
		t.Errorf("TotalRules = %d, want 5", s.TotalRules)
	}
	if s.Passed != 2 {
		t.Errorf("Passed = %d, want 2", s.Passed)
	}
	if s.Failed != 1 {
		t.Errorf("Failed = %d, want 1", s.Failed)
	}
	if s.Errors != 1 {
		t.Errorf("Errors = %d, want 1", s.Errors)
	}
	if s.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", s.Skipped)
	}

	// OverallPass = passed / (passed + failed) = 2/3
	wantRate := 2.0 / 3.0
	if diff := s.OverallPass - wantRate; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("OverallPass = %v, want %v", s.OverallPass, wantRate)
	}
	if s.LatencyMs != 150 {
		t.Errorf("LatencyMs = %v, want 150", s.LatencyMs)
	}
}

func TestRunResult_JSON(t *testing.T) {
	rr := &RunResult{
		Header: RunHeader{
			RunID:           "run-1",
			EngineVersion:   "0.1.0",
			RulePackVersion: "1.0.0",
			DatasetID:       "ds-1",
			Mode:            ModeBatch,
			StartedAt:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			EndedAt:         time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC),
			RecordsChecked:  100,
		},
		Results: []RuleResult{
			{
				RuleID:   "R001",
				RuleType: RuleCompleteness,
				Status:   StatusPass,
				Checked:  100,
				Failed:   1,
				PassRate: 0.99,
			},
		},
		Summary: RunSummary{
			TotalRules:  1,
			Passed:      1,
			OverallPass: 1.0,
		},
	}

	data, err := rr.JSON()
	if err != nil {
		t.Fatalf("JSON() error: %v", err)
	}

	// Verify it's valid JSON.
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// Check expected top-level keys.
	for _, key := range []string{"header", "results", "summary"} {
		if _, ok := parsed[key]; !ok {
			t.Errorf("missing key %q in JSON output", key)
		}
	}

	// Verify header fields round-trip.
	header, ok := parsed["header"].(map[string]any)
	if !ok {
		t.Fatal("header is not an object")
	}
	if header["run_id"] != "run-1" {
		t.Errorf("header.run_id = %v, want %q", header["run_id"], "run-1")
	}
	if header["dataset_id"] != "ds-1" {
		t.Errorf("header.dataset_id = %v, want %q", header["dataset_id"], "ds-1")
	}
}

func TestComputeSummary_AllPass(t *testing.T) {
	results := []RuleResult{
		{RuleID: "R001", Status: StatusPass},
		{RuleID: "R002", Status: StatusPass},
		{RuleID: "R003", Status: StatusPass},
	}

	s := computeSummary(results, 50*time.Millisecond)

	if s.Passed != 3 {
		t.Errorf("Passed = %d, want 3", s.Passed)
	}
	if s.Failed != 0 {
		t.Errorf("Failed = %d, want 0", s.Failed)
	}
	if s.OverallPass != 1.0 {
		t.Errorf("OverallPass = %v, want 1.0", s.OverallPass)
	}
}

func TestComputeSummary_Empty(t *testing.T) {
	s := computeSummary(nil, 0)

	if s.TotalRules != 0 {
		t.Errorf("TotalRules = %d, want 0", s.TotalRules)
	}
	if s.Passed != 0 {
		t.Errorf("Passed = %d, want 0", s.Passed)
	}
	if s.OverallPass != 0 {
		t.Errorf("OverallPass = %v, want 0", s.OverallPass)
	}
}
