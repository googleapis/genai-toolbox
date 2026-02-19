---
title: "CLI"
type: docs
weight: 1
description: >
  This page describes the `toolbox` command-line options.
---

## Reference

| Flag (Short) | Flag (Long)                | Description                                                                                                                                                                      | Default     |
|--------------|----------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------|
| `-a`         | `--address`                | Address of the interface the server will listen on.                                                                                                                              | `127.0.0.1` |
|              | `--disable-reload`         | Disables dynamic reloading of tools file.                                                                                                                                        |             |
| `-h`         | `--help`                   | help for toolbox                                                                                                                                                                 |             |
|              | `--log-level`              | Specify the minimum level logged. Allowed: 'DEBUG', 'INFO', 'WARN', 'ERROR'.                                                                                                     | `info`      |
|              | `--logging-format`         | Specify logging format to use. Allowed: 'standard' or 'JSON'.                                                                                                                    | `standard`  |
| `-p`         | `--port`                   | Port the server will listen on.                                                                                                                                                  | `5000`      |
|              | `--prebuilt`               | Use one or more prebuilt tool configuration by source type. See [Prebuilt Tools Reference](prebuilt-tools.md) for allowed values.                                                          |             |
|              | `--stdio`                  | Listens via MCP STDIO instead of acting as a remote HTTP server.                                                                                                                 |             |
|              | `--telemetry-gcp`          | Enable exporting directly to Google Cloud Monitoring.                                                                                                                            |             |
|              | `--telemetry-otlp`         | Enable exporting using OpenTelemetry Protocol (OTLP) to the specified endpoint (e.g. 'http://127.0.0.1:4318')                                                                    |             |
|              | `--telemetry-service-name` | Sets the value of the service.name resource attribute for telemetry data.                                                                                                        | `toolbox`   |
|              | `--tools-file`             | File path specifying the tool configuration. Cannot be used with --tools-files or --tools-folder.                                                                                |             |
|              | `--tools-files`            | Multiple file paths specifying tool configurations. Files will be merged. Cannot be used with --tools-file or --tools-folder.                                                    |             |
|              | `--tools-folder`           | Directory path containing YAML tool configuration files. All .yaml and .yml files in the directory will be loaded and merged. Cannot be used with --tools-file or --tools-files. |             |
|              | `--ui`                     | Launches the Toolbox UI web server.                                                                                                                                              |             |
|              | `--allowed-origins`        | Specifies a list of origins permitted to access this server for CORs access.                                                                                                     | `*`         |
|              | `--allowed-hosts`          | Specifies a list of hosts permitted to access this server to prevent DNS rebinding attacks.                                                                                      | `*`         |
|              | `--user-agent-metadata`    | Appends additional metadata to the User-Agent.                                                                                                                                   |             |
| `-v`         | `--version`                | version for toolbox                                                                                                                                                              |             |

## Sub Commands

<details>
<summary><code>invoke</code></summary>

Executes a tool directly with the provided parameters. This is useful for testing tool configurations and parameters without needing a full client setup.

**Syntax:**

```bash
toolbox invoke <tool-name> [params]
```

**Arguments:**

- `tool-name`: The name of the tool to execute (as defined in your configuration).
- `params`: (Optional) A JSON string containing the parameters for the tool.

For more detailed instructions, see [Invoke Tools via CLI](../how-to/invoke_tool.md).

</details>

<details>
<summary><code>skills-generate</code></summary>

Generates a skill package from a specified toolset. Each tool in the toolset will have a corresponding Node.js execution script in the generated skill.

**Syntax:**

```bash
toolbox skills-generate --name <name> --description <description> --toolset <toolset> --output-dir <output>
```

**Flags:**

- `--name`: Name of the generated skill.
- `--description`: Description of the generated skill.
- `--toolset`: (Optional) Name of the toolset to convert into a skill. If not provided, all tools will be included.
- `--output-dir`: (Optional) Directory to output generated skills (default: "skills").

For more detailed instructions, see [Generate Agent Skills](../how-to/generate_skill.md).

</details>

## Examples

### Transport Configuration

**Server Settings:**

- `--address`, `-a`: Server listening address (default: "127.0.0.1")
- `--port`, `-p`: Server listening port (default: 5000)

**STDIO:**

- `--stdio`: Run in MCP STDIO mode instead of HTTP server

#### Usage Examples

```bash
# Basic server with custom port configuration
./toolbox --tools-file "tools.yaml" --port 8080

# Server with prebuilt + custom tools configurations
./toolbox --tools-file tools.yaml --prebuilt alloydb-postgres

# Server with multiple prebuilt tools configurations
./toolbox --prebuilt alloydb-postgres,alloydb-postgres-admin
# OR
./toolbox --prebuilt alloydb-postgres --prebuilt alloydb-postgres-admin
```

### Tool Configuration Sources

The CLI supports multiple mutually exclusive ways to specify tool configurations:

**Single File:** (default)

- `--tools-file`: Path to a single YAML configuration file (default: `tools.yaml`)

**Multiple Files:**

- `--tools-files`: Comma-separated list of YAML files to merge

**Directory:**

- `--tools-folder`: Directory containing YAML files to load and merge

**Prebuilt Configurations:**

- `--prebuilt`: Use one or more predefined configurations for specific database types (e.g.,
  'bigquery', 'postgres', 'spanner'). See [Prebuilt Tools
  Reference](prebuilt-tools.md) for allowed values.

{{< notice tip >}}
The CLI enforces mutual exclusivity between configuration source flags,
preventing simultaneous use of the file-based options ensuring only one of
`--tools-file`, `--tools-files`, or `--tools-folder` is
used at a time.
{{< /notice >}}

### Hot Reload

Toolbox enables dynamic reloading by default. To disable, use the
`--disable-reload` flag.

### Toolbox UI

To launch Toolbox's interactive UI, use the `--ui` flag. This allows you to test
tools and toolsets with features such as authorized parameters. To learn more,
visit [Toolbox UI](../how-to/toolbox-ui/index.md).
