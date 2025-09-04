# Trino Integration Tests

This directory contains integration tests for Trino tools. These tests require a running Trino instance to execute.

## Running Integration Tests

### Prerequisites

You need a running Trino instance accessible from your test environment. This can be:
- A local Trino installation
- A Trino Docker container
- A remote Trino cluster

The tests are configured to connect to `localhost:8080` with default settings.

### Running Tests

Run integration tests with the build tag:

```bash
# Run all Trino integration tests
go test -tags=integration ./internal/tools/trino/...

# Run with verbose output
go test -v -tags=integration ./internal/tools/trino/...

# Run specific test
go test -v -tags=integration -run TestTrinoExecuteSQL ./internal/tools/trino/...

# Run with coverage
go test -v -tags=integration -cover ./internal/tools/trino/...
```

### Using Docker for Testing

You can quickly spin up a Trino instance using Docker:

```bash
# Start Trino container
docker run -d \
  --name trino-test \
  -p 8080:8080 \
  trinodb/trino:latest

# Wait for Trino to be ready
sleep 10

# Run tests (no environment variables needed)
go test -v -tags=integration ./internal/tools/trino/...

# Clean up
docker stop trino-test
docker rm trino-test
```

### Test Coverage

The integration tests cover:

1. **trinoexecutesql** - Direct SQL execution
2. **trinolistcatalogs** - Listing available catalogs
3. **trinolistschemas** - Listing schemas in catalogs
4. **trinolisttables** - Listing tables in schemas
5. **trinogettableinfo** - Getting detailed table metadata
6. **trinoschema** - Comprehensive schema introspection with caching
7. **trinoanalyze** - Query plan analysis
8. **trinotablestatistics** - Table statistics retrieval
9. **trinosql** - Parameterized SQL execution

### Test Data

The tests create temporary tables in the configured catalog/schema:
- `test_table` - Basic table for listing tests
- `test_info_table` - Table with sample data for info retrieval
- `test_stats_table` - Table for statistics tests

All test tables are automatically cleaned up after tests complete.

### Troubleshooting

1. **Connection Issues**
   - Verify Trino is running: `curl http://localhost:8080/v1/info`
   - Check network connectivity
   - Ensure Trino is accessible on localhost:8080

2. **Permission Issues**
   - Ensure the test user has CREATE/DROP table permissions
   - Check catalog/schema permissions

3. **Test Failures**
   - Some tests may skip if certain features aren't available
   - Check Trino logs for detailed error messages
   - Ensure the test catalog supports table creation (e.g., memory catalog)

### CI/CD Integration

For CI/CD pipelines, you can use Docker Compose:

```yaml
# docker-compose.test.yml
version: '3'
services:
  trino:
    image: trinodb/trino:latest
    ports:
      - "8080:8080"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/v1/info"]
      interval: 5s
      timeout: 5s
      retries: 10

  tests:
    build: .
    depends_on:
      trino:
        condition: service_healthy
    command: go test -v -tags=integration ./internal/tools/trino/...
```

Run with:
```bash
docker-compose -f docker-compose.test.yml up --abort-on-container-exit
```
