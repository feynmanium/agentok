# Agentok Studio — Project Context

## What This Is
Agentok Studio is a visual drag-and-drop IDE for building multi-agent AI workflows using AG2 (formerly AutoGen). It generates native Python code from visual flow diagrams.

## Architecture
- **Frontend**: Next.js 15 + React 18 + TypeScript + XYFlow + Zustand + Tailwind + Supabase auth
- **API**: FastAPI (Python 3.11+) + AG2 + Jinja2 templates + Supabase + Poetry
- **Database**: PostgreSQL via Supabase (with pgvector)
- **Docs site**: Docusaurus (`/website`)

## Build & Test Commands
- Frontend: `cd frontend && pnpm install && pnpm dev` (dev) / `pnpm build` (prod)
- API: `cd api && poetry install && poetry run uvicorn agentok_api.main:app --reload`
- Full stack: `docker-compose up`

## Key Directories
- `frontend/src/components/flow/node/` — Agent node React components
- `frontend/src/lib/flow.ts` — Node type registry and metadata
- `api/agentok_api/templates/` — Jinja2 code generation templates
- `api/agentok_api/services/` — Backend services (codegen, chat, tools)
- `db/` — Supabase migrations and seed data

## Agent Node Types (current)
Basic: initializer, conversable, user, assistant, groupchat, note
Advanced: captain, websurfer, deepseek
Deprecated/commented: gpt_assistant, retrieve_user, retrieve_assistant, multimodal, llava, math_user_proxy

## Code Generation Pipeline
Frontend flow (nodes+edges) → POST /v1/codegen → Jinja2 renders templates → executable Python script → AG2 runtime

## Conventions
- Node types defined in `frontend/src/lib/flow.ts` (nodeTypes, basicNodes, advancedNodes)
- Each node needs: React component, Jinja2 template, entry in flow.ts registry
- Generated code uses AG2/AutoGen classes (ConversableAgent, AssistantAgent, GroupChat, etc.)
- All API routes under `/v1/` prefix
- State management via Zustand stores
- Supabase for all persistence and auth

## Do NOT
- Modify Supabase auth flow without explicit discussion
- Add node types without corresponding Jinja2 templates
- Use raw SQL outside of migration files
- Break the codegen pipeline (always test generated Python output)

## Current Initiative: Multi-Repo Analysis Platform
See STRATEGY.md for the full plan. We are extending Agentok with:
- New agent node types for repo analysis (repo_scanner, structure_parser, cross_analyst, context_engineer, synthesizer, graph_query)
- Structured JSON output schemas for all extraction agents
- Knowledge graph memory layer with MCP server integration
- Automated diagram generation (Mermaid)
