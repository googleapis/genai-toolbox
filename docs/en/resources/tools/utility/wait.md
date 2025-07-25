---
title: "wait"
type: docs
weight: 1
description: > 
  A "wait" tool pauses execution for a specified duration.
aliases:
- /resources/tools/utility/wait
---

## About

A `wait` tool pauses execution for a specified duration. This can be useful in workflows where a delay is needed between steps.

`wait` takes one input parameter `duration` which is a string representing the time to wait (e.g., "10s", "2m", "1h").

{{< notice info >}}
This tool is intended for developer assistant workflows with human-in-the-loop and shouldn't be used for production agents.
{{< /notice >}}

## Example

```yaml
tools:
  wait_for_tool:
    kind: wait
    description: Use this tool to pause execution for a specified duration.
    timeout: 30s
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "wait".                                                                     |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
| timeout     |                   string                   |     true     | The default duration the tool can wait for.                                                      |
