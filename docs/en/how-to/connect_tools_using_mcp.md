---
title: "Connect Toolbox to AI tools using MCP"
type: docs
weight: 2
description: >
  Connect your AI developer assistant tools to Toolbox using MCP.
---

[Model Context Protocol (MCP)](https://modelcontextprotocol.io/introduction) is an open protocol for connecting Large Language Models (LLMs) to data sources like Cloud SQL. This guide covers how to use [MCP Toolbox for Databases][toolbox] to expose your developer assistant tools to a Cloud SQL for Postgres instance:

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


## Set up a Cloud SQL instance

1. [Enable the Cloud SQL Admin API in the Google Cloud project](https://console.cloud.google.com/flows/enableapi?apiid=sqladmin&redirect=https://console.cloud.google.com).

1. [Create a Cloud SQL for PostgreSQL instance](https://cloud.google.com/sql/docs/postgres/create-instance). These instructions assume that your Cloud SQL instance has a [public IP address](https://cloud.google.com/sql/docs/postgres/configure-ip). By default, Cloud SQL assigns a public IP address to a new instance. Toolbox will connect securely using the [Cloud SQL connectors](https://cloud.google.com/sql/docs/postgres/language-connectors).

1. Configure the required roles and permissions to complete this task. You will need [Cloud SQL > Client](https://cloud.google.com/sql/docs/postgres/roles-and-permissions#proxy-roles-permissions) role (`roles/cloudsql.client`) or equivalent IAM permissions to connect to the instance.

1. Configured [Application Default Credentials (ADC)](https://cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) for your environment.

1. Create or reuse [a database user](https://cloud.google.com/sql/docs/postgres/create-manage-users) and have the username and password ready.


## Install MCP Toolbox

1. Download the latest version of Toolbox as a binary. Select the [correct binary](https://github.com/googleapis/genai-toolbox/releases) corresponding to your OS and CPU architecture. You are required to use Toolbox version V0.5.0+:
    <!-- {x-release-please-start-version} -->
    {{< tabpane persist=header >}}
{{< tab header="linux/amd64" lang="bash" >}}
curl -O https://storage.googleapis.com/genai-toolbox/v0.5.0/linux/amd64/toolbox
{{< /tab >}}

{{< tab header="darwin/arm64" lang="bash" >}}
curl -O https://storage.googleapis.com/genai-toolbox/v0.5.0/darwin/arm64/toolbox
{{< /tab >}}

{{< tab header="darwin/amd64" lang="bash" >}}
curl -O https://storage.googleapis.com/genai-toolbox/v0.5.0/darwin/amd64/toolbox
{{< /tab >}}

{{< tab header="windows/amd64" lang="bash" >}}
curl -O https://storage.googleapis.com/genai-toolbox/v0.5.0/windows/amd64/toolbox
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
    export CLOUD_SQL_PROJECT="your-gcp-project-id"

    # The region where your Cloud SQL instance is located (e.g., us-central1).
    export CLOUD_SQL_REGION="your-instance-region"

    # The name of your Cloud SQL instance.
    export CLOUD_SQL_INSTANCE="your-instance-name"

    # The name of the database you want to connect to within the instance.
    export CLOUD_SQL_DB="your-database-name"

    # The username for connecting to the database.
    export CLOUD_SQL_USER="your-database-user"

    # The password for the specified database user.
    export CLOUD_SQL_PASS="your-database-password"
    ```

2. Create a `tools.yaml` file.

3. Copy and paste the following contents into the `tools.yaml`:

    ```yaml
    sources:
      cloudsql-pg-source:
        kind: cloud-sql-postgres
        project: ${CLOUD_SQL_PROJECT}
        region: ${CLOUD_SQL_REGION}
        instance: ${CLOUD_SQL_INSTANCE}
        database: ${CLOUD_SQL_DB}
        user: ${CLOUD_SQL_USER}
        password: ${CLOUD_SQL_PASS}
    tools:
      execute_sql:
        kind: postgres-execute-sql
        source: cloudsql-pg-source
        description: Use this tool to execute SQL

      list_tables:
        kind: postgres-sql
        source: cloudsql-pg-source
        description: >
          Lists detailed table information (object type, columns, constraints, indexes, triggers, owner, comment)
          as JSON for user-created tables (ordinary or partitioned). Filters by a comma-separated list of names.
          If names are omitted, lists all tables in user schemas
        statement: |
          WITH desired_relkinds AS (
              SELECT ARRAY['r', 'p']::char[] AS kinds -- Always consider both 'TABLE' and 'PARTITIONED TABLE'
          ),
          table_info AS (
              SELECT
                  t.oid AS table_oid,
                  ns.nspname AS schema_name,
                  t.relname AS table_name,
                  pg_get_userbyid(t.relowner) AS table_owner,
                  obj_description(t.oid, 'pg_class') AS table_comment,
                  t.relkind AS object_kind
              FROM
                  pg_class t
              JOIN
                  pg_namespace ns ON ns.oid = t.relnamespace
              CROSS JOIN desired_relkinds dk
              WHERE
                  t.relkind = ANY(dk.kinds) -- Filter by selected table relkinds ('r', 'p')
                  AND (NULLIF(TRIM($1), '') IS NULL OR t.relname = ANY(string_to_array($1,','))) -- $1 is object_names
                  AND ns.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
                  AND ns.nspname NOT LIKE 'pg_temp_%' AND ns.nspname NOT LIKE 'pg_toast_temp_%'
          ),
          columns_info AS (
              SELECT
                  att.attrelid AS table_oid, att.attname AS column_name, format_type(att.atttypid, att.atttypmod) AS data_type,
                  att.attnum AS column_ordinal_position, att.attnotnull AS is_not_nullable,
                  pg_get_expr(ad.adbin, ad.adrelid) AS column_default, col_description(att.attrelid, att.attnum) AS column_comment
              FROM pg_attribute att LEFT JOIN pg_attrdef ad ON att.attrelid = ad.adrelid AND att.attnum = ad.adnum
              JOIN table_info ti ON att.attrelid = ti.table_oid WHERE att.attnum > 0 AND NOT att.attisdropped
          ),
          constraints_info AS (
              SELECT
                  con.conrelid AS table_oid, con.conname AS constraint_name, pg_get_constraintdef(con.oid) AS constraint_definition,
                  CASE con.contype WHEN 'p' THEN 'PRIMARY KEY' WHEN 'f' THEN 'FOREIGN KEY' WHEN 'u' THEN 'UNIQUE' WHEN 'c' THEN 'CHECK' ELSE con.contype::text END AS constraint_type,
                  (SELECT array_agg(att.attname ORDER BY u.attposition) FROM unnest(con.conkey) WITH ORDINALITY AS u(attnum, attposition) JOIN pg_attribute att ON att.attrelid = con.conrelid AND att.attnum = u.attnum) AS constraint_columns,
                  NULLIF(con.confrelid, 0)::regclass AS foreign_key_referenced_table,
                  (SELECT array_agg(att.attname ORDER BY u.attposition) FROM unnest(con.confkey) WITH ORDINALITY AS u(attnum, attposition) JOIN pg_attribute att ON att.attrelid = con.confrelid AND att.attnum = u.attnum WHERE con.contype = 'f') AS foreign_key_referenced_columns
              FROM pg_constraint con JOIN table_info ti ON con.conrelid = ti.table_oid
          ),
          indexes_info AS (
              SELECT
                  idx.indrelid AS table_oid, ic.relname AS index_name, pg_get_indexdef(idx.indexrelid) AS index_definition,
                  idx.indisunique AS is_unique, idx.indisprimary AS is_primary, am.amname AS index_method,
                  (SELECT array_agg(att.attname ORDER BY u.ord) FROM unnest(idx.indkey::int[]) WITH ORDINALITY AS u(colidx, ord) LEFT JOIN pg_attribute att ON att.attrelid = idx.indrelid AND att.attnum = u.colidx WHERE u.colidx <> 0) AS index_columns
              FROM pg_index idx JOIN pg_class ic ON ic.oid = idx.indexrelid JOIN pg_am am ON am.oid = ic.relam JOIN table_info ti ON idx.indrelid = ti.table_oid
          ),
          triggers_info AS (
              SELECT tg.tgrelid AS table_oid, tg.tgname AS trigger_name, pg_get_triggerdef(tg.oid) AS trigger_definition, tg.tgenabled AS trigger_enabled_state
              FROM pg_trigger tg JOIN table_info ti ON tg.tgrelid = ti.table_oid WHERE NOT tg.tgisinternal
          )
          SELECT
              ti.schema_name,
              ti.table_name AS object_name,
              json_build_object(
                  'schema_name', ti.schema_name,
                  'object_name', ti.table_name,
                  'object_type', CASE ti.object_kind
                                  WHEN 'r' THEN 'TABLE'
                                  WHEN 'p' THEN 'PARTITIONED TABLE'
                                  ELSE ti.object_kind::text -- Should not happen due to WHERE clause
                                END,
                  'owner', ti.table_owner,
                  'comment', ti.table_comment,
                  'columns', COALESCE((SELECT json_agg(json_build_object('column_name',ci.column_name,'data_type',ci.data_type,'ordinal_position',ci.column_ordinal_position,'is_not_nullable',ci.is_not_nullable,'column_default',ci.column_default,'column_comment',ci.column_comment) ORDER BY ci.column_ordinal_position) FROM columns_info ci WHERE ci.table_oid = ti.table_oid), '[]'::json),
                  'constraints', COALESCE((SELECT json_agg(json_build_object('constraint_name',cons.constraint_name,'constraint_type',cons.constraint_type,'constraint_definition',cons.constraint_definition,'constraint_columns',cons.constraint_columns,'foreign_key_referenced_table',cons.foreign_key_referenced_table,'foreign_key_referenced_columns',cons.foreign_key_referenced_columns)) FROM constraints_info cons WHERE cons.table_oid = ti.table_oid), '[]'::json),
                  'indexes', COALESCE((SELECT json_agg(json_build_object('index_name',ii.index_name,'index_definition',ii.index_definition,'is_unique',ii.is_unique,'is_primary',ii.is_primary,'index_method',ii.index_method,'index_columns',ii.index_columns)) FROM indexes_info ii WHERE ii.table_oid = ti.table_oid), '[]'::json),
                  'triggers', COALESCE((SELECT json_agg(json_build_object('trigger_name',tri.trigger_name,'trigger_definition',tri.trigger_definition,'trigger_enabled_state',tri.trigger_enabled_state)) FROM triggers_info tri WHERE tri.table_oid = ti.table_oid), '[]'::json)
              ) AS object_details
          FROM table_info ti ORDER BY ti.schema_name, ti.table_name;
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
        "cloud-sql-postgres": {
          "type": "sse",
          "url": "http://127.0.0.1:5000/mcp/sse"
        }
      }
    }
    ```

4. Restart Claude code to apply the new configuration.
{{< /tab >}}

{{% tab header="Claude desktop" lang="en" %}}

0. Install [`npx`](https://docs.npmjs.com/cli/v8/commands/npx).
1. Open [Claude desktop](https://claude.ai/download) and navigate to Settings.
2. Under the Developer tab, tap Edit Config to open the configuration file.
3. Add the following configuration and save:

    ```json
    {
      "mcpServers": {
        "cloud-sql-postgres": {
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

4. Restart Claude desktop.
5. From the new chat screen, you should see a hammer (MCP) icon appear with the new MCP server available.
{{< /tab >}}

{{% tab header="Cline" lang="en" %}}

1. Open the [Cline](https://github.com/cline/cline) extension in VS Code and tap the **MCP Servers** icon.
2. Tap Configure MCP Servers to open the configuration file.
3. Add the following configuration and save:

    ```json
    {
      "mcpServers": {
        "cloud-sql-postgres": {
          "type": "sse",
          "url": "http://127.0.0.1:5000/mcp/sse"
        }
      }
    }
    ```

4. You should see a green active status after the server is successfully connected.
{{< /tab >}}

{{% tab header="Cursor" lang="en" %}}

1. Create a `.cursor` directory in your project root if it doesn't exist.
2. Create a `.cursor/mcp.json` file if it doesn't exist and open it.
3. Add the following configuration:

    ```json
    {
      "mcpServers": {
        "cloud-sql-postgres": {
          "type": "sse",
          "url": "http://127.0.0.1:5000/mcp/sse"
        }
      }
    }
    ```

4. [Cursor](https://www.cursor.com/) and navigate to **Settings > Cursor Settings > MCP**. You should see a green active status after the server is successfully connected.
{{< /tab >}}

{{% tab header="Visual Studio Code (Copilot)" lang="en" %}}

1. Open [VS Code](https://code.visualstudio.com/docs/copilot/overview) and create a `.vscode` directory in your project root if it doesn't exist.
2. Create a `.vscode/mcp.json` file if it doesn't exist and open it.
3. Add the following configuration:

    ```json
    {
      "mcpServers": {
        "cloud-sql-postgres": {
          "type": "sse",
          "url": "http://127.0.0.1:5000/mcp/sse"
        }
      }
    }
    ```
{{< /tab >}}

{{% tab header="Windsurf" lang="en" %}}

1. Open [Windsurf](https://docs.codeium.com/windsurf) and navigate to the Cascade assistant.
2. Tap on the hammer (MCP) icon, then Configure to open the configuration file.
3. Add the following configuration:

    ```json
    {
      "mcpServers": {
        "cloud-sql-postgres": {
          "serverUrl": "http://127.0.0.1:5000/mcp/sse"
        }
      }
    }

    ```
{{< /tab >}}
{{< /tabpane >}}