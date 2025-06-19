---
title: "YugabyteDB"
type: docs
weight: 1
description: >
  YugabyteDB is a high-performance, distributed SQL database. 
---

## About

[YugabyteDB][yugabytedb] is a high-performance, distributed SQL database designed for global, internet-scale applications, with full PostgreSQL compatibility.

[yugabytedb]: https://www.yugabyte.com/

## Example

```yaml
sources:
    my-yb-source:
        kind: yugabytedb
        host: 127.0.0.1
        port: 5433
        database: yugabyte
        user: ${USER_NAME}
        password: ${PASSWORD}
        load_balance: true
        topology_keys: cloud.region.zone1:1,cloud.region.zone2:2
```

## Reference

| **field**                         | **type** | **required** | **description**                                                        |
|-----------------------------------|:--------:|:------------:|------------------------------------------------------------------------|
| kind                              |  string  |     true     | Must be "yugabytedb".                                                    |
| host                              |  string  |     true     | IP address to connect to.                            |
| port                              |  integer  |     true     | Port to connect to.                                       |
| database                          |  string  |     true     | Name of the YugabyteDB database to connect to.            |
| user                              |  string  |     true     | Name of the YugabyteDB user to connect as.           |
| password                          |  string  |     true     | Password of the YugabyteDB user.                    |
| load_balance                      |  boolean  |     false     | If true, enable uniform load balancing.                    |
| topology_keys                     |  string  |     false     | Comma-separated geo-locations in the form cloud.region.zone:priority to enable topology-aware load balancing. Ignored if load_balance is false.                    |
| yb_servers_refresh_interval       |  integer  |     false     | The interval (in seconds) to refresh the servers list; ignored if load_balance is false                    |
| fallback_to_topology_keys_only    |  boolean  |     false     | If set to true and topology_keys are specified, only connect to nodes specified in topology_keys                    |
| failed_host_reconnect_delay_secs  |  integer  |     false     | Time (in seconds) to wait before trying to connect to failed nodes.                    |
