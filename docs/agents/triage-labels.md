# Triage labels

Five canonical triage roles for issue state management. This repo uses the default label strings.

## The five roles

| Role | Label string | Meaning |
|---|---|---|
| needs-triage | `needs-triage` | Maintainer needs to evaluate the issue |
| needs-info | `needs-info` | Waiting on the reporter for more information |
| ready-for-agent | `ready-for-agent` | Fully specified — an AI agent can pick this up with no human context |
| ready-for-human | `ready-for-human` | Needs human implementation or judgement |
| wontfix | `wontfix` | Will not be actioned (closed) |

## How labels are applied

For **local markdown** issues (see `docs/agents/issue-tracker.md`), the status is encoded in the filename: `<number>--<description>--<status>.md`. The `triage` skill moves issues by renaming the file to update the status segment.

For example, an issue at `needs-triage` → `ready-for-agent` would be renamed from:

```
003--add-pagination--needs-triage.md  →  003--add-pagination--ready-for-agent.md
```

## Overriding

If you need different label strings in the future, update this file and the state machine in the `triage` skill will use whatever strings you define here. The `## Agent skills` section in `AGENTS.md` summarizes this file — update the summary there too if you change the labels.
