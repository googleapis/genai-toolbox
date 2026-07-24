---
title: "Groups"
type: docs
weight: 6
description: >
  Groups let you scope MCP primitives such as tools and prompts together under a single name, with a description used as group metadata.
---

A Group is a single named collection that scopes MCP primitives together — currently **tools** and **prompts**, with more (such as resources) planned. Where a [Toolset](../toolsets/) groups only tools, a group bundles these primitives under one name and one MCP endpoint, and carries a `description` that describes the collection.

Connecting to a group's endpoint (`/mcp/{name}`) scopes the corresponding MCP list methods (such as `tools/list` and `prompts/list`) to that group.

## Defining Groups

Declare a group as a `kind: group` document in your configuration file. A group has the following fields:

| Field         | Required | Description                                                              |
| ------------- | -------- | ------------------------------------------------------------------------ |
| `name`        | Yes\*    | Unique name for the group. Used as the endpoint path (`/mcp/{name}`).    |
| `description` | No       | Human-readable description of the group.                                |
| `tools`       | No       | List of tool names to include in the group.                             |
| `prompts`     | No       | List of prompt names to include in the group.                           |

\* `name` is required for every named group. The single [default group](#the-default-group) omits it.

As Toolbox adds support for more MCP primitives, groups will gain corresponding fields (for example, `resources`).

```yaml
kind: group
name: data_analyst
description: Tools and prompts for exploratory data analysis.
tools:
  - list_tables
  - execute_sql
prompts:
  - summarize_results
---
kind: group
name: admin
description: Administrative operations.
tools:
  - create_user
  - list_users
```

## The default group

A single **default (nameless) group** always exists and contains **all** configured primitives (every tool and prompt). Connecting to the default MCP endpoint (`/mcp`) returns everything.

You may declare a `kind: group` document with no `name` to set a `description` for the default group. Because the default group always contains everything, it **cannot** declare `tools`, `prompts`, or any other primitive list:

```yaml
kind: group
description: All tools and prompts available on this server.
```

## Validation rules

At startup, Toolbox validates groups:

- **Unique names.** A named group must have a unique name that satisfies the standard name rules (alphanumeric characters, underscores, and hyphens).
- **One default group.** Declaring more than one nameless group is an error.
- **Default group restrictions.** The default group may set only a `description`; declaring `tools`, `prompts`, or any other primitive list on it is an error.
- **Group wins over a same-named toolset.** If a name is defined by both a `kind: toolset` and a `kind: group`, the group takes precedence and Toolbox logs a warning naming the shadowed toolset.

## Relationship to toolsets

Groups are a superset of toolsets: a toolset is equivalent to a tools-only group. Existing `kind: toolset` configurations continue to work unchanged — they are treated as groups with tools and no other primitives, so no migration is required. That said, we recommend migrating to a `kind: group` even for tools-only collections: a group lets you attach a `description` and scope prompts (and, in the future, other primitives) alongside tools. See [Toolsets](../toolsets/) for more.
