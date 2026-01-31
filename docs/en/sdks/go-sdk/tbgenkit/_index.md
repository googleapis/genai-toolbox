---
title: "tbGenkit Package"
linkTitle: "tbGenkit"
type: docs
weight: 1
---

![MCP Toolbox Logo](https://raw.githubusercontent.com/googleapis/genai-toolbox/main/logo.png)

# MCP Toolbox TBGenkit Package

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

This package allows you to seamlessly integrate the functionalities of
[Toolbox](https://github.com/googleapis/genai-toolbox) allowing you to load and
use tools defined in the service as standard Genkit Tools within your Genkit Go
applications.

This simplifies integrating external functionalities (like APIs, databases, or
custom logic) managed by the Toolbox into your workflows, especially those
involving Large Language Models (LLMs).


<!-- TOC ignore:true -->
<!-- TOC -->

- [Installation](#installation)
- [Quickstart](#quickstart)
- [Convert Toolbox Tool to a Genkit Tool](#convert-toolbox-tool-to-a-genkit-tool)
- [Contributing](#contributing)
- [License](#license)
- [Support](#support)
- [Samples for Reference](#samples-for-reference)
<!-- /TOC -->

## Installation

```bash
go get github.com/googleapis/mcp-toolbox-sdk-go
```
This SDK is supported on Go version 1.24.4 and higher.

## Quickstart

For more information on how to load a ToolboxTool, see [the core package](https://github.com/googleapis/mcp-toolbox-sdk-go/tree/main/core)

## Convert Toolbox Tool to a Genkit Tool

```go
"github.com/googleapis/mcp-toolbox-sdk-go/tbgenkit"

func main() {
  // Assuming the toolbox tool is loaded
  // Make sure to add error checks for debugging
  ctx := context.Background()
  g, err := genkit.Init(ctx)

  genkitTool, err := tbgenkit.ToGenkitTool(toolboxTool, g)

}
```

For end-to-end example on how to use Toolbox with Genkit Go, check out the [/samples/](https://github.com/googleapis/mcp-toolbox-sdk-go/tree/main/tbgenkit/samples) folder

# Contributing

Contributions are welcome! Please refer to the [DEVELOPER.md](/DEVELOPER.md)
file for guidelines on how to set up a development environment and run tests.

# License

This project is licensed under the Apache License 2.0. See the
[LICENSE](https://github.com/googleapis/mcp-toolbox-sdk-go/blob/main/LICENSE) file for details.

# Support

If you encounter issues or have questions, check the existing [GitHub Issues](https://github.com/googleapis/genai-toolbox/issues) for the main Toolbox project.

# Samples for Reference

<details>
<summary>Genkit Go</summary>

```go
//This sample contains a complete example on how to integrate MCP Toolbox Go SDK with Genkit Go using the tbgenkit package.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/googleapis/mcp-toolbox-sdk-go/core"
	"github.com/googleapis/mcp-toolbox-sdk-go/tbgenkit"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

func main() {
	ctx := context.Background()
	toolboxClient, err := core.NewToolboxClient("http://127.0.0.1:5000")
	if err != nil {
		log.Fatalf("Failed to create Toolbox client: %v", err)
	}

	// Load the tools using the MCP Toolbox SDK.
	tools, err := toolboxClient.LoadToolset("my-toolset", ctx)
	if err != nil {
		log.Fatalf("Failed to load tools: %v\nMake sure your Toolbox server is running and the tool is configured.", err)
	}

	// Initialize genkit
  g := genkit.Init(ctx,
		genkit.WithPlugins(&googlegenai.GoogleAI{}),
		genkit.WithDefaultModel("googleai/gemini-1.5-flash"),
	)

	// Convert your tool to a Genkit tool.
	genkitTools := make([]ai.Tool, len(tools))
	for i, tool := range tools {
		newTool, err := tbgenkit.ToGenkitTool(tool, g)
		if err != nil {
			log.Fatalf("Failed to convert tool: %v\n", err)
		}
		genkitTools[i] = newTool
	}

	toolRefs := make([]ai.ToolRef, len(genkitTools))

	for i, tool := range genkitTools {
		toolRefs[i] = tool
	}

	// Generate llm response using prompts and tools.
	resp, err := genkit.Generate(ctx, g,
		ai.WithPrompt("Find hotels in Basel with Basel in it's name."),
		ai.WithTools(toolRefs...),
	)
	if err != nil {
		log.Fatalf("%v\n", err)
	}
	fmt.Println(resp.Text())
}
```

</details>