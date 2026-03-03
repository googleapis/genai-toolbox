---
title: "Core Package"
linkTitle: "Core"
type: docs
weight: 2
description: >
  MCP Toolbox Core SDK for integrating functionalities of MCP Toolbox into your Agentic apps.
---

## Overview

The `core` package provides a Go interface to the MCP Toolbox service, enabling you to load and invoke tools from your own applications.

## Installation

```bash
go get github.com/googleapis/mcp-toolbox-sdk-go/core
```
This SDK is supported on Go version 1.24.4 and higher.

{{< notice note >}}
While the SDK itself is synchronous, you can execute its functions within goroutines to achieve asynchronous behavior.
{{< /notice >}}

{{< notice note >}}
**Breaking Change Notice**: As of version `0.6.0`, this repository has transitioned to a multi-module structure.
*   **For new versions (`v0.6.0`+)**: You must import specific modules (e.g., `go get github.com/googleapis/mcp-toolbox-sdk-go/core`).
*   **For older versions (`v0.5.1` and below)**: The repository remains a single-module library (`go get github.com/googleapis/mcp-toolbox-sdk-go`).
*   Please update your imports and `go.mod` accordingly when upgrading.
{{< /notice >}}

## Quickstart

Here's a minimal example to get you started. Ensure your Toolbox service is
running and accessible.

```go
package main

import (
	"context"
	"fmt"
	"github.com/googleapis/mcp-toolbox-sdk-go/core"
)

func quickstart() string {
	ctx := context.Background()
	inputs := map[string]any{"location": "London"}
	client, err := core.NewToolboxClient("http://localhost:5000")
	if err != nil {
		return fmt.Sprintln("Could not start Toolbox Client", err)
	}
	tool, err := client.LoadTool("get_weather", ctx)
	if err != nil {
		return fmt.Sprintln("Could not load Toolbox Tool", err)
	}
	result, err := tool.Invoke(ctx, inputs)
	if err != nil {
		return fmt.Sprintln("Could not invoke tool", err)
	}
	return fmt.Sprintln(result)
}

func main() {
	fmt.Println(quickstart())
}
```

## Usage

Import and initialize a Toolbox client, pointing it to the URL of your running
Toolbox service.

```go
import "github.com/googleapis/mcp-toolbox-sdk-go/core"

client, err := core.NewToolboxClient("http://localhost:5000")
```

All interactions for loading and invoking tools happen through this client.

{{< notice note >}} 
For advanced use cases, you can provide an external custom `http.Client` during initialization (e.g., `core.NewToolboxClient(URL, core.WithHTTPClient(myClient)`). 
If you provide your own session, you are responsible for managing its lifecycle; `ToolboxClient` *will not* close it.
{{< /notice >}}

{{< notice info >}}
Closing the `ToolboxClient` also closes the underlying network session shared by all tools loaded from that client. As a result, any tool instances you have loaded will cease to function and will raise an error if you attempt to invoke them after the client is closed.
{{< /notice >}}

## Transport Protocols

The SDK supports multiple transport protocols for communicating with the Toolbox server. By default, the client uses the latest supported version of the **Model Context Protocol (MCP)**.

You can explicitly select a protocol using the `core.WithProtocol` option during client initialization. This is useful if you need to use the native Toolbox HTTP protocol or pin the client to a specific legacy version of MCP.

{{< notice note >}}
* **Native Toolbox Transport**: This uses the service's native **REST over HTTP** API.
* **MCP Transports**: These options use the **Model Context Protocol over HTTP**.
{{< /notice >}}

### Supported Protocols

{{< notice note >}}
The native Toolbox protocol (`core.Toolbox`) is deprecated and will be removed on March 4, 2026. Please use `core.MCP` or specific MCP versions.
{{< /notice >}}

| Constant | Description |
| :--- | :--- |
| `core.MCP` | **(Default)** Alias for the latest supported MCP version (currently `v2025-06-18`). |
| `core.Toolbox` |  **Deprecated** The native Toolbox HTTP protocol. |
| `core.MCPv20251125` | MCP Protocol version 2025-11-25. |
| `core.MCPv20250618` | MCP Protocol version 2025-06-18. |
| `core.MCPv20250326` | MCP Protocol version 2025-03-26. |
| `core.MCPv20241105` | MCP Protocol version 2024-11-05. |

