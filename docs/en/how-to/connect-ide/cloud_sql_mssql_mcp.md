---
title: "Cloud SQL for SQL Server using MCP"
type: docs
weight: 2
description: >
  Connect your IDE to Cloud SQl for SQL Server using Toolbox.
---

[Model Context Protocol (MCP)](https://modelcontextprotocol.io/introduction) is an open protocol for connecting Large Language Models (LLMs) to data sources like Cloud SQL. This guide covers how to use [MCP Toolbox for Databases][toolbox] to expose your developer assistant tools to a Cloud SQL for SQL Server instance:

* [Cursor][cursor]
* [Windsurf][windsurf] (Codium)
* [Visual Studio Code ][vscode] (Copilot)
* [Cline][cline]  (VS Code extension)
* [Claude desktop][claudedesktop]
* [Claude code][claudecode]

[toolbox]: https://github.com/googleapis/genai-toolbox
[cursor]: #configure-your-mcp-client
[windsurf]: #configure-your-mcp-client
[vscode]: #configure-your-mcp-client
[cline]: #configure-your-mcp-client
[claudedesktop]: #configure-your-mcp-client
[claudecode]: #configure-your-mcp-client

## Before you begin

1. In the Google Cloud console, on the [project selector page](https://console.cloud.google.com/projectselector2/home/dashboard), select or create a Google Cloud project.

1. [Make sure that billing is enabled for your Google Cloud project](https://cloud.google.com/billing/docs/how-to/verify-billing-enabled#confirm_billing_is_enabled_on_a_project).


## Set up the database

1. [Enable the Cloud SQL Admin API in the Google Cloud project](https://console.cloud.google.com/flows/enableapi?apiid=sqladmin&redirect=https://console.cloud.google.com).

1. [Create or select a Cloud SQL for SQL Server instance](https://cloud.google.com/sql/docs/sqlserver/create-instance). These instructions assume that your Cloud SQL instance has a [public IP address](https://cloud.google.com/sql/docs/sqlserver/configure-ip). By default, Cloud SQL assigns a public IP address to a new instance. Toolbox will connect securely using the [Cloud SQL connectors](https://cloud.google.com/sql/docs/sqlserver/language-connectors).

1. Configure the required roles and permissions to complete this task. You will need [Cloud SQL > Client](https://cloud.google.com/sql/docs/sqlserver/roles-and-permissions#proxy-roles-permissions) role (`roles/cloudsql.client`) or equivalent IAM permissions to connect to the instance.

1. Configured [Application Default Credentials (ADC)](https://cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) for your environment.

1. Create or reuse [a database user](https://cloud.google.com/sql/docs/sqlserver/create-manage-users) and have the username and password ready.


## Install MCP Toolbox

1. Download the latest version of Toolbox as a binary. Select the [correct binary](https://github.com/googleapis/genai-toolbox/releases) corresponding to your OS and CPU architecture. You are required to use Toolbox version V0.6.0+:

   <!-- {x-release-please-start-version} -->
   {{< tabpane persist=header >}}
{{< tab header="linux/amd64" lang="bash" >}}
curl -O https://storage.googleapis.com/genai-toolbox/v0.6.0/linux/amd64/toolbox
{{< /tab >}}

{{< tab header="darwin/arm64" lang="bash" >}}
curl -O https://storage.googleapis.com/genai-toolbox/v0.6.0/darwin/arm64/toolbox
{{< /tab >}}

{{< tab header="darwin/amd64" lang="bash" >}}
curl -O https://storage.googleapis.com/genai-toolbox/v0.6.0/darwin/amd64/toolbox
{{< /tab >}}

{{< tab header="windows/amd64" lang="bash" >}}
curl -O https://storage.googleapis.com/genai-toolbox/v0.6.0/windows/amd64/toolbox
{{< /tab >}}
{{< /tabpane >}}
    <!-- {x-release-please-end} -->


1. Make the binary executable:

    ```bash
    chmod +x toolbox
    ```

1. Verify the installation:

    ```bash
    ./toolbox --version
    ```

## Configure and run Toolbox

This section will create a `tools.yaml` file, which will define which tools your AI Agent will have access to. You can add, remove, or edit tools as needed to make sure you have the best tools for your workflows.

This will configure the following tools:

1. **list_tables**: lists tables and descriptions
3. **execute_sql**: execute any SQL statement

To configure Toolbox, run the following steps:

1. Set the following environment variables:

    ```bash
    # The ID of your Google Cloud Project where the Cloud SQL instance is located.
    export CLOUD_SQL_MSSQL_PROJECT="your-gcp-project-id"

    # The region where your Cloud SQL instance is located (e.g., us-central1).
    export CLOUD_SQL_MSSQL_REGION="your-instance-region"

    # The name of your Cloud SQL instance.
    export CLOUD_SQL_MSSQL_INSTANCE="your-instance-name"

    # The name of the database you want to connect to within the instance.
    export CLOUD_SQL_MSSQL_DATABASE="your-database-name"

    # The username for connecting to the database.
    export CLOUD_SQL_MSSQL_USER="your-database-user"

    # The password for the specified database user.
    export CLOUD_SQL_MSSQL_PASSWORD="your-database-password"
    ```

2. Create a `tools.yaml` file.

3. Copy and paste the following contents into the `tools.yaml`:

    ```yaml
    sources:
      cloudsql-mssql-source:
        kind: cloud-sql-mysql
        project: ${CLOUD_SQL_PROJECT}
        region: ${CLOUD_SQL_REGION}
        instance: ${CLOUD_SQL_INSTANCE}
        database: ${CLOUD_SQL_DB}
        ipAddress: ${CLOUD_SQL_IP_ADDRESS}
        user: ${CLOUD_SQL_USER}
        password: ${CLOUD_SQL_PASS}
    tools:
      execute_sql:
        kind: mssql-execute-sql
        source: cloud-sql-mssql-source
        description: Use this tool to execute SQL.

      list_tables:
          kind: mssql-sql
          source: cloudsql-mssql-source
          description: "Lists detailed schema information (object type, columns, constraints, indexes, triggers, comment) as JSON for user-created tables (ordinary or partitioned). Filters by a comma-separated list of names. If names are omitted, lists all tables in user schemas."
          statement: |
              WITH table_info AS (
                  SELECT
                      t.object_id AS table_oid,
                      s.name AS schema_name,
                      t.name AS table_name,
                      dp.name AS table_owner, -- Schema's owner principal name
                      CAST(ep.value AS NVARCHAR(MAX)) AS table_comment, -- Cast for JSON compatibility
                      CASE
                          WHEN EXISTS ( -- Check if the table has more than one partition for any of its indexes or heap
                              SELECT 1 FROM sys.partitions p
                              WHERE p.object_id = t.object_id AND p.partition_number > 1
                          ) THEN 'PARTITIONED TABLE'
                          ELSE 'TABLE'
                      END AS object_type_detail
                  FROM
                      sys.tables t
                  INNER JOIN
                      sys.schemas s ON t.schema_id = s.schema_id
                  LEFT JOIN
                      sys.database_principals dp ON s.principal_id = dp.principal_id
                  LEFT JOIN
                      sys.extended_properties ep ON ep.major_id = t.object_id AND ep.minor_id = 0 AND ep.class = 1 AND ep.name = 'MS_Description'
                  WHERE
                      t.type = 'U' -- User tables
                      AND s.name NOT IN ('sys', 'INFORMATION_SCHEMA', 'guest', 'db_owner', 'db_accessadmin', 'db_backupoperator', 'db_datareader', 'db_datawriter', 'db_ddladmin', 'db_denydatareader', 'db_denydatawriter', 'db_securityadmin')
                      AND (@table_names IS NULL OR LTRIM(RTRIM(@table_names)) = '' OR t.name IN (SELECT LTRIM(RTRIM(value)) FROM STRING_SPLIT(@table_names, ',')))
              ),
              columns_info AS (
                  SELECT
                      c.object_id AS table_oid,
                      c.name AS column_name,
                      CONCAT(
                          UPPER(TY.name), -- Base type name
                          CASE
                              WHEN TY.name IN ('char', 'varchar', 'nchar', 'nvarchar', 'binary', 'varbinary') THEN
                                  CONCAT('(', IIF(c.max_length = -1, 'MAX', CAST(c.max_length / CASE WHEN TY.name IN ('nchar', 'nvarchar') THEN 2 ELSE 1 END AS VARCHAR(10))), ')')
                              WHEN TY.name IN ('decimal', 'numeric') THEN
                                  CONCAT('(', c.precision, ',', c.scale, ')')
                              WHEN TY.name IN ('datetime2', 'datetimeoffset', 'time') THEN
                                  CONCAT('(', c.scale, ')')
                              ELSE ''
                          END
                      ) AS data_type,
                      c.column_id AS column_ordinal_position,
                      IIF(c.is_nullable = 0, CAST(1 AS BIT), CAST(0 AS BIT)) AS is_not_nullable,
                      dc.definition AS column_default,
                      CAST(epc.value AS NVARCHAR(MAX)) AS column_comment
                  FROM
                      sys.columns c
                  JOIN
                      table_info ti ON c.object_id = ti.table_oid
                  JOIN
                      sys.types TY ON c.user_type_id = TY.user_type_id AND TY.is_user_defined = 0 -- Ensure we get base types
                  LEFT JOIN
                      sys.default_constraints dc ON c.object_id = dc.parent_object_id AND c.column_id = dc.parent_column_id
                  LEFT JOIN
                      sys.extended_properties epc ON epc.major_id = c.object_id AND epc.minor_id = c.column_id AND epc.class = 1 AND epc.name = 'MS_Description'
              ),
              constraints_info AS (
                  -- Primary Keys & Unique Constraints
                  SELECT
                      kc.parent_object_id AS table_oid,
                      kc.name AS constraint_name,
                      REPLACE(kc.type_desc, '_CONSTRAINT', '') AS constraint_type, -- 'PRIMARY_KEY', 'UNIQUE'
                      STUFF((SELECT ', ' + col.name
                          FROM sys.index_columns ic
                          JOIN sys.columns col ON ic.object_id = col.object_id AND ic.column_id = col.column_id
                          WHERE ic.object_id = kc.parent_object_id AND ic.index_id = kc.unique_index_id
                          ORDER BY ic.key_ordinal
                          FOR XML PATH(''), TYPE).value('.', 'NVARCHAR(MAX)'), 1, 2, '') AS constraint_columns,
                      NULL AS foreign_key_referenced_table,
                      NULL AS foreign_key_referenced_columns,
                      CASE kc.type
                          WHEN 'PK' THEN 'PRIMARY KEY (' + STUFF((SELECT ', ' + col.name FROM sys.index_columns ic JOIN sys.columns col ON ic.object_id = col.object_id AND ic.column_id = col.column_id WHERE ic.object_id = kc.parent_object_id AND ic.index_id = kc.unique_index_id ORDER BY ic.key_ordinal FOR XML PATH(''), TYPE).value('.', 'NVARCHAR(MAX)'), 1, 2, '') + ')'
                          WHEN 'UQ' THEN 'UNIQUE (' + STUFF((SELECT ', ' + col.name FROM sys.index_columns ic JOIN sys.columns col ON ic.object_id = col.object_id AND ic.column_id = col.column_id WHERE ic.object_id = kc.parent_object_id AND ic.index_id = kc.unique_index_id ORDER BY ic.key_ordinal FOR XML PATH(''), TYPE).value('.', 'NVARCHAR(MAX)'), 1, 2, '') + ')'
                      END AS constraint_definition
                  FROM sys.key_constraints kc
                  JOIN table_info ti ON kc.parent_object_id = ti.table_oid

                  UNION ALL

                  -- Foreign Keys
                  SELECT
                      fk.parent_object_id AS table_oid,
                      fk.name AS constraint_name,
                      'FOREIGN KEY' AS constraint_type,
                      STUFF((SELECT ', ' + pc.name
                          FROM sys.foreign_key_columns fkc
                          JOIN sys.columns pc ON fkc.parent_object_id = pc.object_id AND fkc.parent_column_id = pc.column_id
                          WHERE fkc.constraint_object_id = fk.object_id
                          ORDER BY fkc.constraint_column_id
                          FOR XML PATH(''), TYPE).value('.', 'NVARCHAR(MAX)'), 1, 2, '') AS constraint_columns,
                      SCHEMA_NAME(rt.schema_id) + '.' + OBJECT_NAME(fk.referenced_object_id) AS foreign_key_referenced_table,
                      STUFF((SELECT ', ' + rc.name
                          FROM sys.foreign_key_columns fkc
                          JOIN sys.columns rc ON fkc.referenced_object_id = rc.object_id AND fkc.referenced_column_id = rc.column_id
                          WHERE fkc.constraint_object_id = fk.object_id
                          ORDER BY fkc.constraint_column_id
                          FOR XML PATH(''), TYPE).value('.', 'NVARCHAR(MAX)'), 1, 2, '') AS foreign_key_referenced_columns,
                      OBJECT_DEFINITION(fk.object_id) AS constraint_definition
                  FROM sys.foreign_keys fk
                  JOIN sys.tables rt ON fk.referenced_object_id = rt.object_id
                  JOIN table_info ti ON fk.parent_object_id = ti.table_oid

                  UNION ALL

                  -- Check Constraints
                  SELECT
                      cc.parent_object_id AS table_oid,
                      cc.name AS constraint_name,
                      'CHECK' AS constraint_type,
                      NULL AS constraint_columns, -- Definition includes column context
                      NULL AS foreign_key_referenced_table,
                      NULL AS foreign_key_referenced_columns,
                      cc.definition AS constraint_definition
                  FROM sys.check_constraints cc
                  JOIN table_info ti ON cc.parent_object_id = ti.table_oid
              ),
              indexes_info AS (
                  SELECT
                      i.object_id AS table_oid,
                      i.name AS index_name,
                      i.type_desc AS index_method, -- CLUSTERED, NONCLUSTERED, XML, etc.
                      i.is_unique,
                      i.is_primary_key AS is_primary,
                      STUFF((SELECT ', ' + c.name
                          FROM sys.index_columns ic
                          JOIN sys.columns c ON i.object_id = c.object_id AND ic.column_id = c.column_id
                          WHERE ic.object_id = i.object_id AND ic.index_id = i.index_id AND ic.is_included_column = 0
                          ORDER BY ic.key_ordinal
                          FOR XML PATH(''), TYPE).value('.', 'NVARCHAR(MAX)'), 1, 2, '') AS index_columns,
                      (
                          'COLUMNS: (' + ISNULL(STUFF((SELECT ', ' + c.name + CASE WHEN ic.is_descending_key = 1 THEN ' DESC' ELSE '' END
                                                  FROM sys.index_columns ic
                                                  JOIN sys.columns c ON i.object_id = c.object_id AND ic.column_id = c.column_id
                                                  WHERE ic.object_id = i.object_id AND ic.index_id = i.index_id AND ic.is_included_column = 0
                                                  ORDER BY ic.key_ordinal FOR XML PATH(''), TYPE).value('.', 'NVARCHAR(MAX)'), 1, 2, ''), 'N/A') + ')' +
                          ISNULL(CHAR(13)+CHAR(10) + 'INCLUDE: (' + STUFF((SELECT ', ' + c.name
                                                  FROM sys.index_columns ic
                                                  JOIN sys.columns c ON i.object_id = c.object_id AND ic.column_id = c.column_id
                                                  WHERE ic.object_id = i.object_id AND ic.index_id = i.index_id AND ic.is_included_column = 1
                                                  ORDER BY ic.index_column_id FOR XML PATH(''), TYPE).value('.', 'NVARCHAR(MAX)'), 1, 2, '') + ')', '') +
                          ISNULL(CHAR(13)+CHAR(10) + 'FILTER: (' + i.filter_definition + ')', '')
                      ) AS index_definition_details
                  FROM
                      sys.indexes i
                  JOIN
                      table_info ti ON i.object_id = ti.table_oid
                  WHERE i.type <> 0 -- Exclude Heaps
                  AND i.name IS NOT NULL -- Exclude unnamed heap indexes; named indexes (PKs are often named) are preferred.
              ),
              triggers_info AS (
                  SELECT
                      tr.parent_id AS table_oid,
                      tr.name AS trigger_name,
                      OBJECT_DEFINITION(tr.object_id) AS trigger_definition,
                      CASE tr.is_disabled WHEN 0 THEN 'ENABLED' ELSE 'DISABLED' END AS trigger_enabled_state
                  FROM
                      sys.triggers tr
                  JOIN
                      table_info ti ON tr.parent_id = ti.table_oid
                  WHERE
                      tr.is_ms_shipped = 0
                      AND tr.parent_class_desc = 'OBJECT_OR_COLUMN' -- DML Triggers on tables/views
              )
              SELECT
                  ti.schema_name,
                  ti.table_name AS object_name,
                  (
                      SELECT
                          ti.schema_name AS schema_name,
                          ti.table_name AS object_name,
                          ti.object_type_detail AS object_type,
                          ti.table_owner AS owner,
                          ti.table_comment AS comment,
                          JSON_QUERY(ISNULL((
                              SELECT
                                  ci.column_name,
                                  ci.data_type,
                                  ci.column_ordinal_position,
                                  ci.is_not_nullable,
                                  ci.column_default,
                                  ci.column_comment
                              FROM columns_info ci
                              WHERE ci.table_oid = ti.table_oid
                              ORDER BY ci.column_ordinal_position
                              FOR JSON PATH
                          ), '[]')) AS columns,
                          JSON_QUERY(ISNULL((
                              SELECT
                                  cons.constraint_name,
                                  cons.constraint_type,
                                  cons.constraint_definition,
                                  JSON_QUERY(
                                      CASE
                                          WHEN cons.constraint_columns IS NOT NULL AND LTRIM(RTRIM(cons.constraint_columns)) <> ''
                                          THEN '[' + (SELECT STRING_AGG('"' + LTRIM(RTRIM(value)) + '"', ',') FROM STRING_SPLIT(cons.constraint_columns, ',')) + ']'
                                          ELSE '[]'
                                      END
                                  ) AS constraint_columns,
                                  cons.foreign_key_referenced_table,
                                  JSON_QUERY(
                                      CASE
                                          WHEN cons.foreign_key_referenced_columns IS NOT NULL AND LTRIM(RTRIM(cons.foreign_key_referenced_columns)) <> ''
                                          THEN '[' + (SELECT STRING_AGG('"' + LTRIM(RTRIM(value)) + '"', ',') FROM STRING_SPLIT(cons.foreign_key_referenced_columns, ',')) + ']'
                                          ELSE '[]'
                                      END
                                  ) AS foreign_key_referenced_columns
                              FROM constraints_info cons
                              WHERE cons.table_oid = ti.table_oid
                              FOR JSON PATH
                          ), '[]')) AS constraints,
                          JSON_QUERY(ISNULL((
                              SELECT
                                  ii.index_name,
                                  ii.index_definition_details AS index_definition,
                                  ii.is_unique,
                                  ii.is_primary,
                                  ii.index_method,
                                  JSON_QUERY(
                                      CASE
                                          WHEN ii.index_columns IS NOT NULL AND LTRIM(RTRIM(ii.index_columns)) <> ''
                                          THEN '[' + (SELECT STRING_AGG('"' + LTRIM(RTRIM(value)) + '"', ',') FROM STRING_SPLIT(ii.index_columns, ',')) + ']'
                                          ELSE '[]'
                                      END
                                  ) AS index_columns
                              FROM indexes_info ii
                              WHERE ii.table_oid = ti.table_oid
                              FOR JSON PATH
                          ), '[]')) AS indexes,
                          JSON_QUERY(ISNULL((
                              SELECT
                                  tri.trigger_name,
                                  tri.trigger_definition,
                                  tri.trigger_enabled_state
                              FROM triggers_info tri
                              WHERE tri.table_oid = ti.table_oid
                              FOR JSON PATH
                          ), '[]')) AS triggers
                      FOR JSON PATH, WITHOUT_ARRAY_WRAPPER -- Creates a single JSON object for this table's details
                  ) AS object_details
              FROM
                  table_info ti
              ORDER BY
                  ti.schema_name, ti.table_name;
          parameters:
            - name: table_names
              type: string
              description: "Optional: A comma-separated list of table names. If empty, details for all tables in user-accessible schemas will be listed."
    ```

4. Start Toolbox to listen on `127.0.0.1:5000`:

    ```bash
    ./toolbox --tools-file tools.yaml --address 127.0.0.1 --port 5000
    ```

{{< notice tip >}}
To stop the Toolbox server when you're finished, press `ctrl+c` to send the terminate signal.
{{< /notice >}}

## Configure your MCP Client

{{< tabpane text=true >}}
{{% tab header="Claude code" lang="en" %}}

1. Install [Claude Code](https://docs.anthropic.com/en/docs/agents-and-tools/claude-code/overview).
2. Create a `.mcp.json` file in your project root if it doesn't exist.
3. Add the following configuration and save:

    ```json
    {
      "mcpServers": {
        "cloud-sql-sqlserver": {
          "type": "sse",
          "url": "http://127.0.0.1:5000/mcp/sse"
        }
      }
    }
    ```

4. Restart Claude code to apply the new configuration.
{{% /tab %}}

{{% tab header="Claude desktop" lang="en" %}}

1. Install [`npx`](https://docs.npmjs.com/cli/v8/commands/npx).
2. Open [Claude desktop](https://claude.ai/download) and navigate to Settings.
3. Under the Developer tab, tap Edit Config to open the configuration file.
4. Add the following configuration and save:

    ```json
    {
      "mcpServers": {
        "cloud-sql-sqlserver": {
          "command": "npx",
          "args": [
            "-y",
            "mcp-remote",
            "http://127.0.0.1:5000/mcp/sse"
          ]
        }
      }
    }
    ```

5. Restart Claude desktop.
6. From the new chat screen, you should see a hammer (MCP) icon appear with the new MCP server available.
{{% /tab %}}

{{% tab header="Cline" lang="en" %}}

1. Open the [Cline](https://github.com/cline/cline) extension in VS Code and tap the **MCP Servers** icon.
2. Tap Configure MCP Servers to open the configuration file.
3. Add the following configuration and save:

    ```json
    {
      "mcpServers": {
        "cloud-sql-sqlserver": {
          "type": "sse",
          "url": "http://127.0.0.1:5000/mcp/sse"
        }
      }
    }
    ```

4. You should see a green active status after the server is successfully connected.
{{% /tab %}}

{{% tab header="Cursor" lang="en" %}}

1. Create a `.cursor` directory in your project root if it doesn't exist.
2. Create a `.cursor/mcp.json` file if it doesn't exist and open it.
3. Add the following configuration:

    ```json
    {
      "mcpServers": {
        "cloud-sql-sqlserver": {
          "type": "sse",
          "url": "http://127.0.0.1:5000/mcp/sse"
        }
      }
    }
    ```

4. [Cursor](https://www.cursor.com/) and navigate to **Settings > Cursor Settings > MCP**. You should see a green active status after the server is successfully connected.
{{% /tab %}}

{{% tab header="Visual Studio Code (Copilot)" lang="en" %}}

1. Open [VS Code](https://code.visualstudio.com/docs/copilot/overview) and create a `.vscode` directory in your project root if it doesn't exist.
2. Create a `.vscode/mcp.json` file if it doesn't exist and open it.
3. Add the following configuration:

    ```json
    {
      "mcpServers": {
        "cloud-sql-sqlserver": {
          "type": "sse",
          "url": "http://127.0.0.1:5000/mcp/sse"
        }
      }
    }
    ```
{{% /tab %}}

{{% tab header="Windsurf" lang="en" %}}

1. Open [Windsurf](https://docs.codeium.com/windsurf) and navigate to the Cascade assistant.
2. Tap on the hammer (MCP) icon, then Configure to open the configuration file.
3. Add the following configuration:

    ```json
    {
      "mcpServers": {
        "cloud-sql-sqlserver": {
          "serverUrl": "http://127.0.0.1:5000/mcp/sse"
        }
      }
    }

    ```
{{% /tab %}}
{{< /tabpane >}}
