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

		fmt.Println("\n", completion.Choices[0].Message.Content)

	}

}
