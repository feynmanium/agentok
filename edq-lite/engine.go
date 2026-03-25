package edqlite

import (
	"fmt"
	"math"
	"time"
)

// Engine is the core evaluation engine for EDQ Lite data quality rules.
type Engine struct {
	engineVersion      string
	maxViolationSamples int
}

// NewEngine creates a new Engine with the given version string.
func NewEngine(version string) *Engine {
	return &Engine{
		engineVersion:      version,
		maxViolationSamples: 10,
	}
}

// generateRunID produces a unique run identifier.
func generateRunID() string {
	return fmt.Sprintf("run-%d", time.Now().UnixNano())
}

// RunBatch evaluates all rules against a full dataset in batch mode.
func (e *Engine) RunBatch(datasetID string, rules []Rule, data Dataset) (*RunResult, error) {
	// Validate all rules first.
	for i := range rules {
		if err := rules[i].Validate(); err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
	}

	startedAt := time.Now()

	results := make([]RuleResult, 0, len(rules))
	for _, rule := range rules {
		results = append(results, e.evaluateRule(rule, data))
	}

	endedAt := time.Now()
	latency := endedAt.Sub(startedAt)

	return &RunResult{
		Header: RunHeader{
			RunID:          generateRunID(),
			EngineVersion:  e.engineVersion,
			DatasetID:      datasetID,
			Mode:           ModeBatch,
			StartedAt:      startedAt,
			EndedAt:        endedAt,
			RecordsChecked: len(data),
		},
		Results: results,
		Summary: computeSummary(results, latency),
	}, nil
}

// EvaluateRecord evaluates RT-enabled rules against a single record.
func (e *Engine) EvaluateRecord(rules []Rule, rec Record) (*RunResult, error) {
	// Filter to RT-enabled rules only.
	var rtRules []Rule
	for _, r := range rules {
		if r.RTEnabled {
			rtRules = append(rtRules, r)
		}
	}

	// Validate the filtered rules.
	for i := range rtRules {
		if err := rtRules[i].Validate(); err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
	}

	startedAt := time.Now()
	data := Dataset{rec}

	results := make([]RuleResult, 0, len(rtRules))
	for _, rule := range rtRules {
		results = append(results, e.evaluateRule(rule, data))
	}

	endedAt := time.Now()
	latency := endedAt.Sub(startedAt)

	return &RunResult{
		Header: RunHeader{
			RunID:          generateRunID(),
			EngineVersion:  e.engineVersion,
			DatasetID:      "",
			Mode:           ModeNearRT,
			StartedAt:      startedAt,
			EndedAt:        endedAt,
			RecordsChecked: 1,
		},
		Results: results,
		Summary: computeSummary(results, latency),
	}, nil
}

// evaluateRule dispatches evaluation to the appropriate handler based on rule type.
func (e *Engine) evaluateRule(rule Rule, data Dataset) RuleResult {
	switch rule.RuleType {
	case RuleCompleteness:
		return e.evalCompleteness(rule, data)
	case RuleValidity:
		return e.evalValidity(rule, data)
	case RuleRange:
		return e.evalRange(rule, data)
	case RuleUniqueness:
		return e.evalUniqueness(rule, data)
	case RuleTimeliness:
		return e.evalTimeliness(rule, data)
	case RuleVolume:
		return e.evalVolume(rule, data)
	case RuleComparison:
		return e.evaluateComparison(rule, data)
	default:
		return e.errorResult(rule, fmt.Sprintf("unsupported rule type: %s", rule.RuleType))
	}
}

// evalCompleteness checks that a field is non-null across records.
func (e *Engine) evalCompleteness(rule Rule, data Dataset) RuleResult {
	checked := len(data)
	failed := 0
	var violations []Violation

	for i, rec := range data {
		if rec.IsNull(rule.Field) {
			failed++
			if len(violations) < e.maxViolationSamples {
				violations = append(violations, Violation{
					RecordIndex: i,
					Field:       rule.Field,
					Value:       nil,
					Reason:      "field is null or missing",
				})
			}
		}
	}

	passRate := computePassRate(checked, failed)
	nullRate := 0.0
	if checked > 0 {
		nullRate = float64(failed) / float64(checked)
	}

	return RuleResult{
		RuleID:    rule.RuleID,
		RuleType:  rule.RuleType,
		Status:    deriveStatus(passRate, rule.Threshold),
		Checked:   checked,
		Failed:    failed,
		PassRate:  passRate,
		Threshold: rule.Threshold,
		Severity:  rule.Severity,
		ObservedStats: ObservedStats{
			NullRate: &nullRate,
		},
		SampleViolations: violations,
	}
}

