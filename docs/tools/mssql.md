# Cloud SQL Mssql Tool

A "mssql" tool executes a pre-defined T-SQL statement against a Cloud SQL for SQL Server
database. It's compatible with any of the following sources:

- [cloud-sql-mssql](../sources/cloud-sql-mssql.md)

The specified T-SQL statement is executed as a [prepared statement][pg-prepare],
and specified parameters will inserted according to their position: e.g. "$1"
will be the first parameter specified, "$@" will be the second parameter, and so
on.

[pg-prepare]: https://www.postgresql.org/docs/current/sql-prepare.html
