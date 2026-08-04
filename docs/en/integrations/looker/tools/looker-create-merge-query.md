---
title: "looker-create-merge-query"
type: docs
weight: 1
description: >
  "looker-create-merge-query" merges the results of queries from different Looker explores.
---

## About

The `looker-create-merge-query` tool combines the results of two or more queries
into a single merged result, the API equivalent of Looker's
[Merged Results](https://cloud.google.com/looker/docs/merged-results). Because
each source query targets its own model and explore, this is how an agent joins
data across explores — and even across different databases.

The merge behaves like a SQL left outer join. The first entry of
`source_queries` is the primary query, and the results of every later entry are
joined onto it using the field mappings given in that entry's `merge_fields`.

Looker's API has no endpoint that runs a merge query, so this tool returns the
created merge query and a URL for viewing the merged results in Looker rather
than rows of data. To bring merged data into a conversation, run each source
query with `looker-query` and combine the results.

## Compatible Sources

{{< compatible-sources >}}

## Parameters

`looker-create-merge-query` takes eight parameters. Only `source_queries` is
required; the rest apply to the merged results rather than to any single source
query.

| **parameter**    |  **type**  | **required** | **description**                                                                            |
|------------------|:----------:|:------------:|--------------------------------------------------------------------------------------------|
| `source_queries` |   array    |     true     | At least two query definitions, one per explore. Order matters — see below.                 |
| `pivots`         |   array    |    false     | Merged result fields to pivot by.                                                           |
| `sorts`          |   array    |    false     | Merged result fields to sort by, e.g. `["orders.created_date desc"]`.                       |
| `limit`          |  integer   |    false     | Row limit of the merged results. Defaults to 500.                                           |
| `column_limit`   |  integer   |    false     | Column limit of the merged results.                                                         |
| `total`          |  boolean   |    false     | Whether to include a totals row. Defaults to false.                                         |
| `dynamic_fields` |   array    |    false     | Table calculations, custom measures and custom dimensions over the merged results.          |
| `vis_config`     |   object   |    false     | Default visualization settings, in the same format as `looker-query-url`'s `vis_config`.    |

Each entry of `source_queries` is an object with these keys:

| **key**             |  **type**  | **required** | **description**                                                                              |
|---------------------|:----------:|:------------:|----------------------------------------------------------------------------------------------|
| `model`             |   string   |     true     | The LookML model containing the explore.                                                      |
| `explore`           |   string   |     true     | The explore to query.                                                                         |
| `fields`            |   array    |     true     | The fields to retrieve from this explore.                                                     |
| `merge_fields`      |   array    |    false     | How this entry lines up with the merged results. Not needed on the primary query.             |
| `name`              |   string   |    false     | A display name for this source query. Defaults to the explore name.                           |
| `filters`           |   object   |    false     | Filters for this source query, e.g. `{"orders.created_date": "30 days"}`.                     |
| `pivots`            |   array    |    false     | Pivots for this source query. Fields must also appear in `fields`.                            |
| `sorts`             |   array    |    false     | Sorts for this source query.                                                                  |
| `limit`             |  integer   |    false     | Row limit for this source query.                                                              |
| `filter_expression` |   string   |    false     | A Looker custom filter expression for this source query.                                      |
| `dynamic_fields`    |   array    |    false     | Dynamic fields computed within this source query.                                             |
| `tz`                |   string   |    false     | The timezone for this source query, e.g. `America/Los_Angeles`.                               |

`merge_fields` is an array of objects with two keys: `source_field_name` (a
field of this source query) and `field_name` (the field of the primary query it
maps onto). Every field used for joining must also be listed in that entry's
`fields`.

For example, joining web events onto orders by date:

```json
{
  "source_queries": [
    {
      "name": "Orders",
      "model": "thelook",
      "explore": "orders",
      "fields": ["orders.created_date", "orders.count"],
      "filters": {"orders.created_date": "30 days"}
    },
    {
      "name": "Events",
      "model": "thelook",
      "explore": "events",
      "fields": ["events.event_date", "events.count"],
      "merge_fields": [
        {
          "source_field_name": "events.event_date",
          "field_name": "orders.created_date"
        }
      ]
    }
  ],
  "sorts": ["orders.created_date desc"]
}
```

## Example

```yaml
kind: tool
name: create_merge_query
type: looker-create-merge-query
source: looker-source
description: |
  This tool combines the results of two or more queries from different explores
  into a single merged result, the API equivalent of Looker's "Merged Results".
  The merge behaves like a SQL left outer join: the first source query is the
  primary one, and the results of every later source query are joined onto it.

  Required Parameters:
  - source_queries: An array of at least two query definitions, one per explore.
    Order matters — the first entry is the primary query. Each entry is a JSON
    object with:
    - model (required): The name of the LookML model (from `get_models`).
    - explore (required): The name of the explore (from `get_explores`).
    - fields (required): A list of field names to include in this source query.
    - merge_fields (optional): How this source query lines up with the merged
      results, as an array of `{"source_field_name": "<field of this source
      query>", "field_name": "<field of the primary query it maps onto>"}`
      objects. Not needed on the primary query. Every field used for joining
      must also appear in `fields`.
    - name (optional): A display name for this source query. Defaults to the
      explore name.
    - filters, pivots, sorts, limit, filter_expression, dynamic_fields, tz
      (optional): The same meaning as the equivalent `query` tool parameters,
      applied to this source query only.

  Optional Parameters (applied to the merged results, not the source queries):
  - pivots: A list of merged result fields to pivot by.
  - sorts: A list of merged result fields to sort by (e.g., `["view.field desc"]`).
  - limit: Row limit of the merged results (default 500).
  - column_limit: Column limit of the merged results.
  - total: Whether to include a totals row (default false).
  - dynamic_fields: Table calculations, custom measures and custom dimensions
    computed over the merged results, in the same format as the `query` tool.
  - vis_config: A JSON object controlling the default visualization, in the same
    format as the `query_url` tool's `vis_config`.

  Output:
  A JSON object with the merge query `id`, its `result_maker_id`, the
  `source_queries` that were created (each with `name`, `model`, `explore`,
  `query_id` and `slug`), and a `url` that opens the merged results in Looker.

  Note: Looker has no API endpoint that runs a merge query, so this tool returns
  the merge query and a URL to view it rather than rows of data. To get merged
  data back into the conversation, run each source query with the `query` tool
  and join the results yourself.
```

## Output Format

The tool returns a JSON object describing the merge query it created:

```json
{
  "id": "a1b2c3d4e5f6",
  "result_maker_id": "12345",
  "source_queries": [
    {
      "name": "Orders",
      "model": "thelook",
      "explore": "orders",
      "query_id": "9876",
      "slug": "AbCdEfGhIjKlMnOpQrStUv"
    },
    {
      "name": "Events",
      "model": "thelook",
      "explore": "events",
      "query_id": "9877",
      "slug": "WxYzAbCdEfGhIjKlMnOpQr"
    }
  ],
  "url": "https://looker.example.com/merge?qids%5B%5D=AbCdEfGhIjKlMnOpQrStUv&qids%5B%5D=WxYzAbCdEfGhIjKlMnOpQr"
}
```

The `url` opens the merged results in the Looker UI, pre-populated with the
source queries. It is omitted when the Looker host URL cannot be resolved.

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "looker-create-merge-query"                |
| source      |  string  |     true     | Name of the source the query should execute on.    |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
