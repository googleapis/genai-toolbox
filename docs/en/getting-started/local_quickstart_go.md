---
title: "Go Quickstart (Local)"
type: docs
weight: 4
description: >
  How to get started running Toolbox locally with [Go](https://github.com/googleapis/mcp-toolbox-sdk-go), PostgreSQL, and orchestration frameworks such as [LangChain Go](https://tmc.github.io/langchaingo/docs/), [GenkitGo](https://genkit.dev/go/docs/get-started-go/), [Go GenAI](https://github.com/googleapis/go-genai) and [OpenAI Go](https://github.com/openai/openai-go).
---

## Before you begin

This guide assumes you have already done the following:

1. Installed [Go (v1.24.2 or higher)].
1. Installed [PostgreSQL 16+ and the `psql` client][install-postgres].

[Go (v1.24.2 or higher)]: https://go.dev/doc/install
[install-postgres]: https://www.postgresql.org/download/

### Cloud Setup (Optional)
{{< regionInclude "quickstart/shared/cloud_setup.md" "cloud_setup" >}}

## Step 1: Set up your database
{{< regionInclude "quickstart/shared/database_setup.md" "database_setup" >}}

## Step 2: Install and configure Toolbox
{{< regionInclude "quickstart/shared/configure_toolbox.md" "configure_toolbox" >}}

## Step 3: Connect your agent to Toolbox

In this section, we will write and run an agent that will load the Tools
from Toolbox.

1. Initialize a go module:

    ```bash
    go mod init main
    ```

1. In a new terminal, install the
   [SDK](https://pkg.go.dev/github.com/googleapis/mcp-toolbox-sdk-go).

    ```bash
    go get github.com/googleapis/mcp-toolbox-sdk-go
    ```

1. Create a new file named `hotelagent.go` and copy the following code to create
   an agent:

    {{< tabpane persist=header >}}
{{< tab header="LangChain Go" lang="go" >}}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/googleapis/mcp-toolbox-sdk-go/core"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/googleai"
)

// ConvertToLangchainTool converts a generic core.ToolboxTool into a LangChainGo llms.Tool.
func ConvertToLangchainTool(toolboxTool *core.ToolboxTool) llms.Tool {

	// Fetch the tool's input schema
	inputschema, err := toolboxTool.InputSchema()
	if err != nil {
		return llms.Tool{}
	}

	var paramsSchema map[string]any
	_ = json.Unmarshal(inputschema, &paramsSchema)

	// Convert into LangChain's llms.Tool
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        toolboxTool.Name(),
			Description: toolboxTool.Description(),
			Parameters:  paramsSchema,
		},
	}
}

const systemPrompt = `
You're a helpful hotel assistant. You handle hotel searching, booking, and
cancellations. When the user searches for a hotel, mention its name, id,
location and price tier. Always mention hotel ids while performing any
searches. This is very important for any operations. For any bookings or
cancellations, please provide the appropriate confirmation. Be sure to
update checkin or checkout dates if mentioned by the user.
Don't ask for confirmations from the user.
`

var queries = []string{
	"Find hotels in Basel with Basel in its name.",
	"Can you book the hotel Hilton Basel for me?",
	"Oh wait, this is too expensive. Please cancel it.",
	"Please book the Hyatt Regency instead.",
	"My check in dates would be from April 10, 2024 to April 19, 2024.",
}

