---
title: "spanner-execute-ddl"
type: docs
weight: 3
description: >
  Execute DDL statements on Spanner.
---

# spanner-execute-ddl

The `spanner-execute-ddl` tool allows you to execute Data Definition Language (DDL) statements on a Spanner database. This tool is useful for managing your database schema directly from the MCP toolbox.

## Requirements

This tool requires the `gcloud` command-line tool to be installed and authenticated. If `gcloud` is not found in your system's PATH, the tool will return an error.

## Example

```yaml
tools:
  my-spanner-ddl-tool:
    kind: spanner-execute-ddl
    source: my-spanner-source
    description: "A tool to execute DDL statements on my Spanner database."
```

## Reference

| **field** | **type** | **required** | **description** |
|-----------|:--------:|:------------:|-----------------|
| kind | string | true | Must be "spanner-execute-ddl". |
| source | string | true | The name of the Spanner source to use. |
| description | string | true | A description of the tool. |
