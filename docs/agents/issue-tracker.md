# Issue tracker: local markdown

This repo uses **local markdown files** for issue tracking. No external issue tracker is configured.

## Convention

Issues live as markdown files under `.scratch/<feature>/` at the repo root. A "feature" is a logical slice of work — e.g. a new capability, a bug fix, a refactor.

```
.scratch/
  auth/
    001--add-login--needs-triage.md
    002--oauth-refresh--ready-for-agent.md
  parser/
    001--handle-unicode--needs-info.md
```

## Filename convention

Each issue file follows: `<number>--<short-description>--<status>.md`

- `number` — sequential per-feature (001, 002, …)
- `short-description` — kebab-case summary of the issue
- `status` — one of the triage labels (see `docs/agents/triage-labels.md`)

## File format

Each issue file is a markdown document:

```markdown
# <Title>

**Status:** <status>
**Created:** <date>

## Description

Free-form description of the issue.

## Acceptance criteria

- [ ] Criterion one
- [ ] Criterion two
```

## Consumed by

- `to-issues` — breaks plans/specs into `.scratch/<feature>/` issue files
- `triage` — moves issues through the label state machine
- `to-prd` — publishes PRDs under `.scratch/<prd>/`
- `qa` — checks `.scratch/` for issues during quality review

## Switching later

If you later migrate to GitHub Issues or another tracker, re-run the `setup-matt-pocock-skills` skill to regenerate this file with the new workflow. The local `.scratch/` files can be bulk-imported.
