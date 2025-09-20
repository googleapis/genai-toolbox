# Examples

This directory contains example code demonstrating how to use various features of the MCP Toolbox for Databases.

## SQL Validation Example

The `sql_validation_example.go` file demonstrates how to use the new SQL validation and sanitization utilities.

### Features Demonstrated

1. **Query Validation**: Shows how to validate SQL queries for security and best practices
2. **Query Sanitization**: Demonstrates how to clean up SQL queries
3. **Integration**: Example of how to integrate validation into your tools

### Running the Example

```bash
go run sql_validation_example.go
```

### What You'll See

The example will show:
- Validation results for various types of queries (safe, suspicious, dangerous)
- Warnings and suggestions for query improvements
- Sanitization results for queries with extra whitespace and comments
- Integration example showing how to reject unsafe queries

### Use Cases

This validation system can be integrated into:
- Database tools to prevent dangerous operations
- Query builders to suggest improvements
- Security scanners to detect potential injection attempts
- Development tools to enforce best practices

### Customization

The validation rules can be customized by modifying the patterns in the `ValidateSQLQuery` function in `internal/util/util.go`.
