---
title: "Prebuilt Tools"
type: docs
weight: 1
description: >
    This page lists all the prebuilt tools available.
---

Prebuilt tools are reusable, pre-packaged toolsets that are designed to extend the capabilities of agents. These tools are built to be generic and adaptable, allowing developers to interact with and take action on databases.

See guides, [Connect from your IDE](../how-to/connect-ide/_index.md), for details on how to connect your AI tools (IDEs) to databases via Toolbox and MCP.

## AlloyDB Postgres

*   `--prebuilt` value: `alloydb-postgres`
*   **Environment Variables:**
    *   `ALLOYDB_POSTGRES_PROJECT`: The GCP project ID.
    *   `ALLOYDB_POSTGRES_REGION`: The region of your AlloyDB instance.
    *   `ALLOYDB_POSTGRES_CLUSTER`: The ID of your AlloyDB cluster.
    *   `ALLOYDB_POSTGRES_INSTANCE`: The ID of your AlloyDB instance.
    *   `ALLOYDB_POSTGRES_DATABASE`: The name of the database to connect to.
    *   `ALLOYDB_POSTGRES_USER`: (Optional) The database username. Defaults to IAM authentication if unspecified.
    *   `ALLOYDB_POSTGRES_PASSWORD`: (Optional) The password for the database user. Defaults to IAM authentication if unspecified.
    *   `ALLOYDB_POSTGRES_IP_TYPE`: (Optional) The IP type i.e. "Public" or "Private" (Default: Public).
*   **Permissions:**
    *   **AlloyDB Client** (`roles/alloydb.client`) to connect to the instance.
    *   Database-level permissions (e.g., `SELECT`, `INSERT`) are required to execute queries.
*   **Tools:**
    *   `execute_sql`: Executes a SQL query.
    *   `list_tables`: Lists tables in the database.

## AlloyDB Postgres Admin

* `--prebuilt` value: `alloydb-postgres-admin`
*   **Environment Variables:**
    *   `API_KEY`: Your API key for the AlloyDB API.
*   **Permissions:**
    *   **AlloyDB Admin** (`roles/alloydb.admin`) IAM role is required on the project.
*   **Tools:**
    *   `alloydb-create-cluster`: Creates a new AlloyDB cluster.
    *   `alloydb-operations-get`: Polls the operations API to track the status of long-running operations.
    *   `alloydb-create-instance`: Creates a new AlloyDB instance within a cluster.
    *   `alloydb-list-clusters`: Lists all AlloyDB clusters in a project.
    *   `alloydb-list-instances`: Lists all instances within an AlloyDB cluster.
    *   `alloydb-list-users`: Lists all database users within an AlloyDB cluster.
    *   `alloydb-create-user`: Creates a new database user in an AlloyDB cluster.

## BigQuery

*   `--prebuilt` value: `bigquery`
*   **Environment Variables:**
    *   `BIGQUERY_PROJECT`: The GCP project ID.
    *   `BIGQUERY_LOCATION`: (Optional) The dataset location.
*   **Permissions:**
    *   **BigQuery User** (`roles/bigquery.user`) to execute queries and view metadata.
    *   **BigQuery Metadata Viewer** (`roles/bigquery.metadataViewer`) to view all datasets.
    *   **BigQuery Data Editor** (`roles/bigquery.dataEditor`) to create or modify datasets and tables.
    *   **Gemini for Google Cloud** (`roles/cloudaicompanion.user`) to use the conversational analytics API.
