---
title: "wait-for"
type: docs
weight: 1
description: > 
  A "wait-for" tool pauses execution for a specified duration.
aliases:
- /resources/tools/utility/wait-for
---

## About

A `wait-for` tool pauses execution for a specified duration. This can be useful in workflows where a delay is needed between steps.

`wait-for` takes one input parameter `duration` which is a string representing the time to wait (e.g., "10s", "2m", "1h").

> **Note:** This tool is intended for developer assistant workflows with
> human-in-the-loop and shouldn't be used for production agents.

## Example

```yaml
tools:
  wait_for_tool:
    kind: wait-for
    description: Use this tool to pause execution for a specified duration.
    timeout: 30s
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "wait-for".                                                                     |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
| timeout     |                   string                   |     true     | The default duration the tool can wait for.                                                      |
