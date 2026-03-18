# Unified Strategy: Agent Context Engineering for Multi-Repo Analysis

> Everything from the prompt chain + external audit — extracted, clustered, gap-filled, corrected for Claude Code accuracy, and organized into an executable plan grounded in both Agentok Studio and Claude Code's native primitives.

---

## Part 0: Audit Corrections Applied

The following corrections from the external audit have been incorporated throughout:

1. **AGENTS.md → CLAUDE.md**: The canonical Claude Code context file is `CLAUDE.md` (or `.claude/CLAUDE.md`), not "AGENTS.md." The chain used community naming; we now use the official mechanism everywhere.
2. **`.claude/rules/`**: Path-scoped rules with YAML frontmatter `paths:` implement "context isolation" natively — no custom tooling needed. Created: `api-templates.md`, `frontend-nodes.md`, `schema-extraction.md`.
3. **`/init`**: Claude Code's `/init` command bootstraps CLAUDE.md by analyzing the repo. This IS "reverse context engineering" — already built in. Use it as Stage 0 step 1, not as a custom agent.
4. **`--json-schema`**: Claude Code print mode supports `claude -p --json-schema <schema>` for validated structured output. Every extraction agent should use this instead of hoping for well-formed JSON.
5. **`--add-dir`**: Multi-repo sessions use `--add-dir /path/to/other/repo` — not mounting or cloning into one directory.
6. **CLI guardrails**: `--max-turns`, `--max-budget-usd`, `--tools`, `--allowedTools`, `--disallowedTools`, `--permission-mode` are critical at scale. Added to governance section.
7. **Evaluation specifics**: SWE-bench for multi-file coding tasks, SWE-Bench+ for dataset quality caveats (leakage, weak tests), SWE-EVO for long-horizon evolution. Ragas for RAG pipeline quality (faithfulness, context relevance, answer relevance).
8. **Provenance**: Every extraction must include source pointers (repo path + commit, doc ID, section/page, quote span). Added to all schemas.
9. **Skills**: Claude Code skills = progressive disclosure. Heavy procedures (full dependency rebuild, security review) should be skills, not always-loaded rules. Frontmatter always loads; body loads on demand.
10. **CLAUDE.md size**: Target <200 lines. Use rules/skills for modularization.

---

## Part 1: Complete Inventory of Everything Mentioned

### A. METHODOLOGIES (how we approach the work)

