---
title: Trino
description: Connect to Trino distributed SQL query engine
weight: 160
---

# Trino Source

The Trino source enables MCP Toolbox to connect to [Trino](https://trino.io/), a distributed SQL query engine for big data analytics. Trino can query data from multiple sources including Hive, Iceberg, Delta Lake, PostgreSQL, MySQL, and many others.

## Configuration

### Basic Configuration

```yaml
sources:
  my-trino:
    kind: trino
    host: localhost
    port: "8080"
    user: myuser
    catalog: hive
    schema: default
```

### With Authentication

#### Basic Authentication
```yaml
sources:
  my-trino:
    kind: trino
    host: trino.example.com
    port: "8080"
    user: myuser
    password: mypassword
    catalog: hive
    schema: default
```

#### JWT Token Authentication
```yaml
sources:
  my-trino:
    kind: trino
    host: trino.example.com
    port: "443"
    user: myuser
    catalog: hive
    schema: default
    accessToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
    sslEnabled: true
```

#### Kerberos Authentication
```yaml
sources:
  my-trino:
    kind: trino
    host: trino.example.com
    port: "8443"
    user: myuser@EXAMPLE.COM
    catalog: hive
    schema: default
    kerberosEnabled: true
    sslEnabled: true
```

### Advanced Configuration

```yaml
sources:
  my-trino:
    kind: trino
    host: trino.example.com
    port: "8443"
    user: myuser
    password: mypassword
    catalog: hive
    schema: analytics
    queryTimeout: "30m"
    sslEnabled: true
```

## Configuration Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `kind` | string | Yes | Must be `trino` |
| `host` | string | Yes | Trino coordinator hostname |
| `port` | string | Yes | Trino coordinator port (usually 8080 for HTTP, 8443 for HTTPS) |
| `user` | string | Yes | Username for authentication |
| `password` | string | No | Password for basic authentication |
| `catalog` | string | Yes | Default catalog to use for queries |
| `schema` | string | Yes | Default schema to use for queries |
| `queryTimeout` | string | No | Query timeout duration (e.g., "30m", "1h") |
| `accessToken` | string | No | JWT access token for authentication |
| `kerberosEnabled` | boolean | No | Enable Kerberos authentication (default: false) |
| `sslEnabled` | boolean | No | Enable SSL/TLS (default: false) |

## Environment Variables

You can use environment variables for configuration:

```yaml
sources:
  my-trino:
    kind: trino
    host: ${TRINO_HOST}
    port: ${TRINO_PORT}
    user: ${TRINO_USER}
    password: ${TRINO_PASSWORD}
    catalog: ${TRINO_CATALOG}
    schema: ${TRINO_SCHEMA}
    queryTimeout: ${TRINO_QUERY_TIMEOUT}
    accessToken: ${TRINO_ACCESS_TOKEN}
    kerberosEnabled: ${TRINO_KERBEROS_ENABLED}
    sslEnabled: ${TRINO_SSL_ENABLED}
```

## Compatible Tools

The following tools are compatible with the Trino source:

- [`trino-execute-sql`](../../tools/trino/trino-execute-sql.md) - Execute arbitrary SQL queries
- [`trino-sql`](../../tools/trino/trino-sql.md) - Execute parameterized SQL queries

## Usage Examples

### Basic Query Execution

```yaml
tools:
  query-sales:
    kind: trino-execute-sql
    source: my-trino
    description: Execute SQL queries against Trino
```

### Parameterized Queries

```yaml
tools:
  get-user-orders:
    kind: trino-sql
    source: my-trino
    description: Get orders for a specific user
    statement: |
      SELECT order_id, product_name, quantity, price
      FROM orders o
      JOIN products p ON o.product_id = p.id
      WHERE o.user_id = $1
      ORDER BY o.created_at DESC
    parameters:
      - name: user_id
        type: string
        description: The user ID to get orders for
        required: true
```

## Connection Details

### URL Format

The Trino source uses the official [Trino Go client](https://github.com/trinodb/trino-go-client) with connection strings in the format:

```
http://user@host:port?catalog=catalog&schema=schema
```

For SSL connections:
```
https://user@host:port?catalog=catalog&schema=schema
```

### Authentication Methods

1. **No Authentication**: Just specify user (for development/testing)
2. **Basic Authentication**: Include password in configuration
3. **JWT Token**: Use `accessToken` parameter for bearer token auth
4. **Kerberos**: Enable with `kerberosEnabled: true`

### SSL/TLS

Enable SSL by setting `sslEnabled: true`. This will use HTTPS instead of HTTP for the connection.

## Trino-Specific Features

### Catalogs and Schemas

Trino organizes data into catalogs and schemas:
- **Catalog**: A data source (e.g., Hive, PostgreSQL, MySQL)
- **Schema**: A collection of tables within a catalog

You can query across different catalogs:
```sql
SELECT *
FROM hive.sales.orders o
JOIN postgresql.users.customers c ON o.customer_id = c.id
```

### Query Federation

Trino's strength is in federating queries across multiple data sources. You can join data from different systems in a single query.

### Performance Considerations

- Set appropriate `queryTimeout` for long-running analytical queries
- Consider connection pooling settings for high-concurrency workloads
- Use pushdown predicates when possible to reduce data movement

## Troubleshooting

### Connection Issues

1. **Verify Trino coordinator is running**:
   ```bash
   curl http://trino-host:8080/v1/info
   ```

2. **Check authentication**: Ensure credentials are correct and user has necessary permissions

3. **Network connectivity**: Verify firewall rules and network access to Trino coordinator

### Query Issues

1. **Catalog/Schema errors**: Verify the specified catalog and schema exist and are accessible
2. **Permission errors**: Ensure user has SELECT permissions on required tables
3. **Timeout errors**: Increase `queryTimeout` for long-running queries

### SSL/TLS Issues

1. **Certificate errors**: Ensure valid SSL certificates are configured on Trino
2. **Port configuration**: Use appropriate HTTPS port (typically 8443)

## Security Best Practices

1. **Use environment variables** for sensitive configuration like passwords and tokens
2. **Enable SSL/TLS** for production deployments
3. **Implement proper authentication** (avoid anonymous access in production)
4. **Follow principle of least privilege** for database user permissions
5. **Rotate access tokens** regularly when using JWT authentication
6. **Use Kerberos** in enterprise environments for centralized authentication

## Further Reading

- [Trino Documentation](https://trino.io/docs/)
- [Trino Go Client](https://github.com/trinodb/trino-go-client)
- [Trino Security](https://trino.io/docs/current/security.html)
- [Trino Catalogs](https://trino.io/docs/current/connector.html)
