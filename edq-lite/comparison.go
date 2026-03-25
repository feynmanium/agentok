package edqlite

import "fmt"

// evaluateComparison evaluates a comparison rule against every record in the dataset.
// It compares the primary field against either another field (CompareField) or a
// literal value (CompareValue) using the operator specified in CompareOp.
func (e *Engine) evaluateComparison(rule Rule, data Dataset) RuleResult {
	result := RuleResult{
		RuleID:    rule.RuleID,
		RuleType:  RuleComparison,
		Threshold: rule.Threshold,
		Severity:  rule.Severity,
	}

	var checked, failed int

	for i, rec := range data {
		if rec.IsNull(rule.Field) {
			continue
		}
		checked++

		var rhs any
		if rule.Params.CompareField != "" {
			// Cross-field comparison: compare against another field in the same record.
			if rec.IsNull(rule.Params.CompareField) {
				// Treat a null comparand as a failure.
				failed++
				if len(result.SampleViolations) < e.maxViolationSamples {
					result.SampleViolations = append(result.SampleViolations, Violation{
						RecordIndex: i,
						Field:       rule.Field,
						Value:       rec[rule.Field],
						Reason: fmt.Sprintf("compare field %q is null",
							rule.Params.CompareField),
					})
				}
				continue
			}
			rhs = rec[rule.Params.CompareField]
		} else {
			// Cross-value comparison: compare against a literal.
			rhs = rule.Params.CompareValue
		}

		lhs := rec[rule.Field]
		if !compareValues(lhs, rhs, rule.Params.CompareOp) {
			failed++
			if len(result.SampleViolations) < e.maxViolationSamples {
				result.SampleViolations = append(result.SampleViolations, Violation{
					RecordIndex: i,
					Field:       rule.Field,
					Value:       lhs,
					Reason: fmt.Sprintf("%v %s %v is false",
						lhs, rule.Params.CompareOp, rhs),
				})
			}
		}
	}

	result.Checked = checked
	result.Failed = failed

	if checked == 0 {
		result.PassRate = 1.0
	} else {
		result.PassRate = float64(checked-failed) / float64(checked)
	}

	if result.PassRate >= rule.Threshold {
		result.Status = StatusPass
	} else {
		result.Status = StatusFail
	}

	totalN := len(data)
	result.ObservedStats.TotalN = &totalN

	return result
}

// compareValues compares two values using the given comparison operator.
// It first attempts a numeric (float64) comparison, then falls back to
// lexicographic string comparison.
func compareValues(a, b any, op ComparisonOp) bool {
	// Try numeric comparison first.
	fa, aOK := toFloat64(a)
	fb, bOK := toFloat64(b)
	if aOK && bOK {
		switch op {
		case CompareEqual:
			return fa == fb
		case CompareNotEqual:
			return fa != fb
		case CompareGreaterThan:
			return fa > fb
		case CompareLessThan:
			return fa < fb
		case CompareGreaterEqual:
			return fa >= fb
		case CompareLessEqual:
			return fa <= fb
		}
		return false
	}

	// Fall back to string comparison.
	sa := fmt.Sprintf("%v", a)
	sb := fmt.Sprintf("%v", b)
	switch op {
	case CompareEqual:
		return sa == sb
	case CompareNotEqual:
		return sa != sb
	case CompareGreaterThan:
		return sa > sb
	case CompareLessThan:
		return sa < sb
	case CompareGreaterEqual:
		return sa >= sb
	case CompareLessEqual:
		return sa <= sb
	}
	return false
}

// toFloat64 attempts to convert an arbitrary value to float64.
func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	}
	return 0, false
}
