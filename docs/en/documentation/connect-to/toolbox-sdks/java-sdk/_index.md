---
title: "Java"
type: docs
weight: 4
description: >
  Java SDKs to connect to the MCP Toolbox server.
---


## Overview

The MCP Toolbox service provides a centralized way to manage and expose tools
(like API connectors, database query tools, etc.) for use by GenAI applications.

These Java SDKs act as clients for that service. They handle the communication needed to:

* Fetch tool definitions from your running MCP Toolbox instance.
* Provide convenient Java objects or functions representing those tools.
* Invoke the tools (calling the underlying APIs/services configured in MCP Toolbox).
* Handle authentication and parameter binding as needed.

By using these SDKs, you can easily leverage your MCP Toolbox-managed tools directly
within your Java applications or AI orchestration frameworks.

## Getting Started

First make sure MCP Toolbox Server is set up and is running (either locally or deployed on Cloud Run). Follow the instructions here: [**MCP Toolbox Getting Started
    Guide**](https://mcp-toolbox.dev/documentation/introduction/#getting-started)

## Installation

This SDK is distributed via a [Maven Central Repository](https://mvnrepository.com/artifact/com.google.cloud.mcp/mcp-toolbox-sdk-java).

### Maven
Add the dependency to your `pom.xml`:
```xml
<dependency>
    <groupId>com.google.cloud.mcp</groupId>
    <artifactId>mcp-toolbox-sdk-java</artifactId>
    <!-- Replace 'VERSION' with the latest version from https://mvnrepository.com/artifact/com.google.cloud.mcp/mcp-toolbox-sdk-java -->
    <version>VERSION</version>
    <scope>compile</scope>
</dependency>
```

### Gradle

```
dependencies {
    // Replace 'VERSION' with the latest version from https://mvnrepository.com/artifact/com.google.cloud.mcp/mcp-toolbox-sdk-java
    implementation("com.google.cloud.mcp:mcp-toolbox-sdk-java:VERSION") 
}
```


{{< notice note >}}
Source code for [Java-sdk](https://github.com/googleapis/mcp-toolbox-sdk-java)
{{< /notice >}}
