---
title: "elasticsearch-list-indices"
type: docs
weight: 3
description: >
  List all indices in the cluster.
---

# elasticsearch-list-indices

List all indices in the cluster.

This tool can be used to discover the available indices in your Elasticsearch
cluster. By default, it will list all indices, but you can also specify a
list of indices to get information about.

It will return details such as the index name, health status and number of documents.

See the [official documentation](https://www.elastic.co/docs/api/doc/elasticsearch/operation/operation-cat-indices) for more information.

## Example

```yaml
tools:
  kind: elasticsearch-list-indices
  source: elasticsearch-source
  description: List all indices in the cluster.
  timeout: 30
  parameters:
    - name: indices
      type: array
      description: The indices to list.
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
| indices  | []string |    false     | The indices to list.           |
| timeout | integer |    false     | The timeout for the query in seconds. Default is 60 (1 minute).                          |