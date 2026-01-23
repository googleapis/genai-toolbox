---
title: "firestore-query-collection"
type: docs
weight: 1
description: > 
  A "firestore-query-collection" tool allow to query collections in Firestore.
aliases:
- /resources/tools/firestore-query-collection
---

## About

The `firestore-query-collection` tool allows you to query Firestore collections
with filters, ordering, and limit capabilities.

## Configuration

To use this tool, you need to configure it in your YAML configuration file:

```yaml
sources:
  my-firestore:
    kind: firestore
    project: my-gcp-project
    database: "(default)"

tools:
  query_collection:
    kind: firestore-query-collection
    source: my-firestore
    description: Query Firestore collections with advanced filtering
```

## Parameters

| **parameters**   |   **type**   | **required** | **default** | **description**                                                       |
|------------------|:------------:|:------------:|:-----------:|-----------------------------------------------------------------------|
| `collectionPath` |    string    |     true     |      -      | The Firestore Rules source code to validate                           |
| `filters`        |    array     |     false    |      -      | Array of filter objects (as JSON strings) to apply to the query       |
| `orderBy`        |    string    |     false    |      -      | JSON string specifying field and direction to order results           |
| `limit`          |    integer   |     false    |     100     | Maximum number of documents to return                                 |
| `analyzeQuery`   |    boolean   |     false    |    false    | If true, returns query explain metrics including execution statistics |

### Filter Format

Each filter in the `filters` array should be a JSON string with the following
structure:

```json
{
  "field": "fieldName",
  "op": "operator",
  "value": "compareValue"
}
```

Supported operators:

- `<` - Less than
- `<=` - Less than or equal to
- `>` - Greater than
- `>=` - Greater than or equal to
- `==` - Equal to
- `!=` - Not equal to
- `array-contains` - Array contains a specific value
- `array-contains-any` - Array contains any of the specified values
- `in` - Field value is in the specified array
- `not-in` - Field value is not in the specified array

Value types supported:

- String: `"value": "text"`
- Number: `"value": 123` or `"value": 45.67`
- Boolean: `"value": true` or `"value": false`
- Array: `"value": ["item1", "item2"]` (for `in`, `not-in`, `array-contains-any`
  operators)

### OrderBy Format

The `orderBy` parameter should be a JSON string with the following structure:

```json
{
  "field": "fieldName",
  "direction": "ASCENDING"
}
```

Direction values:

- `ASCENDING`
- `DESCENDING`

## Example Usage

### Query with filters

```json
{
  "collectionPath": "users",
  "filters": [
    "{\"field\": \"age\", \"op\": \">\", \"value\": 18}",
    "{\"field\": \"status\", \"op\": \"==\", \"value\": \"active\"}"
  ],
  "orderBy": "{\"field\": \"createdAt\", \"direction\": \"DESCENDING\"}",
  "limit": 50
}
```

### Query with array contains filter

```json
{
  "collectionPath": "products",
  "filters": [
    "{\"field\": \"categories\", \"op\": \"array-contains\", \"value\": \"electronics\"}",
    "{\"field\": \"price\", \"op\": \"<\", \"value\": 1000}"
  ],
  "orderBy": "{\"field\": \"price\", \"direction\": \"ASCENDING\"}",
  "limit": 20
}
```

### Query with IN operator

```json
{
  "collectionPath": "orders",
  "filters": [
    "{\"field\": \"status\", \"op\": \"in\", \"value\": [\"pending\", \"processing\"]}"
  ],
  "limit": 100
}
```

### Query with explain metrics

```json
{
  "collectionPath": "users",
  "filters": [
    "{\"field\": \"age\", \"op\": \">=\", \"value\": 21}",
    "{\"field\": \"active\", \"op\": \"==\", \"value\": true}"
  ],
  "orderBy": "{\"field\": \"lastLogin\", \"direction\": \"DESCENDING\"}",
  "limit": 25,
  "analyzeQuery": true
}
```

## Response Format

### Standard Response (analyzeQuery = false)

The tool returns an array of documents, where each document includes:

```json
{
  "id": "documentId",
  "path": "collection/documentId",
  "data": {
    // Document fields
  },
  "createTime": "2025-01-07T12:00:00Z",
  "updateTime": "2025-01-07T12:00:00Z",
  "readTime": "2025-01-07T12:00:00Z"
}
```

### Response with Query Analysis (analyzeQuery = true)

When `analyzeQuery` is set to true, the tool returns a single object containing
documents and explain metrics:

```json
{
  "documents": [
    // Array of document objects as shown above
  ],
  "explainMetrics": {
    "planSummary": {
      "indexesUsed": [
        {
          "query_scope": "Collection",
          "properties": "(field ASC, __name__ ASC)"
        }
      ]
    },
    "executionStats": {
      "resultsReturned": 50,
      "readOperations": 50,
      "executionDuration": "120ms",
      "debugStats": {
        "indexes_entries_scanned": "1000",
        "documents_scanned": "50",
        "billing_details": {
          "documents_billable": "50",
          "index_entries_billable": "1000",
          "min_query_cost": "0"
        }
      }
    }
  }
}
```

## Error Handling

The tool will return errors for:

- Invalid collection path
- Malformed filter JSON
- Unsupported operators
- Query execution failures
- Invalid orderBy format