func main() {
	genaiKey := os.Getenv("GOOGLE_API_KEY")
	toolboxURL := "http://localhost:5000"
	ctx := context.Background()

	// Initialize the Google AI client (LLM).
	llm, err := googleai.New(ctx, googleai.WithAPIKey(genaiKey), googleai.WithDefaultModel("gemini-1.5-flash"))
	if err != nil {
		log.Fatalf("Failed to create Google AI client: %v", err)
	}

	// Initialize the MCP Toolbox client.
	toolboxClient, err := core.NewToolboxClient(toolboxURL)
	if err != nil {
		log.Fatalf("Failed to create Toolbox client: %v", err)
	}

	// Load the tool using the MCP Toolbox SDK.
	tools, err := toolboxClient.LoadToolset("my-toolset", ctx)
	if err != nil {
		log.Fatalf("Failed to load tools: %v\nMake sure your Toolbox server is running and the tool is configured.", err)
	}

	toolsMap := make(map[string]*core.ToolboxTool, len(tools))

	langchainTools := make([]llms.Tool, len(tools))
	// Convert the loaded ToolboxTools into the format LangChainGo requires.
	for i, tool := range tools {
		langchainTools[i] = ConvertToLangchainTool(tool)
		toolsMap[tool.Name()] = tool
	}

	// Start the conversation history.
	messageHistory := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
	}

	for _, query := range queries {
		messageHistory = append(messageHistory, llms.TextParts(llms.ChatMessageTypeHuman, query))

		// Make the first call to the LLM, making it aware of the tool.
		resp, err := llm.GenerateContent(ctx, messageHistory, llms.WithTools(langchainTools))
		if err != nil {
			log.Fatalf("LLM call failed: %v", err)
		}
		respChoice := resp.Choices[0]

		assistantResponse := llms.TextParts(llms.ChatMessageTypeAI, respChoice.Content)
		for _, tc := range respChoice.ToolCalls {
			assistantResponse.Parts = append(assistantResponse.Parts, tc)
		}
		messageHistory = append(messageHistory, assistantResponse)

		// Process each tool call requested by the model.
		for _, tc := range respChoice.ToolCalls {
			toolName := tc.FunctionCall.Name
			tool := toolsMap[toolName]
			var args map[string]any
			if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
				log.Fatalf("Failed to unmarshal arguments for tool '%s': %v", toolName, err)
			}
			toolResult, err := tool.Invoke(ctx, args)
			if err != nil {
				log.Fatalf("Failed to execute tool '%s': %v", toolName, err)
			}
			if toolResult == "" || toolResult == nil {
				toolResult = "Operation completed successfully with no specific return value."
			}

			// Create the tool call response message and add it to the history.
			toolResponse := llms.MessageContent{
				Role: llms.ChatMessageTypeTool,
				Parts: []llms.ContentPart{
					llms.ToolCallResponse{
						Name:    toolName,
						Content: fmt.Sprintf("%v", toolResult),
					},
				},
			}
			messageHistory = append(messageHistory, toolResponse)
		}
		finalResp, err := llm.GenerateContent(ctx, messageHistory)
		if err != nil {
			log.Fatalf("Final LLM call failed after tool execution: %v", err)
		}

		// Add the final textual response from the LLM to the history
		messageHistory = append(messageHistory, llms.TextParts(llms.ChatMessageTypeAI, finalResp.Choices[0].Content))

		fmt.Println(finalResp.Choices[0].Content)

	}

}

{{< /tab >}}

{{< tab header="Genkit Go" lang="go" >}}

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

const systemPrompt = `
You're a helpful hotel assistant. You handle hotel searching, booking, and
cancellations. When the user searches for a hotel, mention its name, id,
location and price tier. Always mention hotel ids while performing any
searches. This is very important for any operations. For any bookings or
cancellations, please provide the appropriate confirmation. Be sure to
update checkin or checkout dates if mentioned by the user.
Don't ask for confirmations from the user.
`

var queries = []string{
	"Find hotels in Basel with Basel in its name.",
	"Can you book the hotel Hilton Basel for me?",
	"Oh wait, this is too expensive. Please cancel it and book the Hyatt Regency instead.",
	"My check in dates would be from April 10, 2024 to April 19, 2024.",
}

