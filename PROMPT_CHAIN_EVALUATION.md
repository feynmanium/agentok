# Prompt Chain Evaluation: Agent Context Engineering Strategy

## Chain Summary

| # | Input Prompt | Output Type | Key Deliverable |
|---|---|---|---|
| 1 | "evaluate each input prompt... identify gaps" | Meta-evaluation | This document |
| 2 | "in 2026 agents context engineering from novice to world class" | Learning path | 4-stage progression (Novice→Competent→Advanced→World-class) |
| 3 | "how about for research data extraction synthesis of data" | Pipeline design | find→filter→extract→structure→synthesize→verify |
| 4 | "100s of github repo analysis and cross repo analysis and reverse context engineering" | Architecture | 8-stage system (seed→inventory→structure→deps→docs→flows→graph→agents) |
| 5 | "diagrams" | Visualization taxonomy | 3 families: system/dependency, flow, knowledge graph |
| 6 | "analysis stages strategy" | Analysis loop | 6 stages: inventory→structure→behavior→flows→risk→memory |

## Identified Gaps

### Gap 1: No grounding in Agentok Studio
The entire chain discusses building multi-agent analysis systems while sitting inside a visual multi-agent workflow builder (Agentok Studio). None of the prompts leverage:
- Existing node types (assistant, groupchat, websurfer, etc.)
- Code generation via Jinja2 templates
- AG2/AutoGen framework integration
- The visual flow canvas for designing these very workflows

### Gap 2: No concrete schemas or data models
Stages reference "Repository records," "Service records," "Dependency edges," "Flow records" but none define actual schemas — no JSON, no Pydantic models, no TypeScript interfaces.

### Gap 3: No tool/integration specifics
Mentions Tree-sitter, RAG, graph DBs, MCP servers abstractly. Never specifies which tools, how they connect, or how they integrate with Claude Code's MCP protocol.

### Gap 4: No executable agent prompts
Discusses "Extraction Agent," "Synthesis Agent," "Repo Explorer," "Cross-Repo Analyst" but never writes actual system prompts for any of them.

### Gap 5: No evaluation criteria or success metrics
"World-class" is undefined. No benchmarks, no acceptance criteria for any stage. No way to know when a stage is "done."

### Gap 6: Input prompts are underspecified
Prompts 3-6 are 2-8 words each. No constraints, no examples, no boundaries. Forces generic responses.

### Gap 7: No iteration or correction
Chain is purely additive. Never refines previous outputs. Stage 6 partially duplicates Stage 4 without acknowledging redundancy.

### Gap 8: Diagrams have no implementation path
Lists diagram types but doesn't specify any renderable format (Mermaid, D2, Graphviz).

## End-to-End Strategy

### Phase 1 — Ground in Agentok
- Map existing node types to needed analysis agents
- Define new custom nodes: `repo_scanner`, `cross_repo_analyst`, `context_reverse_engineer`
- Produce Jinja2 templates for new agent types

### Phase 2 — Define Schemas
- TypeScript interfaces (frontend) and Pydantic models (API)
- Target entities: `RepoRecord`, `ServiceRecord`, `DependencyEdge`, `FlowRecord`, `DataEntity`, `InferredContext`
- Supabase migrations for persistence

### Phase 3 — Inventory Agent
- System prompt: clone repos, extract language/size/owner/topics/last-commit, output RepoRecord JSON
- Tool bindings: `git clone`, `gh api`, filesystem scan
- Wire into Agentok as a runnable node

### Phase 4 — Structure Agent
- System prompt: use Tree-sitter to extract functions/imports/calls per repo
- Output: `DependencyEdge[]` per repo
- Tool: Tree-sitter Python bindings

### Phase 5 — Cross-Repo Analyst Agent
- System prompt: given N RepoRecords + DependencyEdges, find shared patterns, divergences, anti-patterns
- Input: aggregated Phase 3+4 outputs
- Output: comparison report + risk flags

### Phase 6 — Reverse Context Engineer Agent
- System prompt: given repo structure + README + configs, infer AGENTS.md context file
- Output: draft context file per repo (goals, architecture, conventions, constraints, workflows)

### Phase 7 — Knowledge Graph + Memory
- Store: NetworkX (local) or Neo4j (scale)
- Load all entities/edges from Phases 3-6
- Build MCP server for Claude Code queries

### Phase 8 — Diagrams as Code
- Auto-generate Mermaid diagrams from the knowledge graph
- Types: dependency graphs, flow diagrams, knowledge graph slices
- Render in Agentok frontend or export as SVG/PNG

### Phase 9 — Evaluation Loop
- Define success metrics per phase
- Build a scoring agent that audits outputs
- Example: "95% of repos have complete RepoRecords," "inferred AGENTS.md matches human review on 10 test repos"

## Next Steps
1. Pick Phase 1 or 2 to implement first
2. Write concrete agent system prompts (closes Gap 4)
3. Define Pydantic + TypeScript schemas (closes Gap 2)
4. Build one end-to-end pipeline on 5 test repos before scaling to 100+
