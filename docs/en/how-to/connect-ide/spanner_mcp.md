---
title: "Spanner using MCP"
type: docs
weight: 2
description: >
  Connect your IDE to Spanner using Toolbox.
---

[Model Context Protocol (MCP)](https://modelcontextprotocol.io/introduction) is an open protocol for connecting Large Language Models (LLMs) to data sources like Spanner. This guide covers how to use [MCP Toolbox for Databases][toolbox] to expose your developer assistant tools to a Spanner instance:

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

1. [Enable the Spanner API in the Google Cloud project](https://console.cloud.google.com/flows/enableapi?apiid=spanner.googleapis.com&redirect=https://console.cloud.google.com).

1. [Create or select a Spanner instance and database](https://cloud.google.com/spanner/docs/create-query-database-console).

1. Configure the required roles and permissions to complete this task. You will need [Cloud Spanner Database User](https://cloud.google.com/spanner/docs/iam#roles) role (`roles/spanner.databaseUser`) or equivalent IAM permissions to connect to the instance.

1. Configured [Application Default Credentials (ADC)](https://cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) for your environment.

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
    # The ID of your Google Cloud Project where the Spanner instance is located.
    export SPANNER_PROJECT="your-gcp-project-id"

    # The name of your Spanner instance.
    export SPANNER_INSTANCE="your-instance-name"

    # The name of the database you want to connect to within the instance.
    export SPANNER_DATABASE="your-database-name"
    ```

2. Create a `tools.yaml` file.

3. Copy and paste the following contents into the `tools.yaml`:

    ```yaml
    sources:
      spanner-source:
        kind: spanner
        project: ${SPANNER_PROJECT}
        instance: ${SPANNER_INSTANCE}
        database: ${SPANNER_DATABASE}

    tools:
      execute_sql:
        kind: spanner-execute-sql
        source: spanner-source
        description: Use this tool to execute DML SQL

      execute_sql_dql:
        kind: spanner-execute-sql
        source: spanner-source
        description: Use this tool to execute DQL SQL
        readOnly: true

      list_tables:
        kind: spanner-sql
        source: spanner-source
        readOnly: true
        description: "Lists detailed schema information (object type, columns, constraints, indexes) as JSON for user-created tables (ordinary or partitioned). Filters by a comma-separated list of names. If names are omitted, lists all tables in user schemas."
        statement: |
          WITH FilterTableNames AS (
            SELECT DISTINCT TRIM(name) AS TABLE_NAME
            FROM UNNEST(IF(@table_names = '' OR @table_names IS NULL, ['%'], SPLIT(@table_names, ','))) AS name
          ),

          -- 1. Table Information
          table_info_cte AS (
            SELECT
              T.TABLE_SCHEMA,
              T.TABLE_NAME,
              T.TABLE_TYPE,
              T.PARENT_TABLE_NAME, -- For interleaved tables
              T.ON_DELETE_ACTION -- For interleaved tables
            FROM INFORMATION_SCHEMA.TABLES AS T
            WHERE
              T.TABLE_SCHEMA = ''
              AND T.TABLE_TYPE = 'BASE TABLE'
              AND (EXISTS (SELECT 1 FROM FilterTableNames WHERE FilterTableNames.TABLE_NAME = '%') OR T.TABLE_NAME IN (SELECT TABLE_NAME FROM FilterTableNames))
          ),

          -- 2. Column Information (with JSON string for each column)
          columns_info_cte AS (
            SELECT
              C.TABLE_SCHEMA,
              C.TABLE_NAME,
              ARRAY_AGG(
                CONCAT(
                  '{',
                  '"column_name":"', IFNULL(C.COLUMN_NAME, ''), '",',
                  '"data_type":"', IFNULL(C.SPANNER_TYPE, ''), '",',
                  '"ordinal_position":', CAST(C.ORDINAL_POSITION AS STRING), ',',
                  '"is_not_nullable":', IF(C.IS_NULLABLE = 'NO', 'true', 'false'), ',',
                  '"column_default":', IF(C.COLUMN_DEFAULT IS NULL, 'null', CONCAT('"', C.COLUMN_DEFAULT, '"')),
                  '}'
                ) ORDER BY C.ORDINAL_POSITION
              ) AS columns_json_array_elements
            FROM INFORMATION_SCHEMA.COLUMNS AS C
            WHERE EXISTS (SELECT 1 FROM table_info_cte TI WHERE C.TABLE_SCHEMA = TI.TABLE_SCHEMA AND C.TABLE_NAME = TI.TABLE_NAME)
            GROUP BY C.TABLE_SCHEMA, C.TABLE_NAME
          ),

          -- Helper CTE for aggregating constraint columns
          constraint_columns_agg_cte AS (
            SELECT
              CONSTRAINT_CATALOG,
              CONSTRAINT_SCHEMA,
              CONSTRAINT_NAME,
              ARRAY_AGG(CONCAT('"', COLUMN_NAME, '"') ORDER BY ORDINAL_POSITION) AS column_names_json_list
            FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
            GROUP BY CONSTRAINT_CATALOG, CONSTRAINT_SCHEMA, CONSTRAINT_NAME
          ),

          -- 3. Constraint Information (with JSON string for each constraint)
          constraints_info_cte AS (
            SELECT
              TC.TABLE_SCHEMA,
              TC.TABLE_NAME,
              ARRAY_AGG(
                CONCAT(
                  '{',
                  '"constraint_name":"', IFNULL(TC.CONSTRAINT_NAME, ''), '",',
                  '"constraint_type":"', IFNULL(TC.CONSTRAINT_TYPE, ''), '",',
                  '"constraint_definition":',
                    CASE TC.CONSTRAINT_TYPE
                      WHEN 'CHECK' THEN IF(CC.CHECK_CLAUSE IS NULL, 'null', CONCAT('"', CC.CHECK_CLAUSE, '"'))
                      WHEN 'PRIMARY KEY' THEN CONCAT('"', 'PRIMARY KEY (', ARRAY_TO_STRING(COALESCE(KeyCols.column_names_json_list, []), ', '), ')', '"')
                      WHEN 'UNIQUE' THEN CONCAT('"', 'UNIQUE (', ARRAY_TO_STRING(COALESCE(KeyCols.column_names_json_list, []), ', '), ')', '"')
                      WHEN 'FOREIGN KEY' THEN CONCAT('"', 'FOREIGN KEY (', ARRAY_TO_STRING(COALESCE(KeyCols.column_names_json_list, []), ', '), ') REFERENCES ',
                                              IFNULL(RefKeyTable.TABLE_NAME, ''),
                                              ' (', ARRAY_TO_STRING(COALESCE(RefKeyCols.column_names_json_list, []), ', '), ')', '"')
                      ELSE 'null'
                    END, ',',
                  '"constraint_columns":[', ARRAY_TO_STRING(COALESCE(KeyCols.column_names_json_list, []), ','), '],',
                  '"foreign_key_referenced_table":', IF(RefKeyTable.TABLE_NAME IS NULL, 'null', CONCAT('"', RefKeyTable.TABLE_NAME, '"')), ',',
                  '"foreign_key_referenced_columns":[', ARRAY_TO_STRING(COALESCE(RefKeyCols.column_names_json_list, []), ','), ']',
                  '}'
                ) ORDER BY TC.CONSTRAINT_NAME
              ) AS constraints_json_array_elements
            FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS AS TC
            LEFT JOIN INFORMATION_SCHEMA.CHECK_CONSTRAINTS AS CC
              ON TC.CONSTRAINT_CATALOG = CC.CONSTRAINT_CATALOG AND TC.CONSTRAINT_SCHEMA = CC.CONSTRAINT_SCHEMA AND TC.CONSTRAINT_NAME = CC.CONSTRAINT_NAME
            LEFT JOIN INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS AS RC
              ON TC.CONSTRAINT_CATALOG = RC.CONSTRAINT_CATALOG AND TC.CONSTRAINT_SCHEMA = RC.CONSTRAINT_SCHEMA AND TC.CONSTRAINT_NAME = RC.CONSTRAINT_NAME
            LEFT JOIN INFORMATION_SCHEMA.TABLE_CONSTRAINTS AS RefConstraint
              ON RC.UNIQUE_CONSTRAINT_CATALOG = RefConstraint.CONSTRAINT_CATALOG AND RC.UNIQUE_CONSTRAINT_SCHEMA = RefConstraint.CONSTRAINT_SCHEMA AND RC.UNIQUE_CONSTRAINT_NAME = RefConstraint.CONSTRAINT_NAME
            LEFT JOIN INFORMATION_SCHEMA.TABLES AS RefKeyTable
              ON RefConstraint.TABLE_CATALOG = RefKeyTable.TABLE_CATALOG AND RefConstraint.TABLE_SCHEMA = RefKeyTable.TABLE_SCHEMA AND RefConstraint.TABLE_NAME = RefKeyTable.TABLE_NAME
            LEFT JOIN constraint_columns_agg_cte AS KeyCols
              ON TC.CONSTRAINT_CATALOG = KeyCols.CONSTRAINT_CATALOG AND TC.CONSTRAINT_SCHEMA = KeyCols.CONSTRAINT_SCHEMA AND TC.CONSTRAINT_NAME = KeyCols.CONSTRAINT_NAME
            LEFT JOIN constraint_columns_agg_cte AS RefKeyCols
              ON RC.UNIQUE_CONSTRAINT_CATALOG = RefKeyCols.CONSTRAINT_CATALOG AND RC.UNIQUE_CONSTRAINT_SCHEMA = RefKeyCols.CONSTRAINT_SCHEMA AND RC.UNIQUE_CONSTRAINT_NAME = RefKeyCols.CONSTRAINT_NAME AND TC.CONSTRAINT_TYPE = 'FOREIGN KEY'
            WHERE EXISTS (SELECT 1 FROM table_info_cte TI WHERE TC.TABLE_SCHEMA = TI.TABLE_SCHEMA AND TC.TABLE_NAME = TI.TABLE_NAME)
            GROUP BY TC.TABLE_SCHEMA, TC.TABLE_NAME
          ),

          -- Helper CTE for aggregating index key columns (as JSON strings)
          index_key_columns_agg_cte AS (
            SELECT
              TABLE_CATALOG,
              TABLE_SCHEMA,
              TABLE_NAME,
              INDEX_NAME,
              ARRAY_AGG(
                CONCAT(
                  '{"column_name":"', IFNULL(COLUMN_NAME, ''), '",',
                  '"ordering":"', IFNULL(COLUMN_ORDERING, ''), '"}'
                ) ORDER BY ORDINAL_POSITION
              ) AS key_column_json_details
            FROM INFORMATION_SCHEMA.INDEX_COLUMNS
            WHERE ORDINAL_POSITION IS NOT NULL -- Key columns
            GROUP BY TABLE_CATALOG, TABLE_SCHEMA, TABLE_NAME, INDEX_NAME
          ),

          -- Helper CTE for aggregating index storing columns (as JSON strings)
          index_storing_columns_agg_cte AS (
            SELECT
              TABLE_CATALOG,
              TABLE_SCHEMA,
              TABLE_NAME,
              INDEX_NAME,
              ARRAY_AGG(CONCAT('"', COLUMN_NAME, '"') ORDER BY COLUMN_NAME) AS storing_column_json_names
            FROM INFORMATION_SCHEMA.INDEX_COLUMNS
            WHERE ORDINAL_POSITION IS NULL -- Storing columns
            GROUP BY TABLE_CATALOG, TABLE_SCHEMA, TABLE_NAME, INDEX_NAME
          ),

          -- 4. Index Information (with JSON string for each index)
          indexes_info_cte AS (
            SELECT
              I.TABLE_SCHEMA,
              I.TABLE_NAME,
              ARRAY_AGG(
                CONCAT(
                  '{',
                  '"index_name":"', IFNULL(I.INDEX_NAME, ''), '",',
                  '"index_type":"', IFNULL(I.INDEX_TYPE, ''), '",',
                  '"is_unique":', IF(I.IS_UNIQUE, 'true', 'false'), ',',
                  '"is_null_filtered":', IF(I.IS_NULL_FILTERED, 'true', 'false'), ',',
                  '"interleaved_in_table":', IF(I.PARENT_TABLE_NAME IS NULL, 'null', CONCAT('"', I.PARENT_TABLE_NAME, '"')), ',',
                  '"index_key_columns":[', ARRAY_TO_STRING(COALESCE(KeyIndexCols.key_column_json_details, []), ','), '],',
                  '"storing_columns":[', ARRAY_TO_STRING(COALESCE(StoringIndexCols.storing_column_json_names, []), ','), ']',
                  '}'
                ) ORDER BY I.INDEX_NAME
              ) AS indexes_json_array_elements
            FROM INFORMATION_SCHEMA.INDEXES AS I
            LEFT JOIN index_key_columns_agg_cte AS KeyIndexCols
              ON I.TABLE_CATALOG = KeyIndexCols.TABLE_CATALOG AND I.TABLE_SCHEMA = KeyIndexCols.TABLE_SCHEMA AND I.TABLE_NAME = KeyIndexCols.TABLE_NAME AND I.INDEX_NAME = KeyIndexCols.INDEX_NAME
            LEFT JOIN index_storing_columns_agg_cte AS StoringIndexCols
              ON I.TABLE_CATALOG = StoringIndexCols.TABLE_CATALOG AND I.TABLE_SCHEMA = StoringIndexCols.TABLE_SCHEMA AND I.TABLE_NAME = StoringIndexCols.TABLE_NAME AND I.INDEX_NAME = StoringIndexCols.INDEX_NAME AND I.INDEX_TYPE = 'INDEX'
            WHERE EXISTS (SELECT 1 FROM table_info_cte TI WHERE I.TABLE_SCHEMA = TI.TABLE_SCHEMA AND I.TABLE_NAME = TI.TABLE_NAME)
            GROUP BY I.TABLE_SCHEMA, I.TABLE_NAME
          )

          -- Final SELECT to build the JSON output
          SELECT
            TI.TABLE_SCHEMA AS schema_name,
            TI.TABLE_NAME AS object_name,
            CONCAT(
              '{',
              '"schema_name":"', IFNULL(TI.TABLE_SCHEMA, ''), '",',
              '"object_name":"', IFNULL(TI.TABLE_NAME, ''), '",',
              '"object_type":"', IFNULL(TI.TABLE_TYPE, ''), '",',
              '"columns":[', ARRAY_TO_STRING(COALESCE(CI.columns_json_array_elements, []), ','), '],',
              '"constraints":[', ARRAY_TO_STRING(COALESCE(CONSI.constraints_json_array_elements, []), ','), '],',
              '"indexes":[', ARRAY_TO_STRING(COALESCE(II.indexes_json_array_elements, []), ','), '],',
              '}'
            ) AS object_details
          FROM table_info_cte AS TI
          LEFT JOIN columns_info_cte AS CI
            ON TI.TABLE_SCHEMA = CI.TABLE_SCHEMA AND TI.TABLE_NAME = CI.TABLE_NAME
          LEFT JOIN constraints_info_cte AS CONSI
            ON TI.TABLE_SCHEMA = CONSI.TABLE_SCHEMA AND TI.TABLE_NAME = CONSI.TABLE_NAME
          LEFT JOIN indexes_info_cte AS II
            ON TI.TABLE_SCHEMA = II.TABLE_SCHEMA AND TI.TABLE_NAME = II.TABLE_NAME
          ORDER BY TI.TABLE_SCHEMA, TI.TABLE_NAME;

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
        "spanner": {
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
        "spanner": {
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
        "spanner": {
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
        "spanner": {
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
        "spanner": {
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
        "spanner": {
          "serverUrl": "http://127.0.0.1:5000/mcp/sse"
        }
      }
    }

    ```
{{% /tab %}}
{{< /tabpane >}}
