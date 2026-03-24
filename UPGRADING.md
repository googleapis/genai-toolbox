# Upgrading to MCP Toolbox for Databases v1.0.0

Welcome to the v1.0.0 release of the MCP Toolbox for Databases! 

This release stabilizes our core APIs and standardizes our protocol alignments.
As part of this milestone, we have introduced several breaking changes and
deprecations that require updates to your configuration and code.

**📖 New Versioning Policy**
We have officially published our [Versioning Policy](https://googleapis.github.io/genai-toolbox/dev/about/versioning/). Moving forward, we follow standard versioning conventions to classify updates:
* **Major (vX.0.0):** Breaking changes requiring manual updates.
* **Minor (v1.X.0):** New, backward-compatible features and deprecation notices.
* **Patch (v1.0.X):** Backward-compatible bug fixes and security patches.

This guide outlines what has changed and the steps you need to take to upgrade.

## 🚨 Breaking Changes (Action Required)

### 1. Endpoint Standardization: `/api` removed
The legacy `/api` endpoint has been completely removed. All official SDKs have been updated to use the `/mcp` endpoint instead.
* **Migration:** You must update all your implementation to use the `/mcp`
  endpoint exclusively.

### 2. Strict Tool Naming Validation (SEP986)
Tool names are now strictly validated against [ModelContextProtocol SEP986 guidelines](https://github.com/alexhancock/modelcontextprotocol/blob/main/docs/specification/draft/server/tools.mdx#tool-names) prior to MCP initialization.
* **Migration:** Ensure all your tool names **only** contain alphanumeric characters, hyphens (`-`), underscores (`_`), and periods (`.`). Any other special characters will cause initialization to fail.

### 3. Removed CLI Flags
The legacy snake_case flag `--tools_file` has been completely removed.
* **Migration:** Update your deployment scripts to use `--config` instead.

### 4. Singular `kind` Values in Configuration
All primitive kind fields in configuration files have been updated to use singular nouns instead of plural. For example, kind: sources is now kind: source, and kind: tools is now kind: tool.

* **Migration:** Update your configuration files to use the singular form for all `kind`
values. _(Note: If you are using the ./toolbox migrate command to transition to the new flat format, this conversion is handled automatically)._


### 5. Configuration Schema: `authSources` renamed
The `authSources` field is no longer supported in configuration files.
* **Migration:** Rename all instances of `authSources` to `authService` in your
  configuration files.

### 6. CloudSQL for SQL Server: `ipAddress` removed
The `ipAddress` field for the CloudSQL for SQL Server source was redundant and has been removed.
* **Migration:** Remove the `ipAddress` field from your CloudSQL for SQL Server configurations.


## ⚠️ Deprecations & Modernization

### 1. Flat Configuration Format Introduced
We have introduced a new, streamlined "flat" format for configuration files. While the older nested format is still supported for now, **all new features will only be added to the flat format.**

**Schema Restructuring (`kind` vs. `type`):**
Along with the flat format, the configuration schema has been reorganized. The
old `kind` field (which specified the specific primitive types, like
`alloydb-postgres`) has been renamed to `type`. The `kind` field is now strictly
used to declare the core primitive of the block (e.g., `source` or `tool`).

**Example of the new flat format:**

```yaml
kind: source
name: my-source
type: alloydb-postgres
project: my-project
region: my-region
instance: my-instance
---
kind: tool
name: my-simple-tool
type: postgres-execute-sql
source: my-source
description: this is a tool that executes the sql provided.
```

**Migration:**

You can automatically migrate your existing nested configurations to the new flat format using the CLI. Run the following command:

```Bash
./toolbox migrate --config <path-to-your-config>
```
_Note: You can also use the `--configs` or `--config-folder` flags with this command._

### 2. Deprecated CLI Flags
The following CLI flags are deprecated and will be removed in a future release. Please update your scripts:

* `--tools-file` ➡️ Use `--config`
* `--tools-files` ➡️ Use `--configs`
* `--tools-folder` ➡️ Use `--config-folder`

## 💡 Other Notable Updates
* **Enhanced Error Handling:** Errors are now strictly categorized between Agent Errors (allowing the LLM to self-correct) and Client/Server Errors (which signal a hard stop).

* **Telemetry Updates:** The /mcp endpoint telemetry has been revised to fully comply with the OpenTelemetry semantic conventions for MCP.

* **MCP Auth Support:** The ModelContextProtocol's auth specification is now fully supported.

* **Database Name Validation:** Removed the "required field" validation for the database name in CloudSQL for MySQL and generic MySQL sources.

* **Prebuilt Tools:** Toolsets have been resized for better performance, and tool names are now aligned with OneMCP.

* **Security Tooling:** Release binaries now include an attached signature to support modern security tooling.

## 📚 Documentation Moved
Our official documentation has a new home! Please update your bookmarks to [mcp-toolbox.dev](http://mcp-toolbox.dev).