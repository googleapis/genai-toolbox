---
title: "looker-make-look"
type: docs
weight: 1
description: >
  "looker-make-look" generates a Looker look in the users personal folder in
  Looker
aliases:
- /resources/tools/looker-make-look
---

## About

The `looker-make-look` creates a saved Look in the user's
Looker personal folder.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-make-look` takes eight parameters:

1. the `model`
2. the `explore`
3. the `fields` list
4. an optional set of `filters`
5. an optional set of `pivots`
6. an optional set of `sorts`
7. an optional `limit`
8. an optional `tz`
9. an optional `vis_config`
10. the `title`
11. an optional `description`

## Example

```yaml
tools:
    make_look:
        kind: looker-make-look
        source: looker-source
        description: |
          make_look Tool

          This tool creates a new look in Looker, using the query
          parameters and the vis_config specified.

          Most of the parameters are the same as the query_url
          tool. In addition, there is a title and a description
          that must be provided.

          The newly created look will be created in the user's
          personal folder in looker. The look name must be unique.

          The result is a json document with a link to the newly
          created look.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "looker-make-look"                                                                       |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
