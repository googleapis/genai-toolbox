---
title: "GenAI Integration"
linkTitle: "Go GenAI"
type: docs
weight: 1
---

This sample contains a complete example on how to integrate MCP Toolbox Go Core SDK with the Go GenAI SDK.

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/googleapis/mcp-toolbox-sdk-go/core"
	"google.golang.org/genai"
)

// ConvertToGenaiTool translates a ToolboxTool into the genai.FunctionDeclaration format.
func ConvertToGenaiTool(toolboxTool *core.ToolboxTool) *genai.Tool {

	inputschema, err := toolboxTool.InputSchema()
	if err != nil {
		return &genai.Tool{}
	}

	var schema *genai.Schema
	_ = json.Unmarshal(inputschema, &schema)
	// First, create the function declaration.
	funcDeclaration := &genai.FunctionDeclaration{
		Name:        toolboxTool.Name(),
		Description: toolboxTool.Description(),
		Parameters:  schema,
	}

	// Then, wrap the function declaration in a genai.Tool struct.
	return &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{funcDeclaration},
	}
}

// printResponse extracts and prints the relevant parts of the model's response.
func printResponse(resp *genai.GenerateContentResponse) {
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				fmt.Println(part.Text)
			}
		}
	}
}

func main() {
	// Setup
	ctx := context.Background()
	apiKey := os.Getenv("GOOGLE_API_KEY")
	toolboxURL := "http://localhost:5000"

	// Initialize the Google GenAI client using the explicit ClientConfig.
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		log.Fatalf("Failed to create Google GenAI client: %v", err)
	}

	// Initialize the MCP Toolbox client.
	toolboxClient, err := core.NewToolboxClient(toolboxURL)
	if err != nil {
		log.Fatalf("Failed to create Toolbox client: %v", err)
	}

	// Load the tools using the MCP Toolbox SDK.
	tools, err := toolboxClient.LoadToolset("my-toolset", ctx)
	if err != nil {
		log.Fatalf("Failed to load tools: %v\nMake sure your Toolbox server is running and the tool is configured.", err)
	}

  genAITools := make([]*genai.Tool, len(tools))
	toolsMap := make(map[string]*core.ToolboxTool, len(tools))

	for i, tool := range tools {
    // Convert the tools into usable format
		genAITools[i] = ConvertToGenaiTool(tool)
    // Add tool to a map for lookup later
		toolsMap[tool.Name()] = tool
	}

	// Set up the generative model with the available tool.
	modelName := "gemini-2.0-flash"

	query := "Find hotels in Basel with Basel in it's name and share the names with me"

	// Create the initial content prompt for the model.
	contents := []*genai.Content{
		genai.NewContentFromText(query, genai.RoleUser),
	}
	config := &genai.GenerateContentConfig{
		Tools: genAITools,
		ToolConfig: &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				Mode: genai.FunctionCallingConfigModeAny,
			},
		},
	}
	genContentResp, _ := client.Models.GenerateContent(ctx, modelName, contents, config)

	printResponse(genContentResp)

	functionCalls := genContentResp.FunctionCalls()
	if len(functionCalls) == 0 {
		log.Println("No function call returned by the AI. The model likely answered directly.")
		return
	}

	// Process the first function call (the example assumes one for simplicity).
	fc := functionCalls[0]
	log.Printf("--- Gemini requested function call: %s ---\n", fc.Name)
	log.Printf("--- Arguments: %+v ---\n", fc.Args)

	var toolResultString string

	if fc.Name == "search-hotels-by-name" {
		tool := toolsMap["search-hotels-by-name"]
		toolResult, err := tool.Invoke(ctx, fc.Args)
		toolResultString = fmt.Sprintf("%v", toolResult)
		if err != nil {
			log.Fatalf("Failed to execute tool '%s': %v", fc.Name, err)
		}

	} else {
		log.Println("LLM did not request our tool")
	}
	resultContents := []*genai.Content{
		genai.NewContentFromText("The tool returned this result, share it with the user based of their previous querys"+toolResultString, genai.RoleUser),
	}
	finalResponse, err := client.Models.GenerateContent(ctx, modelName, resultContents, &genai.GenerateContentConfig{})
	if err != nil {
		log.Fatalf("Error calling GenerateContent (with function result): %v", err)
	}
	log.Println("=== Final Response from Model (after processing function result) ===")
	printResponse(finalResponse)

}
```