---
title: "ADK Integration"
linkTitle: "Go ADK"
type: docs
weight: 1
---

This sample contains a complete example on how to integrate MCP Toolbox Go SDK with ADK Go using the tbadk package.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/googleapis/mcp-toolbox-sdk-go/tbadk"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

func main() {
	genaiKey := os.Getenv("GEMINI_API_KEY")
	toolboxURL := "http://localhost:5000"
	ctx := context.Background()

	// Initialize MCP Toolbox client
	toolboxClient, err := tbadk.NewToolboxClient(toolboxURL)
	if err != nil {
		log.Fatalf("Failed to create MCP Toolbox client: %v", err)
	}

	toolsetName := "my-toolset"
	toolset, err := toolboxClient.LoadToolset(toolsetName, ctx)
	if err != nil {
		log.Fatalf("Failed to load MCP toolset '%s': %v\nMake sure your Toolbox server is running.", toolsetName, err)
	}

	// Create Gemini model
	model, err := gemini.NewModel(ctx, "gemini-2.5-flash", &genai.ClientConfig{
		APIKey: genaiKey,
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	tools := make([]tool.Tool, len(toolset))
	for i := range toolset {
		tools[i] = &toolset[i]
	}

	llmagent, err := llmagent.New(llmagent.Config{
		Name:        "hotel_assistant",
		Model:       model,
		Description: "Agent to answer questions about hotels.",
		Tools:       tools,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	appName := "hotel_assistant"
	userID := "user-123"

	sessionService := session.InMemoryService()
	resp, err := sessionService.Create(ctx, &session.CreateRequest{
		AppName: appName,
		UserID:  userID,
	})
	if err != nil {
		log.Fatalf("Failed to create the session service: %v", err)
	}
	session := resp.Session

	r, err := runner.New(runner.Config{
		AppName:        appName,
		Agent:          llmagent,
		SessionService: sessionService,
	})
	if err != nil {
		log.Fatalf("Failed to create runner: %v", err)
	}

	query := "Find hotels with Basel in its name."

	fmt.Println(query)
	userMsg := genai.NewContentFromText(query, genai.RoleUser)

	streamingMode := agent.StreamingModeSSE
	for event, err := range r.Run(ctx, userID, session.ID(), userMsg, agent.RunConfig{
		StreamingMode: streamingMode,
	}) {
		if err != nil {
			fmt.Printf("\nAGENT_ERROR: %v\n", err)
		} else {
			if event.LLMResponse.Content != nil {
				for _, p := range event.LLMResponse.Content.Parts {
					if streamingMode != agent.StreamingModeSSE || event.LLMResponse.Partial {
						fmt.Print(p.Text)
					}
				}
			}
		}
	}
	fmt.Println()
}
```