package edqlite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validYAML = `
version: "1.0.0"
dataset_id: "test_dataset"
entity: "test_entity"
owner: "test_owner"
rules:
  - rule_id: "R001"
    dataset_id: "test_dataset"
    rule_type: "completeness"
    field: "email"
    severity: "high"
    threshold: 0.99
    params: {}
  - rule_id: "R002"
    dataset_id: "test_dataset"
    rule_type: "completeness"
    field: "name"
    severity: "medium"
    threshold: 0.95
    params: {}
`

func TestLoadRulePack(t *testing.T) {
	rp, err := LoadRulePack(strings.NewReader(validYAML))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rp.Version != "1.0.0" {
		t.Errorf("version = %q, want %q", rp.Version, "1.0.0")
	}
	if rp.DatasetID != "test_dataset" {
		t.Errorf("dataset_id = %q, want %q", rp.DatasetID, "test_dataset")
	}
	if rp.Entity != "test_entity" {
		t.Errorf("entity = %q, want %q", rp.Entity, "test_entity")
	}
	if rp.Owner != "test_owner" {
		t.Errorf("owner = %q, want %q", rp.Owner, "test_owner")
	}
	if len(rp.Rules) != 2 {
		t.Fatalf("got %d rules, want 2", len(rp.Rules))
	}
	if rp.Rules[0].RuleID != "R001" {
		t.Errorf("rules[0].rule_id = %q, want %q", rp.Rules[0].RuleID, "R001")
	}
	if rp.Rules[0].RuleType != RuleCompleteness {
		t.Errorf("rules[0].rule_type = %q, want %q", rp.Rules[0].RuleType, RuleCompleteness)
	}
	if rp.Rules[0].Field != "email" {
		t.Errorf("rules[0].field = %q, want %q", rp.Rules[0].Field, "email")
	}
	if rp.Rules[0].Threshold != 0.99 {
		t.Errorf("rules[0].threshold = %v, want 0.99", rp.Rules[0].Threshold)
	}
	if rp.Rules[1].RuleID != "R002" {
		t.Errorf("rules[1].rule_id = %q, want %q", rp.Rules[1].RuleID, "R002")
	}
	if rp.Rules[1].Severity != SeverityMedium {
		t.Errorf("rules[1].severity = %q, want %q", rp.Rules[1].Severity, SeverityMedium)
	}
}

func TestLoadRulePack_InvalidYAML(t *testing.T) {
	malformed := `
version: "1.0.0"
  bad_indent: [
    not valid yaml
`
	_, err := LoadRulePack(strings.NewReader(malformed))
	if err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
}

func TestLoadRulePack_ValidationError(t *testing.T) {
	// Missing field for completeness rule triggers validation error.
	yamlMissingField := `
version: "1.0.0"
dataset_id: "test_dataset"
entity: "test_entity"
owner: "test_owner"
rules:
  - rule_id: "R001"
    dataset_id: "test_dataset"
    rule_type: "completeness"
    severity: "high"
    threshold: 0.99
    params: {}
`
	_, err := LoadRulePack(strings.NewReader(yamlMissingField))
	if err == nil {
		t.Fatal("expected validation error for missing field, got nil")
	}
}

func TestApplyEnvironment(t *testing.T) {
	yamlWithEnv := `
version: "1.0.0"
dataset_id: "test_dataset"
entity: "test_entity"
owner: "test_owner"
rules:
  - rule_id: "R001"
    dataset_id: "test_dataset"
    rule_type: "completeness"
    field: "email"
    severity: "high"
    threshold: 0.99
    params: {}
  - rule_id: "R002"
    dataset_id: "test_dataset"
    rule_type: "completeness"
    field: "name"
    severity: "medium"
    threshold: 0.95
    params: {}
  - rule_id: "R003"
    dataset_id: "test_dataset"
    rule_type: "completeness"
    field: "phone"
    severity: "low"
    threshold: 0.90
    params: {}
environment:
  staging:
    threshold_overrides:
      R001: 0.90
    disabled:
      - R003
`
	rp, err := LoadRulePack(strings.NewReader(yamlWithEnv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rp.ApplyEnvironment("staging")

	// R001 threshold should be overridden.
	if len(rp.Rules) != 2 {
		t.Fatalf("got %d rules after apply, want 2", len(rp.Rules))
	}

	var r001 *Rule
	for i := range rp.Rules {
		if rp.Rules[i].RuleID == "R001" {
			r001 = &rp.Rules[i]
		}
	}
	if r001 == nil {
		t.Fatal("R001 not found after apply")
	}
	if r001.Threshold != 0.90 {
		t.Errorf("R001 threshold = %v, want 0.90", r001.Threshold)
	}

	// R003 should be removed.
	for _, r := range rp.Rules {
		if r.RuleID == "R003" {
			t.Error("R003 should have been disabled/removed")
		}
	}
}

func TestApplyEnvironment_Unknown(t *testing.T) {
	rp, err := LoadRulePack(strings.NewReader(validYAML))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	before := len(rp.Rules)
	rp.ApplyEnvironment("nonexistent")

	if len(rp.Rules) != before {
		t.Errorf("rules count changed from %d to %d on unknown env", before, len(rp.Rules))
	}
}

func TestLoadRulePackFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")

	if err := os.WriteFile(path, []byte(validYAML), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	rp, err := LoadRulePackFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rp.Version != "1.0.0" {
		t.Errorf("version = %q, want %q", rp.Version, "1.0.0")
	}
	if len(rp.Rules) != 2 {
		t.Errorf("got %d rules, want 2", len(rp.Rules))
	}
}
