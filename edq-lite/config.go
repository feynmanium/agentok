package edqlite

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// RulePack represents a versioned collection of data-quality rules for a
// single dataset, loaded from YAML configuration.
type RulePack struct {
	Version     string                 `yaml:"version"`
	DatasetID   string                 `yaml:"dataset_id"`
	Entity      string                 `yaml:"entity"`
	Owner       string                 `yaml:"owner"`
	Tags        []string               `yaml:"tags,omitempty"`
	SLA         SLAConfig              `yaml:"sla,omitempty"`
	Rules       []Rule                 `yaml:"rules"`
	Environment map[string]EnvOverride `yaml:"environment,omitempty"`
}

// SLAConfig holds service-level agreement targets for a rule pack.
type SLAConfig struct {
	BatchWindowMin int    `yaml:"batch_window_min,omitempty"`
	RTLatencyMs    int    `yaml:"rt_latency_ms,omitempty"`
	AlertChannel   string `yaml:"alert_channel,omitempty"`
}

// EnvOverride allows per-environment threshold overrides and rule disabling.
type EnvOverride struct {
	ThresholdOverrides map[string]float64 `yaml:"threshold_overrides,omitempty"`
	Disabled           []string           `yaml:"disabled,omitempty"`
}

// LoadRulePack reads YAML from r, unmarshals it into a RulePack, and
// validates every rule before returning.
func LoadRulePack(r io.Reader) (*RulePack, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("config: reading input: %w", err)
	}

	var rp RulePack
	if err := yaml.Unmarshal(data, &rp); err != nil {
		return nil, fmt.Errorf("config: parsing YAML: %w", err)
	}

	if err := rp.Validate(); err != nil {
		return nil, err
	}

	return &rp, nil
}

// LoadRulePackFile opens the file at path and loads it as a RulePack.
func LoadRulePackFile(path string) (*RulePack, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("config: opening file: %w", err)
	}
	defer f.Close()

	return LoadRulePack(f)
}

// ApplyEnvironment applies the overrides for the given environment name.
// Threshold overrides update matching rules by rule ID; disabled rule IDs
// are removed from the pack entirely. If env is not found in the Environment
// map, ApplyEnvironment is a no-op.
func (rp *RulePack) ApplyEnvironment(env string) {
	override, ok := rp.Environment[env]
	if !ok {
		return
	}

	// Build an index of rules by ID for threshold patching.
	ruleIndex := make(map[string]*Rule, len(rp.Rules))
	for i := range rp.Rules {
		ruleIndex[rp.Rules[i].RuleID] = &rp.Rules[i]
	}

	for ruleID, threshold := range override.ThresholdOverrides {
		if r, found := ruleIndex[ruleID]; found {
			r.Threshold = threshold
		}
	}

	// Build a set of disabled rule IDs for efficient lookup.
	if len(override.Disabled) > 0 {
		disabled := make(map[string]struct{}, len(override.Disabled))
		for _, id := range override.Disabled {
			disabled[id] = struct{}{}
		}

		kept := rp.Rules[:0]
		for _, r := range rp.Rules {
			if _, skip := disabled[r.RuleID]; !skip {
				kept = append(kept, r)
			}
		}
		rp.Rules = kept
	}
}

// Validate checks every rule in the pack and returns the first error
// encountered, if any.
func (rp *RulePack) Validate() error {
	for i := range rp.Rules {
		if err := rp.Rules[i].Validate(); err != nil {
			return err
		}
	}
	return nil
}
