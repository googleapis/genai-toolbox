---
title: "YugabyteDB"
type: docs
weight: 1
description: >
  YugabyteDB is a high-performance, distributed SQL database. 
---

## About

[YugabyteDB][yugabytedb] is a high-performance, distributed SQL database
designed for global, internet-scale applications, with full PostgreSQL
compatibility.

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
        loadBalance: true
        topologyKeys: cloud.region.zone1:1,cloud.region.zone2:2
```

## Reference

| **field**                    | **type** | **required** | **description**                                                                                                                                                       |
|------------------------------|:--------:|:------------:|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| kind                         |  string  |     true     | Must be "yugabytedb".                                                                                                                                                 |
| host                         |  string  |     true     | IP address to connect to.                                                                                                                                             |
| port                         | integer  |     true     | Port to connect to. The default port is 5433.                                                                                                                         |
| database                     |  string  |     true     | Name of the YugabyteDB database to connect to. The default database name is yugabyte.                                                                                 |
| user                         |  string  |     true     | Name of the YugabyteDB user to connect as. The default user is yugabyte.                                                                                              |
| password                     |  string  |     true     | Password of the YugabyteDB user. The default password is yugabyte.                                                                                                    |
| loadBalance                  | boolean  |    false     | If true, enable uniform load balancing. The default loadBalance value is false.                                                                                       |
| topologyKeys                 |  string  |    false     | Comma-separated geo-locations in the form cloud.region.zone:priority to enable topology-aware load balancing. Ignored if loadBalance is false. It is null by default. |
| ybServersRefreshInterval     | integer  |    false     | The interval (in seconds) to refresh the servers list; ignored if loadBalance is false. The default value of ybServersRefreshInterval is 300.                         |
| fallbackToTopologyKeysOnly   | boolean  |    false     | If set to true and topologyKeys are specified, only connect to nodes specified in topologyKeys. By defualt, this is set to false.                                     |
| failedHostReconnectDelaySecs | integer  |    false     | Time (in seconds) to wait before trying to connect to failed nodes. The default value of is 5.                                                                        |
