---
name: review-prs
description: >-
  Review a GitHub pull request in the googleapis/mcp-toolbox repo against the
  team's reviewer checklist: PR title/description conventions, linked issue,
  logic errors and unhandled edge cases, breaking changes, test coverage, docs
  updates, security (input handling), and new dependencies. Use whenever a
  maintainer asks you to review, look over, "take a look at", or check whether
  something is ready to merge in mcp-toolbox, e.g. "review #3703", "can you look
  at this PR", "is this good to merge", or when they paste an mcp-toolbox PR link.
  PROPOSE-ONLY: delivers the review in chat for the maintainer to post; never
  approves, requests changes, comments, labels, or merges on its own.
---

# Review PRs (mcp-toolbox)

A review here is a proposal the maintainer edits and posts, not a rubber stamp. The value is
a fast, grounded read of the diff against the team's conventions.

## Goal

Given a PR number or link, deliver a review the maintainer can post in seconds: a suggested
verdict (approve / request changes / comment), the findings that back it grouped by severity
so the important things aren't buried, and a paste-ready summary comment.

## Prerequisites

- `gh` authenticated for `googleapis/mcp-toolbox`, plus the PR number(s). A GitHub MCP
  server substitutes for `gh` if it isn't available: the `gh` commands below map to its
  read/list tools.

## Guidance

**Read source-of-truth live, not from memory** (conventions drift). All three are authoritative;
this skill only adds how to apply them in a propose-only flow.

- [references/maintainer-playbook.md](references/maintainer-playbook.md): the Reviewer's
  Checklist, SLO/release context, and the `release candidate` labeling rule.
- [references/CONTRIBUTING.md](references/CONTRIBUTING.md): PR conventions: title/scope format
  (Conventional Commits, with the `type` table), the "keep PRs small" and "link an issue"
  guidelines, and the review process. Cite it for title/description/process findings.
- [references/DEVELOPER.md](references/DEVELOPER.md): engineering conventions: tool/source
  naming, error categorization, the patterns for adding a source/tool/integration test, the
  CI-enforced docs-structure rules, and the local test/lint commands. Cite it for code, test,
  and docs-structure findings. Prefer it over `GEMINI.md` (`CLAUDE.md`/`AGENTS.md` are symlinks
  to it), which only summarizes.

All three are symlinks to the live files at the repo root, so they track `main`. Cite them by
their repo-root names (`CONTRIBUTING.md`, `DEVELOPER.md`) in findings.

Fetch the PR and its diff:
```bash
gh pr view <n> --repo googleapis/mcp-toolbox --json number,title,body,author,labels,files,additions,deletions,commits,baseRefName,headRefName,state,isDraft,reviewDecision
gh pr diff <n> --repo googleapis/mcp-toolbox
gh pr checks <n> --repo googleapis/mcp-toolbox
```

**Draft or auto-generated PRs first.** For `renovate`/`release-please` PRs the review is
just: are the checks green? If so, propose merge and stop. If the PR `isDraft`, review
lightly and say so, since the author isn't asking for a final pass yet.

**Non-code / policy PRs.** Some PRs (adding a third-party badge, backlink, or promotional
README line, often from a drive-by contributor) raise a project/brand decision the maintainer
must own rather than a code question. Don't manufacture code findings; state plainly that
acceptance is a maintainer policy call, and still check the objective things (title convention,
CI). For an external badge or link, don't vouch for a URL you haven't fetched: mark it
`[UNVERIFIED]` and suggest the maintainer verify the source and target.

**A docs-shaped title never lowers the read bar.** PR #2473, titled "docs: fix typo in getting
started guide", added an npm `preinstall` hook that hijacked `git` via `GITHUB_PATH` to
exfiltrate an RSA-encrypted `GITHUB_TOKEN`. Read every file in the diff of any PR touching
`.hugo/`, `package.json` lifecycle scripts, `.github/workflows/`, or `.ci/`, whatever the title
says. A file in the diff that the title and description don't account for is itself a blocking
finding.

**Read the whole diff before commenting, and look for what's *missing*, not just what's
wrong.** The costly misses are absences and cross-file shape: a refactor applied to 4 of 5
call sites, a fix whose mirror bug still lives elsewhere, a behavior change with no test
update, an error swallowed silently. Skim for the shape, then dive into hunks.

