---
title: cloud-sql-mssql-connect-gce
type: docs
weight: 12
description: Connect a Cloud SQL for SQL Server instance to a Google Compute Engine VM — validates network reachability, recommends a connection method, and emits setup steps plus an optional code snippet.
---

## About

The `cloud-sql-mssql-connect-gce` tool helps an agent (or user) wire up a
Cloud SQL for SQL Server instance to a Google Compute Engine VM. It:

1. Reads the Cloud SQL instance via the `cloud-sql-admin` source and the
   target VM via the Compute Engine API (auto-discovering the VM zone with an
   `AggregatedList` filter when `vm_zone` is omitted).
2. Validates network reachability between the two (same-VPC / private-IP /
   public-IP posture, SSL requirements).
3. Recommends a primary connection method — Cloud SQL Auth Proxy or Direct
   Private IP — plus ranked alternatives. (The Cloud SQL Connector library
   supports Postgres and MySQL only; for SQL Server, prefer the Auth Proxy.)
4. Returns ready-to-paste setup steps, environment/DSN/JDBC snippets, and —
   when the caller passes `language` — a Python / Node.js / Java / Go code
   snippet using the appropriate driver.

Callers only need to pass the instance connection name and the VM name. All
identifiers that flow into shell commands, DSNs, or the GCE list filter are
validated against GCP naming rules before use.

## Compatible Sources

{{< compatible-sources >}}

## Requirements

- The service account executing Toolbox needs `roles/cloudsql.client` on the
  Cloud SQL instance and `roles/compute.viewer` on the project.
- The Cloud SQL Admin API (`sqladmin.googleapis.com`) and Compute Engine API
  (`compute.googleapis.com`) must be enabled.

## Parameters

| **parameter**            | **type** | **required** | **description**                                                                                            |
| ------------------------ | :------: | :----------: | ---------------------------------------------------------------------------------------------------------- |
| instance_connection_name |  string  |     true     | Cloud SQL instance connection name in the format `project:region:instance`.                                |
| vm_name                  |  string  |     true     | Name of the GCE VM instance to connect from.                                                               |
| vm_zone                  |  string  |    false     | Zone of the VM. If omitted, the tool searches all zones in the project via an aggregated list filter.      |
| database_name            |  string  |    false     | Database name to connect to. Defaults to `master`.                                                         |
| language                 |  string  |    false     | Programming language for a code snippet: `python`, `nodejs`, `java`, or `go`. Omit to skip snippet output. |

## Example

```yaml
sources:
  my-cloud-sql-admin-source:
    kind: cloud-sql-admin

tools:
  connect_to_gce:
    kind: tool
    type: cloud-sql-mssql-connect-gce
    source: my-cloud-sql-admin-source
    description: Help me connect a Cloud SQL SQL Server instance to a GCE VM.
```

## Output Format

The tool returns a JSON object containing:

- `instanceConnectionName`, `project`, `region`, `databaseType`,
  `databaseVersion`
- `computeType` (`gce`), `computeResource` (the VM name), `computeLocation`
  (the resolved zone)
- `validation` — checks performed and their status
- `recommendedMethod` and `alternativeMethods` — one of `auth_proxy` or
  `direct_private_ip`, with rationale and requirements
- `connectionStrings` — host/port/DSN/JDBC templates (credential placeholders
  are `USER` / `PASS`)
- `environmentConfig` — env vars and, where relevant, the Auth Proxy launch
  command
- `setupSteps` — ordered actions the user needs to take
- `codeSnippet` — populated only when `language` was set
- `requiredIamRoles`, `requiredApis`

## Reference

### Tool Configuration

| **field**   | **type** | **required** | **description**                                  |
| ----------- | :------: | :----------: | ------------------------------------------------ |
| type        |  string  |     true     | Must be `cloud-sql-mssql-connect-gce`.           |
| source      |  string  |     true     | The name of the `cloud-sql-admin` source to use. |
| description |  string  |    false     | Overrides the default tool description.          |
