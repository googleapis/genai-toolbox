---
title: "redis-execute-cmd"
type: docs
weight: 1
description: >
  A "redis-execute-cmd" tool executes a Redis command against a Redis
  instance.
aliases:
- /resources/tools/redis-execute-cmd
---

## About

A `redis-execute-cmd` tool executes a Redis command against a Redis
instance. It's compatible with any of the following sources:

- [redis](../../sources/redis.md)

`redis-execute-cmd` takes one input parameter `cmd` as an array of strings and runs the redis command with arguments against the `source`.

> **Note:** This tool is intended for developer assistant workflows with
> human-in-the-loop and shouldn't be used for production agents.

## Example

```yaml
tools:
 execute_cmd_tool:
    kind: redis-execute-cmd
    source: my-redis-instance
    description: Use this tool to execute a redis command.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "redis-execute-cmd".                                                                  |
| source      |                   string                   |     true     | Name of the source the command should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
