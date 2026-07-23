---
name: triage-issues
description: >-
  Triage GitHub issues in the googleapis/mcp-toolbox repo: propose the correct
  labels (type / priority / product / status), check for duplicates, verify a bug
  has enough info to act on, and draft a triage comment. Use whenever a maintainer
  asks you to triage, label, categorize, prioritize, or "look at" an issue (or a
  batch) in mcp-toolbox, e.g. "triage #3648", "what labels should this get", "is
  this a dup", triage a batch of issue numbers, or when they paste an mcp-toolbox
  issue link. PROPOSE-ONLY: delivers the triage in chat for the maintainer to apply;
  never edits labels, comments, closes, or assigns on its own.
---

# Triage Issues (mcp-toolbox)

Triage here is mostly manual **labeling**. The `product:` label auto-routes to the owning team
via `.github/blunderbuss.yml`. Propose only, never mutating the issue; when a call isn't obvious,
ask rather than guess.

## Goal

Given an issue number (or several), deliver a triage the maintainer can apply in seconds: labels
across the four axes (each with a one-line *why*), likely duplicates, missing bug info, and a
draft comment when one adds value.

## Prerequisites

- `gh` authenticated for `googleapis/mcp-toolbox`, plus the issue number(s). A GitHub MCP server
  substitutes for `gh` if it isn't available: the `gh` commands below map to its read/list tools.

## Guidance

**Read source-of-truth live, not from memory** (labels/routing drift):
- `gh label list --limit 200`: the only valid label names. Never propose one not listed.
- `.github/blunderbuss.yml`: which `product:` labels route to which team.
- `.github/ISSUE_TEMPLATE/bug_report.yml`: required bug fields (for the completeness check).
- [references/maintainer-playbook.md](references/maintainer-playbook.md): the triage workflow,
  label taxonomy, priority definitions, and SLO targets this skill applies.

Fetch the issue:
```bash
gh issue view <n> --repo googleapis/mcp-toolbox --json number,title,body,author,labels,comments,createdAt
```

**Check bot-noise first.** Automated reports (e.g. "Cloud Build Failure Reporter ... failed",
bot author) just need `periodic-failure`, so skip the rest of the workflow.

**The four axes** (skip an axis when it doesn't apply; say so, don't force a label):

- **`type:`**: bug / feature request / question / cleanup / process / docs. The template
  pre-applies `type: bug`/`type: question`; trust it, but correct it when the body disagrees
  (a "bug" asking for new behavior is a feature request) and say why.
- **`product:`**: one per data source; **highest-leverage label**, since it drives team routing
  (above). Infer from the source/tool named (e.g. "looker run_dashboard tool" → `product: looker`).
  Don't force one on core/product-agnostic issues; note it has no product owner. If a real data
  source has no matching label yet, say so and flag it for `.github/labels.yaml` rather than
  mis-routing with a wrong one.
- **`priority:`**: depends on type; **bugs are only ever p0 or p1** (never p2/p3), since priority
  drives the team's response SLOs (see [references/maintainer-playbook.md](references/maintainer-playbook.md),
  "Bug Triage Workflow" and the SLO table).
  - **Bug** → **p0**: major functionality broken / feature unusable (db connection fails, a tool
    consistently errors, an extension fails to load, data corruption). **p1**: critical breakage
    impacting the next release / inconsistent behavior (intermittent tool timeout, outdated docs
    for a key feature).
  - **Feature request** → **p0**: high-priority, extends major functionality (e.g. prompt support).
    **p1**: significant improvement targeted for the next release (e.g. new auth support). **p2**:
    nice-to-have (tool tweaks, clearer error messages, less verbose output). **p3**: open for
    community contribution.
  The maintainer overrides with context.
- **`status:`** (optional): `waiting for response` when a bug lacks info to act (reviewer awaiting
  the author); `feedback wanted` when waiting on community/author input; `help wanted` for
  unplanned work open to community contribution. Author silence >60 days is grounds to close.

**Community labels** (standalone, not `status:`-prefixed): `good first issue` for well-scoped,
approachable issues; `ready for work` for triaged issues actionable now. Propose `good first issue`
alongside `help wanted` when the fix is small and self-contained.

**Assignment** (not a label): rarely needed, since `product:` auto-routes to the team. Propose an
assignee only when an external contributor volunteers (assign them, to avoid duplicate work) or a
maintainer is picking it up; otherwise leave it unassigned so contributors know it's open.

**Duplicates.** `gh issue list --repo googleapis/mcp-toolbox --state all --search "<key terms>" --limit 20`
with distinctive terms (tool name, error string). If it's a known issue, propose `duplicate` +
close: link and reference the original, and thank the reporter (template below).

**Completeness (bugs).** Compare against `bug_report.yml` required fields (version, env, expected
vs. current, repro) and list exactly what's missing. This feeds the `waiting for response` label
and the draft comment.

**Draft comment.** Prefer the canonical templates below over inventing wording; fill in the
specifics (name the exact missing fields).

Needs more information (missing repro/details):
```text
Thanks for opening this issue! We are having trouble reproducing your problem with the information provided.

To help us investigate further, could you please provide:
- A minimal, reproducible code sample that demonstrates the issue.
- The full error message and stack trace.

We will close this issue in 14 days if we don't hear back. Thanks!
```

Acknowledging a feature request:
```text
Thanks for suggesting this feature! We appreciate you taking the time to provide this feedback. We've added this to our backlog for consideration. We can't provide a specific timeline for implementation right now, but we will update this issue with any progress. In the meantime, we welcome pull requests from the community if you are interested in contributing this feature yourself.
```

Closing as duplicate:
```text
Thanks for reporting this! This is a duplicate of #<original>, so I'm closing this in favor of that issue. Please follow along there for updates.
```

## Rules

- **Propose only.** Never run `gh issue edit/comment/close` or assign. Deliver in chat.
- **Only real labels**: every proposed label must appear in `gh label list`.
- **When type/product is a genuine toss-up, ask.** A confident wrong `product:` mis-routes.
- **Cite what backs a claim** (e.g. "routes to `toolbox-looker-team` per `blunderbuss.yml`").

## Output format

```text
## Issue #<n>: <title>

**Labels:**
- `type: <x>`: <why>
- `product: <x>`: <why + routing, or "no product owner: core/...">
- `priority: <x>`: <why>
- `status: <x>`: <why, if any>

**Duplicates:** <#nums, or "none for terms: ...">
**Completeness (bugs):** <what's missing, or "complete">
**Draft comment:** <paste-ready, or "none needed">

**Apply:** gh issue edit <n> --repo googleapis/mcp-toolbox --add-label "type: ...,product: ...,priority: ..."
```

Include the ready-to-run `gh issue edit` line so the maintainer applies with one paste, but leave
running it to them. For a batch, one block per issue plus a one-line summary table.
