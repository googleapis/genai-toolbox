---
title: "Sources"
type: docs
weight: 1
description: > 
  Sources represent your different data sources that a tool can interact with.
---

A Source represents a data sources that a tool can interact with. You can define
Sources as a map in the `sources` section of your `tools.yaml` file. Typically,
a source configuration will contain any information needed to connect with and
interact with the database.

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

```yaml
kind: sources
name: my-cloud-sql-source
type: cloud-sql-postgres
project: my-project-id
region: us-central1
instance: my-instance-name
database: my_db
user: ${USER_NAME}
password: ${PASSWORD}
```

In implementation, each source is a different connection pool or client that used
to connect to the database and execute the tool.

## Available Sources
