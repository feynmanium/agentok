---
paths:
  - "api/agentok_api/models/**"
  - "api/agentok_api/schemas/**"
  - "frontend/src/types/**"
---

# Schema & Extraction Rules

- All extraction schemas must be defined as BOTH Pydantic models (API) and TypeScript interfaces (frontend)
- Every schema must include provenance fields: source_repo, source_file, source_commit, extracted_at
- Schemas must be compatible with `claude -p --json-schema` for CLI validation
- Use enum types for categorical fields (language, license_type, dependency_kind)
- All cross-entity references use string IDs, not nested objects
- Schema changes require corresponding Supabase migration in `db/`
