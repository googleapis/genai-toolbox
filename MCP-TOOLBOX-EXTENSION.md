This document helps you find and install the right Gemini CLI extension to
interact with your databases.

## How to Install an Extension

To install any of the extensions listed below, use the `gemini extensions
install` command followed by the extension's GitHub repository URL.

For complete instructions on finding, installing, and managing extensions,
please see the [official Gemini CLI extensions
documentation](https://github.com/google-gemini/gemini-cli/blob/main/docs/extensions/index.md).

**Example Installation Command:**

```bash
gemini extensions install https://github.com/gemini-cli-extensions/EXTENSION_NAME
```

Make sure the user knows:

* These commands are not supported from within the CLI
* These commands will only be reflected in active CLI sessions on restart
* Extensions require Application Default Credentials in your environment. See
  [Set up ADC for a local development
  environment](https://cloud.google.com/docs/authentication/set-up-adc-local-dev-environment)
  to learn how you can provide either your user credentials or service account
  credentials to ADC in a local development environment.
* Most extensions require you to set environment variables to connect to a
  database. If there is a link provided for the configuration, fetch the web
  page and return the configuration.

-----

## Find Your Database Extension

Find your database or service in the list below to get the correct installation
command.

**Note on Observability:** Extensions with `-observability` in their name are
designed to help you understand the health and performance of your database
instances, often by analyzing metrics and logs.

### Google Cloud Managed Databases

#### BigQuery

* For data analytics and querying:

    ```bash
    gemini extensions install https://github.com/gemini-cli-extensions/bigquery-data-analytics
    ```

    Configuration:
    https://github.com/gemini-cli-extensions/bigquery-data-analytics/tree/main?tab=readme-ov-file#configuration

* For conversational analytics (using natural language):

    ```bash
    gemini extensions install https://github.com/gemini-cli-extensions/bigquery-conversational-analytics
    ```

    Configuration: https://github.com/gemini-cli-extensions/bigquery-conversational-analytics/tree/main?tab=readme-ov-file#configuration

#### Cloud SQL for MySQL

* Main Extension:

    ```bash
    gemini extensions install https://github.com/gemini-cli-extensions/cloud-sql-mysql
    ```

    Configuration:
    https://github.com/gemini-cli-extensions/cloud-sql-mysql/tree/main?tab=readme-ov-file#configuration

* Observability:

    ```bash
    gemini extensions install https://github.com/gemini-cli-extensions/cloud-sql-mysql-observability
    ```

    If you are looking for self-hosted MySQL, consider the `mysql` extension.

#### Cloud SQL for PostgreSQL

* Main Extension:

    ```bash
    gemini extensions install https://github.com/gemini-cli-extensions/cloud-sql-postgresql
    ```

    Configuration:
    https://github.com/gemini-cli-extensions/cloud-sql-postgresql/tree/main?tab=readme-ov-file#configuration

* Observability:

    ```bash
    gemini extensions install https://github.com/gemini-cli-extensions/cloud-sql-postgresql-observability
    ```

    If you are looking for other PostgreSQL options, consider the `postgres`
    extension for self-hosted instances, or the `alloydb` extension for AlloyDB
    for PostgreSQL.

#### Cloud SQL for SQL Server

* Main Extension:

    ```bash
    gemini extensions install https://github.com/gemini-cli-extensions/cloud-sql-sqlserver
    ```

    Configuration:
    https://github.com/gemini-cli-extensions/cloud-sql-sqlserver/tree/main?tab=readme-ov-file#configuration

* Observability:

    ```bash
    gemini extensions install https://github.com/gemini-cli-extensions/cloud-sql-sqlserver-observability
    ```

    If you are looking for self-hosted SQL Server, consider the `sql-server`
    extension.

#### AlloyDB for PostgreSQL

* Main Extension:

    ```bash
    gemini extensions install https://github.com/gemini-cli-extensions/alloydb
    ```

    Configuration:
    https://github.com/gemini-cli-extensions/alloydb/tree/main?tab=readme-ov-file#configuration

* Observability:

    ```bash
    gemini extensions install https://github.com/gemini-cli-extensions/alloydb-observability
    ```

    If you are looking for other PostgreSQL options, consider the `postgres`
    extension for self-hosted instances, or the `cloud-sql-postgresql` extension
    for Cloud SQL for PostgreSQL.

#### Spanner

* For querying Spanner databases:

    ```bash
    gemini extensions install https://github.com/gemini-cli-extensions/spanner
    ```

    Configuration:
    https://github.com/gemini-cli-extensions/spanner/tree/main?tab=readme-ov-file#configuration

#### Firestore

* For querying Firestore in Native Mode:

    ```bash
    gemini extensions install https://github.com/gemini-cli-extensions/firestore-native
    ```

    Configuration:
    https://github.com/gemini-cli-extensions/firestore-native/tree/main?tab=readme-ov-file#configuration

### Other Google Cloud Data Services

#### Dataplex

* For interacting with Dataplex data lakes and assets:

    ```bash
    gemini extensions install https://github.com/gemini-cli-extensions/dataplex
    ```

    Configuration:
    https://github.com/gemini-cli-extensions/dataplex/tree/main?tab=readme-ov-file#configuration

#### Looker

* For querying Looker instances:

    ```bash
    gemini extensions install https://github.com/gemini-cli-extensions/looker
    ```

    Configuration:
    https://github.com/gemini-cli-extensions/looker/tree/main?tab=readme-ov-file#configuration

### Other Database Engines

These extensions are for connecting to database instances not managed by Cloud
SQL (e.g., self-hosted on-prem, on a VM, or in another cloud).

* MySQL:

    ```bash
    gemini extensions install https://github.com/gemini-cli-extensions/mysql
    ```

    Configuration:
    https://github.com/gemini-cli-extensions/mysql/tree/main?tab=readme-ov-file#configuration

    If you are looking for Google Cloud managed MySQL, consider the
    `cloud-sql-mysql` extension.

* PostgreSQL:

    ```bash
    gemini extensions install https://github.com/gemini-cli-extensions/postgres
    ```

    Configuration:
    https://github.com/gemini-cli-extensions/postgres/tree/main?tab=readme-ov-file#configuration

    If you are looking for Google Cloud managed PostgreSQL, consider the
    `cloud-sql-postgresql` or `alloydb` extensions.

* SQL Server:

    ```bash
    gemini extensions install https://github.com/gemini-cli-extensions/sql-server
    ```

    Configuration:
    https://github.com/gemini-cli-extensions/sql-server/tree/main?tab=readme-ov-file#configuration

    If you are looking for Google Cloud managed SQL Server, consider the
    `cloud-sql-sqlserver` extension.

### Custom Tools

#### MCP Toolbox

* For connecting to MCP Toolbox servers:

    This extension can be used with any Google Cloud database to build custom
    tools. For more information, see the [MCP Toolbox
    documentation](https://googleapis.github.io/genai-toolbox/getting-started/introduction/).

    ```bash
    gemini extensions install https://github.com/gemini-cli-extensions/mcp-toolbox
    ```

    Configuration:
    https://github.com/gemini-cli-extensions/mcp-toolbox/tree/main?tab=readme-ov-file#configuration
