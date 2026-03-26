---
title: "looker-get-connection-databases"
type: docs
weight: 1
description: >
  A "looker-get-connection-databases" tool returns all the databases in a connection.

---

## About

A `looker-get-connection-databases` tool returns all the databases in a connection.

`looker-get-connection-databases` accepts a `conn` parameter.

## Compatible Sources

{{< compatible-sources >}}

## Example

```yaml
kind: tool
name: get_connection_databases
type: looker-get-connection-databases
source: looker-source
description: |
  This tool retrieves a list of databases available through a specified Looker connection.
  This is only applicable for connections that support multiple databases.
  Use `get_connections` to check if a connection supports multiple databases.

  Parameters:
  - connection_name (required): The name of the database connection, obtained from `get_connections`.

  Output:
  A JSON array of strings, where each string is the name of an available database.
  If the connection does not support multiple databases, an empty list or an error will be returned.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "looker-get-connection-databases".         |
| source      |  string  |     true     | Name of the source Looker instance.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
