---
title: "Go SDK"
type: docs
weight: 7
description: >
  Go SDKs to connect to the MCP Toolbox server.
---


## Overview

![MCP Toolbox
Logo](https://raw.githubusercontent.com/googleapis/genai-toolbox/main/logo.png)

# MCP Toolbox SDKs for Go

[![License: Apache
2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Docs](https://img.shields.io/badge/Docs-MCP_Toolbox-blue)](https://googleapis.github.io/genai-toolbox/)
[![Discord](https://img.shields.io/badge/Discord-%235865F2.svg?style=flat&logo=discord&logoColor=white)](https://discord.gg/Dmm69peqjh)
[![Medium](https://img.shields.io/badge/Medium-12100E?style=flat&logo=medium&logoColor=white)](https://medium.com/@mcp_toolbox)
[![Go Report Card](https://goreportcard.com/badge/github.com/googleapis/mcp-toolbox-sdk-go)](https://goreportcard.com/report/github.com/googleapis/mcp-toolbox-sdk-go)
[![Module Version](https://img.shields.io/github/v/release/googleapis/mcp-toolbox-sdk-go)](https://img.shields.io/github/v/release/googleapis/mcp-toolbox-sdk-go)
[![Go Version](https://img.shields.io/github/go-mod/go-version/googleapis/mcp-toolbox-sdk-go)](https://img.shields.io/github/go-mod/go-version/googleapis/mcp-toolbox-sdk-go)

This repository contains the Go SDK designed to seamlessly integrate the
functionalities of the [MCP
Toolbox](https://github.com/googleapis/genai-toolbox) into your Gen AI
applications. The SDK allow you to load tools defined in Toolbox and use them
as standard Go tools within popular orchestration frameworks
or your custom code.

This simplifies the process of incorporating external functionalities (like
Databases or APIs) managed by Toolbox into your GenAI applications.

<!-- TOC -->

- [Overview](#overview)
- [Which Package Should I Use?](#which-package-should-i-use)
- [Available Packages](#available-packages)
- [Getting Started](#getting-started)

<!-- /TOC -->

## Overview

The MCP Toolbox service provides a centralized way to manage and expose tools
(like API connectors, database query tools, etc.) for use by GenAI applications.

The Go SDK act as clients for that service. They handle the communication needed to:

* Fetch tool definitions from your running Toolbox instance.
* Provide convenient Go structs representing those tools.
* Invoke the tools (calling the underlying APIs/services configured in Toolbox).
* Handle authentication and parameter binding as needed.

By using the SDK, you can easily leverage your Toolbox-managed tools directly
within your Go applications or AI orchestration frameworks.

## Which Package Should I Use?

Choosing the right package depends on how you are building your application:

- [**`core`**](core/):
  This is a framework-agnostic way to connect tools to popular frameworks
  like Google GenAI, LangChain, etc.

- [**`tbadk`**](tbadk/):
  This package provides a way to connect tools to ADK Go.

- [**`tbgenkit`**](tbgenkit/):
  This package provides functionality to convert the Tool fetched using the core package
  into a Genkit Go compatible tool.

## Available Packages

This repository hosts the following Go packages. See the package-specific
README for detailed installation and usage instructions:

| Package | Target Use Case | Integration | Path | Details (README) |
| :------ | :----------| :---------- | :---------------------- | :---------- |
| [`core`](core/) | Framework-agnostic / Custom applications | Use directly / Custom | `core/` | 📄 [View README](https://github.com/googleapis/mcp-toolbox-sdk-go/blob/main/core/README.md) |
| [`tbadk`](tbadk/) | ADK Go | Use directly | `tbadk/` | 📄 [View README](https://github.com/googleapis/mcp-toolbox-sdk-go/blob/main/tbadk/README.md) |
| [`tbgenkit`](tbgenkit/) | Genkit Go | Along with core | `tbgenkit/` | 📄 [View README](https://github.com/googleapis/mcp-toolbox-sdk-go/blob/main/tbgenkit/README.md) |

## Getting Started

To get started using Toolbox tools with an application, follow these general steps:

1. **Set up and Run the Toolbox Service:**

    Before using the SDKs, you need the MCP Toolbox server running. Follow
    the instructions here: [**Toolbox Getting Started
    Guide**](https://github.com/googleapis/genai-toolbox?tab=readme-ov-file#getting-started)

2. **Install the Appropriate SDK:**

    Choose the package based on your needs (see "[Which Package Should I Use?](#which-package-should-i-use)" above)
    Use this command to install the SDK module

    ```bash
    # For the core, framework-agnostic SDK
    go get github.com/googleapis/mcp-toolbox-sdk-go
    ```

3. **Use the SDK:**

    Consult the README for your chosen package (linked in the "[Available
    Packages](#available-packages)" section above) for detailed instructions on
    how to connect the client, load tool definitions, invoke tools, configure
    authentication/binding, and integrate them into your application or
    framework.

[Github](https://github.com/googleapis/mcp-toolbox-sdk-go)