// evalValidity checks that field values are within the allowed set.
func (e *Engine) evalValidity(rule Rule, data Dataset) RuleResult {
	allowed := make(map[string]struct{}, len(rule.Params.AllowedValues))
	for _, v := range rule.Params.AllowedValues {
		allowed[v] = struct{}{}
	}

	checked := len(data)
	failed := 0
	var violations []Violation

	for i, rec := range data {
		s, ok := rec.GetString(rule.Field)
		if !ok {
			failed++
			if len(violations) < e.maxViolationSamples {
				violations = append(violations, Violation{
					RecordIndex: i,
					Field:       rule.Field,
					Value:       rec[rule.Field],
					Reason:      "field is null or not a string",
				})
			}
			continue
		}
		if _, valid := allowed[s]; !valid {
			failed++
			if len(violations) < e.maxViolationSamples {
				violations = append(violations, Violation{
					RecordIndex: i,
					Field:       rule.Field,
					Value:       s,
					Reason:      fmt.Sprintf("value %q not in allowed set", s),
				})
			}
		}
	}

	passRate := computePassRate(checked, failed)
	return RuleResult{
		RuleID:           rule.RuleID,
		RuleType:         rule.RuleType,
		Status:           deriveStatus(passRate, rule.Threshold),
		Checked:          checked,
		Failed:           failed,
		PassRate:         passRate,
		Threshold:        rule.Threshold,
		Severity:         rule.Severity,
		SampleViolations: violations,
	}
}

// evalRange checks that numeric field values fall within min/max bounds.
func (e *Engine) evalRange(rule Rule, data Dataset) RuleResult {
	checked := len(data)
	failed := 0
	var violations []Violation

	var minVal, maxVal float64
	var sum float64
	validCount := 0
	first := true

	for i, rec := range data {
		f, ok := rec.GetFloat(rule.Field)
		if !ok {
			failed++
			if len(violations) < e.maxViolationSamples {
				violations = append(violations, Violation{
					RecordIndex: i,
					Field:       rule.Field,
					Value:       rec[rule.Field],
					Reason:      "field is null or not numeric",
				})
			}
			continue
		}

		// Track stats.
		if first {
			minVal = f
			maxVal = f
			first = false
		} else {
			if f < minVal {
				minVal = f
			}
			if f > maxVal {
				maxVal = f
			}
		}
		sum += f
		validCount++

		// Check bounds.
		outOfRange := false
		var reason string
		if rule.Params.Min != nil && f < *rule.Params.Min {
			outOfRange = true
			reason = fmt.Sprintf("value %g < min %g", f, *rule.Params.Min)
		}
		if rule.Params.Max != nil && f > *rule.Params.Max {
			outOfRange = true
			reason = fmt.Sprintf("value %g > max %g", f, *rule.Params.Max)
		}
		if outOfRange {
			failed++
			if len(violations) < e.maxViolationSamples {
				violations = append(violations, Violation{
					RecordIndex: i,
					Field:       rule.Field,
					Value:       f,
					Reason:      reason,
				})
			}
		}
	}

	passRate := computePassRate(checked, failed)

	stats := ObservedStats{}
	if validCount > 0 {
		mean := sum / float64(validCount)
		stats.Min = &minVal
		stats.Max = &maxVal
		stats.Mean = &mean
	}

	return RuleResult{
		RuleID:           rule.RuleID,
		RuleType:         rule.RuleType,
		Status:           deriveStatus(passRate, rule.Threshold),
		Checked:          checked,
		Failed:           failed,
		PassRate:         passRate,
		Threshold:        rule.Threshold,
		Severity:         rule.Severity,
		ObservedStats:    stats,
		SampleViolations: violations,
	}
}

// evalUniqueness checks that field values are unique across all records.
func (e *Engine) evalUniqueness(rule Rule, data Dataset) RuleResult {
	checked := len(data)
	failed := 0
	var violations []Violation

	type occurrence struct {
		count int
		first int
	}
	seen := make(map[string]occurrence, checked)

	for i, rec := range data {
		s, ok := rec.GetString(rule.Field)
		if !ok {
			// Null/missing fields are not counted as duplicates.
			continue
		}
		occ, exists := seen[s]
		if exists {
			// This is a duplicate.
			failed++
			if len(violations) < e.maxViolationSamples {
				violations = append(violations, Violation{
					RecordIndex: i,
					Field:       rule.Field,
					Value:       s,
					Reason:      fmt.Sprintf("duplicate value %q (first seen at index %d)", s, occ.first),
				})
			}
			occ.count++
			seen[s] = occ
		} else {
			seen[s] = occurrence{count: 1, first: i}
		}
	}

	passRate := computePassRate(checked, failed)
	uniqueCount := len(seen)

	return RuleResult{
		RuleID:    rule.RuleID,
		RuleType:  rule.RuleType,
		Status:    deriveStatus(passRate, rule.Threshold),
		Checked:   checked,
		Failed:    failed,
		PassRate:  passRate,
		Threshold: rule.Threshold,
		Severity:  rule.Severity,
		ObservedStats: ObservedStats{
			UniqueN: &uniqueCount,
		},
		SampleViolations: violations,
	}
}

