---
title: "CLI"
type: docs
weight: 1
description: >
  This page describes the `toolbox` command-line options.
---

## Reference

| Flag (Short) | Flag (Long)                | Description                                                                                                                                                                                   | Default     |
|--------------|----------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------|
| `-a`         | `--address`                | Address of the interface the server will listen on.                                                                                                                                           | `127.0.0.1` |
|              | `--disable-reload`         | Disables dynamic reloading of tools file.                                                                                                                                                     |             |
| `-h`         | `--help`                   | help for toolbox                                                                                                                                                                              |             |
|              | `--log-level`              | Specify the minimum level logged. Allowed: 'DEBUG', 'INFO', 'WARN', 'ERROR'.                                                                                                                  | `info`      |
|              | `--logging-format`         | Specify logging format to use. Allowed: 'standard' or 'JSON'.                                                                                                                                 | `standard`  |
| `-p`         | `--port`                   | Port the server will listen on.                                                                                                                                                               | `5000`      |
|              | `--prebuilt`               | Use a prebuilt tool configuration by source type. Cannot be used with --tools-file. See [Prebuilt Tools Reference](prebuilt-tools.md) for allowed values.                                     |             |
|              | `--stdio`                  | Listens via MCP STDIO instead of acting as a remote HTTP server.                                                                                                                              |             |
|              | `--telemetry-gcp`          | Enable exporting directly to Google Cloud Monitoring.                                                                                                                                         |             |
|              | `--telemetry-otlp`         | Enable exporting using OpenTelemetry Protocol (OTLP) to the specified endpoint (e.g. 'http://127.0.0.1:4318')                                                                                 |             |
|              | `--telemetry-service-name` | Sets the value of the service.name resource attribute for telemetry data.                                                                                                                     | `toolbox`   |
|              | `--tools-file`             | File path specifying the tool configuration. Cannot be used with --prebuilt, --tools-files, or --tools-folder.                                                                                |             |
|              | `--tools-files`            | Multiple file paths specifying tool configurations. Files will be merged. Cannot be used with --prebuilt, --tools-file, or --tools-folder.                                                    |             |
|              | `--tools-folder`           | Directory path containing YAML tool configuration files. All .yaml and .yml files in the directory will be loaded and merged. Cannot be used with --prebuilt, --tools-file, or --tools-files. |             |
|              | `--ui`                     | Launches the Toolbox UI web server.                                                                                                                                                           |             |
|              | `--allowed-origins`        | Specifies a list of origins permitted to access this server.                                                                                                                                  | `*`         |
| `-v`         | `--version`                | version for toolbox                                                                                                                                                                           |             |

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

- `--prebuilt`: Use predefined configurations for specific database types (e.g.,
  'bigquery', 'postgres', 'spanner'). See [Prebuilt Tools
  Reference](prebuilt-tools.md) for allowed values.

{{< notice tip >}}
The CLI enforces mutual exclusivity between configuration source flags,
preventing simultaneous use of `--prebuilt` with file-based options, and
ensuring only one of `--tools-file`, `--tools-files`, or `--tools-folder` is
used at a time.
{{< /notice >}}

### Hot Reload

Toolbox enables dynamic reloading by default. To disable, use the
`--disable-reload` flag.

### Toolbox UI

To launch Toolbox's interactive UI, use the `--ui` flag. This allows you to test
tools and toolsets with features such as authorized parameters. To learn more,
visit [Toolbox UI](../how-to/toolbox-ui/index.md).
