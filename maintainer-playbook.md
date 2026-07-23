# Open Source Maintainer Playbook

## Overview

This playbook aligns on a consistent process for maintaining MCP Toolbox, ensuring quality, and providing a positive experience for contributors.

## Background

Our team maintains the MCP Toolbox family of open source repositories on GitHub:

* [MCP Toolbox](https://github.com/googleapis/genai-toolbox)
* [MCP Toolbox \- Python SDK](https://github.com/googleapis/mcp-toolbox-sdk-python)
* [MCP Toolbox \- JS SDK](https://github.com/googleapis/mcp-toolbox-sdk-js)
* [MCP Toolbox \- Go SDK](https://github.com/googleapis/mcp-toolbox-sdk-go)

This playbook establishes our guidelines for maintaining our repositories, aiming to bring clarity, efficiency, and predictability to our open-source operations.

## The Issue Lifecycle: From Report to Release

This section outlines the end-to-end pipeline for how a bug or feature request moves from initial report to a final release. Each step links to a more detailed section later in this playbook.

1. [**Identify Issue**](#identify-issue): An issue is opened in one of our GitHub repositories by a community member or a team member.

2. [**Triage**](#bug-triage-workflow): A team member acknowledges the issue, applies appropriate labels for categorization and priority, and verifies the report.

3. [**Resolution**](#resolution): The issue is assigned and work begins.

4. [**Review & Merge**](#handling-pull-requests): The PR undergoes a thorough review by the team. Once approved and all checks pass, a maintainer merges it into the main branch.

5. [**Release**](#releases): The merged changes are bundled into the next versioned release and published.

## Issue Workflow

### Identify Issue

We track open issues and PRs across all repositories so maintainers have visibility into what needs attention. Each team should monitor incoming work against the team's response and closure targets (SLOs).

There are 3 primary types of work that need to be addressed, in prioritized order:

1. Out of SLO (past the target response/closure time)
2. Near to SLO (approaching the target)
3. Untriaged

While we want to immediately remediate anything that is out of SLO, ideally issues and PRs would not get to that point.

Everyone should check their open issues/PRs at least **once a week**.

Current SLO targets:

| Type | Priority | Metric | Objective |
| :---- | :---- | :---- | :---- |
| Feature Request | P0 | Response | 5 days |
| Process | P0 | Response | 5 days |
| Bug / Customer Issue | P0 | Response | 2 days |
| | | Closure | 14 days |
| | P1 | Response | 7 days |
| | | Closure | 90 days |
| | P2 | Response | 30 days |

\* Response requires at least a response from the reviewer

\* Closure requires the issue/PR to be closed.

### Bug Triage Workflow

Once you have identified a bug, assess it and provide it with an **initial acknowledgement**.

#### Triage Checklist

* [ ] **Check for Duplicates:** Is this a known issue? If so, link to the original, thank the user, and close as duplicate referencing the other issue.
* [ ] **Verify Reproducibility:** Can we reproduce the reported bug with the information provided? If not, request more information.
* [ ] **Apply Labels:** Add the **`Priority <>`** & **`Type <>`** & **Product \<\>** (if applicable) label on the GitHub Issue / PR as deemed appropriate. SLOs are based on the "Priority" label. Add the **Status \<\>** label if applicable.
* [ ] **Assign/Unassign Owner:** Assign a team member to investigate further if necessary. If you are planning to work on the issue, keep yourself assigned and pull it into your sprint. If you are not planning to work on the issue, unassign yourself so that contributors are aware that the issue is not assigned.

#### **Labels**

* Types
  * Bug
  * Feature request (FR)
  * Questions
  * Docs \- requires additional documentation
  * Process \- regular workflow processes, may include testing, release, etc.
  * Cleanup \- System improvements, or internal cleanup/hygiene concern
* Priorities
  * P0
    * Bug \- Major functionality broken that renders a feature unusable.
      * Example: issues with the database connection or a tool consistently erroring
        * An extension fails to load, preventing users from accessing any of its tools.
        * A critical data plane tool consistently returns incorrect results, leading to data corruption.
    * FR \- Reduces friction, a high priority feature to extend major functionality
      * Example: prompt support
  * P1
    * Bug \- Critical feature breakage which impacts the next release.
      * Example: tool or extension doesn't work consistently
        * A newly added tool for creating instances sometimes times out, requiring manual retries.
        * The documentation for a key feature is outdated, causing confusion for developers.
    * FR \- Significant feature improvements or additions targeted for the next release.
      * Example: adding support for an additional authentication method
  * P2
    * Bugs should not be P2s
    * FR \- Nice to have.
      * Example: tweaks to tools
        * The error message for a common permission issue is unclear and could be more actionable.
        * A tool's output is verbose and could be summarized for better readability.
  * P3
    * Bugs should not be P3s
    * Feature requests that are open for contribution can be labeled P3.
      * Example: a request to add a new, non-critical feature to an existing extension.
* Product
  * Each product should have a label. Add labels in [`labels.yaml`](.github/labels.yaml) if one is missing.
* Status
  * help wanted \- Unplanned work open for contributions from the community.
  * feedback wanted \- Waiting for feedback from community or issue author. If the contributor did not respond for \>60 days, we should just close the PR.
  * waiting for response \- Reviewer awaiting feedback or responses from author. If the contributor did not respond for \>60 days, we should just close the PR.

Here are some sample templates:

Acknowledging a Feature Request:

```text
Thanks for suggesting this feature! We appreciate you taking the time to provide this feedback. We've added this to our backlog for consideration. We can't provide a specific timeline for implementation right now, but we will update this issue with any progress. In the meantime, we welcome pull requests from the community if you are interested in contributing this feature yourself.
```

Needs More Information:

```text
Thanks for opening this issue! We are having trouble reproducing your problem with the information provided.

To help us investigate further, could you please provide:
- A minimal, reproducible code sample that demonstrates the issue.
- The full error message and stack trace.

We will close this issue in 14 days if we don't hear back. Thanks!
```

### Resolution

For bugs, this involves writing code to fix the problem. For features, it involves implementation. For PRs, it involves assessment and review of the features being added.

After triage, the issue should be assigned to a team member for resolution.
If an external contributor expresses interest in working on an issue, assign it
to them to prevent duplicate work.

Flaky tests that we don't own will not be the priority. We should prioritize tests we own. Try to push work back to the upstream product teams. If third-party tests are constantly flaky, consider removing them from the test suites and escalate to the appropriate point of contact.

If we are opening a PR for a Feature Request or a Bug, make sure to link the issue in the description.

For auto-generated PRs (e.g. dependency updates and release automation), make sure the tests and PR checks pass and merge them.

### Handling Pull Requests

#### Reviewer's Checklist

If you are reviewing a PR, here are a few things to consider:

* [ ] Does this PR have a corresponding issue? If so, is it linked?
* [ ] Does the PR's title and description clearly explain *what* it does and *why*?
* [ ] Are there any logic errors or edge cases that haven't been considered?
* [ ] Does this change introduce any breaking changes? If so, are they documented and necessary?
* [ ] Does the PR title follow our guideline?
* [ ] Does the code follow our style guide? (Run the linter)
* [ ] Does the PR include new tests for the added functionality or bug fix?
* [ ] Do the tests cover both happy paths and edge cases?
* [ ] If this changes how a user interacts with the code, is the `README.md` or relevant documentation updated?
* [ ] Are there clear code comments for any complex parts of the logic?
* [ ] Does this PR handle user input? If so, is it properly sanitized?
* [ ] Does it add any new dependencies? If so, have they been vetted?
* [ ] Add the `release candidate` label if it needs to be in the next release.

All the pre-submit tests should pass and the documentation change should be reviewed before approval.

### Releases

Once the PR is merged, make sure you leave a comment on the open issue making external contributors aware that the fix should be available in the upcoming version.

Here's an example:

```text
This has been resolved in PR #[PR number]. The fix will be available in our next release (vX.Y.Z). Thanks again to @[contributor-username] for the contribution! Closing this issue now.
```

Release PRs are created by release automation and assigned to a team member. Keep an eye out for those or re-assign them to another team member as necessary.

Release plan:

* MCP Toolbox: Generally two a month.
* SDKs: As deemed necessary.
