# Examples

This directory contains example code demonstrating how to use various features of the MCP Toolbox for Databases.

## SQL Validation and Performance Analysis Example

The `sql_validation_example.go` file demonstrates how to use the new SQL validation, sanitization, and performance analysis utilities.

### Features Demonstrated

1. **Query Validation**: Shows how to validate SQL queries for security and best practices
2. **Query Sanitization**: Demonstrates how to clean up SQL queries
3. **Performance Analysis**: Analyzes query complexity and provides optimization suggestions
4. **Integration**: Example of how to integrate validation into your tools

### Running the Example

```bash
go run sql_validation_example.go
```

### What You'll See

The example will show:
- **Security Validation**: Results for various types of queries (safe, suspicious, dangerous)
- **Security Warnings**: Alerts for potential SQL injection attempts
- **Query Sanitization**: Cleaned up queries with removed comments and normalized whitespace
- **Performance Analysis**: Complexity scores, cost estimates, and optimization suggestions
- **Integration Example**: How to reject unsafe queries before execution

### Performance Analysis Features

The new performance analyzer provides:
- **Complexity Scoring**: 1-10 scale based on query complexity factors
- **Cost Estimation**: Low/Medium/High cost estimates
- **Issue Detection**: Identifies performance problems like:
  - Functions in WHERE clauses
  - LIKE queries with leading wildcards
  - Missing indexes on JOIN columns
  - High complexity queries
- **Optimization Suggestions**: Actionable recommendations for:
  - Adding appropriate indexes
  - Restructuring queries for better performance
  - Converting subqueries to JOINs
  - Adding LIMIT clauses

### Use Cases

This comprehensive system can be integrated into:
- **Database Tools**: Prevent dangerous operations and optimize queries
- **Query Builders**: Suggest improvements and best practices
- **Security Scanners**: Detect potential injection attempts
- **Development Tools**: Enforce best practices and performance standards
- **Code Review Tools**: Automatically analyze SQL queries in code reviews
- **Monitoring Systems**: Track query performance and complexity over time

### Customization

The validation and performance analysis rules can be customized by modifying the patterns in:
- `ValidateSQLQuery` function in `internal/util/util.go` for security validation
- `AnalyzeQueryPerformance` function in `internal/util/util.go` for performance analysis
