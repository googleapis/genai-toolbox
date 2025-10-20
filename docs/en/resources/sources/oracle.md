---
title: "Oracle"
type: docs
weight: 1
description: >
  Oracle Database is a widely-used relational database management system.
---

## About

[Oracle Database][oracle-docs] is a multi-model database management system produced and marketed by Oracle Corporation. It is commonly used for running online transaction processing (OLTP), data warehousing (DW), and mixed (OLTP & DW) database workloads.

[oracle-docs]: https://www.oracle.com/database/

## Available Tools

- [`oracle-sql`](../tools/oracle/oracle-sql.md)
  Execute pre-defined prepared SQL queries in Oracle.

- [`oracle-execute-sql`](../tools/oracle/oracle-execute-sql.md)
  Run parameterized SQL queries in Oracle.

## Requirements

### Database User

This source uses standard authentication. You will need to [create an Oracle user][oracle-users] to log in to the database with the necessary permissions.

[oracle-users]:
    https://docs.oracle.com/en/database/oracle/oracle-database/21/sqlrf/CREATE-USER.html

## Connection Methods

You can configure the connection to your Oracle database using one of the following three methods. **You should only use one method** in your source configuration.

### Basic Connection (Host/Port/Service Name)

This is the most straightforward method, where you provide the connection details as separate fields:

- `host`: The IP address or hostname of the database server.
- `port`: The port number the Oracle listener is running on (typically 1521).
- `serviceName`: The service name for the database instance you wish to connect to.

### Connection String

As an alternative, you can provide all the connection details in a single `connectionString`. This is a convenient way to consolidate the connection information. The typical format is `hostname:port/servicename`.

### TNS Alias

For environments that use a `tnsnames.ora` configuration file, you can connect using a TNS (Transparent Network Substrate) alias.

- `tnsAlias`: Specify the alias name defined in your `tnsnames.ora` file.
- `tnsAdmin` (Optional): If your configuration file is not in a standard location, you can use this field to provide the path to the directory containing it. This setting will override the `TNS_ADMIN` environment variable.

## Example

```yaml
sources:
    my-oracle-source:
        kind: oracle
        # --- Choose one connection method ---
        # 1. Host, Port, and Service Name
        host: 127.0.0.1
        port: 1521
        serviceName: XEPDB1

        # 2. Direct Connection String
        connectionString: "127.0.0.1:1521/XEPDB1"

        # 3. TNS Alias (requires tnsnames.ora)
        tnsAlias: "MY_DB_ALIAS"
        tnsAdmin: "/opt/oracle/network/admin" # Optional: overrides TNS_ADMIN env var

        user: ${USER_NAME}
        password: ${PASSWORD}

```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

| **field**        | **type** | **required** | **description**                                                                                                             |
|------------------|:--------:|:------------:|-----------------------------------------------------------------------------------------------------------------------------|
| kind             |  string  |     true     | Must be "oracle".                                                                                                           |
| user             |  string  |     true     | Name of the Oracle user to connect as (e.g. "my-oracle-user").                                                              |
| password         |  string  |     true     | Password of the Oracle user (e.g. "my-password").                                                                           |
| host             |  string  |    false     | IP address or hostname to connect to (e.g. "127.0.0.1"). Required if not using `connectionString` or `tnsAlias`.            |
| port             | integer  |    false     | Port to connect to (e.g. "1521"). Required if not using `connectionString` or `tnsAlias`.                                   |
| serviceName      |  string  |    false     | The Oracle service name of the database to connect to. Required if not using `connectionString` or `tnsAlias`.              |
| connectionString |  string  |    false     | A direct connection string (e.g. "hostname:port/servicename"). Use as an alternative to `host`, `port`, and `serviceName`.  |
| tnsAlias         |  string  |    false     | A TNS alias from a `tnsnames.ora` file. Use as an alternative to `host`/`port` or `connectionString`.                       |
| tnsAdmin         |  string  |    false     | Path to the directory containing the `tnsnames.ora` file. This overrides the `TNS_ADMIN` environment variable if it is set. |
