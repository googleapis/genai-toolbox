---
title: "OpenAI Integration"
linkTitle: "Go OpenAI"
type: docs
weight: 1
---

This sample contains a complete example on how to integrate MCP Toolbox Go Core SDK with the OpenAI Go SDK.

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/googleapis/mcp-toolbox-sdk-go/core"
	openai "github.com/openai/openai-go"
)

// ConvertToOpenAITool converts a ToolboxTool into the go-openai library's Tool format.
func ConvertToOpenAITool(toolboxTool *core.ToolboxTool) openai.ChatCompletionToolParam {
	// Get the input schema
	jsonSchemaBytes, err := toolboxTool.InputSchema()
	if err != nil {
		return openai.ChatCompletionToolParam{}
	}

	// Unmarshal the JSON bytes into FunctionParameters
	var paramsSchema openai.FunctionParameters
	if err := json.Unmarshal(jsonSchemaBytes, &paramsSchema); err != nil {
		return openai.ChatCompletionToolParam{}
	}

	// Create and return the final tool parameter struct.
	return openai.ChatCompletionToolParam{
		Function: openai.FunctionDefinitionParam{
			Name:        toolboxTool.Name(),
			Description: openai.String(toolboxTool.Description()),
			Parameters:  paramsSchema,
		},
	}
}

func main() {
	// Setup
	ctx := context.Background()
	toolboxURL := "http://localhost:5000"
	openAIClient := openai.NewClient()

	// Initialize the MCP Toolbox client.
	toolboxClient, err := core.NewToolboxClient(toolboxURL)
	if err != nil {
		log.Fatalf("Failed to create Toolbox client: %v", err)
	}

	// Load the tools using the MCP Toolbox SDK.
	tools, err := toolboxClient.LoadToolset("my-toolset", ctx)
	if err != nil {
		log.Fatalf("Failed to load tool : %v\nMake sure your Toolbox server is running and the tool is configured.", err)
	}

	openAITools := make([]openai.ChatCompletionToolParam, len(tools))
	toolsMap := make(map[string]*core.ToolboxTool, len(tools))

	for i, tool := range tools {
		// Convert the Toolbox tool into the openAI FunctionDeclaration format.
		openAITools[i] = ConvertToOpenAITool(tool)
		// Add tool to a map for lookup later
		toolsMap[tool.Name()] = tool

	}
	question := "Find hotels in Basel with Basel in it's name "

	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(question),
		},
		Tools: openAITools,
		Seed:  openai.Int(0),
		Model: openai.ChatModelGPT4o,
	}

	// Make initial chat completion request
	completion, err := openAIClient.Chat.Completions.New(ctx, params)
	if err != nil {
		panic(err)
	}

	toolCalls := completion.Choices[0].Message.ToolCalls

	// Return early if there are no tool calls
	if len(toolCalls) == 0 {
		fmt.Printf("No function call")
		return
	}

	// If there is a was a function call, continue the conversation
	params.Messages = append(params.Messages, completion.Choices[0].Message.ToParam())
	for _, toolCall := range toolCalls {
		if toolCall.Function.Name == "search-hotels-by-name" {
			// Extract the location from the function call arguments
			var args map[string]interface{}
			tool := toolsMap["search-hotels-by-name"]
			err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
			if err != nil {
				panic(err)
			}

			result, err := tool.Invoke(ctx, args)
			if err != nil {
				log.Fatal("Could not invoke tool", err)
			}

			params.Messages = append(params.Messages, openai.ToolMessage(result.(string), toolCall.ID))
		}
	}

	completion, err = openAIClient.Chat.Completions.New(ctx, params)
	if err != nil {
		panic(err)
	}

	fmt.Println(completion.Choices[0].Message.Content)
}
```