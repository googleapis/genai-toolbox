---
title: "Elasticsearch"
type: docs
weight: 1
description: >
  Elasticsearch is a distributed, free and open search and analytics engine 
  for all types of data, including textual, numerical, geospatial, structured, 
  and unstructured.
---

# Elasticsearch Source

[Elasticsearch][elasticsearch-docs] is a distributed, free and open search and analytics engine
for all types of data, including textual, numerical, geospatial, structured,
and unstructured.

If you are new to Elasticsearch, you can learn how to 
[set up a cluster and start indexing data][elasticsearch-quickstart].

Elasticsearch uses [ES|QL][elasticsearch-esql] for querying data. ES|QL
is a powerful query language that allows you to search and aggregate data in
Elasticsearch.

See the [official documentation](https://www.elastic.co/guide/en/elasticsearch/reference/current/index.html) for more information.

[elasticsearch-docs]: https://www.elastic.co/guide/en/elasticsearch/reference/current/index.html
[elasticsearch-quickstart]: https://www.elastic.co/guide/en/elasticsearch/reference/current/getting-started.html
[elasticsearch-esql]: https://www.elastic.co/guide/en/elasticsearch/reference/current/esql.html

## Available Tools

- [`elasticsearch-esql`](../tools/elasticsearch/elasticsearch-esql.md)
  Execute ES|QL queries.

## Requirements

### API Key

Toolbox uses an [API key][api-key] to authorize and authenticate when
interacting with [Elasticsearch][elasticsearch-docs].

In addition to [setting the API key for your server][set-api-key], you need to
ensure the API key has the correct permissions for the queries you intend to
run. See [API key management][api-key-management] for more information on
applying permissions to an API key.

[api-key]: https://www.elastic.co/guide/en/elasticsearch/reference/current/security-api-create-api-key.html
[set-api-key]: https://www.elastic.co/guide/en/elasticsearch/reference/current/security-api-create-api-key.html
[api-key-management]: https://www.elastic.co/guide/en/elasticsearch/reference/current/security-api-get-api-key.html

## Example

```yaml
sources:
  my-elasticsearch-source:
    kind: "elasticsearch"
    addresses:
      - "http://localhost:9200"
    apikey: "my-api-key"
```

## Reference

| **field** | **type** | **required** | **description**                                                               |
|-----------|:--------:|:------------:|-------------------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "elasticsearch".                                                      |
| addresses | []string |     true     | List of Elasticsearch hosts to connect to.                                    |
| apikey    |  string  |     true     | The API key to use for authentication.                                        |
