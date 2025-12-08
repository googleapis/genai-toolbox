---
title: "looker-generate-embed-url"
type: docs
weight: 1
description: >
  "looker-generate-embed-url" generates an embeddable URL for Looker content.
aliases:
- /resources/tools/looker-generate-embed-url
---

## About

The `looker-generate-embed-url` tool generates an embeddable URL for a given
piece of Looker content. The url generated is created for the user authenticated
to the Looker source. When opened in the browser it will create a Looker Embed
session.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-generate-embed-url` takes two parameters:

1. the `type` of content (e.g., "dashboards", "looks", "query-visualization")
2. the `id` of the content

It's recommended to use other tools from the Looker MCP toolbox with this tool
to do things like fetch dashboard id's, generate a query, etc that can be
supplied to this tool.

## Example

```yaml
tools:
    generate_embed_url:
        kind: looker-generate-embed-url
        source: looker-source
        description: |
          generate_embed_url Tool

          This tool generates a signed, private embed URL for specific Looker content,
          allowing users to access it directly.

          Parameters:
          - type (required): The type of content to embed. Common values include:
            - `dashboards`
            - `looks`
            - `explore`
          - id (required): The unique identifier for the content.
            - For dashboards and looks, use the numeric ID (e.g., "123").
            - For explores, use the format "model_name/explore_name".    generate_embed_url:
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-generate-embed-url"                |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
