---
title: "Data Lineage Source"
weight: 1
linkTitle: "Source"
---

## About

The Data Lineage integration allows the MCP Toolbox to connect to the Google Cloud Data Lineage API. It enables large language models to query and analyze data lineage, representing the flow of data between source (upstream) and target (downstream) assets.

This integration supports:

- **Entity-Level Lineage:** Tracking data flow between entire assets (e.g., tables, files).
- **Column-Level Lineage (CLL):** Tracking data flow between specific fields or columns within assets.

## Available Tools

{{< list-tools >}}

## Example

Here is an example configuration for the Data Lineage source:

```yaml
kind: source
name: my-lineage-source
type: datalineage
project: my-gcp-project-id
```

## Reference

| **field** | **type** | **required** | **description**                                                  |
| :-------- | :------: | :----------: | :--------------------------------------------------------------- |
| name      |  string  |     true     | Unique name for this source instance.                            |
| type      |  string  |     true     | Must be "datalineage".                                           |
| project   |  string  |     true     | The Google Cloud Project ID where the lineage events are stored. |
