# sqlcommenter

Internal package that annotates outgoing SQL queries and BigQuery jobs with
observability metadata, following the [SQLCommenter specification][spec].

The annotations make every query visible in **Cloud SQL Insights** grouped by
the MCP tool that issued it, the service that owns the toolbox instance, and
the framework version.

---

## How it works

### Postgres / Cloud SQL Postgres / AlloyDB / MySQL / Cloud SQL MySQL

A trailing comment is appended to the SQL string before it reaches the driver:

```sql
SELECT * FROM orders WHERE id = $1
/*action='get-orders',application='my-service',controller='sales-agent',db_driver='pgx',framework='mcp-toolbox'*/
```

Cloud SQL Insights indexes these keys and lets you filter and group queries by
`action` (tool name), `controller` (agent name), `application` (service name),
`db_driver`, and `framework` in the Insights dashboard.

### BigQuery

BigQuery ignores SQL comments for observability purposes, so labels are attached
to the **QueryJobConfig** instead. They are visible in `INFORMATION_SCHEMA.JOBS`
and the BigQuery console under job details:

```
labels.action      = "get-orders"
labels.application = "my-service"
labels.framework   = "mcp-toolbox"
```

---

## Controller tag — how the agent name is resolved

The `controller` tag identifies which AI agent issued the query. It is
resolved automatically with this priority (highest → lowest):

1. **Dynamic — MCP `clientInfo.name`** (automatic, zero config)
   When an MCP client connects it sends an `initialize` message containing
   `clientInfo: { name: "sales-agent", version: "1.0" }`. The toolbox server
   captures this name per session and injects it into the context before
   every `tools/call`. Multiple agents connecting to the same toolbox
   instance each get their own `controller` tag with no configuration.

2. **Static — `TOOLBOX_AGENT_NAME` env var** (fallback for non-MCP callers)
   Set this when all callers share a single agent identity, or when calling
   the REST API directly rather than via MCP.

3. **Tool name** (last resort)
   If neither of the above is set, `controller` mirrors `action` (the tool
   name), so at least the query is still distinguishable.

## Configuration

| Environment variable | Purpose | Default |
|---|---|---|
| `TOOLBOX_APP_NAME` | The `application` tag — identifies the service running the toolbox (e.g. `order-service`). | `mcp-toolbox` |
| `TOOLBOX_AGENT_NAME` | Static fallback for `controller` when MCP `clientInfo.name` is not available. | *(tool name)* |

---

## Package API

```go
// --- Tool layer (called in each tool's Invoke) ---
ctx = sqlcommenter.WithToolName(ctx, t.Name)  // sets "action"
ctx = sqlcommenter.WithDBDriver(ctx, "pgx")   // sets "db_driver"; "mysql" for MySQL

// Append a SQLCommenter comment (Postgres / MySQL path).
stmt := sqlcommenter.AppendComment(ctx, sql)

// Build BigQuery job labels (BigQuery source RunSQL path).
query.Labels = sqlcommenter.JobLabels(ctx)

// --- Server layer (called automatically; no manual wiring needed) ---
ctx = sqlcommenter.WithAgentName(ctx, clientInfo.Name)  // injected per session
```

---

| `internal/sources/pgx_util.go` | **Centralized Handler** — Executes SQL with SQLCommenter telemetry for all Postgres-based drivers |
| `internal/server/mcp.go` | Stores `clientName` per session on `initialize`; handles **onRemove** session cleanup to prevent leaks |
| `internal/server/server.go` | Initializes `mcpClientNames` and lifecycle callbacks for memory safety |

Cloud SQL Postgres, Cloud SQL MySQL, and AlloyDB are covered automatically —
they reuse the same `postgres-execute-sql` / `mysql-execute-sql` tool types,
only backed by different source drivers.

---

## Comment format

Keys and values are **percent-encoded** (RFC 3986 via `net/url.QueryEscape`)
and emitted in **lexicographic key order**, matching the SQLCommenter spec.
A trailing semicolon in the original SQL is preserved — the comment is placed
before it so the statement remains syntactically valid.

Keys emitted per driver:

| Key | Value | Postgres/MySQL | BigQuery |
|---|---|:---:|:---:|
| `action` | Tool name (e.g. `get-orders`) | ✓ comment | ✓ label |
| `application` | `$TOOLBOX_APP_NAME` | ✓ comment | ✓ label |
| `controller` | Agent name (auto from MCP) → `$TOOLBOX_AGENT_NAME` → tool name | ✓ comment | — |
| `db_driver` | `pgx` or `mysql` | ✓ comment | — |
| `framework` | `mcp-toolbox` | ✓ comment | ✓ label |

[spec]: https://google.github.io/sqlcommenter/
