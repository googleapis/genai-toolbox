---
title: "Invoke Tools via CLI"
type: docs
weight: 10
description: >
  Learn how to invoke your tools directly from the command line using the `invoke` command.
---

The `invoke` command allows you to invoke tools defined in your configuration directly from the CLI. This is useful for:

- **Ephemeral Invocation:** Executing a tool without spinning up a full MCP server/client.
- **Debugging:** Isolating tool execution logic and testing with various parameter combinations.

{{< notice tip >}}
**Keep configurations minimal:** The `invoke` command initializes *all* resources (sources, tools, etc.) defined in your configuration files during execution. To ensure fast response times, consider using a minimal configuration file containing only the tools you need for the specific invocation.
{{< /notice >}}

## Before you begin

1. Make sure you have the `toolbox` binary installed or built.
2. Make sure you have a valid tool configuration file (e.g., `tools.yaml`).

### Command Usage

The basic syntax for the command is:

```bash
toolbox <tool-source> invoke <tool-name> [params]
```

- `<tool-source>`: Can be `--tools-file`, `--tools-files`, `--tools-folder`, and `--prebuilt`. See the [CLI Reference](../reference/cli.md) for details.
- `<tool-name>`: The name of the tool you want to call. This must match the name defined in your `tools.yaml`.
- `[params]`: (Optional) A JSON string representing the arguments for the tool.

## Examples

### 1. Calling a Tool without Parameters

If your tool takes no parameters, simply provide the tool name:

```bash
toolbox  --tools-file tools.yaml invoke my-simple-tool
```

### 2. Calling a Tool with Parameters

For tools that require arguments, pass them as a JSON string. Ensure you escape quotes correctly for your shell.

**Example: A tool that takes parameters**

Assuming a tool named `mytool` taking `a` and `b`:

```bash
toolbox --tools-file tools.yaml invoke mytool '{"a": 10, "b": 20}' 
```

**Example: A tool that queries a database**

```bash
toolbox  --tools-file tools.yaml invoke db-query '{"sql": "SELECT * FROM users LIMIT 5"}'
```

### 3. Using Prebuilt Configurations

You can also use the `--prebuilt` flag to load prebuilt toolsets.

```bash
toolbox --prebuilt cloudsql-postgres invoke cloudsql-postgres-list-instances
```

## Troubleshooting

- **Tool not found:** Ensure the `<tool-name>` matches exactly what is in your YAML file and that the file is correctly loaded via `--tools-file`.
- **Invalid parameters:** Double-check your JSON syntax. The error message will usually indicate if the JSON parsing failed or if the parameters didn't match the tool's schema.
- **Auth errors:** The `invoke` command currently does not support flows requiring client-side authorization (like OAuth flow initiation via the CLI). It works best for tools using service-side authentication (e.g., Application Default Credentials).