**Review dimensions** (how to *apply* the playbook's checklist here). Skip a dimension when
it doesn't apply: say so, don't invent a finding.

- **Title & description.** The title must follow Conventional Commits with the right
  `type(scope)` per `CONTRIBUTING.md` (e.g. `feat(source/postgres): ...`); a `!` or
  `BREAKING CHANGE` is required for breaking changes. The body should follow
  `.github/PULL_REQUEST_TEMPLATE.md`: describe *what* and *why*, complete the PR checklist,
  and link the issue as `Fixes #<n>`. A PR with no linked issue is worth flagging (team norm:
  open an issue first), but don't block solely on it; just note it.
- **Correctness.** This is the highest-value part. Trace the changed logic and look for the
  bugs CI won't catch: unhandled error returns, nil/empty-input paths, off-by-one and
  boundary conditions, concurrency (the repo tests with `-race`), and behavior that
  contradicts the PR's stated intent. Cite `file:line` and explain the failure case, not
  just "looks risky". The **#1 source of bugs here** is type conversion at the MCP boundary:
  the schema needs JSON-serializable output, but database drivers return native types that
  don't serialize (e.g. MySQL returns `[]byte` for decimals, nulls come back as `nil`/`None`).
  Require explicit type handling that maps cleanly to the tool's JSON schema; reject implicit
  casts or a missing type switch. On any new or changed error path, check the error taxonomy
  (`DEVELOPER.md`, error handling): `AgentError` for input/execution errors the agent can fix
  itself (HTTP 200, `isError: true`) versus `ClientServerError` for infrastructure failures it
  can't. Getting these backwards is invisible to CI and decides whether the agent retries or
  gives up.
- **Breaking changes.** Changed config field names/YAML shape, tool names, removed/renamed
  exported symbols, or altered default behavior break users. If you find one, check the
  title carries `!`/`BREAKING CHANGE` and the description justifies it; if not, that's a
  blocking finding.
- **Refactor purity.** A `refactor:` PR must not change behavior; the logic should be
  equivalent. If a bug fix or default-behavior change is bundled in, ask for it to be split
  into a separate `fix:`/`feat:` PR so it is reviewable and revertable on its own.
- **Source reuse (new sources).** The most-enforced architectural rule in this repo, stated in
  `DEVELOPER.md` (adding a new database source or tool): a new `internal/sources/<db>/` is not
  accepted when the database is wire-compatible with a source that already exists. If the PR adds
  a source, require the description to establish that the wire protocol differs from every
  existing source; if it doesn't, the fix is to configure the existing source instead, and that is
  blocking. Precedent to cite if the author pushes back: the MariaDB source was removed after
  merge on these grounds (#1908) and SingleStore landed only with an explicit maintenance
  commitment (#1333). The same logic applies to a tool duplicating an existing tool's behavior
  under a new name.
- **Architecture (no boilerplate).** New tools embed `tools.BaseTool[Config]` and new
  sources follow the registration pattern rather than re-declaring the standard interface
  methods (`GetName`, `Manifest`, etc.); reject copy-pasted boilerplate. The authoritative
  list of what `BaseTool` provides is in `DEVELOPER.md` (adding a new tool).
- **Tool and parameter descriptions.** Every `description:` in a tool or parameter definition is
  an LLM prompt, not developer documentation. Review it for whether an agent could pick this tool
  and fill its parameters from that text alone, at a token cost worth paying. Flag descriptions
  that merely restate the field name, omit units/format/allowed values, or run long without
  adding information.
- **Tests.** New functionality or a bug fix should add tests. For a bug fix, look for a test
  that fails without the fix. Check that both happy path and edge cases are covered, and
  that a new source/tool follows the repo's unit + integration test pattern (and is wired
  into `.ci/integration.cloudbuild.yaml` when adding a source), per `DEVELOPER.md`. Missing
  tests on new logic is usually a request-changes. Placement is reviewed as closely as coverage:
  source-specific helpers stay unexported in `tests/<db>/<db>_integration_test.go` and must not
  land in the shared `tests/common.go`. Where a test looks flaky, the fixes reviewers ask for by
  name are UUID-scoped resource names, `t.Cleanup` teardown, polling instead of `time.Sleep`, and
  subset assertions rather than exact-match on result sets that a shared instance can add rows to.
  Integration tests need GCP credentials that an external contributor's PR can't trigger in CI,
  so green CI on an external PR does not mean they ran; when the code looks ready and passes
  locally, the next step is a maintainer running them via the `tests: run` label or a `/gcbrun`
  comment (`DEVELOPER.md`). Note that rather than treating the un-run tests as a blocker.
- **Docs.** If the change alters how a user configures or interacts with the toolbox, the
  matching docs under `docs/en/` must be updated. New sources/tools have CI-enforced page
  structure (H2 ordering, `_index.md` frontmatter-only, title conventions); see the
  docs-structure rules in `DEVELOPER.md` (enforced in CI by `.ci/lint-docs-*.sh`). A
  violation will break the build, so flag it as blocking.
- **Security.** If the PR handles user/LLM input or builds queries, check for injection
  (SQL/command), unsanitized interpolation, and secrets logged or committed. Flag concrete
  vectors with `file:line`, not generic warnings.
- **Dependencies.** New entries in `go.mod` deserve a note: is the dependency necessary,
  maintained, and appropriately licensed? Call out additions so the maintainer can vet them.

**Findings this repo rejects.** These four are settled decisions, not oversights, and each looks
like a defect by general Go standards, so they are easy to raise by reflex. Raising one costs the
contributor a cycle and the review its credibility.

- **No defensive checks for states the call graph already rules out.** Don't ask for a nil or
  bounds check on a value the caller guarantees (#3388).
- **Direct type assertions on parameters are the house pattern.** Don't ask for comma-ok in
  parameter handling; a panic on an internally impossible type is acceptable here.
- **Duplication across MCP protocol versions is deliberate.** Each version keeps its own copy of
  shared structures so versions can diverge independently; don't propose factoring them together
  (#3167, #3211).
- **Wildcard `--allowed-hosts` and CORS defaults are a product decision.** A CVE was requested for
  this and declined (#2750, #3113). Don't file it as a vulnerability.

A genuine bug in code that happens to touch one of these areas is still a finding. What's out of
scope is re-litigating the convention.

**CI status.** Read `gh pr checks`. Failing lint/tests are objective blockers: name which
check failed rather than re-deriving it by hand. Don't claim the linter passes; report what
CI says, or run `golangci-lint run` / `go test` locally only if the branch is checked out
and note results as advisory. The CLA check has one recurring non-obvious failure: commits
co-authored by an AI agent (for example `cursoragent@cursor.com` or a Claude agent address) fail
it even when the human author has signed. The fix is to squash to a single commit authored solely
by the human, so suggest that rather than pointing the contributor at the CLA docs.

**Existing bot review.** If `gemini-code-assist` has already commented, don't restate its points
as your own. It is the highest-volume reviewer in the repo and frequently wrong, having been
overruled on Go stdlib behavior, third-party API surfaces, and compile errors that did not exist.
Verify anything you carry forward against the diff yourself and drop the rest. Its review carries
no approval weight either way.

**Severity is the point.** A wall of equal-weight comments is noise. Separate **blocking**
(correctness bug, breaking change without `!`, missing tests on new logic, CI red, docs that
break the build) from **non-blocking** (style, naming, test-coverage gaps on existing code)
from **nits** (typos, wording). The verdict follows: any blocking finding means request
changes; only non-blocking/nits means approve with comments; an unresolved judgment call
means comment and ask. Calibrate against how the team actually votes: humans on this repo
approve roughly 4.5 times for every changes-requested, so treat "request changes" as a real
signal reserved for the blocking list, not the default for any imperfection. Don't spend findings
on mere preference (early-return vs nested-if, naming that matches the file) or on what CI already
catches (formatting, lint); call something wrong only when you can say *why* it is worse, not just
different. When there are no blockers, say so plainly, since "no blockers, a couple of nits" tells
the maintainer it's mergeable as-is.

**`release candidate`.** Per the playbook, propose adding the `release candidate` label
(defined in `.github/labels.yaml`) when the change should ship in the next release. Propose
it; the maintainer applies it.

## Rules

- **Propose only.** Never run `gh pr review`, `gh pr comment`, `gh pr edit`, `gh pr merge`,
  or apply labels. Deliver the review in chat.
- **Ground every finding** in the strongest evidence available for its kind: a
  correctness/security/breaking claim cites `file:line`; a convention claim cites
  `CONTRIBUTING.md`/`DEVELOPER.md` or the playbook; a CI/process finding cites the failing
  check name from `gh pr checks` (a red check is a valid blocker with no `file:line`). If you
  couldn't verify something (runtime behavior you can't trace, a URL you didn't fetch), mark
  it `[UNVERIFIED]` rather than asserting it.
- **When it's a genuine judgment call, ask** rather than issuing a confident wrong verdict,
  since a wrong "request changes" costs a contributor a cycle.

## Output format

```
## Review #<n>: <title>

**Suggested verdict:** <approve / request changes / comment>: <one-line reason>

**Title & issue:** <conventional-commit check; linked issue or "none, suggest linking">
**CI:** <passing / which checks failing, per gh pr checks>

**Blocking:**
- `file:line`: <finding + the failure case> [cite]

**Non-blocking:**
- `file:line`: <finding> [cite]

**Nits:**
- <typo/wording>

**Tests:** <added & adequate / what's missing>
**Docs:** <updated / what's missing, or n/a>
**Dependencies:** <new deps to vet, or none>
**release candidate:** <suggest label / not needed>

**Draft comment:**
<paste-ready summary the maintainer can post>
```

If there are no findings in a section, omit it rather than writing "none". For a batch of
PRs, one block per PR plus a one-line summary table (PR, verdict, blocker count).