func main() {
	ctx := context.Background()

	// Create Toolbox Client
	toolboxClient, err := core.NewToolboxClient("http://127.0.0.1:5000")
	if err != nil {
		log.Fatalf("Failed to create Toolbox client: %v", err)
	}

	// Load the tools using the MCP Toolbox SDK.
	tools, err := toolboxClient.LoadToolset("my-toolset", ctx)
	if err != nil {
		log.Fatalf("Failed to load tools: %v\nMake sure your Toolbox server is running and the tool is configured.", err)
	}

	// Initialize Genkit
	g, err := genkit.Init(ctx,
		genkit.WithPlugins(&googlegenai.GoogleAI{}),
		genkit.WithDefaultModel("googleai/gemini-1.5-flash"),
	)
	if err != nil {
		log.Fatalf("Failed to init genkit: %v\n", err)
	}

	// Create a conversation history
	conversationHistory := []*ai.Message{
		ai.NewSystemTextMessage(systemPrompt),
	}

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

	for _, query := range queries {
		conversationHistory = append(conversationHistory, ai.NewUserTextMessage(query))
		response, err := genkit.Generate(ctx, g,
			ai.WithMessages(conversationHistory...),
			ai.WithTools(toolRefs...),
			ai.WithReturnToolRequests(true),
		)

		if err != nil {
			log.Fatalf("%v\n", err)
		}
		conversationHistory = append(conversationHistory, response.Message)

		parts := []*ai.Part{}

		for _, req := range response.ToolRequests() {
			tool := genkit.LookupTool(g, req.Name)
			if tool == nil {
				log.Fatalf("tool %q not found", req.Name)
			}

			output, err := tool.RunRaw(ctx, req.Input)
			if err != nil {
				log.Fatalf("tool %q execution failed: %v", tool.Name(), err)
			}

			parts = append(parts,
				ai.NewToolResponsePart(&ai.ToolResponse{
					Name:   req.Name,
					Ref:    req.Ref,
					Output: output,
				}))

		}

		if len(parts) > 0 {
			resp, err := genkit.Generate(ctx, g,
				ai.WithMessages(append(response.History(), ai.NewMessage(ai.RoleTool, nil, parts...))...),
				ai.WithTools(toolRefs...),
			)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("\n", resp.Text())
			conversationHistory = append(conversationHistory, resp.Message)
		} else {
			fmt.Println("\n", response.Text())
		}

	}

}

{{< /tab >}}

{{< tab header="Go GenAI" lang="go" >}}

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

	var paramsSchema *genai.Schema
	_ = json.Unmarshal(inputschema, &paramsSchema)
	// First, create the function declaration.
	funcDeclaration := &genai.FunctionDeclaration{
		Name:        toolboxTool.Name(),
		Description: toolboxTool.Description(),
		Parameters:  paramsSchema,
	}

	// Then, wrap the function declaration in a genai.Tool struct.
	return &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{funcDeclaration},
	}
}

func printResponse(resp *genai.GenerateContentResponse) {
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				fmt.Println(part.Text)
			}
		}
	}
}

const systemPrompt = `
You're a helpful hotel assistant. You handle hotel searching, booking, and
cancellations. When the user searches for a hotel, mention its name, id,
location and price tier. Always mention hotel ids while performing any
searches. This is very important for any operations. For any bookings or
cancellations, please provide the appropriate confirmation. Be sure to
update checkin or checkout dates if mentioned by the user.
Don't ask for confirmations from the user.
`

var queries = []string{
	"Find hotels in Basel with Basel in its name.",
	"Can you book the hotel Hilton Basel for me?",
	"Oh wait, this is too expensive. Please cancel it.",
	"Please book the Hyatt Regency instead.",
	"My check in dates would be from April 10, 2024 to April 19, 2024.",
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

	// Load the tool using the MCP Toolbox SDK.
	tools, err := toolboxClient.LoadToolset("my-toolset", ctx)
	if err != nil {
		log.Fatalf("Failed to load tools: %v\nMake sure your Toolbox server is running and the tool is configured.", err)
	}

	genAITools := make([]*genai.Tool, len(tools))
	toolsMap := make(map[string]*core.ToolboxTool, len(tools))

	for i, tool := range tools {
		genAITools[i] = ConvertToGenaiTool(tool)
		toolsMap[tool.Name()] = tool
	}

	// Set up the generative model with the available tool.
	modelName := "gemini-2.0-flash"

	// Create the initial content prompt for the model.
	messageHistory := []*genai.Content{
		genai.NewContentFromText(systemPrompt, genai.RoleUser),
	}
	config := &genai.GenerateContentConfig{
		Tools: genAITools,
		ToolConfig: &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				Mode: genai.FunctionCallingConfigModeAny,
			},
		},
	}

	for _, query := range queries {

		messageHistory = append(messageHistory, genai.NewContentFromText(query, genai.RoleUser))

		genContentResp, err := client.Models.GenerateContent(ctx, modelName, messageHistory, config)
		if err != nil {
			log.Fatalf("LLM call failed for query '%s': %v", query, err)
		}

		if len(genContentResp.Candidates) > 0 && genContentResp.Candidates[0].Content != nil {
			messageHistory = append(messageHistory, genContentResp.Candidates[0].Content)
		}

		functionCalls := genContentResp.FunctionCalls()

		toolResponseParts := []*genai.Part{}

		for _, fc := range functionCalls {

			toolToInvoke, found := toolsMap[fc.Name]
			if !found {
				log.Fatalf("Tool '%s' not found in loaded tools map. Check toolset configuration.", fc.Name)
			}

			toolResult, invokeErr := toolToInvoke.Invoke(ctx, fc.Args)
			if invokeErr != nil {
				log.Fatalf("Failed to execute tool '%s': %v", fc.Name, invokeErr)
			}

			// Enhanced Tool Result Handling (retained to prevent nil issues)
			toolResultString := ""
			if toolResult != nil {
				jsonBytes, marshalErr := json.Marshal(toolResult)
				if marshalErr == nil {
					toolResultString = string(jsonBytes)
				} else {
					toolResultString = fmt.Sprintf("%v", toolResult)
				}
			}

			responseMap := map[string]any{"result": toolResultString}

			toolResponseParts = append(toolResponseParts, genai.NewPartFromFunctionResponse(fc.Name, responseMap))
		}
		// Add all accumulated tool responses for this turn to the message history.
		toolResponseContent := genai.NewContentFromParts(toolResponseParts, "function")
		messageHistory = append(messageHistory, toolResponseContent)

		finalResponse, err := client.Models.GenerateContent(ctx, modelName, messageHistory, &genai.GenerateContentConfig{})
		if err != nil {
			log.Fatalf("Error calling GenerateContent (with function result): %v", err)
		}

		printResponse(finalResponse)
		// Add the final textual response from the LLM to the history
		if len(finalResponse.Candidates) > 0 && finalResponse.Candidates[0].Content != nil {
			messageHistory = append(messageHistory, finalResponse.Candidates[0].Content)
		}
	}
}