// evalTimeliness checks that records are within acceptable latency bounds.
func (e *Engine) evalTimeliness(rule Rule, data Dataset) RuleResult {
	tsField := rule.Params.TimestampField
	if tsField == "" {
		tsField = rule.Field
	}

	checked := len(data)
	failed := 0
	var violations []Violation
	now := time.Now()
	maxLatency := time.Duration(rule.Params.MaxLatencySec) * time.Second

	for i, rec := range data {
		t, ok := rec.GetTime(tsField)
		if !ok {
			failed++
			if len(violations) < e.maxViolationSamples {
				violations = append(violations, Violation{
					RecordIndex: i,
					Field:       tsField,
					Value:       rec[tsField],
					Reason:      "timestamp field is null or not a time.Time",
				})
			}
			continue
		}

		latency := now.Sub(t)
		if latency < 0 {
			latency = -latency
		}

		if latency > maxLatency {
			failed++
			if len(violations) < e.maxViolationSamples {
				violations = append(violations, Violation{
					RecordIndex: i,
					Field:       tsField,
					Value:       t.Format(time.RFC3339),
					Reason:      fmt.Sprintf("latency %.1fs exceeds max %ds", latency.Seconds(), rule.Params.MaxLatencySec),
				})
			}
		}
	}

	passRate := computePassRate(checked, failed)
	return RuleResult{
		RuleID:           rule.RuleID,
		RuleType:         rule.RuleType,
		Status:           deriveStatus(passRate, rule.Threshold),
		Checked:          checked,
		Failed:           failed,
		PassRate:         passRate,
		Threshold:        rule.Threshold,
		Severity:         rule.Severity,
		SampleViolations: violations,
	}
}

// evalVolume checks that the dataset size falls within expected bounds.
func (e *Engine) evalVolume(rule Rule, data Dataset) RuleResult {
	rowCount := len(data)
	checked := 1
	failed := 0
	var violations []Violation

	if rule.Params.MinRows != nil && rowCount < *rule.Params.MinRows {
		failed = 1
		violations = append(violations, Violation{
			Field:  "_dataset",
			Value:  rowCount,
			Reason: fmt.Sprintf("row count %d < min %d", rowCount, *rule.Params.MinRows),
		})
	}
	if rule.Params.MaxRows != nil && rowCount > *rule.Params.MaxRows {
		failed = 1
		if len(violations) == 0 {
			violations = append(violations, Violation{
				Field:  "_dataset",
				Value:  rowCount,
				Reason: fmt.Sprintf("row count %d > max %d", rowCount, *rule.Params.MaxRows),
			})
		}
	}

	passRate := computePassRate(checked, failed)
	totalCount := rowCount
	return RuleResult{
		RuleID:    rule.RuleID,
		RuleType:  rule.RuleType,
		Status:    deriveStatus(passRate, rule.Threshold),
		Checked:   checked,
		Failed:    failed,
		PassRate:  passRate,
		Threshold: rule.Threshold,
		Severity:  rule.Severity,
		ObservedStats: ObservedStats{
			TotalN: &totalCount,
		},
		SampleViolations: violations,
	}
}

// errorResult builds a RuleResult with StatusError.
func (e *Engine) errorResult(rule Rule, msg string) RuleResult {
	return RuleResult{
		RuleID:       rule.RuleID,
		RuleType:     rule.RuleType,
		Status:       StatusError,
		Threshold:    rule.Threshold,
		Severity:     rule.Severity,
		ErrorMessage: msg,
	}
}

// computePassRate returns the pass rate given checked and failed counts.
func computePassRate(checked, failed int) float64 {
	if checked == 0 {
		return 1.0
	}
	rate := float64(checked-failed) / float64(checked)
	// Clamp to [0, 1] to handle potential edge cases.
	return math.Max(0, math.Min(1, rate))
}

// deriveStatus determines pass/fail based on pass rate vs threshold.
func deriveStatus(passRate, threshold float64) RuleStatus {
	if passRate >= threshold {
		return StatusPass
	}
	return StatusFail
}
