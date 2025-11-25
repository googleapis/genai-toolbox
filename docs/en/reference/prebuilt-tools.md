---
title: "Prebuilt Tools"
type: docs
weight: 1
description: >
    This page lists all the prebuilt tools available.
---

Prebuilt tools are reusable, pre-packaged toolsets that are designed to extend
the capabilities of agents. These tools are built to be generic and adaptable,
allowing developers to interact with and take action on databases.

See guides, [Connect from your IDE](../how-to/connect-ide/_index.md), for
details on how to connect your AI tools (IDEs) to databases via Toolbox and MCP.

## AlloyDB Postgres

*   `--prebuilt` value: `alloydb-postgres`
*   **Environment Variables:**
    *   `ALLOYDB_POSTGRES_PROJECT`: The GCP project ID.
    *   `ALLOYDB_POSTGRES_REGION`: The region of your AlloyDB instance.
    *   `ALLOYDB_POSTGRES_CLUSTER`: The ID of your AlloyDB cluster.
    *   `ALLOYDB_POSTGRES_INSTANCE`: The ID of your AlloyDB instance.
    *   `ALLOYDB_POSTGRES_DATABASE`: The name of the database to connect to.
    *   `ALLOYDB_POSTGRES_USER`: (Optional) The database username. Defaults to
        IAM authentication if unspecified.
    *   `ALLOYDB_POSTGRES_PASSWORD`: (Optional) The password for the database
        user. Defaults to IAM authentication if unspecified.
    *   `ALLOYDB_POSTGRES_IP_TYPE`: (Optional) The IP type i.e. "Public" or
        "Private" (Default: Public).
*   **Permissions:**
    *   **AlloyDB Client** (`roles/alloydb.client`) to connect to the instance.
    *   Database-level permissions (e.g., `SELECT`, `INSERT`) are required to
        execute queries.
*   **Tools:**
    *   `execute_sql`: Executes a SQL query.
    *   `list_tables`: Lists tables in the database.
    *   `list_autovacuum_configurations`: Lists autovacuum configurations in the
        database.
    *   `list_memory_configurations`: Lists memory-related configurations in the
        database.
    *   `list_top_bloated_tables`: List top bloated tables in the database.
    *   `list_replication_slots`: Lists replication slots in the database.
    *   `list_invalid_indexes`: Lists invalid indexes in the database.
    *   `get_query_plan`: Generate the execution plan of a statement.
    *   `list_views`: Lists views in the database from pg_views with a default
        limit of 50 rows. Returns schemaname, viewname and the ownername.
    *   `list_schemas`: Lists schemas in the database. 
    *   `database_overview`: Fetches the current state of the PostgreSQL server. 
    *   `list_triggers`: Lists triggers in the database. 
    *   `list_indexes`: List available user indexes in a PostgreSQL database.
    *   `list_sequences`: List sequences in a PostgreSQL database.

## AlloyDB Postgres Admin

* `--prebuilt` value: `alloydb-postgres-admin`
*   **Permissions:**
    *   **AlloyDB Viewer** (`roles/alloydb.viewer`) is required for `list` and
        `get` tools.
    *   **AlloyDB Admin** (`roles/alloydb.admin`) is required for `create` tools.
*   **Tools:**
    *   `create_cluster`: Creates a new AlloyDB cluster.
    *   `list_clusters`: Lists all AlloyDB clusters in a project.
    *   `get_cluster`: Gets information about a specified AlloyDB cluster.
    *   `create_instance`: Creates a new AlloyDB instance within a cluster.
    *   `list_instances`: Lists all instances within an AlloyDB cluster.
    *   `get_instance`: Gets information about a specified AlloyDB instance.
    *   `create_user`: Creates a new database user in an AlloyDB cluster.
    *   `list_users`: Lists all database users within an AlloyDB cluster.
    *   `get_user`: Gets information about a specified database user in an
        AlloyDB cluster.
    *   `wait_for_operation`: Polls the operations API to track the status of
        long-running operations.

## AlloyDB Postgres Observability

*   `--prebuilt` value: `alloydb-postgres-observability`
*   **Permissions:**
    *   **Monitoring Viewer** (`roles/monitoring.viewer`) is required on the
        project to view monitoring data.
