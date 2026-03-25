package edqlite

import (
	"testing"
	"time"
)

func TestRecord_IsNull(t *testing.T) {
	r := Record{
		"present": "value",
		"nil_val": nil,
	}

	if r.IsNull("present") {
		t.Error("IsNull(present) = true, want false")
	}
	if !r.IsNull("nil_val") {
		t.Error("IsNull(nil_val) = false, want true")
	}
	if !r.IsNull("missing") {
		t.Error("IsNull(missing) = false, want true")
	}
}

func TestRecord_GetString(t *testing.T) {
	r := Record{
		"name":    "Alice",
		"age":     30,
		"nil_val": nil,
	}

	// Valid string field.
	s, ok := r.GetString("name")
	if !ok || s != "Alice" {
		t.Errorf("GetString(name) = (%q, %v), want (%q, true)", s, ok, "Alice")
	}

	// Non-string field.
	s, ok = r.GetString("age")
	if ok {
		t.Errorf("GetString(age) = (%q, true), want (\"\", false)", s)
	}

	// Missing field.
	s, ok = r.GetString("missing")
	if ok || s != "" {
		t.Errorf("GetString(missing) = (%q, %v), want (\"\", false)", s, ok)
	}

	// Nil field.
	s, ok = r.GetString("nil_val")
	if ok || s != "" {
		t.Errorf("GetString(nil_val) = (%q, %v), want (\"\", false)", s, ok)
	}
}

func TestRecord_GetFloat(t *testing.T) {
	r := Record{
		"f64": float64(3.14),
		"f32": float32(2.71),
		"i":   42,
		"i64": int64(100),
		"str": "not a number",
	}

	tests := []struct {
		field   string
		wantVal float64
		wantOK  bool
	}{
		{"f64", 3.14, true},
		{"f32", float64(float32(2.71)), true},
		{"i", 42.0, true},
		{"i64", 100.0, true},
		{"str", 0, false},
		{"missing", 0, false},
	}

	for _, tc := range tests {
		v, ok := r.GetFloat(tc.field)
		if ok != tc.wantOK {
			t.Errorf("GetFloat(%s) ok = %v, want %v", tc.field, ok, tc.wantOK)
		}
		if tc.wantOK && v != tc.wantVal {
			t.Errorf("GetFloat(%s) = %v, want %v", tc.field, v, tc.wantVal)
		}
	}
}

func TestRecord_GetTime(t *testing.T) {
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	r := Record{
		"ts":  now,
		"str": "2026-03-25",
	}

	// Valid time field.
	got, ok := r.GetTime("ts")
	if !ok {
		t.Fatal("GetTime(ts) ok = false, want true")
	}
	if !got.Equal(now) {
		t.Errorf("GetTime(ts) = %v, want %v", got, now)
	}

	// Non-time field.
	_, ok = r.GetTime("str")
	if ok {
		t.Error("GetTime(str) ok = true, want false")
	}

	// Missing field.
	_, ok = r.GetTime("missing")
	if ok {
		t.Error("GetTime(missing) ok = true, want false")
	}
}
