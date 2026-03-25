// Package edqlite implements EDQ Lite — a lightweight, deterministic data-quality
// rule engine for batch and selective near-real-time entry points.
//
// # Purpose
//
// Provide standardized, auditable hard-rule checks against tabular records,
// producing structured results suitable for DPA evidence, governance dashboards,
// and downstream alerting. EDQ Lite is an engine only; scheduling, triggering,
// and result distribution are owned by consuming services.
//
// # Rule Types
//
//   - Completeness: non-null thresholds on required fields
//   - Validity/Domain: membership in enumerations or reference sets
//   - Range/Bounds: min/max or vendor-aligned windows per field
//   - Uniqueness: primary/business key uniqueness per entity
//   - Timeliness: max ingestion-to-availability latency
//   - VolumeSanity: expected row-count bands
//   - Comparison: cross-record or cross-field equality/consistency checks
//
// # Execution Modes
//
//   - Batch: evaluate rule packs against full datasets (primary mode)
//   - NearRT: evaluate a rule subset against individual records with
//     circuit-breaker protection and latency budgets
//
// # Output Contract
//
// Every run produces a RunResult containing a RunHeader, per-rule RuleResult
// entries with pass/fail status and violation samples, and aggregate metrics.
// Results are JSON-serializable for storage in canonical audit locations.
//
// # Configuration
//
// Rule packs are declared in YAML with dataset metadata, SLA targets, and
// environment overrides. The engine uses semantic versioning for rule packs
// and its own runtime version.
package edqlite
