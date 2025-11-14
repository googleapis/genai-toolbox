---
title: "postgres-list-indexes"
type: docs
weight: 1
description: >
 The "postgres-list-indexes" tool lists indexes in a Postgres database.
aliases:
- /resources/tools/postgres-list-indexes
---

## About

The `postgres-list-indexes` tool lists available user indexes in the database excluding those in `pg_catalog` and `information_schema`. It's compatible with any of the following sources:

- [alloydb-postgres](../../sources/alloydb-pg.md)
- [cloud-sql-postgres](../../sources/cloud-sql-pg.md)
- [postgres](../../sources/postgres.md)

`postgres-list-indexes` lists detailed information as JSON for indexes. The tool takes the following input parameters:

- `table_name` (optional): A text to filter results by table name. The input is used within a LIKE clause. Default: `""`
- `index_name` (optional): A text to filter results by index name. The input is used within a LIKE clause. Default: `""`
- `schema_name` (optional): A text to filter results by schema name. The input is used within a LIKE clause. Default: `""`
- `limit` (optional): The maximum number of rows to return. Default: `50`.

## Example

```yaml
tools:
  list_indexes:
    kind: postgres-list-indexes
    source: postgres-source
    description: |
      Lists available user indexes in the database, excluding system schemas (pg_catalog, 
      information_schema). For each index, the following properties are returned: 
      schema name, table name, index name, index type (access method), a boolean 
      indicating if it's a unique index, a boolean indicating if it's for a primary key,
      the index definition, index size in bytes, the number of index scans, the number of 
      index tuples read, the number of table tuples fetched via index scans, and a boolean 
      indicating if the index has been used at least once. 
```

The response is a json array with the following elements:
```json
{
 "schema_name": "schema name", 
 "table_name": "table name",
 "index_name": "index name",
 "index_type": "index access method (e.g btree, hash, gin)",
 "is_unique": "boolean indicating if the index is unique",
 "is_primary": "boolean indicating if the index is for a primary key",
 "index_definition": "index definition statement",
 "index_size_bytes": "index size in bytes",
 "index_scans": "Number of index scans initiated on this index",
 "tuples_read": "Number of index entries returned by scans on this index",
 "tuples_fetched": "Number of live table rows fetched by simple index scans using this index", 
 "is_used": "boolean indicating if the index has been scanned at least once"
}
```

## Reference

| **field**   | **type** | **required**  | **description**                                      |
|-------------|:--------:|:-------------:|------------------------------------------------------|
| kind        |  string  |     true      | Must be "postgres-list-indexes".                      |
| source      |  string  |     true      | Name of the source the SQL should execute on.        |
| description |  string  |     false     | Description of the tool that is passed to the agent. |
