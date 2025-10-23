---
title: "Oracle OCI"
type: docs
weight: 2
description: >
  Oracle Database connection using Oracle Call Interface (OCI) via godror driver wrapper.
---

## About

The `oracle-oci` source provides Oracle Database connectivity using the [godror driver][godror-docs], which is a Go wrapper around the Oracle Call Interface (OCI) using Oracle Instant Client. This driver provides full support for Oracle-specific features including TNS aliases and Oracle Wallet authentication.

[godror-docs]: https://godror.github.io/godror/

## When to Use oracle-oci

Use `oracle-oci` when you need:
- **Oracle Autonomous Database** connections (requires wallet authentication)
- **TNS aliases** from tnsnames.ora configuration files
- **Oracle Wallet** for mutual TLS authentication
- **Full Oracle OCI features** and compatibility

For simple direct connections without these requirements, consider using the [`oracle`](../oracle.md) source which is pure Go and requires no external dependencies.

## Available Tools

- [`oracle-sql`](../tools/oracle/oracle-sql.md)
  Execute pre-defined prepared SQL queries in Oracle.

- [`oracle-execute-sql`](../tools/oracle/oracle-execute-sql.md)
  Run parameterized SQL queries in Oracle.

## Requirements

### Oracle Instant Client

The `oracle-oci` source requires Oracle Instant Client to be installed on the system where the toolbox runs.

**Download:** https://www.oracle.com/database/technologies/instant-client/downloads.html

Choose the appropriate version for your operating system (Basic or Basic Light package is sufficient).

### Database User

You will need to [create an Oracle user][oracle-users] with the necessary permissions to access your database.

[oracle-users]:
    https://docs.oracle.com/en/database/oracle/oracle-database/21/sqlrf/CREATE-USER.html

### Build Requirements

The `oracle-oci` source is optional and must be explicitly enabled at build time using build tags.

```bash
# Set environment variables
export LD_LIBRARY_PATH=/path/to/instantclient:$LD_LIBRARY_PATH
export CGO_ENABLED=1

# Build with oracle-oci support
go build -tags oracleoci -o toolbox
```

{{< notice note >}}
The default build (`go build`) does NOT include `oracle-oci` support. You must use the `-tags oracleoci` flag.
{{< /notice >}}

## Connection Methods

You can configure the connection to your Oracle database using one of the following three methods. **You should only use one method** in your source configuration.

{{< notice important >}}
**Oracle Wallet / mTLS Authentication:** If your database requires Oracle Wallet or mutual TLS (mTLS) authentication (such as Oracle Autonomous Database), you **MUST use the TNS Alias method**. The Connection String and Host+Port+ServiceName methods do not support wallet-based authentication.
{{< /notice >}}

### TNS Alias (Required for Wallet Authentication)

**Use this method when:**
- Connecting to Oracle Autonomous Database
- Your database requires Oracle Wallet for mutual TLS (mTLS)
- Using tnsnames.ora for connection configuration

For Oracle Autonomous Database or any Oracle deployment using tnsnames.ora configuration:

- `tnsAlias`: Specify the alias name defined in your `tnsnames.ora` file.
- `tnsAdmin` (Optional): Path to the directory containing the `tnsnames.ora` and wallet files. This overrides the `TNS_ADMIN` environment variable.

This method enables wallet-based authentication and is the **only method** that supports mutual TLS connections.

### Connection String

**Use this method when:**
- Connecting to standard Oracle databases (not Autonomous Database)
- No Oracle Wallet or mTLS authentication is required
- You prefer a single-string connection format

Provide all connection details in a single `connectionString`. The typical format is `hostname:port/servicename`.

{{< notice note >}}
This method does **not** support Oracle Wallet or mTLS authentication. For Autonomous Database, use the TNS Alias method.
{{< /notice >}}

### Host + Port + ServiceName

**Use this method when:**
- Connecting to standard Oracle databases (not Autonomous Database)
- No Oracle Wallet or mTLS authentication is required
- You prefer separate fields for connection details

Provide the connection details as separate fields:

- `host`: The IP address or hostname of the database server.
- `port`: The port number the Oracle listener is running on (typically 1521).
- `serviceName`: The service name for the database instance you wish to connect to.

{{< notice note >}}
This method does **not** support Oracle Wallet or mTLS authentication. For Autonomous Database, use the TNS Alias method.
{{< /notice >}}

## Choosing a Connection Method

| Database Type | Wallet/mTLS Required? | Recommended Method |
|---------------|----------------------|-------------------|
| Oracle Autonomous Database | ✅ Yes | **TNS Alias** (only option) |
| Standard Oracle with Wallet | ✅ Yes | **TNS Alias** (only option) |
| Standard Oracle without Wallet | ❌ No | Any method (TNS Alias, Connection String, or Host+Port) |

## Examples

