---
title: "Snowflake"
type: docs
weight: 2
description: >
  How to get started running Toolbox with MCP Inspector and Snowflake as the source.
---

## Overview

[Model Context Protocol](https://modelcontextprotocol.io) is an open protocol
that standardizes how applications provide context to LLMs. Check out this page
on how to [connect to Toolbox via MCP](../../how-to/connect_via_mcp.md).

## Before you begin

This guide assumes you have already done the following:

1.  [Create a Snowflake account](https://signup.snowflake.com/).
1.  Connect to the instance using [SnowSQL](https://docs.snowflake.com/en/user-guide/snowsql), or any other Snowflake client.

## Step 1: Set up your environment

Copy the environment template and update it with your Snowflake credentials:

```bash
cp examples/snowflake-env.sh my-snowflake-env.sh
```

Edit `my-snowflake-env.sh` with your actual Snowflake connection details:

```bash
export SNOWFLAKE_ACCOUNT="your-account-identifier"
export SNOWFLAKE_USER="your-username"
export SNOWFLAKE_PASSWORD="your-password"
export SNOWFLAKE_DATABASE="your-database"
export SNOWFLAKE_SCHEMA="your-schema"
export SNOWFLAKE_WAREHOUSE="COMPUTE_WH"
export SNOWFLAKE_ROLE="ACCOUNTADMIN"
```

## Step 2: Install Toolbox

In this section, we will download and install the Toolbox binary.

1. Download the latest version of Toolbox as a binary:

    {{< notice tip >}}
   Select the
   [correct binary](https://github.com/googleapis/genai-toolbox/releases)
   corresponding to your OS and CPU architecture.
    {{< /notice >}}
    <!-- {x-release-please-start-version} -->
    ```bash
    export OS="linux/amd64" # one of linux/amd64, darwin/arm64, darwin/amd64, or windows/amd64
    export VERSION="0.10.0"
    curl -O https://storage.googleapis.com/genai-toolbox/v$VERSION/$OS/toolbox
    ```
    <!-- {x-release-please-end} -->

1. Make the binary executable:

    ```bash
    chmod +x toolbox
    ```

## Step 3: Configure the tools

You have two options:

#### Option A: Use the prebuilt configuration

```bash
./toolbox --prebuilt snowflake
```

#### Option B: Use the custom configuration

Create a `tools.yaml` file and add the following content. You must replace the placeholders with your actual Snowflake configuration.

```yaml
sources:
  snowflake-source:
    kind: snowflake
    account: ${SNOWFLAKE_ACCOUNT}
    user: ${SNOWFLAKE_USER}
    password: ${SNOWFLAKE_PASSWORD}
    database: ${SNOWFLAKE_DATABASE}
    schema: ${SNOWFLAKE_SCHEMA}
    warehouse: ${SNOWFLAKE_WAREHOUSE}
    role: ${SNOWFLAKE_ROLE}

tools:
    execute_sql:
        kind: snowflake-execute-sql
        source: snowflake-source
        description: Use this tool to execute SQL.

    list_tables:
        kind: snowflake-sql
        source: snowflake-source
        description: "Lists detailed schema information for user-created tables."
        statement: |
           SELECT table_name, table_type 
           FROM information_schema.tables 
           WHERE table_schema = current_schema()
           ORDER BY table_name;
```

For more info on tools, check out the
[Tools](../../resources/tools/) section.

## Step 4: Run the Toolbox server

Run the Toolbox server, pointing to the `tools.yaml` file created earlier:

```bash
./toolbox --tools-file "tools.yaml"
```

## Step 5: Connect to MCP Inspector

1. Run the MCP Inspector:

    ```bash
    npx @modelcontextprotocol/inspector
    ```

1. Type `y` when it asks to install the inspector package.

1. It should show the following when the MCP Inspector is up and running (please take note of `<YOUR_SESSION_TOKEN>`):

    ```bash
    Starting MCP inspector...
    ⚙️ Proxy server listening on localhost:6277
    🔑 Session token: <YOUR_SESSION_TOKEN>
       Use this token to authenticate requests or set DANGEROUSLY_OMIT_AUTH=true to disable auth

    🚀 MCP Inspector is up and running at:
       http://localhost:6274/?MCP_PROXY_AUTH_TOKEN=<YOUR_SESSION_TOKEN>
    ```

1. Open the above link in your browser.

1. For `Transport Type`, select `Streamable HTTP`.

1. For `URL`, type in `http://127.0.0.1:5000/mcp`.

1. For `Configuration` -> `Proxy Session Token`, make sure `<YOUR_SESSION_TOKEN>` is present.

1. Click Connect.

1. Select `List Tools`, you will see a list of tools configured in `tools.yaml`.

1. Test out your tools here!

## What's next

- Learn more about [MCP Inspector](../../how-to/connect_via_mcp.md).
- Learn more about [Toolbox Resources](../../resources/).
- Learn more about [Toolbox How-to guides](../../how-to/).
