package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	edqlite "github.com/dustland/agentok/edq-lite"
)

func main() {
	configPath := flag.String("config", "", "path to rule pack YAML config (required)")
	env := flag.String("env", "", "environment name for threshold overrides (optional)")
	flag.Parse()

	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "error: -config flag is required")
		fmt.Fprintln(os.Stderr, "usage: edqlite -config <path> [-env <environment>]")
		os.Exit(2)
	}

	rp, err := edqlite.LoadRulePackFile(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(2)
	}

	if *env != "" {
		rp.ApplyEnvironment(*env)
	}

	// Generate synthetic demo dataset.
	data := generateSyntheticData(rp, 100)

	eng := edqlite.NewEngine("1.0.0")
	result, err := eng.RunBatch(rp.DatasetID, rp.Rules, data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running evaluation: %v\n", err)
		os.Exit(1)
	}

	jsonBytes, err := result.JSON()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error serializing result: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(jsonBytes))

	if result.Summary.Failed > 0 {
		os.Exit(1)
	}
}

// generateSyntheticData creates a demo dataset with fields inferred from the
// rule pack. It produces mostly valid data with a small number of realistic
// violations to exercise the rules.
func generateSyntheticData(rp *edqlite.RulePack, n int) edqlite.Dataset {
	// Collect all field names referenced by rules.
	fields := make(map[string]edqlite.RuleType)
	for _, r := range rp.Rules {
		if r.Field != "" {
			fields[r.Field] = r.RuleType
		}
		if r.Params.CompareField != "" {
			fields[r.Params.CompareField] = edqlite.RuleType("compare_target")
		}
		if r.Params.TimestampField != "" {
			fields[r.Params.TimestampField] = edqlite.RuleTimeliness
		}
	}

	// Collect allowed values for validity rules.
	allowedByField := make(map[string][]string)
	for _, r := range rp.Rules {
		if r.RuleType == edqlite.RuleValidity && len(r.Params.AllowedValues) > 0 {
			allowedByField[r.Field] = r.Params.AllowedValues
		}
	}

	// Collect range bounds.
	type rangeBound struct {
		min, max float64
	}
	rangeByField := make(map[string]rangeBound)
	for _, r := range rp.Rules {
		if r.RuleType == edqlite.RuleRange {
			rb := rangeBound{min: 0, max: 1000}
			if r.Params.Min != nil {
				rb.min = *r.Params.Min
			}
			if r.Params.Max != nil {
				rb.max = *r.Params.Max
			}
			rangeByField[r.Field] = rb
		}
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	data := make(edqlite.Dataset, n)

	for i := 0; i < n; i++ {
		rec := edqlite.Record{}

		for field, ruleType := range fields {
			switch {
			case allowedByField[field] != nil:
				vals := allowedByField[field]
				rec[field] = vals[rng.Intn(len(vals))]
			case rangeByField[field] != (rangeBound{}):
				rb := rangeByField[field]
				rec[field] = rb.min + rng.Float64()*(rb.max-rb.min)
			case ruleType == edqlite.RuleTimeliness:
				rec[field] = time.Now().Add(-time.Duration(rng.Intn(600)) * time.Second)
			case ruleType == edqlite.RuleUniqueness:
				rec[field] = fmt.Sprintf("%s-%05d", field, i)
			default:
				// Generic numeric field.
				rec[field] = rng.Float64() * 1000
			}
		}

		data[i] = rec
	}

	return data
}
