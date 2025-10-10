---
title: "firestore-add-documents"
type: docs
weight: 1
description: >
  A "firestore-add-documents" tool adds document to a given collection path.
aliases:
- /resources/tools/firestore-add-documents
---
## Description

The `firestore-add-documents` tool allows you to add new documents to a
Firestore collection. It supports all Firestore data types using Firestore's
native JSON format. The tool automatically generates a unique document ID for
each new document.

## Parameters

| Parameter        | Type    | Required | Description                                                                                                                                                                                                   |
|------------------|---------|----------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `collectionPath` | string  | Yes      | The path of the collection where the document will be added                                                                                                                                                   |
| `documentData`   | map     | Yes      | The data to be added as a document to the given collection. Must use [Firestore's native JSON format](https://cloud.google.com/firestore/docs/reference/rest/Shared.Types/ArrayValue#Value) with typed values |
| `returnData`     | boolean | No       | If set to true, the output will include the data of the created document. Defaults to false to help avoid overloading the context                                                                             |

## Output

The tool returns a map containing:

| Field          | Type   | Description                                                                                                                    |
|----------------|--------|--------------------------------------------------------------------------------------------------------------------------------|
| `documentPath` | string | The full resource name of the created document (e.g., `projects/{projectId}/databases/{databaseId}/documents/{document_path}`) |
| `createTime`   | string | The timestamp when the document was created                                                                                    |
| `documentData` | map    | The data that was added (only included when `returnData` is true)                                                              |

## Data Type Format

The tool requires Firestore's native JSON format for document data. Each field
must be wrapped with its type indicator:

### Basic Types
- **String**: `{"stringValue": "your string"}`
- **Integer**: `{"integerValue": "123"}` or `{"integerValue": 123}`
- **Double**: `{"doubleValue": 123.45}`
- **Boolean**: `{"booleanValue": true}`
- **Null**: `{"nullValue": null}`
- **Bytes**: `{"bytesValue": "base64EncodedString"}`
- **Timestamp**: `{"timestampValue": "2025-01-07T10:00:00Z"}` (RFC3339 format)

### Complex Types
- **GeoPoint**: `{"geoPointValue": {"latitude": 34.052235, "longitude": -118.243683}}`
- **Array**: `{"arrayValue": {"values": [{"stringValue": "item1"}, {"integerValue": "2"}]}}`
- **Map**: `{"mapValue": {"fields": {"key1": {"stringValue": "value1"}, "key2": {"booleanValue": true}}}}`
- **Reference**: `{"referenceValue": "collection/document"}`

## Examples

### Basic Document Creation

```yaml
tools:
  add-company-doc:
    kind: firestore-add-documents
    source: my-firestore
    description: Add a new company document
```

Usage:
```json
{
  "collectionPath": "companies",
  "documentData": {
    "name": {
      "stringValue": "Acme Corporation"
    },
    "establishmentDate": {
      "timestampValue": "2000-01-15T10:30:00Z"
    },
    "location": {
      "geoPointValue": {
        "latitude": 34.052235,
        "longitude": -118.243683
      }
    },
    "active": {
      "booleanValue": true
    },
    "employeeCount": {
      "integerValue": "1500"
    },
    "annualRevenue": {
      "doubleValue": 1234567.89
    }
  }
}
```

### With Nested Maps and Arrays

```json
{
  "collectionPath": "companies",
  "documentData": {
    "name": {
      "stringValue": "Tech Innovations Inc"
    },
    "contactInfo": {
      "mapValue": {
        "fields": {
          "email": {
            "stringValue": "info@techinnovations.com"
          },
          "phone": {
            "stringValue": "+1-555-123-4567"
          },
          "address": {
            "mapValue": {
              "fields": {
                "street": {
                  "stringValue": "123 Innovation Drive"
                },
                "city": {
                  "stringValue": "San Francisco"
                },
                "state": {
                  "stringValue": "CA"
                },
                "zipCode": {
                  "stringValue": "94105"
                }
              }
            }
          }
        }
      }
    },
    "products": {
      "arrayValue": {
        "values": [
          {
            "stringValue": "Product A"
          },
          {
            "stringValue": "Product B"
          },
          {
            "mapValue": {
              "fields": {
                "productName": {
                  "stringValue": "Product C Premium"
                },
                "version": {
                  "integerValue": "3"
                },
                "features": {
                  "arrayValue": {
                    "values": [
                      {
                        "stringValue": "Advanced Analytics"
                      },
                      {
                        "stringValue": "Real-time Sync"
                      }
                    ]
                  }
                }
              }
            }
          }
        ]
      }
    }
  },
  "returnData": true
}
```

### Complete Example with All Data Types

```json
{
  "collectionPath": "test-documents",
  "documentData": {
    "stringField": {
      "stringValue": "Hello World"
    },
    "integerField": {
      "integerValue": "42"
    },
    "doubleField": {
      "doubleValue": 3.14159
    },
    "booleanField": {
      "booleanValue": true
    },
    "nullField": {
      "nullValue": null
    },
    "timestampField": {
      "timestampValue": "2025-01-07T15:30:00Z"
    },
    "geoPointField": {
      "geoPointValue": {
        "latitude": 37.7749,
        "longitude": -122.4194
      }
    },
    "bytesField": {
      "bytesValue": "SGVsbG8gV29ybGQh"
    },
    "arrayField": {
      "arrayValue": {
        "values": [
          {
            "stringValue": "item1"
          },
          {
            "integerValue": "2"
          },
          {
            "booleanValue": false
          }
        ]
      }
    },
    "mapField": {
      "mapValue": {
        "fields": {
          "nestedString": {
            "stringValue": "nested value"
          },
          "nestedNumber": {
            "doubleValue": 99.99
          }
        }
      }
    }
  }
}
```

## Authentication

The tool can be configured to require authentication:

```yaml
tools:
  secure-add-docs:
    kind: firestore-add-documents
    source: prod-firestore
    description: Add documents with authentication required
    authRequired:
      - google-oauth
      - api-key
```

## Error Handling

Common errors include:

- Invalid collection path
- Missing or invalid document data
- Permission denied (if Firestore security rules block the operation)
- Invalid data type conversions

## Best Practices

1. **Always use typed values**: Every field must be wrapped with its appropriate
   type indicator (e.g., `{"stringValue": "text"}`)
2. **Integer values can be strings**: The tool accepts integer values as strings
   (e.g., `{"integerValue": "1500"}`)
3. **Use returnData sparingly**: Only set to true when you need to verify the
   exact data that was written
4. **Validate data before sending**: Ensure your data matches Firestore's native
   JSON format
5. **Handle timestamps properly**: Use RFC3339 format for timestamp strings
6. **Base64 encode binary data**: Binary data must be base64 encoded in the
   `bytesValue` field
7. **Consider security rules**: Ensure your Firestore security rules allow
   document creation in the target collection

## Related Tools

- [`firestore-get-documents`](firestore-get-documents.md) - Retrieve documents
  by their paths
- [`firestore-query-collection`](firestore-query-collection.md) - Query
  documents in a collection
- [`firestore-delete-documents`](firestore-delete-documents.md) - Delete
  documents from Firestore
