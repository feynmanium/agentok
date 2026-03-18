---
paths:
  - "frontend/src/components/flow/node/**"
  - "frontend/src/lib/flow.ts"
---

# Frontend Node Rules

- Every new node type must be added to BOTH `nodeTypes` and either `basicNodes` or `advancedNodes` in `flow.ts`
- Node components must use the XYFlow Node API and accept standard node props
- Node metadata requires: id, name, description, class_type, icon
- Icons come from `@/components/icons` — use existing icons or add new ones there
- Add the node type string to the `isConversable()` check if the node participates in conversations
- All user-visible strings must use i18n keys (label field = i18n key)
