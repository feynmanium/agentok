# Unified Strategy: Agent Context Engineering for Multi-Repo Analysis

> Everything mentioned across the prompt chain — extracted, clustered, gap-filled, and organized into an executable end-to-end plan grounded in Agentok Studio.

---

## Part 1: Complete Inventory of Everything Mentioned

### A. METHODOLOGIES (how we approach the work)

| ID | Methodology | Source | Status |
|----|-------------|--------|--------|
| M1 | Context Engineering (write, select, compress, isolate) | Prompt 2 | Named, not implemented |
| M2 | Schema-guided extraction (structured templates → multi-pass) | Prompt 3 | Named, not implemented |
| M3 | Layered synthesis (per-item → themes → cross-item → narrative) | Prompt 3 | Named, not implemented |
| M4 | Reverse context engineering (infer AGENTS.md from code) | Prompt 4 | Named, not implemented |
| M5 | Iterative feedback loop (run agent → log failures → adjust context) | Prompt 2 | Named, not implemented |
| M6 | RAG-based retrieval for just-in-time context loading | Prompt 4 | Named, not implemented |
| M7 | Multi-pass extraction (locate → copy+normalize) | Prompt 3 | Named, not implemented |
| **M8** | **Visual workflow design (drag-and-drop agent flows)** | **Agentok existing** | **EXISTS in codebase** |
| **M9** | **Template-based code generation (Jinja2 → Python)** | **Agentok existing** | **EXISTS in codebase** |
| M10 | Benchmark-driven evaluation (SWE-Bench style) | Prompt 2 | Named, not implemented |

### B. PROCESSES (repeatable sequences of steps)

| ID | Process | Steps | Status |
|----|---------|-------|--------|
| P1 | Research data pipeline | find → filter → extract → structure → synthesize → verify | Described, no code |
| P2 | Repo analysis pipeline | seed → clone → scan → parse → index → query | Described, no code |
| P3 | Cross-repo comparison | gather N repos → extract per-repo views → compare → report | Described, no code |
| P4 | Context inference | scan repo → cluster files → infer norms → draft context file → meta-analyze | Described, no code |
| P5 | Skill progression | practice task → design context → run agent → log struggles → adjust → repeat | Described, no code |
| **P6** | **Agentok codegen pipeline** | **frontend flow → POST /v1/codegen → Jinja2 render → Python script → execute** | **EXISTS in codebase** |
| **P7** | **Chat execution** | **user message → /v1/chats → spawn AG2 agents → stream results** | **EXISTS in codebase** |

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
| T1 | Clone 100+ repos into local workspace | — | Not started |
| T2 | Write RepoRecord schema (Pydantic + TypeScript) | — | Not started |
| T3 | Write ServiceRecord schema | T2 | Not started |
| T4 | Write DependencyEdge schema | T2 | Not started |
| T5 | Write FlowRecord schema | T3 | Not started |
| T6 | Write DataEntity schema | T3 | Not started |
| T7 | Write InferredContext schema | T2 | Not started |
| T8 | Build Inventory Agent system prompt | T2 | Not started |
| T9 | Build Structure Agent system prompt | T4 | Not started |
| T10 | Build Cross-Repo Analyst system prompt | T2, T4 | Not started |
| T11 | Build Reverse Context Engineer system prompt | T7 | Not started |
| T12 | Build Synthesis Agent system prompt | T5, T6 | Not started |
| T13 | Build Risk/Integration Agent system prompt | T4, T5 | Not started |
| T14 | Build Scoring/Evaluation Agent system prompt | All schemas | Not started |
| T15 | Install and configure Tree-sitter Python bindings | — | Not started |
| T16 | Choose and set up graph store (NetworkX vs Neo4j) | — | Not started |
| T17 | Build MCP server for graph queries | T16, T6 | Not started |
| T18 | Create Agentok node type: repo_scanner | T8 | Not started |
| T19 | Create Agentok node type: cross_repo_analyst | T10 | Not started |
| T20 | Create Agentok node type: context_reverse_engineer | T11 | Not started |
| T21 | Create Jinja2 templates for new node types | T18-T20 | Not started |
| T22 | Supabase migrations for new entity tables | T2-T7 | Not started |
| T23 | Auto-generate Mermaid diagrams from graph data | T6, T16 | Not started |
| T24 | Define success metrics per agent | T14 | Not started |
| T25 | Run end-to-end on 5 test repos | T8-T12 | Not started |

