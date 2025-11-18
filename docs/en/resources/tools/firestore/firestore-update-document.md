---
title: "firestore-update-document"
type: docs
weight: 1
description: >
  A "firestore-update-document" tool updates an existing document in Firestore.
aliases:
- /resources/tools/firestore-update-document
---
## Description

The `firestore-update-document` tool allows you to update existing documents in
Firestore. It supports all Firestore data types using Firestore's native JSON
format. The tool can perform both full document updates (replacing all fields)
or selective field updates using an update mask. When using an update mask,
fields referenced in the mask but not present in the document data will be
deleted from the document, following Firestore's native behavior.

## Parameters

| Parameter      | Type    | Required | Description                                                                                                                                                                                                                                            |
|----------------|---------|----------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `documentPath` | string  | Yes      | The path of the document which needs to be updated                                                                                                                                                                                                     |
| `documentData` | map     | Yes      | The data to update in the document. Must use [Firestore's native JSON format](https://cloud.google.com/firestore/docs/reference/rest/Shared.Types/ArrayValue#Value) with typed values                                                                  |
| `updateMask`   | array   | No       | The selective fields to update. If not provided, all fields in documentData will be updated. When provided, only the specified fields will be updated. Fields referenced in the mask but not present in documentData will be deleted from the document |
| `returnData`   | boolean | No       | If set to true, the output will include the data of the updated document. Defaults to false to help avoid overloading the context                                                                                                                      |

## Output

The tool returns a map containing:

| Field          | Type   | Description                                                                                 |
|----------------|--------|---------------------------------------------------------------------------------------------|
| `documentPath` | string | The full path of the updated document                                                       |
| `updateTime`   | string | The timestamp when the document was updated                                                 |
| `documentData` | map    | The current data of the document after the update (only included when `returnData` is true) |

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

## Update Modes

### Full Document Update (Merge All)

When `updateMask` is not provided, the tool performs a merge operation that
updates all fields specified in `documentData` while preserving other existing
fields in the document.

### Selective Field Update

When `updateMask` is provided, only the fields listed in the mask are updated.
This allows for precise control over which fields are modified, added, or
deleted. To delete a field, include it in the `updateMask` but omit it from
`documentData`.

## Reference

| **field**   | **type** | **required** | **description**                                      |
|-------------|:--------:|:------------:|------------------------------------------------------|
| kind        |  string  |     true     | Must be "firestore-update-document".                 |
| source      |  string  |     true     | Name of the Firestore source to update documents in. |
| description |  string  |     true     | Description of the tool that is passed to the LLM.   |

## Examples

### Basic Document Update (Full Merge)

```yaml
tools:
  update-user-doc:
    kind: firestore-update-document
    source: my-firestore
    description: Update a user document
```

Usage:

```json
{
  "documentPath": "users/user123",
  "documentData": {
    "name": {
      "stringValue": "Jane Doe"
    },
    "lastUpdated": {
      "timestampValue": "2025-01-15T10:30:00Z"
    },
    "status": {
      "stringValue": "active"
    },
    "score": {
      "integerValue": "150"
    }
  }
}
```

### Selective Field Update with Update Mask

```json
{
  "documentPath": "users/user123",
  "documentData": {
    "email": {
      "stringValue": "newemail@example.com"
    },
    "profile": {
      "mapValue": {
        "fields": {
          "bio": {
            "stringValue": "Updated bio text"
          },
          "avatar": {
            "stringValue": "https://example.com/new-avatar.jpg"
          }
        }
      }
    }
  },
  "updateMask": ["email", "profile.bio", "profile.avatar"]
}
```

### Update with Field Deletion

To delete fields, include them in the `updateMask` but omit them from
`documentData`:

```json
{
  "documentPath": "users/user123",
  "documentData": {
    "name": {
      "stringValue": "John Smith"
    }
  },
  "updateMask": ["name", "temporaryField", "obsoleteData"],
  "returnData": true
}
```

In this example:

- `name` will be updated to "John Smith"
- `temporaryField` and `obsoleteData` will be deleted from the document (they
  are in the mask but not in the data)

### Complex Update with Nested Data

```json
{
  "documentPath": "companies/company456",
  "documentData": {
    "metadata": {
      "mapValue": {
        "fields": {
          "lastModified": {
            "timestampValue": "2025-01-15T14:30:00Z"
          },
          "modifiedBy": {
            "stringValue": "admin@company.com"
          }
        }
      }
    },
    "locations": {
      "arrayValue": {
        "values": [
          {
            "mapValue": {
              "fields": {
                "city": {
                  "stringValue": "San Francisco"
                },
                "coordinates": {
                  "geoPointValue": {
                    "latitude": 37.7749,
                    "longitude": -122.4194
                  }
                }
              }
            }
          },
          {
            "mapValue": {
              "fields": {
                "city": {
                  "stringValue": "New York"
                },
                "coordinates": {
                  "geoPointValue": {
                    "latitude": 40.7128,
                    "longitude": -74.0060
                  }
                }
              }
            }
          }
        ]
      }
    },
    "revenue": {
      "doubleValue": 5678901.23
    }
  },
  "updateMask": ["metadata", "locations", "revenue"]
}
```

### Update with All Data Types

```json
{
  "documentPath": "test-documents/doc789",
  "documentData": {
    "stringField": {
      "stringValue": "Updated string"
    },
    "integerField": {
      "integerValue": "999"
    },
    "doubleField": {
      "doubleValue": 2.71828
    },
    "booleanField": {
      "booleanValue": false
    },
    "nullField": {
      "nullValue": null
    },
    "timestampField": {
      "timestampValue": "2025-01-15T16:45:00Z"
    },
    "geoPointField": {
      "geoPointValue": {
        "latitude": 51.5074,
        "longitude": -0.1278
      }
    },
    "bytesField": {
      "bytesValue": "VXBkYXRlZCBkYXRh"
    },
    "arrayField": {
      "arrayValue": {
        "values": [
          {
            "stringValue": "updated1"
          },
          {
            "integerValue": "200"
          },
          {
            "booleanValue": true
          }
        ]
      }
    },
    "mapField": {
      "mapValue": {
        "fields": {
          "nestedString": {
            "stringValue": "updated nested value"
          },
          "nestedNumber": {
            "doubleValue": 88.88
          }
        }
      }
    },
    "referenceField": {
      "referenceValue": "users/updatedUser"
    }
  },
  "returnData": true
}
```

## Authentication

The tool can be configured to require authentication:

```yaml
tools:
  secure-update-doc:
    kind: firestore-update-document
    source: prod-firestore
    description: Update documents with authentication required
    authRequired:
      - google-oauth
      - api-key
```

## Error Handling

Common errors include:

- Document not found (when using update with a non-existent document)
- Invalid document path
- Missing or invalid document data
- Permission denied (if Firestore security rules block the operation)
- Invalid data type conversions

## Best Practices

1. **Use update masks for precision**: When you only need to update specific
   fields, use the `updateMask` parameter to avoid unintended changes
2. **Always use typed values**: Every field must be wrapped with its appropriate
   type indicator (e.g., `{"stringValue": "text"}`)
3. **Integer values can be strings**: The tool accepts integer values as strings
   (e.g., `{"integerValue": "1500"}`)
4. **Use returnData sparingly**: Only set to true when you need to verify the
   exact data after the update
5. **Validate data before sending**: Ensure your data matches Firestore's native
   JSON format
6. **Handle timestamps properly**: Use RFC3339 format for timestamp strings
7. **Base64 encode binary data**: Binary data must be base64 encoded in the
   `bytesValue` field
8. **Consider security rules**: Ensure your Firestore security rules allow
   document updates
9. **Delete fields using update mask**: To delete fields, include them in the
   `updateMask` but omit them from `documentData`
10. **Test with non-production data first**: Always test your updates on
    non-critical documents first

## Differences from Add Documents

- **Purpose**: Updates existing documents vs. creating new ones
- **Document must exist**: For standard updates (though not using updateMask
  will create if missing with given document id)
- **Update mask support**: Allows selective field updates
- **Field deletion**: Supports removing specific fields by including them in the
  mask but not in the data
- **Returns updateTime**: Instead of createTime

## Related Tools

- [`firestore-add-documents`](firestore-add-documents.md) - Add new documents to
  Firestore
- [`firestore-get-documents`](firestore-get-documents.md) - Retrieve documents
  by their paths
- [`firestore-query-collection`](firestore-query-collection.md) - Query
  documents in a collection
- [`firestore-delete-documents`](firestore-delete-documents.md) - Delete
  documents from Firestore