### Oracle Autonomous Database (with Wallet)

```yaml
sources:
    my-autonomous-db:
        kind: oracle-oci
        tnsAlias: "mydb_high"                        # TNS alias from tnsnames.ora
        tnsAdmin: "/home/user/wallet/Wallet_MyDB"    # Path to wallet directory
        user: ${ORACLE_USER}
        password: ${ORACLE_PASSWORD}

tools:
    query-sales:
        kind: oracle-sql
        source: my-autonomous-db
        description: Query sales data by region
        parameters:
          - name: region
            type: string
            description: Sales region code
        statement: SELECT * FROM sales WHERE region = :1
```

### Direct Connection (Host + Port + ServiceName)

```yaml
sources:
    my-oracle-db:
        kind: oracle-oci
        host: 127.0.0.1
        port: 1521
        serviceName: XEPDB1
        user: ${ORACLE_USER}
        password: ${ORACLE_PASSWORD}
```

### Connection String

```yaml
sources:
    my-oracle-db:
        kind: oracle-oci
        connectionString: "myhost.example.com:1521/PRODDB"
        user: ${ORACLE_USER}
        password: ${ORACLE_PASSWORD}
```

### With Custom TNS_ADMIN Location

```yaml
sources:
    my-oracle-db:
        kind: oracle-oci
        tnsAlias: "MY_DB"
        tnsAdmin: "/opt/oracle/network/admin"  # Overrides TNS_ADMIN env var
        user: ${ORACLE_USER}
        password: ${ORACLE_PASSWORD}
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

| **field**        | **type** | **required** | **description**                                                                                                             |
|------------------|:--------:|:------------:|-----------------------------------------------------------------------------------------------------------------------------|
| kind             |  string  |     true     | Must be "oracle-oci".                                                                                                       |
| user             |  string  |     true     | Name of the Oracle user to connect as (e.g. "my-oracle-user").                                                              |
| password         |  string  |     true     | Password of the Oracle user (e.g. "my-password").                                                                           |
| host             |  string  |    false     | IP address or hostname to connect to (e.g. "127.0.0.1"). Required if not using `connectionString` or `tnsAlias`.            |
| port             | integer  |    false     | Port to connect to (e.g. "1521"). Optional when using `host`, defaults to 1521 if not specified.                            |
| serviceName      |  string  |    false     | The Oracle service name of the database to connect to. Required if not using `connectionString` or `tnsAlias`.              |
| connectionString |  string  |    false     | A direct connection string (e.g. "hostname:port/servicename"). Use as an alternative to `host`, `port`, and `serviceName`.  |
| tnsAlias         |  string  |    false     | A TNS alias from a `tnsnames.ora` file. Use as an alternative to `host`/`port` or `connectionString`.                       |
| tnsAdmin         |  string  |    false     | Path to the directory containing the `tnsnames.ora` file and wallet. This overrides the `TNS_ADMIN` environment variable.   |

## Comparison: oracle vs oracle-oci

| Feature                  | oracle (go-ora)        | oracle-oci (godror)    |
|--------------------------|------------------------|------------------------|
| External Dependencies    | None                   | Oracle Instant Client  |
| Build Tags Required      | No                     | Yes (`-tags oracleoci`)|
| TNS Alias Support        | No                     | Yes                    |
| Oracle Wallet Support    | No                     | Yes                    |
| Autonomous Database      | No                     | Yes                    |
| Pure Go                  | Yes                    | No (uses CGO)          |
| Connection Methods       | 3 (host, string, TNS*) | 3 (host, string, TNS)  |

\* TNS configuration fields exist in `oracle` source but are not functional with go-ora driver.

## Troubleshooting

### Error: "unknown source kind: oracle-oci"

This means the toolbox was built without oracle-oci support. Rebuild with:
```bash
export CGO_ENABLED=1
export LD_LIBRARY_PATH=/path/to/instantclient:$LD_LIBRARY_PATH
go build -tags oracleoci -o toolbox
```

### Error: "cannot find Instant Client library"

Set the `LD_LIBRARY_PATH` (Linux/macOS) or `PATH` (Windows) to include the Oracle Instant Client directory:
```bash
export LD_LIBRARY_PATH=/path/to/instantclient:$LD_LIBRARY_PATH
./toolbox --tools-file tools.yaml
```

### Error: "ORA-12154: TNS:could not resolve the connect identifier"

- Verify `tnsAlias` matches an entry in your `tnsnames.ora` file
- Check that `tnsAdmin` points to the correct directory containing `tnsnames.ora`
- Ensure the `TNS_ADMIN` environment variable is set if not using the `tnsAdmin` config field

### Wallet Authentication Issues

- Verify wallet files (`cwallet.sso`, `ewallet.p12`, etc.) are in the directory specified by `tnsAdmin`
- Ensure file permissions allow the toolbox process to read wallet files