### E. OBJECTIVES (what each piece achieves)

| ID | Objective | Measures |
|----|-----------|----------|
| O1 | Full inventory of 100+ repos with structured metadata | % repos with complete RepoRecords |
| O2 | Structural understanding of every repo (symbols, deps, calls) | % repos parsed by Tree-sitter |
| O3 | Cross-repo pattern detection (shared libs, anti-patterns, drift) | # patterns identified, false positive rate |
| O4 | Inferred context files for every repo | % repos with AGENTS.md, human review match rate |
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

### H. STAGES (ordered phases of the overall program)

| Stage | Name | Key Deliverables | Entry Criteria | Exit Criteria |
|-------|------|------------------|----------------|---------------|
| 0 | **Foundation** | Schemas defined, Tree-sitter installed, graph store chosen | Decision to start | All 6 schemas written + validated |
| 1 | **Inventory** | RepoRecords for all repos, Inventory Agent working | Stage 0 done | 95% repos have complete records |
| 2 | **Structure** | AST data + DependencyEdges per repo | Stage 1 done | 90% repos parsed, edges validated |
| 3 | **Behavior** | Requirements + docs extracted per service/repo | Stage 2 done | Key services have requirement records |
| 4 | **Flows & Data** | FlowRecords + DataEntity + lineage edges | Stage 3 done | Critical flows documented |
| 5 | **Cross-Repo Analysis** | Pattern reports, anti-pattern flags, comparison views | Stage 2 done (can parallel with 3-4) | Reports reviewed by human |
| 6 | **Reverse Context** | Inferred AGENTS.md per repo | Stage 2 done (can parallel with 3-5) | Human review match rate >80% |
| 7 | **Knowledge Graph** | All entities loaded, MCP server running, queryable | Stages 1-6 producing data | 10 test queries answered correctly |
| 8 | **Diagrams** | Auto-generated Mermaid for deps, flows, knowledge graph | Stage 7 done | Diagrams render for any repo/cross-repo view |
| 9 | **Evaluation** | Scoring agent, benchmarks, regression suite | Stage 7 done | Automated quality checks on every run |
| 10 | **Agentok Integration** | New node types, Jinja2 templates, published workflows | Stages 7-9 stable | Full pipeline runnable from Agentok UI |

### I. SEQUENCE OF EVENTS (execution timeline)

```
Week 1-2: Stage 0 (Foundation)
├── Define all 6 Pydantic + TypeScript schemas
├── Install Tree-sitter, choose graph store
├── Write system prompts for Inventory + Structure agents
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
| No evaluation methodology | Vague "benchmarks" mentioned | M10 now has concrete measures per stage (see Objectives) |
| No error handling methodology | What happens when agents fail? | Added: iterative prompt refinement loop (M5) with structured logging |

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
| No cost/resource strategy | Running 100+ repos through LLMs is expensive | Batch processing, caching, incremental updates implied in pipeline design |

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
| `context_engineer` | Context Engineer | AssistantAgent | `Icons.agent` | Infers AGENTS.md from repo structure |
| `synthesizer` | Synthesizer | AssistantAgent | `Icons.note` | Layered synthesis across extracted data |
| `graph_query` | Graph Query | ConversableAgent | `Icons.globe` | Queries knowledge graph via MCP |

### New Jinja2 Templates Needed

| Template | Generates |
|----------|-----------|
| `repo_scanner.j2` | Python script that clones repos, runs scans, outputs RepoRecord JSON |
| `structure_parser.j2` | Python script using Tree-sitter bindings |
| `cross_analyst.j2` | Python script that loads N repo records and runs comparison |
| `context_engineer.j2` | Python script that reads repo files and generates AGENTS.md |
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
