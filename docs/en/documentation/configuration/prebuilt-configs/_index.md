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

{{< notice warning >}}
These prebuilt configs are intended for 'build-time' use cases, where agents are helping trusted developers build things. They are not secure enough for 'run time' use cases, where the agent will be talking to potentially untrusted developers.
{{< /notice >}}

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

## Available Prebuilt Configs

{{< list-prebuilt-configs >}}
