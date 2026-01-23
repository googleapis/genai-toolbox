---
title: "Snowflake"
type: docs
weight: 1
description: >
  Snowflake is a cloud-based data platform.

---

## About

[Snowflake][sf-docs] is a cloud data platform that provides a data warehouse-as-a-service designed for the cloud.

[sf-docs]: https://docs.snowflake.com/

## Available Tools

- [`snowflake-sql`](../tools/snowflake/snowflake-sql.md)  
  Execute SQL queries as prepared statements in Snowflake.

- [`snowflake-execute-sql`](../tools/snowflake/snowflake-execute-sql.md)  
  Run parameterized SQL statements in Snowflake.

## Requirements

### Database User

This source only uses standard authentication. You will need to create a
Snowflake user to login to the database with.

## Example

```yaml
sources:
    my-sf-source:
        kind: snowflake
        account: ${SNOWFLAKE_ACCOUNT}
        user: ${SNOWFLAKE_USER}
        password: ${SNOWFLAKE_PASSWORD}
        database: ${SNOWFLAKE_DATABASE}
        schema: ${SNOWFLAKE_SCHEMA}
        warehouse: ${SNOWFLAKE_WAREHOUSE}
        role: ${SNOWFLAKE_ROLE}
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

| **field** | **type** | **required** | **description**                                                        |
|-----------|:--------:|:------------:|------------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "snowflake".                                                   |
| account   |  string  |     true     | Your Snowflake account identifier.                                     |
| user      |  string  |     true     | Name of the Snowflake user to connect as (e.g. "my-sf-user").          |
| password  |  string  |     true     | Password of the Snowflake user (e.g. "my-password").                   |
| database  |  string  |     true     | Name of the Snowflake database to connect to (e.g. "my_db").           |
| schema    |  string  |     true     | Name of the schema to use (e.g. "my_schema").                          |
| warehouse |  string  |     false    | The virtual warehouse to use. Defaults to "COMPUTE_WH".                |
| role      |  string  |     false    | The security role to use. Defaults to "ACCOUNTADMIN".                  |
| timeout   |  integer |     false    | The connection timeout in seconds. Defaults to 60.                     |