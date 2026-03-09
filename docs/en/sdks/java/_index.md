---
title: "Java"
type: docs
weight: 4
description: >
  JAVA SDKs to connect to the MCP Toolbox server.
---


## Overview

The MCP Toolbox service provides a centralized way to manage and expose tools
(like API connectors, database query tools, etc.) for use by GenAI applications.

These JAVA SDKs act as clients for that service. They handle the communication needed to:

* Fetch tool definitions from your running Toolbox instance.
* Provide convenient Java objects or functions representing those tools.
* Invoke the tools (calling the underlying APIs/services configured in Toolbox).
* Handle authentication and parameter binding as needed.

By using these SDKs, you can easily leverage your Toolbox-managed tools directly
within your JAVA applications or AI orchestration frameworks.

## Getting Started

First make sure Toolbox Server is set up and is running (either locally or deployed on Cloud Run). Follow the instructions here: [**Toolbox Getting Started
    Guide**](https://github.com/googleapis/genai-toolbox?tab=readme-ov-file#getting-started)

## Installation

This SDK is distributed via a Maven Central Repository.

### Maven
Add the dependency to your `pom.xml`:
```xml
<!-- Source: https://mvnrepository.com/artifact/com.google.cloud.mcp/mcp-toolbox-sdk-java -->
<dependency>
    <groupId>com.google.cloud.mcp</groupId>
    <artifactId>mcp-toolbox-sdk-java</artifactId>
    <version>0.2.0</version> <!-- {x-version-update:mcp-toolbox-sdk-java:current} -->
    <scope>compile</scope>
</dependency>
```

### Gradle

```
dependencies {
    // Source: https://mvnrepository.com/artifact/com.google.cloud.mcp/mcp-toolbox-sdk-java
    implementation("com.google.cloud.mcp:mcp-toolbox-sdk-java:0.2.0") 
}
```


{{< notice note >}}
Source code for [JAVA-sdk](https://github.com/googleapis/mcp-toolbox-sdk-java)
{{< /notice >}}
