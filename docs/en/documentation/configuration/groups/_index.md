---
title: "Groups"
type: docs
weight: 6
description: >
  Groups let you scope tools and prompts together under a single name, with a description used as group metadata.
---

A Group is a single named collection that scopes both **tools** and **prompts** together. Where a [Toolset](../toolsets/) groups only tools, a group bundles tools and prompts under one name and one MCP endpoint, and carries a `description` that describes the collection.

Connecting to a group's endpoint (`/mcp/{name}`) scopes both `tools/list` and `prompts/list` to that group. Groups are also introspectable over MCP through the `groups/list` and `groups/get` methods.

## Defining Groups

Declare a group as a `kind: group` document in your configuration file. A group has four fields:

| Field         | Required | Description                                                              |
| ------------- | -------- | ------------------------------------------------------------------------ |
| `name`        | Yes\*    | Unique name for the group. Used as the endpoint path (`/mcp/{name}`).    |
| `description` | No       | Human-readable description of the group, surfaced via `groups/list`.     |
| `tools`       | No       | List of tool names to include in the group.                             |
| `prompts`     | No       | List of prompt names to include in the group.                           |

\* `name` is required for every named group. The single [default group](#the-default-group) omits it.

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

A single **default (nameless) group** always exists and contains **all** configured tools and prompts. Connecting to the default MCP endpoint returns everything.

You may declare a `kind: group` document with no `name` to set a `description` for the default group. Because the default group always contains all tools and prompts, it **cannot** declare `tools` or `prompts`:

```yaml
kind: group
description: All tools and prompts available on this server.
```

## Validation rules

At startup, Toolbox validates groups:

- **Unique names.** A named group must have a unique name that satisfies the standard name rules (alphanumeric characters, underscores, and hyphens).
- **One default group.** Declaring more than one nameless group is an error.
- **Default group restrictions.** The default group may set only a `description`; declaring `tools` or `prompts` on it is an error.
- **Group wins over a same-named toolset.** If a name is defined by both a `kind: toolset` and a `kind: group`, the group takes precedence and Toolbox logs a warning naming the shadowed toolset.

## Relationship to toolsets

Groups are a superset of toolsets: a toolset is equivalent to a tools-only group. Existing `kind: toolset` configurations continue to work unchanged — they are treated as groups with tools and no prompts, so no migration is required. Use a group when you need to scope prompts alongside tools. See [Toolsets](../toolsets/) for more.

## Introspecting groups over MCP

Two MCP methods let clients discover groups. Both are available across all supported MCP protocol versions.

### `groups/list`

Returns every named group with its `name` and `description`:

```json
{
  "groups": [
    { "name": "data_analyst", "description": "Tools and prompts for exploratory data analysis." },
    { "name": "admin", "description": "Administrative operations." }
  ]
}
```

### `groups/get`

Takes a group `name` and returns that group's tools and prompts together:

```json
{
  "name": "data_analyst",
  "tools": [ /* ... */ ],
  "prompts": [ /* ... */ ]
}
```
