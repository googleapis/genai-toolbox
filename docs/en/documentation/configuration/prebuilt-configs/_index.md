---
title: "Prebuilt Configs"
type: docs
weight: 2
description: >
    This page lists all the prebuilt configs available.
---

Prebuilt configs are reusable, pre-packaged toolsets that are designed to extend
the capabilities of agents. These configs are built to be generic and adaptable,
allowing developers to interact with and take action on databases.

See guides, [Connect from your IDE](../../connect-to/ides/_index.md), for
details on how to connect your AI tools (IDEs) to databases via Toolbox and MCP.

{{< notice tip >}}
You can now use `--prebuilt` along `--config`, `--configs`, or
`--config-folder` to combine prebuilt configs with custom tools.

You can also combine multiple prebuilt configs.

**Filtering Toolsets:**
You can load a specific toolset from a prebuilt configuration by appending a `/` and the toolset name, for example: `--prebuilt=postgres/data` to only load the SQL tools.

See [Usage Examples](../../../reference/cli.md#usage-examples).
{{< /notice >}}

## Security for dynamic SQL tools

Some prebuilt configs expose dynamic `execute_sql`-style tools where the agent
supplies raw SQL. Tool annotations and MCP client confirmations are useful UX
guardrails, but they are not a database security boundary.

Run these tools with a dedicated database identity that only has the privileges
the agent should exercise. For exploratory agents, this usually means
`SELECT`-only access to the specific schemas, tables, or views the agent may
read. Avoid owner, admin, migration, or application-write accounts.

Prefer custom parameterized tools for fixed workflows. Use dynamic SQL tools for
trusted exploratory read-only access, and rely on database-native permissions or
read-only session controls where the engine supports them. Do not rely on regex
keyword blacklists to make an arbitrary SQL endpoint safe.

## Available Prebuilt Configs

{{< list-prebuilt-configs >}}
