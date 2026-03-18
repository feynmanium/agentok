---
paths:
  - "api/agentok_api/templates/**"
  - "api/agentok_api/services/codegen*"
---

# Code Generation Rules

- Every Jinja2 template must produce valid, standalone Python 3.11+ code
- Generated code must import from AG2 (autogen) — never raw OpenAI SDK
- All agent instantiation must include: name, system_message, llm_config
- Templates must handle the human input status signals: `__STATUS_WAIT_FOR_HUMAN_INPUT__`, `__STATUS_RECEIVED_HUMAN_INPUT__`
- New templates must be registered in `main.j2` via import + include
- Test generated output by running it with `python -c` before committing
