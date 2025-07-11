---
title: "dataplex-search-entries"
type: docs
weight: 1
description: > 
  A "dataplex-search-entries" tool returns all entries in Dataplex Catalog.
aliases:
- /resources/tools/dataplex-search-entries
---

## About

A `dataplex-search-entries` tool returns all entries in Dataplex Catalog (e.g. tables, views, models) that matches given user query.
It's compatible with the following sources:

- [dataplex](../sources/dataplex.md)

dataplex-search-entries requires a `query` parameter as input based on which entries are filtered and returned to the user.

## Example

```yaml
tools:
  dataplex-search-entries:
    kind: dataplex-search-entries
    source: my-dataplex-source
    description: Use this tool to get all the entries based on user query.
    parameters:
      - name: query
        type: string
        description: "Keyword search query for entries"
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "dataplex-search-entries".                                                               |
| source      |                   string                   |     true     | Name of the source the tool should execute on.                                                   |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
| parameters  | [parameters](_index#specifying-parameters) |     true     | List of [parameters](_index#specifying-parameters) that will be passed                           |
