---
title: "Configuration"
type: docs
weight: 6
description: >
  How to configure Toolbox's tools.yaml file.
---

The primary way to configure Toolbox is through the `tools.yaml` file. If you
have multiple files, you can tell toolbox which to load with the `--tools-file
tools.yaml` flag.

You can find more detailed reference documentation to all resource types in the
[Resources](../resources/).

### Using Environment Variables

To avoid hardcoding certain secret fields like passwords, usernames, API keys
etc., you could use environment variables instead with the format `${ENV_NAME}`.

```yaml
  user: ${USER_NAME}
  password: ${PASSWORD}
```
A default value can be specified like `${ENV_NAME:default}`.

```yaml
  port: ${DB_PORT:3306}
```

### Sources

The `sources` section of your `tools.yaml` defines what data sources your
Toolbox should have access to. Most tools will have at least one source to
execute against.

```yaml
sources:
  my-pg-source:
    kind: postgres
    host: 127.0.0.1
    port: 5432
    database: toolbox_db
    user: ${USER_NAME}
    password: ${PASSWORD}
```

For more details on configuring different types of sources, see the
[Sources](../resources/sources/).

### Tools

The `tools` section of your `tools.yaml` defines the actions your agent can
take: what type of tool it is, which source(s) it affects, what parameters it
uses, etc.

```yaml
tools:
  search-hotels-by-name:
    kind: postgres-sql
    source: my-pg-source
    description: Search for hotels based on name.
    parameters:
      - name: name
        type: string
        description: The name of the hotel.
    statement: SELECT * FROM hotels WHERE name ILIKE '%' || $1 || '%';
```

For more details on configuring different types of tools, see the
[Tools](../resources/tools/).

### Toolsets

The `toolsets` section of your `tools.yaml` allows you to define groups of tools
that you want to be able to load together. This can be useful for defining
different sets for different agents or different applications.

```yaml
toolsets:
  my_first_toolset:
    - my_first_tool
    - my_second_tool
  my_second_toolset:
    - my_second_tool
    - my_third_tool
```

You can load toolsets by name:

```python
# This will load all tools
all_tools = client.load_toolset()

# This will only load the tools listed in 'my_second_toolset'
my_second_toolset = client.load_toolset("my_second_toolset")
```