{{< /tab >}}

{{< tab header="OpenAI Go" lang="go" >}}

package main

import (
	"context"
	"encoding/json"
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

const systemPrompt = `
You're a helpful hotel assistant. You handle hotel searching, booking, and
cancellations. When the user searches for a hotel, mention its name, id,
location and price tier. Always mention hotel ids while performing any
searches. This is very important for any operations. For any bookings or
cancellations, please provide the appropriate confirmation. Be sure to
update checkin or checkout dates if mentioned by the user.
Don't ask for confirmations from the user.
`

var queries = []string{
	"Find hotels in Basel with Basel in its name.",
	"Can you book the hotel Hilton Basel for me?",
	"Oh wait, this is too expensive. Please cancel it and book the Hyatt Regency instead.",
	"My check in dates would be from April 10, 2024 to April 19, 2024.",
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

	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
		},
		Tools: openAITools,
		Seed:  openai.Int(0),
		Model: openai.ChatModelGPT4o,
	}

	for _, query := range queries {

		params.Messages = append(params.Messages, openai.UserMessage(query))

		// Make initial chat completion request
		completion, err := openAIClient.Chat.Completions.New(ctx, params)
		if err != nil {
			panic(err)
		}

		toolCalls := completion.Choices[0].Message.ToolCalls

		// Return early if there are no tool calls
		if len(toolCalls) == 0 {
			log.Println("No function call")
		}

		// If there was a function call, continue the conversation
		params.Messages = append(params.Messages, completion.Choices[0].Message.ToParam())
		for _, toolCall := range toolCalls {

			toolName := toolCall.Function.Name
			toolToInvoke := toolsMap[toolName]

			var args map[string]any
			err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
			if err != nil {
				panic(err)
			}

			result, err := toolToInvoke.Invoke(ctx, args)
			if err != nil {
				log.Fatal("Could not invoke tool", err)
			}

			params.Messages = append(params.Messages, openai.ToolMessage(result.(string), toolCall.ID))
		}

		completion, err = openAIClient.Chat.Completions.New(ctx, params)
		if err != nil {
			panic(err)
		}

		params.Messages = append(params.Messages, openai.AssistantMessage(query))

		println("\n", completion.Choices[0].Message.Content)

	}

}

{{< /tab >}}
{{< /tabpane >}}

1. Ensure all dependencies are installed:

    ```sh
    go mod tidy
    ```

1. Run your agent, and observe the results:

    ```sh
    go run hotelagent.go
    ```

{{< notice info >}}
For more information, visit the [Go SDK
repo](https://github.com/googleapis/mcp-toolbox-sdk-go).
{{</ notice >}}
