package edqlite

import (
	"encoding/json"
	"time"
)

// RunMode indicates whether a run was batch or near-real-time.
type RunMode string

const (
	ModeBatch  RunMode = "batch"
	ModeNearRT RunMode = "rt"
)

// RuleStatus is the outcome of a single rule evaluation.
type RuleStatus string

const (
	StatusPass  RuleStatus = "pass"
	StatusFail  RuleStatus = "fail"
	StatusError RuleStatus = "error"
	StatusSkip  RuleStatus = "skip"
)

// RunResult is the complete output of an EDQ Lite evaluation run (Section 7).
type RunResult struct {
	Header  RunHeader    `json:"header"`
	Results []RuleResult `json:"results"`
	Summary RunSummary   `json:"summary"`
}

// RunHeader contains run-level metadata.
type RunHeader struct {
	RunID            string    `json:"run_id"`
	EngineVersion    string    `json:"engine_version"`
	RulePackVersion  string    `json:"rule_pack_version"`
	DatasetID        string    `json:"dataset_id"`
	Mode             RunMode   `json:"mode"`
	StartedAt        time.Time `json:"started_at"`
	EndedAt          time.Time `json:"ended_at"`
	RecordsChecked   int       `json:"records_checked"`
	ExecutorProfile  string    `json:"executor_profile,omitempty"`
}

// RuleResult is the per-rule outcome.
type RuleResult struct {
	RuleID           string         `json:"rule_id"`
	RuleType         RuleType       `json:"rule_type"`
	Status           RuleStatus     `json:"status"`
	Checked          int            `json:"checked"`
	Failed           int            `json:"failed"`
	PassRate         float64        `json:"pass_rate"`
	Threshold        float64        `json:"threshold"`
	Severity         Severity       `json:"severity"`
	ObservedStats    ObservedStats  `json:"observed_stats"`
	SampleViolations []Violation    `json:"sample_violations,omitempty"`
	WaiverApplied    bool           `json:"waiver_applied"`
	LineageRefs      []string       `json:"lineage_refs,omitempty"`
	ErrorMessage     string         `json:"error_message,omitempty"`
}

// ObservedStats captures measured statistics during evaluation.
type ObservedStats struct {
	NullRate  *float64 `json:"null_rate,omitempty"`
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	Mean      *float64 `json:"mean,omitempty"`
	UniqueN   *int     `json:"unique_count,omitempty"`
	TotalN    *int     `json:"total_count,omitempty"`
	LatencyMs *float64 `json:"latency_ms,omitempty"`
}

// Violation is a sampled failing record.
type Violation struct {
	RecordIndex int    `json:"record_index,omitempty"`
	Field       string `json:"field"`
	Value       any    `json:"value"`
	Reason      string `json:"reason"`
}

// RunSummary aggregates across all rules.
type RunSummary struct {
	TotalRules    int     `json:"total_rules"`
	Passed        int     `json:"passed"`
	Failed        int     `json:"failed"`
	Errors        int     `json:"errors"`
	Skipped       int     `json:"skipped"`
	OverallPass   float64 `json:"overall_pass_rate"`
	LatencyMs     float64 `json:"latency_ms"`
	DPAEvidenceURL string `json:"dpa_evidence_url,omitempty"`
}

// JSON returns the RunResult as indented JSON bytes.
func (r *RunResult) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// computeSummary populates the RunSummary from results.
func computeSummary(results []RuleResult, latency time.Duration) RunSummary {
	s := RunSummary{
		TotalRules: len(results),
		LatencyMs:  float64(latency.Milliseconds()),
	}
	for _, r := range results {
		switch r.Status {
		case StatusPass:
			s.Passed++
		case StatusFail:
			s.Failed++
		case StatusError:
			s.Errors++
		case StatusSkip:
			s.Skipped++
		}
	}
	evaluated := s.Passed + s.Failed
	if evaluated > 0 {
		s.OverallPass = float64(s.Passed) / float64(evaluated)
	}
	return s
}
