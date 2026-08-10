---
title: "Database Insights Source"
linkTitle: "Source"
type: docs
weight: 1
description: "The \"databaseinsights\" source provides a client for the Google Cloud Database Insights API.\n"
no_list: true
---

## About

The `databaseinsights` source provides a client to interact with the Google Cloud Database Insights API (`databaseinsights.googleapis.com`). This source supports the Advanced Query Insights tools that are available via the remote [Database Insights MCP Server](https://docs.cloud.google.com/alloydb/docs/reference/mcp/databaseinsights/mcp), allowing tools to retrieve query performance statistics, wait event metrics, time-series trends, and index recommendations for Google Cloud databases like AlloyDB.

Authentication is handled via Application Default Credentials (ADC) using OAuth 2.0 access tokens.

## Available Tools

{{< list-tools >}}

## Example

```yaml
kind: source
name: database-insights-source
type: databaseinsights
---
kind: source
name: custom-project-database-insights
type: databaseinsights
project: my-quota-project
```

## Reference

| **field** | **type** | **required** | **description** |
| -------------- | :------: | :----------: | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| type | string | true | Must be "databaseinsights". |
| project | string | false | The Google Cloud project ID to use as a fallback billing/quota project for API calls if not extracted from the resource path. |
| endpoint | string | false | Override API endpoint URL (e.g., for staging or testing environments). Defaults to `https://databaseinsights.googleapis.com`. |
