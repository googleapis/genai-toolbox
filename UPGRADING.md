# Upgrading to MCP Toolbox for Databases v1.0.0

Welcome to the v1.0.0 release of the MCP Toolbox for Databases!

This release stabilizes core APIs and aligns the system with the Model Context Protocol (MCP). It introduces several breaking changes and deprecations. Please review and update your setup accordingly.

---

## 📖 Versioning Policy

We follow semantic versioning:

* **Major (vX.0.0):** Breaking changes requiring manual migration
* **Minor (v1.X.0):** Backward-compatible features and deprecations
* **Patch (v1.0.X):** Bug fixes and security updates

---

## 🚨 Breaking Changes (Action Required)

### 1. Repository Rename

The repository has been renamed to:

`googleapis/mcp-toolbox`

### Migration steps:

```bash
# Rename your local folder (adjust old name if different)
mv <old-repo-folder> mcp-toolbox
cd mcp-toolbox

# Update remote URL
git remote set-url origin git@github.com:googleapis/mcp-toolbox.git

# Verify
git remote -v
```

---

### 2. Endpoint Change: `/api` deprecated

The `/api` endpoint is deprecated. Use `/mcp` instead.

Some legacy systems may still require `/api`, but it is no longer enabled by default.

Enable legacy support (if required):

```bash
./toolbox --enable-api
```

All SDKs should migrate to:

```
/mcp
```

---

### 3. Tool Naming Rules (MCP Standard)

Tool names must follow MCP naming rules:

* Allowed: `a-z`, `A-Z`, `0-9`, `-`, `_`, `.`
* No spaces or special characters

Invalid names will fail MCP initialization.

Reference: MCP specification for tool naming.

---

### 4. CLI Flag Changes

Removed:

* `--tools-file`
* `--tools-files`
* `--tools-folder`

Use instead:

```bash
--config
--configs
--config-folder
```

---

### 5. Configuration Schema Update

* `kind` now represents the resource type (`source` or `tool`)
* `type` defines the specific implementation

Example:

```yaml
kind: source
name: my-source
type: alloydb-postgres
project: my-project
region: my-region
instance: my-instance
---
kind: tool
name: my-tool
type: postgres-execute-sql
source: my-source
description: Executes SQL queries on the configured database
```

---

### 6. Configuration Field Rename

* `authSources` → `authService`

Update all configuration files accordingly.

---

### 7. CloudSQL SQL Server Change

The `ipAddress` field has been removed.

Remove it from all configurations.

---

## ⚠️ Deprecations & Modernization

### 1. Flat Configuration Format

A new flat configuration format is now recommended.

Legacy nested format is still supported but will be phased out.

---

### Migration Tool

```bash
./toolbox migrate --config <path>
```

Supports:

* single config file
* multiple configs
* config folders

---

## 💡 Other Updates

* Improved error classification (agent vs system errors)
* Updated telemetry to OpenTelemetry MCP standards
* Full MCP authorization support added
* CloudSQL MySQL validation relaxed for database name field
* Prebuilt toolsets optimized for performance

---

## 📚 Documentation Update

Official documentation has moved to:

https://mcp-toolbox.dev

Please update bookmarks accordingly.
