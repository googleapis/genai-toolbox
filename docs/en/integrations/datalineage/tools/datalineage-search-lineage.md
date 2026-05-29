---
title: "datalineage-search-lineage"
type: docs
weight: 1
description: >
  A "datalineage-search-lineage" tool allows to retrieve a streaming response of lineage links connected to the requested assets.
---

## About

A `datalineage-search-lineage` tool retrieves lineage links (representing data flow between source and target assets) by performing a breadth-first search in the given direction.

It supports both entity-level and column-level lineage (CLL).

## Compatible Sources

{{< compatible-sources >}}

## Requirements

### IAM Permissions

To retrieve lineage links, the authorized Google Cloud identity must have the following permissions:

- `datalineage.events.get` on the project where the link is stored (for entity-level lineage).
- `datalineage.events.getFields` on the project where the link is stored (for column-level lineage).
- `datalineage.processes.get` on the project where the process is stored (if retrieving process details).

## Parameters

### `locations` (Array of Strings, Required)

The locations to search in (e.g., `['us', 'eu', 'global']`).

- Must contain at least **one** location.
- The first location in the list is used as the **parent location** to initiate the search and for billing/quota.
- The search will retrieve links from all valid locations provided in the list.

### `root_entities` (Array of Objects, Required)

The starting entities (roots) for the lineage graph traversal. The maximum number of entities is 20.
Each object represents an entity reference and must contain:

- `fully_qualified_name` (String, Required): The FQN of the entity (e.g., a BigQuery table FQN).
- `fields` (Array of Strings, Optional): The fields/columns of the entity to search for **Column-Level Lineage (CLL)**. Supports wildcards.

### `direction` (String, Required)

The direction of the search. Supported values:

- `UPSTREAM`: Search for assets that contribute to the root entities (sources).
- `DOWNSTREAM`: Search for assets that are derived from the root entities (targets).

### `max_depth` (Integer, Optional)

The maximum depth of the search in the lineage graph.

- **API Default:** `5`
- **Maximum:** `100`
- **Note:** If omitted, the GCP API default of `5` will be used.

### `max_results` (Integer, Optional)

The maximum number of links to return in the response.

- **API Default:** `1000`
- **Maximum:** `10000`
- **Note:** If omitted, the GCP API default of `1000` will be used.

### `max_process_per_link` (Integer, Optional)

The maximum number of processes to return per link.

- **API Default:** `0` (no processes returned).
- **Maximum:** `100`
- **Note:** If omitted, the GCP API default of `0` will be used. Must be greater than `0` if `request_process_details` is `true`.

### `request_process_details` (Boolean, Optional)

If set to `true`, instructs the tool to retrieve full process details (such as `displayName`, `attributes`, and `origin`) for the processes associated with the links.

- **Default:** `false`
- **Requirement:** Requires `max_process_per_link` to be explicitly set to a value greater than `0`. If `max_process_per_link` is `0` or omitted, a validation error will be returned.
- **Behavior:** When `true`, it sets the system `x-goog-fieldmask` header to `"links,links.processes.process,unreachable"` to request full process details from the API.

## Example

```yaml
kind: tool
name: search_lineage
type: datalineage-search-lineage
source: my-lineage-source
description: Use this tool to search data lineage links for BigQuery tables.
```

## Output Format

The tool returns a structured JSON object containing the following fields:

- `links` (Array of Objects): A list of accumulated lineage links. Each object represents a lineage link containing details like `name`, `source` entity, `target` entity, `endTime`, `startTime`, and optionally associated `processes` (if process details were requested).
- `unreachable` (Array of Strings, Optional): A list of GCP locations that failed to respond during the multi-location search (e.g., `projects/123456789/locations/us-east1`). This field is omitted if there are no unreachable locations.

### Example Response

```json
{
  "links": [
    {
      "name": "projects/my-project/locations/us/links/link-id",
      "source": {
        "fullyQualifiedName": "bigquery:project.dataset.source_table"
      },
      "target": {
        "fullyQualifiedName": "bigquery:project.dataset.target_table"
      },
      "startTime": "2026-01-01T01:01:01.010Z",
      "endTime": "2026-01-01T01:01:01.010Z"
    }
  ],
  "unreachable": ["projects/my-project/locations/us-east1"]
}
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
| :---------- | :------: | :----------: | :------------------------------------------------- |
| type        |  string  |     true     | Must be "datalineage-search-lineage".              |
| source      |  string  |     true     | Name of the source the tool should execute on.     |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
