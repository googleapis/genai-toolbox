---
title: "redis"
type: docs
weight: 1
description: > 
  A "redis" tool executes a set of pre-defined Redis commands against a Redis instance.

---

## About

A redis tool executes a series of pre-defined Redis commands against a
Redis source.

The specified Redis commands are executed sequentially. Each command is
represented as a string list, where the first element is the command name (e.g.,
SET, GET, HGETALL) and subsequent elements are its arguments.

### Dynamic Command Parameters

Command arguments can be templated using the `$variableName` annotation. The
array type parameters will be expanded once into multiple arguments. Take the
following config for example:

```yaml
  commands:
      - [SADD, userNames, $userNames] # Array will be flattened into multiple arguments.
  parameters:
    - name: userNames
      type: array
      description: The user names to be set.  
```

If the input is an array of strings `["Alice", "Sid", "Bob"]`,  The final command
to be executed after argument expansion will be `[SADD, userNames, Alice, Sid, Bob]`.

## Example

```yaml
tools:
  user_data_tool:
    kind: redis
    source: my-redis-instance
    description: |
      Use this tool to interact with user data stored in Redis.
      It can set, retrieve, and delete user-specific information.
    commands:
      - [SADD, userNames, $userNames] # Array will be flattened into multiple arguments.
      - [GET, $userId]
    parameters:
      - name: userId
        type: string
        description: The unique identifier for the user.
      - name: userNames
        type: array
        description: The user names to be set.  
```