*   **Tools:**
    *   `analyze_contribution`: Use this tool to perform contribution analysis, also called key driver analysis.
    *   `ask_data_insights`: Use this tool to perform data analysis, get insights, or answer complex questions about the contents of specific BigQuery tables. For more information on required roles, API setup, and IAM configuration, see the setup and authentication section of the [Conversational Analytics API documentation](https://cloud.google.com/gemini/docs/conversational-analytics-api/overview).
    *   `execute_sql`: Executes a SQL statement.
    *   `forecast`: Use this tool to forecast time series data.
    *   `get_dataset_info`: Gets dataset metadata.
    *   `get_table_info`: Gets table metadata.
    *   `list_dataset_ids`: Lists datasets.
    *   `list_table_ids`: Lists tables.

## Cloud SQL for MySQL

*   `--prebuilt` value: `cloud-sql-mysql`
*   **Environment Variables:**
    *   `CLOUD_SQL_MYSQL_PROJECT`: The GCP project ID.
    *   `CLOUD_SQL_MYSQL_REGION`: The region of your Cloud SQL instance.
    *   `CLOUD_SQL_MYSQL_INSTANCE`: The ID of your Cloud SQL instance.
    *   `CLOUD_SQL_MYSQL_DATABASE`: The name of the database to connect to.
    *   `CLOUD_SQL_MYSQL_USER`: The database username.
    *   `CLOUD_SQL_MYSQL_PASSWORD`: The password for the database user.
    *   `CLOUD_SQL_MYSQL_IP_TYPE`: The IP type i.e. "Public
     or "Private" (Default: Public).
*   **Permissions:**
    *   **Cloud SQL Client** (`roles/cloudsql.client`) to connect to the instance.
    *   Database-level permissions (e.g., `SELECT`, `INSERT`) are required to execute queries.
*   **Tools:**
    *   `execute_sql`: Executes a SQL query.
    *   `list_tables`: Lists tables in the database.

## Cloud SQL for PostgreSQL

*   `--prebuilt` value: `cloud-sql-postgres`
*   **Environment Variables:**
    *   `CLOUD_SQL_POSTGRES_PROJECT`: The GCP project ID.
    *   `CLOUD_SQL_POSTGRES_REGION`: The region of your Cloud SQL instance.
    *   `CLOUD_SQL_POSTGRES_INSTANCE`: The ID of your Cloud SQL instance.
    *   `CLOUD_SQL_POSTGRES_DATABASE`: The name of the database to connect to.
    *   `CLOUD_SQL_POSTGRES_USER`: (Optional) The database username. Defaults to IAM authentication if unspecified.
    *   `CLOUD_SQL_POSTGRES_PASSWORD`: (Optional) The password for the database user. Defaults to IAM authentication if unspecified.
    *   `CLOUD_SQL_POSTGRES_IP_TYPE`: (Optional) The IP type i.e. "Public" or "Private" (Default: Public).
*   **Permissions:**
    *   **Cloud SQL Client** (`roles/cloudsql.client`) to connect to the instance.
    *   Database-level permissions (e.g., `SELECT`, `INSERT`) are required to execute queries.
*   **Tools:**
    *   `execute_sql`: Executes a SQL query.
    *   `list_tables`: Lists tables in the database.

## Cloud SQL for SQL Server

*   `--prebuilt` value: `cloud-sql-mssql`
*   **Environment Variables:**
    *   `CLOUD_SQL_MSSQL_PROJECT`: The GCP project ID.
    *   `CLOUD_SQL_MSSQL_REGION`: The region of your Cloud SQL instance.
    *   `CLOUD_SQL_MSSQL_INSTANCE`: The ID of your Cloud SQL instance.
    *   `CLOUD_SQL_MSSQL_DATABASE`: The name of the database to connect to.
    *   `CLOUD_SQL_MSSQL_IP_ADDRESS`: The IP address of the Cloud SQL instance.
    *   `CLOUD_SQL_MSSQL_USER`: The database username.
    *   `CLOUD_SQL_MSSQL_PASSWORD`: The password for the database user.
    *   `CLOUD_SQL_MSSQL_IP_TYPE`: (Optional) The IP type i.e. "Public" or "Private" (Default: Public).
*   **Permissions:**
    *   **Cloud SQL Client** (`roles/cloudsql.client`) to connect to the instance.
    *   Database-level permissions (e.g., `SELECT`, `INSERT`) are required to execute queries.
*   **Tools:**
    *   `execute_sql`: Executes a SQL query.
    *   `list_tables`: Lists tables in the database.

## Dataplex

*   `--prebuilt` value: `dataplex`
*   **Environment Variables:**
    *   `DATAPLEX_PROJECT`: The GCP project ID.
*   **Permissions:**
    *   **Dataplex Reader** (`roles/dataplex.viewer`) to search and look up entries.
    *   **Dataplex Editor** (`roles/dataplex.editor`) to modify entries.
*   **Tools:**
    *   `dataplex_search_entries`: Searches for entries in Dataplex Catalog.
    *   `dataplex_lookup_entry`: Retrieves a specific entry from Dataplex Catalog.
    *   `dataplex_search_aspect_types`: Finds aspect types relevant to the query.

## Firestore

*   `--prebuilt` value: `firestore`
*   **Environment Variables:**
    *   `FIRESTORE_PROJECT`: The GCP project ID.
    *   `FIRESTORE_DATABASE`: (Optional) The Firestore database ID. Defaults to "(default)".
*   **Permissions:**
    *   **Cloud Datastore User** (`roles/datastore.user`) to get documents, list collections, and query collections.
    *   **Firebase Rules Viewer** (`roles/firebaserules.viewer`) to get and validate Firestore rules.
*   **Tools:**
    *   `firestore-get-documents`: Gets multiple documents from Firestore by their paths.
    *   `firestore-list-collections`: Lists Firestore collections for a given parent path.
    *   `firestore-delete-documents`: Deletes multiple documents from Firestore.
    *   `firestore-query-collection`: Retrieves one or more Firestore documents from a collection.
    *   `firestore-get-rules`: Retrieves the active Firestore security rules.
    *   `firestore-validate-rules`: Checks the provided Firestore Rules source for syntax and validation errors.

## Looker

*   `--prebuilt` value: `looker`
*   **Environment Variables:**
    *   `LOOKER_BASE_URL`: The URL of your Looker instance.
    *   `LOOKER_CLIENT_ID`: The client ID for the Looker API.
    *   `LOOKER_CLIENT_SECRET`: The client secret for the Looker API.
    *   `LOOKER_VERIFY_SSL`: Whether to verify SSL certificates.
*   **Permissions:**
    *   A Looker account with permissions to access the desired models, explores, and data is required.
*   **Tools:**
    *   `get_models`: Retrieves the list of LookML models.
    *   `get_explores`: Retrieves the list of explores in a model.
    *   `get_dimensions`: Retrieves the list of dimensions in an explore.
    *   `get_measures`: Retrieves the list of measures in an explore.
    *   `get_filters`: Retrieves the list of filters in an explore.
    *   `get_parameters`: Retrieves the list of parameters in an explore.
    *   `query`: Runs a query against the LookML model.
    *   `query_sql`: Generates the SQL for a query.
    *   `query_url`: Generates a URL for a query in Looker.
    *   `get_looks`: Searches for saved looks.
    *   `run_look`: Runs the query associated with a look.
    *   `make_look`: Creates a new look.
    *   `get_dashboards`: Searches for saved dashboards.
    *   `make_dashboard`: Creates a new dashboard.
    *   `add_dashboard_element`: Adds a tile to a dashboard.

## Microsoft SQL Server

*   `--prebuilt` value: `mssql`
*   **Environment Variables:**
    *   `MSSQL_HOST`: The hostname or IP address of the SQL Server instance.
    *   `MSSQL_PORT`: The port number for the SQL Server instance.
    *   `MSSQL_DATABASE`: The name of the database to connect to.
    *   `MSSQL_USER`: The database username.
    *   `MSSQL_PASSWORD`: The password for the database user.
*   **Permissions:**
    *   Database-level permissions (e.g., `SELECT`, `INSERT`) are required to execute queries.
*   **Tools:**
    *   `execute_sql`: Executes a SQL query.
    *   `list_tables`: Lists tables in the database.

## MySQL

*   `--prebuilt` value: `mysql`
*   **Environment Variables:**
    *   `MYSQL_HOST`: The hostname or IP address of the MySQL server.
    *   `MYSQL_PORT`: The port number for the MySQL server.
    *   `MYSQL_DATABASE`: The name of the database to connect to.
    *   `MYSQL_USER`: The database username.
    *   `MYSQL_PASSWORD`: The password for the database user.
*   **Permissions:**
    *   Database-level permissions (e.g., `SELECT`, `INSERT`) are required to execute queries.
*   **Tools:**
    *   `execute_sql`: Executes a SQL query.
    *   `list_tables`: Lists tables in the database.

## OceanBase

*   `--prebuilt` value: `oceanbase`
*   **Environment Variables:**
    *   `OCEANBASE_HOST`: The hostname or IP address of the OceanBase server.
    *   `OCEANBASE_PORT`: The port number for the OceanBase server.
    *   `OCEANBASE_DATABASE`: The name of the database to connect to.
    *   `OCEANBASE_USER`: The database username.
    *   `OCEANBASE_PASSWORD`: The password for the database user.
*   **Permissions:**
    *   Database-level permissions (e.g., `SELECT`, `INSERT`) are required to execute queries.
*   **Tools:**
    *   `execute_sql`: Executes a SQL query.
    *   `list_tables`: Lists tables in the database.

## PostgreSQL

*   `--prebuilt` value: `postgres`
*   **Environment Variables:**
    *   `POSTGRES_HOST`: The hostname or IP address of the PostgreSQL server.
    *   `POSTGRES_PORT`: The port number for the PostgreSQL server.
    *   `POSTGRES_DATABASE`: The name of the database to connect to.
    *   `POSTGRES_USER`: The database username.
    *   `POSTGRES_PASSWORD`: The password for the database user.
    *   `POSTGRES_QUERY_PARAMS`: (Optional) Raw query to be added to the db connection string.
*   **Permissions:**
    *   Database-level permissions (e.g., `SELECT`, `INSERT`) are required to execute queries.
*   **Tools:**
    *   `execute_sql`: Executes a SQL query.
    *   `list_tables`: Lists tables in the database.

## Spanner (GoogleSQL dialect)

*   `--prebuilt` value: `spanner`
*   **Environment Variables:**
    *   `SPANNER_PROJECT`: The GCP project ID.
    *   `SPANNER_INSTANCE`: The Spanner instance ID.
    *   `SPANNER_DATABASE`: The Spanner database ID.
*   **Permissions:**
    *   **Cloud Spanner Database Reader** (`roles/spanner.databaseReader`) to execute DQL queries and list tables.
    *   **Cloud Spanner Database User** (`roles/spanner.databaseUser`) to execute DML queries.
*   **Tools:**
    *   `execute_sql`: Executes a DML SQL query.
    *   `execute_sql_dql`: Executes a DQL SQL query.
    *   `list_tables`: Lists tables in the database.

## Spanner (PostgreSQL dialect)

*   `--prebuilt` value: `spanner-postgres`
*   **Environment Variables:**
    *   `SPANNER_PROJECT`: The GCP project ID.
    *   `SPANNER_INSTANCE`: The Spanner instance ID.
    *   `SPANNER_DATABASE`: The Spanner database ID.
*   **Permissions:**
    *   **Cloud Spanner Database Reader** (`roles/spanner.databaseReader`) to execute DQL queries and list tables.
    *   **Cloud Spanner Database User** (`roles/spanner.databaseUser`) to execute DML queries.
*   **Tools:**
    *   `execute_sql`: Executes a DML SQL query using the PostgreSQL interface for Spanner.
    *   `execute_sql_dql`: Executes a DQL SQL query using the PostgreSQL interface for Spanner.
    *   `list_tables`: Lists tables in the database.

## SQLite

*   `--prebuilt` value: `sqlite`
*   **Environment Variables:**
    *   `SQLITE_DATABASE`: The path to the SQLite database file (e.g., `./sample.db`).
*   **Permissions:**
    *   File system read/write permissions for the specified database file.
*   **Tools:**
    *   `execute_sql`: Executes a SQL query.
    *   `list_tables`: Lists tables in the database.

## Neo4j

*   `--prebuilt` value: `neo4j`
*   **Environment Variables:**
    *   `NEO4J_URI`: The URI of the Neo4j instance (e.g., `bolt://localhost:7687`).
    *   `NEO4J_DATABASE`: The name of the Neo4j database to connect to.
    *   `NEO4J_USERNAME`: The username for the Neo4j instance.
    *   `NEO4J_PASSWORD`: The password for the Neo4j instance.
*   **Permissions:**
    *   **Database-level permissions** are required to execute Cypher queries.
*   **Tools:**
    *   `execute_cypher`: Executes a Cypher query.
    *   `get_schema`: Retrieves the schema of the Neo4j database.
