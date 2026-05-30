# Domain docs: multi-context layout

This repo uses a **multi-context** domain documentation layout, suitable for monorepos or projects with distinct subsystems.

## Layout

```
CONTEXT-MAP.md          ← entry point; lists available contexts
CONTEXT.md              ← (optional) root-level context if the root has domain meaning
docs/adr/               ← architecture decision records for the whole project
docs/adr/001-*.md
docs/adr/002-*.md
...
<subsystem>/
  CONTEXT.md            ← per-context domain language
  docs/adr/             ← subsystem-specific ADRs (optional)
```

## How to read

1. Start at `CONTEXT-MAP.md` — it lists every named context and where its `CONTEXT.md` lives.
2. For a given context, read its `CONTEXT.md` to learn the domain language, entities, and invariants for that subsystem.
3. For architecture decisions, check `docs/adr/` at the root first (project-wide decisions), then the subsystem's own `docs/adr/` if applicable.

## Consumer rules

Skills that read domain docs (`improve-codebase-architecture`, `diagnose`, `tdd`) follow this lookup order:

- Look for `CONTEXT-MAP.md` at the repo root. If it exists, read it to find the relevant context(s).
- If no `CONTEXT-MAP.md`, fall back to `CONTEXT.md` at the root (single-context mode).
- Always check `docs/adr/` at the root for architectural decisions.
- When operating on a specific file or directory, prefer the nearest `CONTEXT.md` in the ancestor chain.