### Example

If you wish to use the native Toolbox protocol:

```go
import "github.com/googleapis/mcp-toolbox-sdk-go/core"

client, err := core.NewToolboxClient(
    "http://localhost:5000",
    core.WithProtocol(core.Toolbox),
)
```
If you want to pin the MCP Version 2025-03-26:

```go
import "github.com/googleapis/mcp-toolbox-sdk-go/core"

client, err := core.NewToolboxClient(
    "http://localhost:5000",
    core.WithProtocol(core.MCPv20250326),
)
```

## Loading Tools

You can load tools individually or in groups (toolsets) as defined in your
Toolbox service configuration. Loading a toolset is convenient when working with
multiple related functions, while loading a single tool offers more granular
control.

### Load a toolset

A toolset is a collection of related tools. You can load all tools in a toolset
or a specific one:

```go
// Load default toolset by providing an empty string as the name
tools, err := client.LoadToolset("", ctx)

// Load a specific toolset
tools, err := client.LoadToolset("my-toolset", ctx)
```


### Load a single tool

Loads a specific tool by its unique name. This provides fine-grained control.

```go
tool, err = client.LoadTool("my-tool", ctx)
```

## Invoking Tools

Once loaded, tools behave like Go structs. You invoke them using `Invoke` method
by passing arguments corresponding to the parameters defined in the tool's
configuration within the Toolbox service.

```go
tool, err = client.LoadTool("my-tool", ctx)
inputs := map[string]any{"location": "London"}
result, err := tool.Invoke(ctx, inputs)
```

{{< notice tip >}}
For a more comprehensive guide on setting up the Toolbox service itself, which you'll need running to use this SDK, please refer to the [Toolbox Quickstart Guide](https://googleapis.github.io/genai-toolbox/getting-started/local_quickstart).
{{< /notice >}}

## Client to Server Authentication

This section describes how to authenticate the ToolboxClient itself when
connecting to a Toolbox server instance that requires authentication. This is
crucial for securing your Toolbox server endpoint, especially when deployed on
platforms like Cloud Run, GKE,  or any environment where unauthenticated access is restricted.

