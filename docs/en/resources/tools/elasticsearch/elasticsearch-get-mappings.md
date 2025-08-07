---
title: "elasticsearch-get-mappings"
type: docs
weight: 4
description: >
  Get the mapping of a specific index.
---

# elasticsearch-get-mappings

Get the mapping of a specific index.

This tool is useful for understanding the structure of your indices. It will
return the mapping of the specified indices, which includes the fields and
their types.

See the [official documentation](https://www.elastic.co/guide/en/elasticsearch/reference/current/indices-get-mapping.html) for more information.

## Example

```yaml
tools:
  kind: elasticsearch-get-mappings
  source: elasticsearch-source
  description: Get the mapping of a specific index.
  parameters:
    - name: indices
      type: array
      description: The indices to get the mapping for.
      default:
        - "*"
      items:
        name: index
        type: string
        description: The name of the index.
```

## Parameters

| **name** | **type** | **required** | **description**                |
|----------|:--------:|:------------:|--------------------------------|
| indices  | []string |    false     | The indices to get the mapping for. |
| timeout | integer |    false     | The timeout for the query in seconds. Default is 60 (1 minute).                          |