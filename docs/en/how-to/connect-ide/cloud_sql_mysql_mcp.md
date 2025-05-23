---
title: "Cloud SQL for MySQL using MCP"
type: docs
weight: 2
description: >
  Connect your IDE to Cloud SQl for MySQL using Toolbox.
---

[Model Context Protocol (MCP)](https://modelcontextprotocol.io/introduction) is an open protocol for connecting Large Language Models (LLMs) to data sources like Cloud SQL. This guide covers how to use [MCP Toolbox for Databases][toolbox] to expose your developer assistant tools to a Cloud SQL for MySQL instance:

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

1. [Create a Cloud SQL for MySQL instance](https://cloud.google.com/sql/docs/mysql/create-instance). These instructions assume that your Cloud SQL instance has a [public IP address](https://cloud.google.com/sql/docs/mysql/configure-ip). By default, Cloud SQL assigns a public IP address to a new instance. Toolbox will connect securely using the [Cloud SQL connectors](https://cloud.google.com/sql/docs/mysql/language-connectors).

1. Configure the required roles and permissions to complete this task. You will need [Cloud SQL > Client](https://cloud.google.com/sql/docs/mysql/roles-and-permissions#proxy-roles-permissions) role (`roles/cloudsql.client`) or equivalent IAM permissions to connect to the instance.

1. Configured [Application Default Credentials (ADC)](https://cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) for your environment.

1. Create or reuse [a database user](https://cloud.google.com/sql/docs/mysql/create-manage-users) and have the username and password ready.


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
    export CLOUD_SQL_MYSQL_PROJECT="your-gcp-project-id"

    # The region where your Cloud SQL instance is located (e.g., us-central1).
    export CLOUD_SQL_MYSQL_REGION="your-instance-region"

    # The name of your Cloud SQL instance.
    export CLOUD_SQL_MYSQL_INSTANCE="your-instance-name"

    # The name of the database you want to connect to within the instance.
    export CLOUD_SQL_MYSQL_DATABASE="your-database-name"

    # The username for connecting to the database.
    export CLOUD_SQL_MYSQL_USER="your-database-user"

    # The password for the specified database user.
    export CLOUD_SQL_MYSQL_PASSWORD="your-database-password"
    ```

2. Create a `tools.yaml` file.

3. Copy and paste the following contents into the `tools.yaml`:

    ```yaml
    sources:
      cloud-sql-mysql-source:
            kind: cloud-sql-mysql
            project: ${CLOUD_SQL_MYSQL_PROJECT}
            region: ${CLOUD_SQL_MYSQL_REGION}
            instance: ${CLOUD_SQL_MYSQL_INSTANCE}
            database: ${CLOUD_SQL_MYSQL_DATABASE}
            user: ${CLOUD_SQL_MYSQL_USER}
            password: ${CLOUD_SQL_MYSQL_PASSWORD}

    tools:
      execute_sql:
        kind: mysql-execute-sql
        source: cloud-sql-mysql-source
        description: Use this tool to execute SQL.

      list_tables:
        kind: mysql-sql
        source: cloud-sql-mysql-source
        description: "Lists detailed schema information (object type, columns, constraints, indexes, triggers, comment) as JSON for user-created tables (ordinary or partitioned). Filters by a comma-separated list of names. If names are omitted, lists all tables in user schemas."
        statement: |
          SELECT
              T.TABLE_SCHEMA AS schema_name,
              T.TABLE_NAME AS object_name,
              CONVERT( JSON_OBJECT(
                  'schema_name', T.TABLE_SCHEMA,
                  'object_name', T.TABLE_NAME,
                  'object_type', 'TABLE',
                  'owner', (
                      SELECT
                          IFNULL(U.GRANTEE, 'N/A')
                      FROM
                          INFORMATION_SCHEMA.SCHEMA_PRIVILEGES U
                      WHERE
                          U.TABLE_SCHEMA = T.TABLE_SCHEMA
                      LIMIT 1
                  ),
                  'comment', IFNULL(T.TABLE_COMMENT, ''),
                  'columns', (
                      SELECT
                          IFNULL(
                              JSON_ARRAYAGG(
                                  JSON_OBJECT(
                                      'column_name', C.COLUMN_NAME,
                                      'data_type', C.COLUMN_TYPE,
                                      'ordinal_position', C.ORDINAL_POSITION,
                                      'is_not_nullable', IF(C.IS_NULLABLE = 'NO', TRUE, FALSE),
                                      'column_default', C.COLUMN_DEFAULT,
                                      'column_comment', IFNULL(C.COLUMN_COMMENT, '')
                                  )
                              ),
                              JSON_ARRAY()
                          )
                      FROM
                          INFORMATION_SCHEMA.COLUMNS C
                      WHERE
                          C.TABLE_SCHEMA = T.TABLE_SCHEMA AND C.TABLE_NAME = T.TABLE_NAME
                      ORDER BY C.ORDINAL_POSITION
                  ),
                  'constraints', (
                      SELECT
                          IFNULL(
                              JSON_ARRAYAGG(
                                  JSON_OBJECT(
                                      'constraint_name', TC.CONSTRAINT_NAME,
                                      'constraint_type',
                                          CASE TC.CONSTRAINT_TYPE
                                              WHEN 'PRIMARY KEY' THEN 'PRIMARY KEY'
                                              WHEN 'FOREIGN KEY' THEN 'FOREIGN KEY'
                                              WHEN 'UNIQUE' THEN 'UNIQUE'
                                              ELSE TC.CONSTRAINT_TYPE
                                          END,
                                      'constraint_definition', '',
                                      'constraint_columns', (
                                          SELECT
                                              IFNULL(JSON_ARRAYAGG(KCU.COLUMN_NAME), JSON_ARRAY())
                                          FROM
                                              INFORMATION_SCHEMA.KEY_COLUMN_USAGE KCU
                                          WHERE
                                              KCU.CONSTRAINT_SCHEMA = TC.CONSTRAINT_SCHEMA
                                              AND KCU.CONSTRAINT_NAME = TC.CONSTRAINT_NAME
                                              AND KCU.TABLE_NAME = TC.TABLE_NAME
                                          ORDER BY KCU.ORDINAL_POSITION
                                      ),
                                      'foreign_key_referenced_table', IF(TC.CONSTRAINT_TYPE = 'FOREIGN KEY', RC.REFERENCED_TABLE_NAME, NULL),
                                      'foreign_key_referenced_columns', IF(TC.CONSTRAINT_TYPE = 'FOREIGN KEY',
                                          (SELECT IFNULL(JSON_ARRAYAGG(FKCU.REFERENCED_COLUMN_NAME), JSON_ARRAY())
                                          FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE FKCU
                                          WHERE FKCU.CONSTRAINT_SCHEMA = TC.CONSTRAINT_SCHEMA
                                            AND FKCU.CONSTRAINT_NAME = TC.CONSTRAINT_NAME
                                            AND FKCU.TABLE_NAME = TC.TABLE_NAME
                                            AND FKCU.REFERENCED_TABLE_NAME IS NOT NULL
                                          ORDER BY FKCU.ORDINAL_POSITION),
                                          NULL
                                      )
                                  )
                              ),
                              JSON_ARRAY()
                          )
                      FROM
                          INFORMATION_SCHEMA.TABLE_CONSTRAINTS TC
                      LEFT JOIN
                          INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS RC
                          ON TC.CONSTRAINT_SCHEMA = RC.CONSTRAINT_SCHEMA
                          AND TC.CONSTRAINT_NAME = RC.CONSTRAINT_NAME
                          AND TC.TABLE_NAME = RC.TABLE_NAME
                      WHERE
                          TC.TABLE_SCHEMA = T.TABLE_SCHEMA AND TC.TABLE_NAME = T.TABLE_NAME
                  ),
                  'indexes', (
                      SELECT
                          IFNULL(
                              JSON_ARRAYAGG(
                                  JSON_OBJECT(
                                      'index_name', IndexData.INDEX_NAME,
                                      'is_unique', IF(IndexData.NON_UNIQUE = 0, TRUE, FALSE),
                                      'is_primary', IF(IndexData.INDEX_NAME = 'PRIMARY', TRUE, FALSE),
                                      'index_columns', IFNULL(IndexData.INDEX_COLUMNS_ARRAY, JSON_ARRAY())
                                  )
                              ),
                              JSON_ARRAY()
                          )
                      FROM (
                          SELECT
                              S.TABLE_SCHEMA,
                              S.TABLE_NAME,
                              S.INDEX_NAME,
                              MIN(S.NON_UNIQUE) AS NON_UNIQUE, -- Aggregate NON_UNIQUE here to get unique status for the index
                              JSON_ARRAYAGG(S.COLUMN_NAME) AS INDEX_COLUMNS_ARRAY -- Aggregate columns into an array for this index
                          FROM
                              INFORMATION_SCHEMA.STATISTICS S
                          WHERE
                              S.TABLE_SCHEMA = T.TABLE_SCHEMA AND S.TABLE_NAME = T.TABLE_NAME
                          GROUP BY
                              S.TABLE_SCHEMA, S.TABLE_NAME, S.INDEX_NAME
                      ) AS IndexData
                      ORDER BY IndexData.INDEX_NAME
                  ),
                  'triggers', (
                      SELECT
                          IFNULL(
                              JSON_ARRAYAGG(
                                  JSON_OBJECT(
                                      'trigger_name', TR.TRIGGER_NAME,
                                      'trigger_definition', TR.ACTION_STATEMENT
                                  )
                              ),
                              JSON_ARRAY()
                          )
                      FROM
                          INFORMATION_SCHEMA.TRIGGERS TR
                      WHERE
                          TR.EVENT_OBJECT_SCHEMA = T.TABLE_SCHEMA AND TR.EVENT_OBJECT_TABLE = T.TABLE_NAME
                      ORDER BY TR.TRIGGER_NAME
                  )
              ) USING utf8mb4) AS object_details
          FROM
              INFORMATION_SCHEMA.TABLES T
          WHERE
              T.TABLE_SCHEMA NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')
              AND (NULLIF(TRIM(?), '') IS NULL OR FIND_IN_SET(T.TABLE_NAME, ?))
              AND T.TABLE_TYPE = 'BASE TABLE'
          ORDER BY
              T.TABLE_SCHEMA, T.TABLE_NAME;
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
        "cloud-sql-mysql": {
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
        "cloud-sql-mysql": {
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
        "cloud-sql-mysql": {
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
        "cloud-sql-mysql": {
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
        "cloud-sql-mysql": {
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
        "cloud-sql-mysql": {
          "serverUrl": "http://127.0.0.1:5000/mcp/sse"
        }
      }
    }

    ```
{{% /tab %}}
{{< /tabpane >}}
