---
title: "Genkit Integration"
linkTitle: "Go Genkit"
type: docs
weight: 1
---

This sample contains a complete example on how to integrate MCP Toolbox Go SDK with Genkit Go using the tbgenkit package.

```go
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