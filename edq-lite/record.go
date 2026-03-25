package edqlite

import "time"

// Record represents a single data record to be checked.
// Fields are stored as a map of field name to value.
// Nil values represent NULL/missing fields.
type Record map[string]any

// IsNull returns true if the field is missing or nil.
func (r Record) IsNull(field string) bool {
	v, exists := r[field]
	return !exists || v == nil
}

// GetString returns the string value of a field, or "" if missing/non-string.
func (r Record) GetString(field string) (string, bool) {
	v, ok := r[field]
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// GetFloat returns the float64 value of a field.
func (r Record) GetFloat(field string) (float64, bool) {
	v, ok := r[field]
	if !ok || v == nil {
		return 0, false
	}
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

// GetTime returns the time.Time value of a field.
func (r Record) GetTime(field string) (time.Time, bool) {
	v, ok := r[field]
	if !ok || v == nil {
		return time.Time{}, false
	}
	t, ok := v.(time.Time)
	return t, ok
}

// Dataset is a slice of records for batch evaluation.
type Dataset []Record