*   **Tools:**
    *   `get_system_metrics`: Fetches system level cloud monitoring data
        (timeseries metrics) for an AlloyDB instance using a PromQL query.
    *   `get_query_metrics`: Fetches query level cloud monitoring data
        (timeseries metrics) for queries running in an AlloyDB instance using a
        PromQL query.

## BigQuery

*   `--prebuilt` value: `bigquery`
*   **Environment Variables:**
    *   `BIGQUERY_PROJECT`: The GCP project ID.
    *   `BIGQUERY_LOCATION`: (Optional) The dataset location.
    *   `BIGQUERY_USE_CLIENT_OAUTH`: (Optional) If `true`, forwards the client's
        OAuth access token for authentication. Defaults to `false`.
*   **Permissions:**
    *   **BigQuery User** (`roles/bigquery.user`) to execute queries and view
        metadata.
    *   **BigQuery Metadata Viewer** (`roles/bigquery.metadataViewer`) to view
        all datasets.
    *   **BigQuery Data Editor** (`roles/bigquery.dataEditor`) to create or
        modify datasets and tables.
    *   **Gemini for Google Cloud** (`roles/cloudaicompanion.user`) to use the
        conversational analytics API.
*   **Tools:**
    *   `analyze_contribution`: Use this tool to perform contribution analysis,
        also called key driver analysis.
    *   `ask_data_insights`: Use this tool to perform data analysis, get
        insights, or answer complex questions about the contents of specific
        BigQuery tables. For more information on required roles, API setup, and
        IAM configuration, see the setup and authentication section of the
        [Conversational Analytics API
        documentation](https://cloud.google.com/gemini/docs/conversational-analytics-api/overview).
    *   `execute_sql`: Executes a SQL statement.
    *   `forecast`: Use this tool to forecast time series data.
    *   `get_dataset_info`: Gets dataset metadata.
    *   `get_table_info`: Gets table metadata.
    *   `list_dataset_ids`: Lists datasets.
    *   `list_table_ids`: Lists tables.
    *   `search_catalog`: Search for entries based on the provided query.

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
    *   **Cloud SQL Client** (`roles/cloudsql.client`) to connect to the
        instance.
    *   Database-level permissions (e.g., `SELECT`, `INSERT`) are required to
        execute queries.
*   **Tools:**
    *   `execute_sql`: Executes a SQL query.
    *   `list_tables`: Lists tables in the database.
    *   `get_query_plan`: Provides information about how MySQL executes a SQL
        statement.
    *   `list_active_queries`: Lists ongoing queries.
    *   `list_tables_missing_unique_indexes`: Looks for tables that do not have
        primary or unique key contraint.
    *   `list_table_fragmentation`: Displays table fragmentation in MySQL.

## Cloud SQL for MySQL Observability

*   `--prebuilt` value: `cloud-sql-mysql-observability`
*   **Permissions:**
    *   **Monitoring Viewer** (`roles/monitoring.viewer`) is required on the
        project to view monitoring data.
*   **Tools:**
    *   `get_system_metrics`: Fetches system level cloud monitoring data
        (timeseries metrics) for a MySQL instance using a PromQL query.
    *   `get_query_metrics`: Fetches query level cloud monitoring data
        (timeseries metrics) for queries running in a MySQL instance using a
        PromQL query.

## Cloud SQL for MySQL Admin

*   `--prebuilt` value: `cloud-sql-mysql-admin`
*   **Permissions:**
    *   **Cloud SQL Viewer** (`roles/cloudsql.viewer`): Provides read-only
        access to resources.
        * `get_instance`
        * `list_instances`
        * `list_databases`
        * `wait_for_operation`
    *   **Cloud SQL Editor** (`roles/cloudsql.editor`): Provides permissions to
        manage existing resources.
        * All `viewer` tools
        * `create_database`
    *   **Cloud SQL Admin** (`roles/cloudsql.admin`): Provides full control over
        all resources.
        * All `editor` and `viewer` tools
        * `create_instance`
        * `create_user`
*   **Tools:**
    *   `create_instance`: Creates a new Cloud SQL for MySQL instance.
    *   `get_instance`: Gets information about a Cloud SQL instance.
    *   `list_instances`: Lists Cloud SQL instances in a project.
    *   `create_database`: Creates a new database in a Cloud SQL instance.
    *   `list_databases`: Lists all databases for a Cloud SQL instance.
    *   `create_user`: Creates a new user in a Cloud SQL instance.
    *   `wait_for_operation`: Waits for a Cloud SQL operation to complete.

## Cloud SQL for PostgreSQL

*   `--prebuilt` value: `cloud-sql-postgres`
*   **Environment Variables:**
    *   `CLOUD_SQL_POSTGRES_PROJECT`: The GCP project ID.
    *   `CLOUD_SQL_POSTGRES_REGION`: The region of your Cloud SQL instance.
    *   `CLOUD_SQL_POSTGRES_INSTANCE`: The ID of your Cloud SQL instance.
    *   `CLOUD_SQL_POSTGRES_DATABASE`: The name of the database to connect to.
    *   `CLOUD_SQL_POSTGRES_USER`: (Optional) The database username. Defaults to
        IAM authentication if unspecified.
    *   `CLOUD_SQL_POSTGRES_PASSWORD`: (Optional) The password for the database
        user. Defaults to IAM authentication if unspecified.
    *   `CLOUD_SQL_POSTGRES_IP_TYPE`: (Optional) The IP type i.e. "Public" or
        "Private" (Default: Public).
*   **Permissions:**
    *   **Cloud SQL Client** (`roles/cloudsql.client`) to connect to the
        instance.
    *   Database-level permissions (e.g., `SELECT`, `INSERT`) are required to
        execute queries.
*   **Tools:**
    *   `execute_sql`: Executes a SQL query.
    *   `list_tables`: Lists tables in the database.
    *   `list_autovacuum_configurations`: Lists autovacuum configurations in the
        database.
    *   `list_memory_configurations`: Lists memory-related configurations in the
        database.
    *   `list_top_bloated_tables`: List top bloated tables in the database.
    *   `list_replication_slots`: Lists replication slots in the database.
    *   `list_invalid_indexes`: Lists invalid indexes in the database.
    *   `get_query_plan`: Generate the execution plan of a statement.
    *   `list_views`: Lists views in the database from pg_views with a default
        limit of 50 rows. Returns schemaname, viewname and the ownername.
    *   `list_schemas`: Lists schemas in the database. 
    *   `database_overview`: Fetches the current state of the PostgreSQL server. 
    *   `list_triggers`: Lists triggers in the database. 
    *   `list_indexes`: List available user indexes in a PostgreSQL database.
    *   `list_sequences`: List sequences in a PostgreSQL database.

## Cloud SQL for PostgreSQL Observability

*   `--prebuilt` value: `cloud-sql-postgres-observability`
*   **Permissions:**
    *   **Monitoring Viewer** (`roles/monitoring.viewer`) is required on the
        project to view monitoring data.
*   **Tools:**
    *   `get_system_metrics`: Fetches system level cloud monitoring data
        (timeseries metrics) for a Postgres instance using a PromQL query.
    *   `get_query_metrics`: Fetches query level cloud monitoring data
        (timeseries metrics) for queries running in Postgres instance using a
        PromQL query.

## Cloud SQL for PostgreSQL Admin

*   `--prebuilt` value: `cloud-sql-postgres-admin`
*   **Permissions:**
    *   **Cloud SQL Viewer** (`roles/cloudsql.viewer`): Provides read-only
        access to resources.
        * `get_instance`
        * `list_instances`
        * `list_databases`
        * `wait_for_operation`
    *   **Cloud SQL Editor** (`roles/cloudsql.editor`): Provides permissions to
        manage existing resources.
        * All `viewer` tools
        * `create_database`
    *   **Cloud SQL Admin** (`roles/cloudsql.admin`): Provides full control over
        all resources.
        * All `editor` and `viewer` tools
        * `create_instance`
        * `create_user`
*   **Tools:**
    *   `create_instance`: Creates a new Cloud SQL for PostgreSQL instance.
    *   `get_instance`: Gets information about a Cloud SQL instance.
    *   `list_instances`: Lists Cloud SQL instances in a project.
    *   `create_database`: Creates a new database in a Cloud SQL instance.
    *   `list_databases`: Lists all databases for a Cloud SQL instance.
    *   `create_user`: Creates a new user in a Cloud SQL instance.
    *   `wait_for_operation`: Waits for a Cloud SQL operation to complete.

## Cloud SQL for SQL Server

*   `--prebuilt` value: `cloud-sql-mssql`
*   **Environment Variables:**
    *   `CLOUD_SQL_MSSQL_PROJECT`: The GCP project ID.
    *   `CLOUD_SQL_MSSQL_REGION`: The region of your Cloud SQL instance.
    *   `CLOUD_SQL_MSSQL_INSTANCE`: The ID of your Cloud SQL instance.
    *   `CLOUD_SQL_MSSQL_DATABASE`: The name of the database to connect to.
    *   `CLOUD_SQL_MSSQL_USER`: The database username.
    *   `CLOUD_SQL_MSSQL_PASSWORD`: The password for the database user.
    *   `CLOUD_SQL_MSSQL_IP_TYPE`: (Optional) The IP type i.e. "Public" or
        "Private" (Default: Public).
*   **Permissions:**
    *   **Cloud SQL Client** (`roles/cloudsql.client`) to connect to the
        instance.
    *   Database-level permissions (e.g., `SELECT`, `INSERT`) are required to
        execute queries.
*   **Tools:**
    *   `execute_sql`: Executes a SQL query.
    *   `list_tables`: Lists tables in the database.

## Cloud SQL for SQL Server Observability

*   `--prebuilt` value: `cloud-sql-mssql-observability`
*   **Permissions:**
    *   **Monitoring Viewer** (`roles/monitoring.viewer`) is required on the
        project to view monitoring data.
*   **Tools:**
    *   `get_system_metrics`: Fetches system level cloud monitoring data
        (timeseries metrics) for a SQL Server instance using a PromQL query.

## Cloud SQL for SQL Server Admin

*   `--prebuilt` value: `cloud-sql-mssql-admin`
*   **Permissions:**
    *   **Cloud SQL Viewer** (`roles/cloudsql.viewer`): Provides read-only
        access to resources.
        * `get_instance`
        * `list_instances`
        * `list_databases`
        * `wait_for_operation`
    *   **Cloud SQL Editor** (`roles/cloudsql.editor`): Provides permissions to
        manage existing resources.
        * All `viewer` tools
        * `create_database`
    *   **Cloud SQL Admin** (`roles/cloudsql.admin`): Provides full control over
        all resources.
        * All `editor` and `viewer` tools
        * `create_instance`
        * `create_user`
*   **Tools:**
    *   `create_instance`: Creates a new Cloud SQL for SQL Server instance.
    *   `get_instance`: Gets information about a Cloud SQL instance.
    *   `list_instances`: Lists Cloud SQL instances in a project.
    *   `create_database`: Creates a new database in a Cloud SQL instance.
    *   `list_databases`: Lists all databases for a Cloud SQL instance.
    *   `create_user`: Creates a new user in a Cloud SQL instance.
    *   `wait_for_operation`: Waits for a Cloud SQL operation to complete.

## Dataplex

*   `--prebuilt` value: `dataplex`
*   **Environment Variables:**
    *   `DATAPLEX_PROJECT`: The GCP project ID.
*   **Permissions:**
    *   **Dataplex Reader** (`roles/dataplex.viewer`) to search and look up
        entries.
    *   **Dataplex Editor** (`roles/dataplex.editor`) to modify entries.
*   **Tools:**
    *   `dataplex_search_entries`: Searches for entries in Dataplex Catalog.
    *   `dataplex_lookup_entry`: Retrieves a specific entry from Dataplex
        Catalog.
    *   `dataplex_search_aspect_types`: Finds aspect types relevant to the
        query.

## Firestore

*   `--prebuilt` value: `firestore`
*   **Environment Variables:**
    *   `FIRESTORE_PROJECT`: The GCP project ID.
    *   `FIRESTORE_DATABASE`: (Optional) The Firestore database ID. Defaults to
        "(default)".
*   **Permissions:**
    *   **Cloud Datastore User** (`roles/datastore.user`) to get documents, list
        collections, and query collections.
    *   **Firebase Rules Viewer** (`roles/firebaserules.viewer`) to get and
        validate Firestore rules.
*   **Tools:**
    *   `get_documents`: Gets multiple documents from Firestore by their paths.
    *   `add_documents`: Adds a new document to a Firestore collection.
    *   `update_document`: Updates an existing document in Firestore.
    *   `list_collections`: Lists Firestore collections for a given parent path.
    *   `delete_documents`: Deletes multiple documents from Firestore.
    *   `query_collection`: Retrieves one or more Firestore documents from a
        collection.
    *   `get_rules`: Retrieves the active Firestore security rules.
    *   `validate_rules`: Checks the provided Firestore Rules source for syntax
        and validation errors.

## Looker

*   `--prebuilt` value: `looker`
*   **Environment Variables:**
    *   `LOOKER_BASE_URL`: The URL of your Looker instance.
    *   `LOOKER_CLIENT_ID`: The client ID for the Looker API.
    *   `LOOKER_CLIENT_SECRET`: The client secret for the Looker API.
    *   `LOOKER_VERIFY_SSL`: Whether to verify SSL certificates.
    *   `LOOKER_USE_CLIENT_OAUTH`: Whether to use OAuth for authentication.
    *   `LOOKER_SHOW_HIDDEN_MODELS`: Whether to show hidden models.
    *   `LOOKER_SHOW_HIDDEN_EXPLORES`: Whether to show hidden explores.
    *   `LOOKER_SHOW_HIDDEN_FIELDS`: Whether to show hidden fields.
*   **Permissions:**
    *   A Looker account with permissions to access the desired models,
        explores, and data is required.
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
    *   `run_dashboard`: Runs the queries associated with a dashboard.
    *   `make_dashboard`: Creates a new dashboard.
    *   `add_dashboard_element`: Adds a tile to a dashboard.
    *   `health_pulse`: Test the health of a Looker instance.
    *   `health_analyze`: Analyze the LookML usage of a Looker instance.
    *   `health_vacuum`: Suggest LookML elements that can be removed.
    *   `dev_mode`: Activate developer mode.
    *   `get_projects`: Get the LookML projects in a Looker instance.
    *   `get_project_files`: List the project files in a project.
    *   `get_project_file`: Get the content of a LookML file.
    *   `create_project_file`: Create a new LookML file.
    *   `update_project_file`: Update an existing LookML file.
    *   `delete_project_file`: Delete a LookML file.
    *   `get_connections`: Get the available connections in a Looker instance.
    *   `get_connection_schemas`: Get the available schemas in a connection.
    *   `get_connection_databases`: Get the available databases in a connection.
    *   `get_connection_tables`: Get the available tables in a connection.
    *   `get_connection_table_columns`: Get the available columns for a table.

## Looker Conversational Analytics

*   `--prebuilt` value: `looker-conversational-analytics`
*   **Environment Variables:**
    *   `LOOKER_BASE_URL`: The URL of your Looker instance.
    *   `LOOKER_CLIENT_ID`: The client ID for the Looker API.
    *   `LOOKER_CLIENT_SECRET`: The client secret for the Looker API.
    *   `LOOKER_VERIFY_SSL`: Whether to verify SSL certificates.
    *   `LOOKER_USE_CLIENT_OAUTH`: Whether to use OAuth for authentication.
    *   `LOOKER_PROJECT`: The GCP Project to use for Conversational Analytics.
    *   `LOOKER_LOCATION`: The GCP Location to use for Conversational Analytics.
*   **Permissions:**
    *   A Looker account with permissions to access the desired models,
        explores, and data is required.
    *   **Looker Instance User** (`roles/looker.instanceUser`): IAM role to
        access Looker.
    *   **Gemini for Google Cloud User** (`roles/cloudaicompanion.user`): IAM
        role to access Conversational Analytics.
    *   **Gemini Data Analytics Stateless Chat User (Beta)**
        (`roles/geminidataanalytics.dataAgentStatelessUser`): IAM role to
        access Conversational Analytics.
*   **Tools:**
    *   `ask_data_insights`: Ask a question of the data.
    *   `get_models`: Retrieves the list of LookML models.
    *   `get_explores`: Retrieves the list of explores in a model.

## Microsoft SQL Server

*   `--prebuilt` value: `mssql`
*   **Environment Variables:**
    *   `MSSQL_HOST`: The hostname or IP address of the SQL Server instance.
    *   `MSSQL_PORT`: The port number for the SQL Server instance.
    *   `MSSQL_DATABASE`: The name of the database to connect to.
    *   `MSSQL_USER`: The database username.
    *   `MSSQL_PASSWORD`: The password for the database user.
*   **Permissions:**
    *   Database-level permissions (e.g., `SELECT`, `INSERT`) are required to
        execute queries.
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
    *   Database-level permissions (e.g., `SELECT`, `INSERT`) are required to
        execute queries.
*   **Tools:**
    *   `execute_sql`: Executes a SQL query.
    *   `list_tables`: Lists tables in the database.
    *   `get_query_plan`: Provides information about how MySQL executes a SQL
        statement.
    *   `list_active_queries`: Lists ongoing queries.
    *   `list_tables_missing_unique_indexes`: Looks for tables that do not have
        primary or unique key contraint.
    *   `list_table_fragmentation`: Displays table fragmentation in MySQL.

## OceanBase

*   `--prebuilt` value: `oceanbase`
*   **Environment Variables:**
    *   `OCEANBASE_HOST`: The hostname or IP address of the OceanBase server.
    *   `OCEANBASE_PORT`: The port number for the OceanBase server.
    *   `OCEANBASE_DATABASE`: The name of the database to connect to.
    *   `OCEANBASE_USER`: The database username.
    *   `OCEANBASE_PASSWORD`: The password for the database user.
*   **Permissions:**
    *   Database-level permissions (e.g., `SELECT`, `INSERT`) are required to
        execute queries.
*   **Tools:**
    *   `execute_sql`: Executes a SQL query.
    *   `list_tables`: Lists tables in the database.

## PostgreSQL

*   `--prebuilt` value: `postgres`
*   **Environment Variables:**
    *   `POSTGRES_HOST`: (Optional) The hostname or IP address of the PostgreSQL server.
    *   `POSTGRES_PORT`: (Optional) The port number for the PostgreSQL server.
    *   `POSTGRES_DATABASE`: The name of the database to connect to.
    *   `POSTGRES_USER`: The database username.
    *   `POSTGRES_PASSWORD`: The password for the database user.
    *   `POSTGRES_QUERY_PARAMS`: (Optional) Raw query to be added to the db
        connection string.
*   **Permissions:**
    *   Database-level permissions (e.g., `SELECT`, `INSERT`) are required to
        execute queries.
*   **Tools:**
    *   `execute_sql`: Executes a SQL query.
    *   `list_tables`: Lists tables in the database.
    *   `list_autovacuum_configurations`: Lists autovacuum configurations in the
        database.
    *   `list_memory_configurations`: Lists memory-related configurations in the
        database.
    *   `list_top_bloated_tables`: List top bloated tables in the database.
    *   `list_replication_slots`: Lists replication slots in the database.
    *   `list_invalid_indexes`: Lists invalid indexes in the database.
    *   `get_query_plan`: Generate the execution plan of a statement.
    *   `list_views`: Lists views in the database from pg_views with a default
        limit of 50 rows. Returns schemaname, viewname and the ownername.
    *   `list_schemas`: Lists schemas in the database. 
    *   `database_overview`: Fetches the current state of the PostgreSQL server. 
    *   `list_triggers`: Lists triggers in the database. 
    *   `list_indexes`: List available user indexes in a PostgreSQL database.
    *   `list_sequences`: List sequences in a PostgreSQL database.

## Google Cloud Serverless for Apache Spark

*   `--prebuilt` value: `serverless-spark`
*   **Environment Variables:**
    *   `SERVERLESS_SPARK_PROJECT`: The GCP project ID
    *   `SERVERLESS_SPARK_LOCATION`: The GCP Location.
*   **Permissions:**
    *   **Dataproc Serverless Viewer** (`roles/dataproc.serverlessViewer`) to
        view serverless batches.
    *   **Dataproc Serverless Editor** (`roles/dataproc.serverlessEditor`) to
        view serverless batches.
*   **Tools:**
    *   `list_batches`: Lists Spark batches.

## Spanner (GoogleSQL dialect)

*   `--prebuilt` value: `spanner`
*   **Environment Variables:**
    *   `SPANNER_PROJECT`: The GCP project ID.
    *   `SPANNER_INSTANCE`: The Spanner instance ID.
    *   `SPANNER_DATABASE`: The Spanner database ID.
*   **Permissions:**
    *   **Cloud Spanner Database Reader** (`roles/spanner.databaseReader`) to
        execute DQL queries and list tables.
    *   **Cloud Spanner Database User** (`roles/spanner.databaseUser`) to
        execute DML queries.
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
    *   **Cloud Spanner Database Reader** (`roles/spanner.databaseReader`) to
        execute DQL queries and list tables.
    *   **Cloud Spanner Database User** (`roles/spanner.databaseUser`) to
        execute DML queries.
*   **Tools:**
    *   `execute_sql`: Executes a DML SQL query using the PostgreSQL interface
        for Spanner.
    *   `execute_sql_dql`: Executes a DQL SQL query using the PostgreSQL
        interface for Spanner.
    *   `list_tables`: Lists tables in the database.

## SQLite

*   `--prebuilt` value: `sqlite`
*   **Environment Variables:**
    *   `SQLITE_DATABASE`: The path to the SQLite database file (e.g.,
        `./sample.db`).
*   **Permissions:**
    *   File system read/write permissions for the specified database file.
*   **Tools:**
    *   `execute_sql`: Executes a SQL query.
    *   `list_tables`: Lists tables in the database.

## Neo4j

*   `--prebuilt` value: `neo4j`
*   **Environment Variables:**
    *   `NEO4J_URI`: The URI of the Neo4j instance (e.g.,
        `bolt://localhost:7687`).
    *   `NEO4J_DATABASE`: The name of the Neo4j database to connect to.
    *   `NEO4J_USERNAME`: The username for the Neo4j instance.
    *   `NEO4J_PASSWORD`: The password for the Neo4j instance.
*   **Permissions:**
    *   **Database-level permissions** are required to execute Cypher queries.
*   **Tools:**
    *   `execute_cypher`: Executes a Cypher query.
    *   `get_schema`: Retrieves the schema of the Neo4j database.

## Google Cloud Healthcare API
*   `--prebuilt` value: `cloud-healthcare`
*   **Environment Variables:**
    *   `CLOUD_HEALTHCARE_PROJECT`: The GCP project ID.
    *   `CLOUD_HEALTHCARE_REGION`: The Cloud Healthcare API dataset region.
    *   `CLOUD_HEALTHCARE_DATASET`: The Cloud Healthcare API dataset ID.
    *   `CLOUD_HEALTHCARE_USE_CLIENT_OAUTH`: (Optional) If `true`, forwards the client's
        OAuth access token for authentication. Defaults to `false`.
*   **Permissions:**
    *   **Healthcare FHIR Resource Reader** (`roles/healthcare.fhirResourceReader`) to read an
        search FHIR resources.
    *   **Healthcare DICOM Viewer** (`roles/healthcare.dicomViewer`) to retrieve DICOM images from a
        DICOM store.
*   **Tools:**
    *   `get_dataset`: Gets information about a Cloud Healthcare API dataset.
    *   `list_dicom_stores`: Lists DICOM stores in a Cloud Healthcare API dataset.
    *   `list_fhir_stores`: Lists FHIR stores in a Cloud Healthcare API dataset.
    *   `get_fhir_store`: Gets information about a FHIR store.
    *   `get_fhir_store_metrics`: Gets metrics for a FHIR store.
    *   `get_fhir_resource`: Gets a FHIR resource from a FHIR store.
    *   `fhir_patient_search`: Searches for patient resource(s) based on a set of criteria.
    *   `fhir_patient_everything`: Retrieves resources related to a given patient.
    *   `fhir_fetch_page`: Fetches a page of FHIR resources.
    *   `get_dicom_store`: Gets information about a DICOM store.
    *   `get_dicom_store_metrics`: Gets metrics for a DICOM store.
    *   `search_dicom_studies`: Searches for DICOM studies.
    *   `search_dicom_series`: Searches for DICOM series.
    *   `search_dicom_instances`: Searches for DICOM instances.
    *   `retrieve_rendered_dicom_instance`: Retrieves a rendered DICOM instance.