This client-to-server authentication ensures that the Toolbox server can verify
the identity of the client making the request before any tool is loaded or
called. It is different from [Authenticating Tools](#authenticating-tools),
which deals with providing credentials for specific tools within an already
connected Toolbox session.

### When is Client-to-Server Authentication Needed?

You'll need this type of authentication if your Toolbox server is configured to
deny unauthenticated requests. For example:

- Your Toolbox server is deployed on Cloud Run and configured to "Require authentication."
- Your server is behind an Identity-Aware Proxy (IAP) or a similar
  authentication layer.
- You have custom authentication middleware on your self-hosted Toolbox server.

Without proper client authentication in these scenarios, attempts to connect or
make calls (like `LoadTool`) will likely fail with `Unauthorized` errors.

### How it works

The `ToolboxClient` allows you to specify TokenSources that dynamically generate HTTP headers for
every request sent to the Toolbox server. The most common use case is to add an
Authorization header with a bearer token (e.g., a Google ID token).

These header-generating functions are called just before each request, ensuring
that fresh credentials or header values can be used.

### Configuration

You can configure these dynamic headers as seen below:


```go
import "github.com/googleapis/mcp-toolbox-sdk-go/core"

tokenProvider := func() string {
	return "header3_value"
}

staticTokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "header2_value"})
dynamicTokenSource := core.NewCustomTokenSource(tokenProvider)

client, err := core.NewToolboxClient(
  "toolbox-url",
  core.WithClientHeaderString("header1", "header1_value"),
  core.WithClientHeaderTokenSource("header2", staticTokenSource),
  core.WithClientHeaderTokenSource("header3", dynamicTokenSource),
)
```

### Authenticating with Google Cloud Servers

For Toolbox servers hosted on Google Cloud (e.g., Cloud Run) and requiring
`Google ID token` authentication, the helper module
[auth_methods](https://github.com/googleapis/mcp-toolbox-sdk-go/blob/main/core/auth.go) provides utility functions.

### Step by Step Guide for Cloud Run

1. **Configure Permissions**: [Grant](https://cloud.google.com/run/docs/securing/managing-access#service-add-principals) the `roles/run.invoker` IAM role on the Cloud
   Run service to the principal. This could be your `user account email` or a
   `service account`.
2. **Configure Credentials**
    - Local Development: Set up
   [ADC](https://cloud.google.com/docs/authentication/set-up-adc-local-dev-environment).
    - Google Cloud Environments: When running within Google Cloud (e.g., Compute
      Engine, GKE, another Cloud Run service, Cloud Functions), ADC is typically
      configured automatically, using the environment's default service account.
3. **Connect to the Toolbox Server**
    ```go
    import "github.com/googleapis/mcp-toolbox-sdk-go/core"
    import "context"

    ctx := context.Background()

    token, err := core.GetGoogleIDToken(ctx, URL)

    client, err := core.NewToolboxClient(
      URL,
      core.WithClientHeaderString("Authorization", token),
    )

    // Now, you can use the client as usual.
    ```

## Authenticating Tools

{{< notice info >}}
**Always use HTTPS** to connect your application with the Toolbox service, especially in **production environments** or whenever the communication involves **sensitive data** (including scenarios where tools requireauthentication tokens). Using plain HTTP lacks encryption and exposes your application and data to significant security risks, such as eavesdropping and tampering.
{{< /notice >}}

Tools can be configured within the Toolbox service to require authentication,
ensuring only authorized users or applications can invoke them, especially when
accessing sensitive data.

### When is Authentication Needed?

Authentication is configured per-tool within the Toolbox service itself. If a
tool you intend to use is marked as requiring authentication in the service, you
must configure the SDK client to provide the necessary credentials (currently
Oauth2 tokens) when invoking that specific tool.

### Supported Authentication Mechanisms

The Toolbox service enables secure tool usage through **Authenticated Parameters**.
For detailed information on how these mechanisms work within the Toolbox service and how to configure them, please refer to [Toolbox Service Documentation - Authenticated Parameters](https://googleapis.github.io/genai-toolbox/resources/tools/#authenticated-parameters).

### Step 1: Configure Tools in Toolbox Service

First, ensure the target tool(s) are configured correctly in the Toolbox service
to require authentication. Refer to the [Toolbox Service Documentation -
Authenticated
Parameters](https://googleapis.github.io/genai-toolbox/resources/tools/#authenticated-parameters)
for instructions.

### Step 2: Configure SDK Client

Your application needs a way to obtain the required Oauth2 token for the
authenticated user. The SDK requires you to provide a function capable of
retrieving this token *when the tool is invoked*.

#### Provide an ID Token Retriever Function

You must provide the SDK with a function that returns the
necessary token when called. The implementation depends on your application's
authentication flow (e.g., retrieving a stored token, initiating an OAuth flow).

{{< notice info >}}
The name used when registering the getter function with the SDK (e.g., `"my_api_token"`) must exactly match the `name` of the corresponding `authServices` defined in the tool's configuration within the Toolbox service.
{{< /notice >}}

```go
func getAuthToken() string {
  // ... Logic to retrieve ID token (e.g., from local storage, OAuth flow)
  // This example just returns a placeholder. Replace with your actual token retrieval.
	return "YOUR_ID_TOKEN" // Placeholder
}
```

{{< notice tip >}}
Your token retriever function is invoked every time an authenticated parameter requires a token for a tool call. Consider implementing caching logic within this function to avoid redundant token fetching or generation, especially for tokens with longer validity periods or if the retrieval process is resource-intensive.
{{< /notice >}}

#### Option A: Add Default Authentication to a Client

You can add default tool level authentication to a client.
Every tool / toolset loaded by the client will contain the auth token.

```go

ctx := context.Background()

client, err := core.NewToolboxClient("http://127.0.0.1:5000",
	core.WithDefaultToolOptions(
		core.WithAuthTokenString("my-auth-1", "auth-value"),
	),
)

AuthTool, err := client.LoadTool("my-tool", ctx)
```

#### Option B: Add Authentication to a Loaded Tool

You can add the token retriever function to a tool object *after* it has been
loaded. This modifies the specific tool instance.

```go

ctx := context.Background()

client, err := core.NewToolboxClient("http://127.0.0.1:5000")

tool, err := client.LoadTool("my-tool", ctx)

AuthTool, err := tool.ToolFrom(
  core.WithAuthTokenSource("my-auth", headerTokenSource),
  core.WithAuthTokenString("my-auth-1", "value"),
  )
```

#### Option C: Add Authentication While Loading Tools

You can provide the token retriever(s) directly during the `LoadTool` or
`LoadToolset` calls. This applies the authentication configuration only to the
tools loaded in that specific call, without modifying the original tool objects
if they were loaded previously.

```go
AuthTool, err := client.LoadTool("my-tool", ctx, core.WithAuthTokenString("my-auth-1", "value"))

// or

AuthTools, err := client.LoadToolset(
  "my-toolset",
  ctx,
  core.WithAuthTokenString("my-auth-1", "value"),
)
```

{{< notice note >}}
Adding auth tokens during loading only affect the tools loaded within that call.
{{< /notice >}}

### Complete Authentication Example

```go
import "github.com/googleapis/mcp-toolbox-sdk-go/core"
import "fmt"

func getAuthToken() string {
  // ... Logic to retrieve ID token (e.g., from local storage, OAuth flow)
  // This example just returns a placeholder. Replace with your actual token retrieval.
	return "YOUR_ID_TOKEN" // Placeholder
}

func main() {
  ctx := context.Background()
  inputs := map[string]any{"input": "some input"}

  dynamicTokenSource := core.NewCustomTokenSource(getAuthToken)

  client, err := core.NewToolboxClient("http://127.0.0.1:5000")
  tool, err := client.LoadTool("my-tool", ctx)
  AuthTool, err := tool.ToolFrom(core.WithAuthTokenSource("my_auth", dynamicTokenSource))

  result, err := AuthTool.Invoke(ctx, inputs)

  fmt.Println(result)
}
```

{{< notice note >}}
An auth token getter for a specific name (e.g., "GOOGLE_ID") will replace any client header with the same name followed by "_token" (e.g., "GOOGLE_ID_token").
{{< /notice >}}

## Binding Parameter Values

The SDK allows you to pre-set, or "bind", values for specific tool parameters
before the tool is invoked or even passed to an LLM. These bound values are
fixed and will not be requested or modified by the LLM during tool use.

### Why Bind Parameters?

- **Protecting sensitive information:**  API keys, secrets, etc.
- **Enforcing consistency:** Ensuring specific values for certain parameters.
- **Pre-filling known data:**  Providing defaults or context.

{{< notice info >}}
The parameter names used for binding (e.g., `"api_key"`) must exactly match the parameter names defined in the tool's configuration within the Toolbox service.
{{< /notice >}}

{{< notice note >}}
You do not need to modify the tool's configuration in the Toolbox service to bind parameter values using the SDK.
{{< /notice >}}

#### Option A: Add Default Bound Parameters to a Client

You can add default tool level bound parameters to a client. Every tool / toolset  
loaded by the client will have the bound parameter.

```go

ctx := context.Background()

client, err := core.NewToolboxClient("http://127.0.0.1:5000",
	core.WithDefaultToolOptions(
		core.WithBindParamString("param1", "value"),
	),
)

boundTool, err := client.LoadTool("my-tool", ctx)
```

### Option B: Binding Parameters to a Loaded Tool

Bind values to a tool object *after* it has been loaded. This modifies the
specific tool instance.

```go
client, err := core.NewToolboxClient("http://127.0.0.1:5000")

tool, err := client.LoadTool("my-tool", ctx)

boundTool, err := tool.ToolFrom(
  core.WithBindParamString("param1", "value"),
  core.WithBindParamString("param2", "value")
  )
```

### Option C: Binding Parameters While Loading Tools

Specify bound parameters directly when loading tools. This applies the binding
only to the tools loaded in that specific call.

```go
boundTool, err := client.LoadTool("my-tool", ctx, core.WithBindParamString("param", "value"))

// OR

boundTool, err := client.LoadToolset("", ctx, core.WithBindParamString("param", "value"))
```

{{< notice note >}} Bound values during loading only affect the tools loaded in that call. {{< /notice >}}

### Binding Dynamic Values

Instead of a static value, you can bind a parameter to a synchronous or
asynchronous function. This function will be called *each time* the tool is
invoked to dynamically determine the parameter's value at runtime.
Functions with the return type (data_type, error) can be provided.

```go
getDynamicValue := func() (string, error) { return "req-123", nil }

dynamicBoundTool, err := tool.ToolFrom(core.WithBindParamStringFunc("param", getDynamicValue))
```

{{< notice info >}} You don't need to modify tool configurations to bind parameter values. {{< /notice >}}

## Default Parameters

Tools defined in the MCP Toolbox server can specify default values for their optional parameters. When invoking a tool using the SDK, if an input for a parameter with a default value is not provided, the SDK will automatically populate the request with the default value.

```go
tool, err = client.LoadTool("my-tool", ctx)

// If 'my-tool' has a parameter 'param2' with a default value of "default-value",
// we can omit 'param2' from the inputs.
inputs := map[string]any{"param1": "value"}

// The invocation will automatically use param2="default-value" if not provided
result, err := tool.Invoke(ctx, inputs)
```


# Using with Orchestration Frameworks

To see how the MCP Toolbox Go SDK works with orchestration frameworks, check out these end-to-end examples given below.

<details>
<summary>Google GenAI</summary>

```go
// This sample demonstrates integration with the standard Google GenAI framework.
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

</details>

<details>
<summary>LangChain</summary>

```go
// This sample demonstrates how to use Toolbox tools as function definitions in LangChain Go.
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

	// Load the tools using the MCP Toolbox SDK.
	tools, err := toolboxClient.LoadToolset("my-toolset", ctx)
	if err != nil {
		log.Fatalf("Failed to load tools: %v\nMake sure your Toolbox server is running and the tool is configured.", err)
	}

	toolsMap := make(map[string]*core.ToolboxTool, len(tools))

	langchainTools := make([]llms.Tool, len(tools))
	for i, tool := range tools {
    // Convert the loaded ToolboxTools into the format LangChainGo requires.
		langchainTools[i] = ConvertToLangchainTool(tool)
    // Add tool to a map for lookup later
		toolsMap[tool.Name()] = tool
	}

	// Start the conversation history.
	messageHistory := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, "Find hotels in Basel with Basel in it's name."),
	}

	// Make the first call to the LLM, making it aware of the tool.
	resp, err := llm.GenerateContent(ctx, messageHistory, llms.WithTools(langchainTools))
	if err != nil {
		log.Fatalf("LLM call failed: %v", err)
	}

	// Add the model's response (which should be a tool call) to the history.
	respChoice := resp.Choices[0]
	assistantResponse := llms.TextParts(llms.ChatMessageTypeAI, respChoice.Content)
	for _, tc := range respChoice.ToolCalls {
		assistantResponse.Parts = append(assistantResponse.Parts, tc)
	}
	messageHistory = append(messageHistory, assistantResponse)

	// Process each tool call requested by the model.
	for _, tc := range respChoice.ToolCalls {
		toolName := tc.FunctionCall.Name

		switch tc.FunctionCall.Name {
		case "search-hotels-by-name":
			var args map[string]any
			if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
				log.Fatalf("Failed to unmarshal arguments for tool '%s': %v", toolName, err)
			}
			tool := toolsMap["search-hotels-by-name"]
			toolResult, err := tool.Invoke(ctx, args)
			if err != nil {
				log.Fatalf("Failed to execute tool '%s': %v", toolName, err)
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
		default:
			log.Fatalf("got unexpected function call: %v", tc.FunctionCall.Name)
		}
	}

	// Final LLM Call for Natural Language Response
	log.Println("Sending tool response back to LLM for a final answer...")

	// Call the LLM again with the updated history, which now includes the tool's result.
	finalResp, err := llm.GenerateContent(ctx, messageHistory)
	if err != nil {
		log.Fatalf("Final LLM call failed: %v", err)
	}

	// Display the Result
	fmt.Println("\n======================================")
	fmt.Println("Final Response from LLM:")
	fmt.Println(finalResp.Choices[0].Content)
	fmt.Println("======================================")
}

```
</details>

<details>
<summary>OpenAI</summary>

```go
// This sample demonstrates integration with the OpenAI Go client.
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

// If there was a function call, continue the conversation
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

</details>