| ID | Methodology | Source | Status |
|----|-------------|--------|--------|
| M1 | Context Engineering (write, select, compress, isolate) | Prompt 2 | **Implemented via CLAUDE.md + .claude/rules/** |
| M2 | Schema-guided extraction (structured templates → multi-pass) | Prompt 3 | Named; use `--json-schema` for enforcement |
| M3 | Layered synthesis (per-item → themes → cross-item → narrative) | Prompt 3 | Named, not implemented |
| M4 | Reverse context engineering (infer CLAUDE.md from code) | Prompt 4 | **Partially built-in via `/init`**; custom agent extends it |
| M5 | Iterative feedback loop (run agent → log failures → adjust context) | Prompt 2 | Named, not implemented |
| M6 | RAG-based retrieval for just-in-time context loading | Prompt 4 | Named, not implemented |
| M7 | Multi-pass extraction (locate → copy+normalize) | Prompt 3 | Named, not implemented |
| **M8** | **Visual workflow design (drag-and-drop agent flows)** | **Agentok existing** | **EXISTS in codebase** |
| **M9** | **Template-based code generation (Jinja2 → Python)** | **Agentok existing** | **EXISTS in codebase** |
| M10 | Benchmark-driven evaluation (SWE-Bench + SWE-Bench+ + SWE-EVO + Ragas) | Prompt 2 + Audit | Named, specific tools identified |
| **M11** | **Claude Code native context (CLAUDE.md + rules + skills)** | **Audit** | **Implemented: CLAUDE.md + 3 rule files** |
| M12 | Structured CLI output (`claude -p --json-schema`) | Audit | Ready to use |
| M13 | Provenance tracking (source pointers on every extraction) | Audit | Specified in schemas |

### B. PROCESSES (repeatable sequences of steps)

| ID | Process | Steps | Status |
|----|---------|-------|--------|
| P1 | Research data pipeline | find → filter → extract (`--json-schema`) → structure → synthesize → verify (Ragas metrics) | Described, tool-specific |
| P2 | Repo analysis pipeline | seed → clone → scan → parse (Tree-sitter) → index → query | Described, no code |
| P3 | Cross-repo comparison | `--add-dir` N repos → extract per-repo views → compare → report | Described, CLI-specific |
| P4 | Context inference | `/init` → review generated CLAUDE.md → refine → split into .claude/rules/ → meta-analyze | **Partially built-in** |
| P5 | Skill progression | practice task → design context → run agent → log struggles → adjust → repeat | Described, no code |
| **P6** | **Agentok codegen pipeline** | **frontend flow → POST /v1/codegen → Jinja2 render → Python script → execute** | **EXISTS in codebase** |
| **P7** | **Chat execution** | **user message → /v1/chats → spawn AG2 agents → stream results** | **EXISTS in codebase** |
| P8 | CLI automation pipeline | `claude -p --json-schema <schema> --max-turns N --max-budget-usd X` per repo | Audit, ready to use |
| P9 | Scale governance | permission-mode + tool restrictions + snapshot versioning per batch | Audit, not implemented |

### C. FLOWS (specific end-to-end data/work movements)

| ID | Flow | From → To | Status |
|----|------|-----------|--------|
| F1 | Repo inventory flow | GitHub API → clone → scan → RepoRecord JSON | Not built |
| F2 | Structure extraction flow | Repo files → Tree-sitter → AST → symbols/imports/calls JSON | Not built |
| F3 | Dependency mapping flow | Per-repo symbols → cross-file/cross-repo edges → DependencyEdge[] | Not built |
| F4 | Docs/requirements flow | READMEs + tickets + docs → structured Requirement records | Not built |
| F5 | Data lineage flow | Code + config → DataEntity records → producer/consumer edges | Not built |
| F6 | Knowledge graph load flow | All records → graph DB nodes/edges → queryable memory | Not built |
| F7 | Diagram generation flow | Graph data → Mermaid/D2 syntax → rendered SVG/PNG | Not built |
| F8 | Synthesis flow | N extracted records → theme clustering → cross-study comparison → narrative | Not built |
| **F9** | **Agentok visual flow** | **Drag nodes → connect edges → generate code → execute** | **EXISTS** |

### D. TASKS (discrete units of work)

| ID | Task | Depends On | Status |
|----|------|------------|--------|
| **T0a** | **Run `/init` to bootstrap CLAUDE.md** | — | **DONE** |
| **T0b** | **Create .claude/rules/ with path-scoped rules** | — | **DONE** |
| **T0c** | **Define persistence model (what goes where)** | — | **DONE** (see Part 6) |
| T1 | Clone 100+ repos into local workspace | — | Not started |
| T2 | Write RepoRecord schema (Pydantic + TypeScript + JSON Schema for `--json-schema`) | — | Not started |
| T3 | Write ServiceRecord schema (with provenance fields) | T2 | Not started |
| T4 | Write DependencyEdge schema (with provenance fields) | T2 | Not started |
| T5 | Write FlowRecord schema (with provenance fields) | T3 | Not started |
| T6 | Write DataEntity schema (with provenance fields) | T3 | Not started |
| T7 | Write InferredContext schema (CLAUDE.md-compatible output) | T2 | Not started |
| T8 | Build Inventory Agent system prompt + CLI recipe | T2 | Not started |
| T9 | Build Structure Agent system prompt + CLI recipe | T4 | Not started |
| T10 | Build Cross-Repo Analyst system prompt + CLI recipe | T2, T4 | Not started |
| T11 | Build Reverse Context Engineer system prompt (extends `/init`) | T7 | Not started |
| T12 | Build Synthesis Agent system prompt + CLI recipe | T5, T6 | Not started |
| T13 | Build Risk/Integration Agent system prompt + CLI recipe | T4, T5 | Not started |
| T14 | Build Scoring/Evaluation Agent (SWE-bench tasks + Ragas metrics) | All schemas | Not started |
| T15 | Install and configure Tree-sitter Python bindings | — | Not started |
| T16 | Choose and set up graph store (NetworkX local → Neo4j scale) | — | Not started |
| T17 | Build MCP server for graph queries | T16, T6 | Not started |
| T18 | Create Agentok node type: repo_scanner | T8 | Not started |
| T19 | Create Agentok node type: cross_repo_analyst | T10 | Not started |
| T20 | Create Agentok node type: context_reverse_engineer | T11 | Not started |
| T21 | Create Jinja2 templates for new node types | T18-T20 | Not started |
| T22 | Supabase migrations for new entity tables | T2-T7 | Not started |
| T23 | Auto-generate Mermaid diagrams from graph data | T6, T16 | Not started |
| T24 | Define success metrics per agent + regression suite | T14 | Not started |
| T25 | Run end-to-end on 5 test repos | T8-T12 | Not started |
| **T26** | **Define tool-permission policy for scale runs** | — | Not started |
| **T27** | **Build snapshot/versioning for repo analysis batches** | T1 | Not started |
| **T28** | **Write 10 CLI recipes with exact flags** | T8-T13 | Not started |
| **T29** | **Build MCP connectors for external docs (Drive/Jira/Confluence)** | T17 | Not started |

### E. OBJECTIVES (what each piece achieves)

| ID | Objective | Measures |
|----|-----------|----------|
| O1 | Full inventory of 100+ repos with structured metadata | % repos with complete RepoRecords |
| O2 | Structural understanding of every repo (symbols, deps, calls) | % repos parsed by Tree-sitter |
| O3 | Cross-repo pattern detection (shared libs, anti-patterns, drift) | # patterns identified, false positive rate |
| O4 | Inferred CLAUDE.md for every repo | % repos with CLAUDE.md, human review match rate |
| O5 | Queryable knowledge graph across all repos | Query response accuracy on test questions |
| O6 | Auto-generated diagrams for any repo or cross-repo view | % diagrams that render correctly |
| O7 | End-to-end reproducible pipeline | Time from new repo → full analysis complete |
| O8 | Reusable Agentok workflows for all analysis types | # workflows published, re-execution success rate |

### F. GOALS (higher-level outcomes)

| ID | Goal | Timeframe |
|----|------|-----------|
| G1 | "Novice → World-class" in agent context engineering | Ongoing progression |
| G2 | Analyze 100+ repos with agents, not manual effort | 3-6 months |
| G3 | Cross-repo architectural intelligence on demand | After G2 |
| G4 | Reverse-engineer context for any new repo automatically | After G2 |
| G5 | Research-grade data extraction and synthesis capability | Parallel with G2 |
| G6 | Agentok Studio as the visual IDE for all analysis workflows | After G2 |

### G. STRATEGIES (approaches to achieve goals)

| ID | Strategy | Supports Goals |
|----|----------|----------------|
| S1 | Ground everything in Agentok's existing architecture | G6 |
| S2 | Schema-first development (define data models before agents) | G2, G5 |
| S3 | Start small (5 repos), prove pipeline, then scale to 100+ | G2 |
| S4 | Agent-per-concern (inventory, structure, analysis, synthesis as separate agents) | G2, G3 |
| S5 | Knowledge graph as central memory across all agents | G3, G4 |
| S6 | Diagrams-as-code for all visualizations | G3, G6 |
| S7 | Continuous evaluation loop (scoring agent audits other agents) | G1 |
| S8 | MCP server as bridge between graph memory and Claude Code | G3, G4 |
| **S9** | **Claude Code native primitives as default building blocks** (CLAUDE.md, rules, skills, `--json-schema`, `--add-dir`) | **All goals** |
| **S10** | **CLI guardrails for cost/safety** (`--max-turns`, `--max-budget-usd`, `--permission-mode`, tool restrictions) | **G2** |
| **S11** | **Provenance-first extraction** (every record traces to source repo+commit+file+line) | **G2, G5** |
| **S12** | **Multi-tier evaluation** (SWE-bench for coding, Ragas for RAG, golden tasks for custom, human review gates) | **G1** |

### H. STAGES (ordered phases of the overall program)

| Stage | Name | Key Deliverables | Entry Criteria | Exit Criteria |
|-------|------|------------------|----------------|---------------|
| 0 | **Foundation** | CLAUDE.md + rules (**DONE**), schemas defined, Tree-sitter installed, graph store chosen, CLI recipes drafted | Decision to start | All 6 schemas written + validated, 10 CLI recipes documented |
| 1 | **Inventory** | RepoRecords for all repos, Inventory Agent working | Stage 0 done | 95% repos have complete records |
| 2 | **Structure** | AST data + DependencyEdges per repo | Stage 1 done | 90% repos parsed, edges validated |
| 3 | **Behavior** | Requirements + docs extracted per service/repo | Stage 2 done | Key services have requirement records |
| 4 | **Flows & Data** | FlowRecords + DataEntity + lineage edges | Stage 3 done | Critical flows documented |
| 5 | **Cross-Repo Analysis** | Pattern reports, anti-pattern flags, comparison views | Stage 2 done (can parallel with 3-4) | Reports reviewed by human |
| 6 | **Reverse Context** | Inferred CLAUDE.md per repo (extends `/init`) | Stage 2 done (can parallel with 3-5) | Human review match rate >80% |
| 7 | **Knowledge Graph** | All entities loaded, MCP server running, queryable | Stages 1-6 producing data | 10 test queries answered correctly |
| 8 | **Diagrams** | Auto-generated Mermaid for deps, flows, knowledge graph | Stage 7 done | Diagrams render for any repo/cross-repo view |
| 9 | **Evaluation** | Scoring agent, benchmarks, regression suite | Stage 7 done | Automated quality checks on every run |
| 10 | **Agentok Integration** | New node types, Jinja2 templates, published workflows | Stages 7-9 stable | Full pipeline runnable from Agentok UI |

### I. SEQUENCE OF EVENTS (execution timeline)

```
Week 1-2: Stage 0 (Foundation)
├── [DONE] Bootstrap CLAUDE.md via /init + refine
├── [DONE] Create .claude/rules/ with path-scoped rules
├── Define all 6 Pydantic + TypeScript + JSON Schema files
├── Install Tree-sitter, choose graph store
├── Write 10 CLI recipes with exact flags
├── Write system prompts for Inventory + Structure agents
├── Define tool-permission policy (T26)
└── Clone 5 test repos

Week 3-4: Stage 1 (Inventory)
├── Run Inventory Agent on 5 test repos
├── Validate RepoRecord outputs
├── Fix agent prompt based on failures
└── Scale to full repo set

Week 5-7: Stage 2 (Structure)
├── Run Structure Agent on all repos
├── Build DependencyEdge outputs
├── Validate against manual spot-checks
└── Begin cross-repo edge aggregation

Week 6-8: Stages 3+4 (Behavior + Flows) [parallel with late Stage 2]
├── Extract requirements from docs/READMEs
├── Reconstruct critical flows
├── Build DataEntity + lineage records
└── Validate against known architectures

Week 7-9: Stages 5+6 (Cross-Repo + Reverse Context) [parallel]
├── Run Cross-Repo Analyst on aggregated data
├── Run Reverse Context Engineer per repo
├── Human review of outputs
└── Iterate prompts based on review

Week 9-11: Stage 7 (Knowledge Graph)
├── Load all entities into graph store
├── Build MCP server
├── Test queries from Claude Code
└── Iterate on query patterns

Week 11-13: Stages 8+9 (Diagrams + Evaluation) [parallel]
├── Auto-generate Mermaid from graph
├── Build scoring agent
├── Define regression test suite
└── Run full pipeline end-to-end

Week 13-16: Stage 10 (Agentok Integration)
├── Create new node types in frontend
├── Create Jinja2 templates in API
├── Build visual workflow for full pipeline
└── Test, document, publish
```

---

## Part 2: Gap Analysis by Cluster

### Cluster A: METHODOLOGIES — Gaps Filled

| Gap | What Was Missing | Now Added |
|-----|-----------------|-----------|
| No connection to Agentok's existing methodology | M8, M9 were never mentioned | Mapped as foundation; new agents build ON existing codegen pipeline |
| No evaluation methodology | Vague "benchmarks" mentioned | M10 now specifies SWE-bench + SWE-Bench+ + SWE-EVO + Ragas |
| No error handling methodology | What happens when agents fail? | M5 with structured logging + audit logs (Part 8) |
| **No Claude Code native primitives** | **Entire chain ignored CLAUDE.md, rules, skills, CLI flags** | **M11-M13 added; CLAUDE.md + 3 rule files created; 10 CLI recipes written** |
| **No provenance methodology** | **Extractions had no source tracking** | **M13: every record must include repo+commit+file+line provenance** |

### Cluster B: PROCESSES — Gaps Filled

| Gap | What Was Missing | Now Added |
|-----|-----------------|-----------|
| P1-P5 existed only as prose | No executable steps | Each now maps to specific Tasks (T1-T25) |
| No process for scaling from 5 → 100+ repos | Implied but never designed | S3 strategy + Stage 1 exit criteria handles this |
| No process for human review/approval gates | Agents run unsupervised | Stage exit criteria now require human validation |

### Cluster C: FLOWS — Gaps Filled

| Gap | What Was Missing | Now Added |
|-----|-----------------|-----------|
| No data format specifications | "JSON" mentioned generically | Schemas (T2-T7) define exact shapes |
| No error/retry flows | What if clone fails? Parse fails? | Each flow needs error handling — added as requirement in Stage 0 |
| F9 (Agentok's existing flow) never connected to F1-F8 | Two separate worlds | Stage 10 explicitly bridges them via new node types + templates |

### Cluster D: TASKS — Gaps Filled

| Gap | What Was Missing | Now Added |
|-----|-----------------|-----------|
| No dependency ordering | Tasks listed flat | Dependency column added (T9 depends on T4, etc.) |
| No agent prompt tasks | Talked about agents, never wrote prompts | T8-T14 are explicit prompt-writing tasks |
| No infrastructure tasks | Graph DB, MCP server assumed to exist | T15-T17 cover tooling setup |
| No Agentok extension tasks | Never planned to extend the platform | T18-T22 cover new nodes, templates, migrations |

### Cluster E-F: OBJECTIVES & GOALS — Gaps Filled

| Gap | What Was Missing | Now Added |
|-----|-----------------|-----------|
| "World-class" undefined | No measurable definition | Broken into 8 measurable objectives (O1-O8) |
| No timeframes | Everything was "eventually" | Goals now have relative timeframes |
| No intermediate milestones | Big goals with no checkpoints | Stage exit criteria serve as milestones |

### Cluster G: STRATEGIES — Gaps Filled

| Gap | What Was Missing | Now Added |
|-----|-----------------|-----------|
| No prioritization | 8 strategies, no order | S2 (schema-first) and S3 (start small) are prerequisites for all others |
| No risk mitigation strategy | What if Tree-sitter can't parse a language? | Fallback: regex-based extraction for unsupported languages |
| No cost/resource strategy | Running 100+ repos through LLMs is expensive | **S10: explicit CLI guardrails (`--max-budget-usd`, `--max-turns`) + batch recipe (Recipe 10)** |
| **No governance strategy** | **Permissions, secrets, tool restrictions ignored** | **S10 + Part 8 governance table + T26 (permission policy)** |
| **No provenance strategy** | **No way to trace claims to sources** | **S11: provenance-first extraction, mandatory on all schemas** |

### Cluster H-I: STAGES & SEQUENCE — Gaps Filled

| Gap | What Was Missing | Now Added |
|-----|-----------------|-----------|
| Stages 4 and 6 (from original chain) were redundant | Duplicated without acknowledging | Merged into unified 11-stage sequence (0-10) |
| No parallelism identified | Everything seemed sequential | Stages 3-4 and 5-6 and 8-9 can run in parallel |
| No entry/exit criteria | No way to know when a stage is "done" | Every stage now has explicit criteria |
| No timeline | Unbounded | 16-week indicative timeline added |

---

## Part 3: Agentok Platform Mapping

How each new capability maps to Agentok Studio's existing architecture:

### New Node Types Needed

| Node ID | Name | AG2 Class | Icon | Purpose |
|---------|------|-----------|------|---------|
| `repo_scanner` | Repo Scanner | ConversableAgent | `Icons.search` | Clones + inventories repos → RepoRecord |
| `structure_parser` | Structure Parser | ConversableAgent | `Icons.robot` | Tree-sitter AST extraction → DependencyEdge[] |
| `cross_analyst` | Cross-Repo Analyst | AssistantAgent | `Icons.group` | Compares N repos → pattern report |
| `context_engineer` | Context Engineer | AssistantAgent | `Icons.agent` | Infers CLAUDE.md from repo structure (extends `/init`) |
| `synthesizer` | Synthesizer | AssistantAgent | `Icons.note` | Layered synthesis across extracted data |
| `graph_query` | Graph Query | ConversableAgent | `Icons.globe` | Queries knowledge graph via MCP |

### New Jinja2 Templates Needed

| Template | Generates |
|----------|-----------|
| `repo_scanner.j2` | Python script that clones repos, runs scans, outputs RepoRecord JSON |
| `structure_parser.j2` | Python script using Tree-sitter bindings |
| `cross_analyst.j2` | Python script that loads N repo records and runs comparison |
| `context_engineer.j2` | Python script that reads repo files and generates CLAUDE.md (extends `/init` output) |
| `graph_query.j2` | Python script that queries NetworkX/Neo4j |

### Example Agentok Visual Workflow

```
[Initializer] → [Repo Scanner] → [Structure Parser] → [Cross-Repo Analyst]
                                         ↓                       ↓
                                  [Context Engineer]      [Synthesizer]
                                         ↓                       ↓
                                   [Graph Query] ←──────────────┘
                                         ↓
                                   [Diagram Gen]
```

This entire pipeline would be designable in Agentok's drag-and-drop canvas, generating executable Python via the existing codegen system.

---

## Part 4: Priority Matrix

| Priority | What | Why First |
|----------|------|-----------|
| **P0** | Schemas (T2-T7) | Everything depends on data shapes |
| **P0** | Tree-sitter setup (T15) | Core parsing capability |
| **P1** | Inventory Agent prompt (T8) | First agent to produce real data |
| **P1** | Structure Agent prompt (T9) | Second agent, feeds everything downstream |
| **P2** | Graph store setup (T16) | Needed before cross-repo work |
| **P2** | Cross-Repo Analyst prompt (T10) | First high-value output |
| **P2** | Reverse Context Engineer prompt (T11) | Second high-value output |
| **P3** | MCP server (T17) | Bridges graph to Claude Code |
| **P3** | Agentok node types (T18-T21) | Makes everything visual/reusable |
| **P4** | Evaluation agent (T14) | Quality assurance layer |
| **P4** | Diagrams (T23) | Visualization layer |

---

## Part 5: Redesigned Prompt Chain v2 (Audit-Corrected)

The original prompt chain used 2-8 word inputs and got generic outputs. Below are replacement prompts with proper constraints, output contracts, and persistence decisions.

### Prompt 1 (replaces "novice to world class"):

**Input:**
> "Design a 12-week training path to become an expert Claude Code CLI operator focused on context engineering and reliability. Assume 5 hours/week. Deliverables: (1) a reusable CLAUDE.md + .claude/rules/ template, (2) 10 CLI recipes with exact commands and flags citing the official CLI reference, (3) an evaluation harness plan covering coding tasks (SWE-bench style) and RAG/extraction tasks (Ragas metrics). Constraints: local Mac, no cloud infra, budget-conscious."

**Output contract:**
- Week-by-week plan with specific exercises
- 10 CLI recipes as executable shell commands
- Evaluation harness specification with task suites and scoring

**What to persist:** CLAUDE.md template, CLI recipe file, eval harness config

---

### Prompt 2 (replaces "research data extraction synthesis"):

**Input:**
> "I need a data-extraction workflow for research documents using Claude Code. Create: (1) a JSON Schema for extracted records that works with `claude -p --json-schema`, (2) a 2-pass extraction method where pass 1 locates evidence passages and pass 2 normalizes into the schema, (3) a synthesis method that only consumes extracted JSON and includes provenance pointers (doc_id, section, page, quote_span), (4) an evaluation plan using Ragas metrics (faithfulness, context relevance, answer relevance). Format: executable CLI commands."

**Output contract:**
- JSON Schema file (`.json`)
- Two prompt templates (locate pass + normalize pass)
- Synthesis prompt template
- Ragas evaluation script skeleton
- Provenance field specification

**What to persist:** Schema in `api/agentok_api/schemas/`, prompts in `.claude/rules/` or skills

---

### Prompt 3 (replaces "100s of repos + cross-repo"):

**Input:**
> "I will analyze 200 GitHub repos locally using Claude Code. Design the full architecture and operating procedure. Must include: (1) repo ingestion via `--add-dir`, (2) structural indexing with Tree-sitter, (3) cross-repo queries and comparison outputs, (4) tool-permission policy using `--permission-mode` + `--allowedTools` + `--disallowedTools`, (5) cost controls via `--max-turns` and `--max-budget-usd`, (6) snapshot/versioning strategy, (7) a 'reverse context engineering' procedure that extends `/init` to generate and refine CLAUDE.md + .claude/rules/ per repo, (8) diagram generation in Mermaid format. All extraction outputs must validate against JSON Schemas."

**Output contract:**
- Architecture diagram (Mermaid)
- CLI session patterns (exact commands)
- Permission policy document
- Cost budget per-repo and per-batch
- Snapshot/versioning procedure
- Per-repo CLAUDE.md generation procedure

**What to persist:** Architecture in STRATEGY.md, CLI patterns in recipes, permissions in `.claude/settings.json`

---

### Prompt 4 (replaces "diagrams"):

**Input:**
> "For the multi-repo analysis system, define diagram specifications for: (1) service dependency graph (nodes=repos/services, edges=calls/imports/events), (2) data lineage flow (producers→entities→consumers), (3) knowledge graph slice (repos↔services↔docs↔requirements). For each: specify Mermaid syntax, which stage artifacts feed it, and how to auto-generate from the JSON Schema outputs. Provide a working Mermaid example for each."

**Output contract:**
- 3 Mermaid diagram templates with placeholder data
- Mapping: stage artifact → diagram type → generation script
- Auto-generation script skeleton (Python, reads JSON, outputs .mmd)

---

### Prompt 5 (replaces "analysis stages strategy"):

**Input:**
> "Define the evaluation and quality gates for the multi-repo analysis pipeline. For each of the 10 stages, specify: (1) what 'done' looks like (quantitative), (2) automated checks that run before advancing, (3) human review gates (what needs manual sign-off), (4) regression tests (what to re-run when prompts change). Include specific tool choices: SWE-bench for coding eval, Ragas for extraction eval, and custom golden-task sets for cross-repo analysis."

**Output contract:**
- Per-stage quality gate table
- Automated check scripts (skeleton)
- Human review checklist
- Regression test suite design

---

## Part 6: Persistence Model ("What Goes Where")

The audit identified that the chain repeatedly gestured at "memory" without defining what is stored where. This is the concrete split:

| Tier | What | Where | When Loaded |
|------|------|-------|-------------|
| **Always-loaded rules** | Project purpose, build/test commands, coding conventions, security rules, do/don'ts | `CLAUDE.md` (<200 lines) | Every session, automatically |
| **Path-scoped rules** | Template rules, node rules, schema rules | `.claude/rules/*.md` with YAML `paths:` frontmatter | When matching files are opened |
| **Skills (progressive disclosure)** | Heavy workflows: full dependency rebuild, security review, cross-repo comparison | `.claude/skills/` (future) | Frontmatter always; body on demand |
| **Structured artifacts** | RepoRecords, DependencyEdges, FlowRecords, DataEntities, InferredContexts | JSON files validated by `--json-schema`, stored in `data/` dir | On demand via agent queries |
| **Graph memory** | All entities + edges as queryable graph | NetworkX (local pickle) → Neo4j (scale) | Via MCP server |
| **External connectors** | Docs, tickets, research PDFs from Drive/Jira/Confluence | MCP servers | On demand |
| **CLI session state** | Auto-memory notes across sessions | Claude Code auto-memory (first 200 lines loaded) | Every session, automatically |
| **Batch audit logs** | What agent ran, what context was loaded, what it produced | JSON logs per run in `data/logs/` | For debugging and evaluation |

---

## Part 7: CLI Recipes (10 Exact Commands)

These are the concrete Claude Code CLI patterns for the pipeline:

### Recipe 1: Bootstrap a repo's context
```bash
cd /path/to/repo && claude /init
# Then review and refine the generated CLAUDE.md
```

### Recipe 2: Inventory a single repo (structured output)
```bash
claude -p "Analyze this repository and output a RepoRecord." \
  --json-schema ./schemas/repo_record.json \
  --max-turns 5 \
  --max-budget-usd 0.50
```

### Recipe 3: Multi-repo session
```bash
claude --add-dir /repos/service-a --add-dir /repos/service-b \
  -p "Compare these two repos: architecture, shared dependencies, divergent patterns."
```

### Recipe 4: Structure extraction with Tree-sitter
```bash
claude -p "Use tree-sitter to parse all Python files in this repo. Output DependencyEdge records." \
  --json-schema ./schemas/dependency_edge.json \
  --max-turns 10 \
  --max-budget-usd 1.00
```

### Recipe 5: Cross-repo pattern detection
```bash
claude --add-dir /repos/svc-{a,b,c,d,e} \
  -p "Identify shared architectural patterns and anti-patterns across these 5 repos." \
  --max-turns 15 \
  --max-budget-usd 2.00
```

### Recipe 6: Reverse context engineering (extends /init)
```bash
claude -p "Generate a CLAUDE.md for this repo. Include: purpose, architecture, conventions, constraints, key modules, build/test commands, do/don't rules." \
  --max-turns 8 \
  --max-budget-usd 0.75
```

### Recipe 7: Research document extraction
```bash
claude -p "Extract structured records from this PDF. Use 2-pass: (1) locate evidence passages, (2) normalize into schema." \
  --json-schema ./schemas/research_record.json \
  --max-turns 6 \
  --max-budget-usd 0.50
```

### Recipe 8: Synthesis from extracted records
```bash
claude -p "Synthesize these extracted records into a comparison report. Cite provenance (file, section, page) for every claim." \
  --max-turns 10 \
  --max-budget-usd 1.00
```

### Recipe 9: Diagram generation
```bash
claude -p "Generate a Mermaid dependency diagram from these DependencyEdge JSON records. Output valid Mermaid syntax only." \
  --max-turns 3 \
  --max-budget-usd 0.25
```

### Recipe 10: Batch run with safety guardrails
```bash
for repo in /repos/*/; do
  claude -p "Analyze this repo and output a RepoRecord." \
    --json-schema ./schemas/repo_record.json \
    --max-turns 5 \
    --max-budget-usd 0.50 \
    --permission-mode plan \
    --allowedTools "Read,Glob,Grep,Bash(git log:*),Bash(wc:*)" \
    > "data/inventory/$(basename $repo).json"
done
```

---

## Part 8: Governance & Scale Controls

| Control | Mechanism | When Applied |
|---------|-----------|--------------|
| **Cost ceiling per repo** | `--max-budget-usd 0.50-2.00` | Every batch run |
| **Turn limits** | `--max-turns 5-15` depending on task | Every CLI invocation |
| **Tool restrictions** | `--allowedTools` whitelist per task type | Batch runs (Recipe 10) |
| **Permission mode** | `--permission-mode plan` for automated runs | Unattended batch execution |
| **Snapshot versioning** | Git tag + JSON export per batch run | After each batch completes |
| **Secrets isolation** | Never pass `.env` or credentials to analysis agents | Always |
| **Human review gates** | Stage exit criteria require spot-check | Between stages |
| **Audit logging** | JSON log per agent run (input, context loaded, output, cost) | Every run |
