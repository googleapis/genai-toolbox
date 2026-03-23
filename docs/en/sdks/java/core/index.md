---
title: "Core"
type: docs
weight: 2
description: >
  MCP Toolbox SDK for integrating functionalities of MCP Toolbox into your Agentic apps.
---

## Overview

The package provides a java interface to the MCP Toolbox service, enabling you to load and invoke tools from your own applications.

## Installation

This SDK is distributed via a [Maven Central Repository](https://mvnrepository.com/artifact/com.google.cloud.mcp/mcp-toolbox-sdk-java).

### Maven
Add the dependency to your `pom.xml`:
```xml
<dependency>
    <groupId>com.google.cloud.mcp</groupId>
    <artifactId>mcp-toolbox-sdk-java</artifactId>
    <!-- Replace 'VERSION' with the latest version from https://mvnrepository.com/artifact/com.google.cloud.mcp/mcp-toolbox-sdk-java -->
    <version>VERSION</version>
    <scope>compile</scope>
</dependency>
```

### Gradle

```
dependencies {
    // Replace 'VERSION' with the latest version from https://mvnrepository.com/artifact/com.google.cloud.mcp/mcp-toolbox-sdk-java
    implementation("com.google.cloud.mcp:mcp-toolbox-sdk-java:VERSION") 
}
```

## Quickstart

Here is the minimal code needed to connect to a MCP Toolbox and invoke a tool.

```java
import com.google.cloud.mcp.McpToolboxClient;
import java.util.Map;

public class App {
    public static void main(String[] args) {
        // 1. Create the Client
        McpToolboxClient client = McpToolboxClient.builder()
            .baseUrl("https://my-toolbox-service.a.run.app/mcp") 
            .build();

        // 2. Invoke a Tool
        client.invokeTool("get-toy-price", Map.of("description", "plush dinosaur"))
            .thenAccept(result -> {
                // Pick the first item from the response.
                System.out.println("Tool Output: " + result.content().get(0).text());
            })
            .exceptionally(ex -> {
                System.err.println("Error: " + ex.getMessage());
                return null;
            })
            .join(); // Wait for completion
    }
}
```

For a detailed example, check the [ExampleUsage.java file](https://github.com/googleapis/mcp-toolbox-sdk-java/blob/main/example/src/main/java/cloudcode/helloworld/ExampleUsage.java) in the example folder of [Java SDK Repo](https://github.com/googleapis/mcp-toolbox-sdk-java).

{{< notice tip >}}
The SDK is Async-First, using Java's `CompletableFuture` to bridge both patterns naturally.
- Chain methods using `.thenCompose()`, `.thenAccept()`, and `.exceptionally()` for non-blocking execution.
- If you prefer synchronous execution, simply call `.join()` on the result to block until completion.

```java
// Async (Non-blocking)
client.invokeTool("tool-name", args).thenAccept(result -> ...);
// Sync (Blocking)
ToolResult result = client.invokeTool("tool-name", args).join();
```
{{< /notice >}}

## Usage

### Load the Client

The `McpToolboxClient` is your entry point. It is thread-safe and designed to be instantiated once and reused.

```java
// Local Development
McpToolboxClient client = McpToolboxClient.builder()
    .baseUrl("http://localhost:5000/mcp")
    .build();

// Cloud Run Production
McpToolboxClient client = McpToolboxClient.builder()
    .baseUrl("https://my-toolbox-service.a.run.app/mcp")
    // .apiKey("...") // Optional: Overrides automatic Google Auth
    .build();
```

### Load a Toolset

You can load all available tools or a specific subset (toolset) if your server supports it. This returns a Map of tool definitions.

```java
// Load all tools (alias for listTools)
client.loadToolset().thenAccept(tools -> {
    System.out.println("Available Tools: " + tools.keySet());
    
    tools.forEach((name, definition) -> {
        System.out.println("Tool: " + name);
        System.out.println("Description: " + definition.description());
    });
});
```

```java
// Load a specific toolset (e.g., 'retail-tools')
client.loadToolset("retail-tools").thenAccept(tools -> {
    System.out.println("Tools in Retail Set: " + tools.keySet());
});
```

### Load a Tool

If you know the specific tool you want to use, you can load its definition directly. This is useful for validation or inspecting the required parameters before execution.

```java
client.loadTool("get-toy-price").thenAccept(toolDef -> {
    System.out.println("Loaded Tool: " + toolDef.description());
    System.out.println("Parameters: " + toolDef.parameters());
});
```


### Invoke a Tool

Invoking a tool sends a request to the MCP Server to execute the logic (SQL, API call, etc.). Arguments are passed as a `Map<String, Object>`.

```java
import java.util.Map;

Map<String, Object> args = Map.of(
    "description", "plush dinosaur",
    "limit", 5
);

client.invokeTool("get-toy-price", args).thenAccept(result -> {
    // Pick the first item from the response.
    System.out.println("Result: " + result.content().get(0).text());
});
```

## Authentication

### Client to Server Authentication

This section describes how to authenticate the `ToolboxClient` itself when connecting to a MCP Toolbox server instance that requires authentication. This is crucial for securing your MCP Toolbox server endpoint, especially when deployed on platforms like Cloud Run, GKE, or any environment where unauthenticated access is restricted.

### When is Client-to-Server Authentication Needed

You'll need this if your MCP Toolbox server is configured to deny unauthenticated requests. For example:

* Your MCP Toolbox server is deployed on **Google Cloud Run** and configured to "Require authentication" (default).  
* Your server is behind an Identity-Aware Proxy (IAP).  
* You have custom authentication middleware.

Without proper client authentication, attempts to connect (like `listTools`) will fail with `401 Unauthorized` or `403 Forbidden` errors.

### How it works

The Java SDK handles the generation of **Authorization headers** (Bearer tokens) using the **Google Auth Library**. It follows the **Application Default Credentials (ADC)** strategy to find the correct credentials based on the environment where your code is running.

{{< notice note >}}
To get started with local development, you'll need to set up [Application Default Credentials (ADC)](https://cloud.google.com/docs/authentication/set-up-adc-local-dev-environment).
{{< /notice >}}

### Authenticating with Google Cloud Servers (Cloud Run)

For MCP Toolbox servers hosted on Google Cloud (e.g., Cloud Run), the SDK provides seamless OIDC authentication.

#### 1\. Configure Permissions

Grant the **`roles/run.invoker`** IAM role on the Cloud Run service to the principal calling the service.

* **Local Dev:** Grant this role to your *User Account Email*.  
* **Production:** Grant this role to the *Service Account* attached to your application.

#### 2\. Configure Credentials

**Option A: Local Development** If running on your laptop, use the `gcloud` CLI to set up your user credentials.

```
gcloud auth application-default login
```

The SDK will automatically detect these credentials and generate an OIDC ID Token intended for your MCP Toolbox URL.

**Option B: Google Cloud Environments** When running within Google Cloud (e.g., Compute Engine, GKE, another Cloud Run service, Cloud Functions), ADC is configured automatically. The SDK uses the environment's default service account. No extra code or configuration is required.

**Option C: On-Premise / CI/CD** If running outside of Google Cloud (e.g., Jenkins, AWS), create a Service Account Key (JSON) and set the environment variable:

```
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/key.json"
```

| Environment | Mechanism | Setup Required |
| :---- | :---- | :---- |
| **Local Dev** | Uses User Credentials | Run gcloud auth application-default login |
| **Cloud Run** | Uses Service Account | **None.** (Automatic) |
| **CI/CD** | Uses Service Account Key | Set GOOGLE\_APPLICATION\_CREDENTIALS=/path/to/key.json |

{{< notice note >}}
If you provide an `.apiKey()` in the builder, it overrides the automatic ADC mechanism.
{{< /notice >}}

### Authenticating the Tools

Tools can be configured within the MCP Toolbox service to require authentication, ensuring only authorized users or applications can invoke them, especially when accessing sensitive data.

{{< notice info >}}
Always use HTTPS to connect your application with the MCP Toolbox service, especially in production environments or whenever the communication involves sensitive data (including scenarios where tools require authentication tokens). Using plain HTTP lacks encryption and exposes your application and data to significant security risks, such as eavesdropping and tampering.
{{< /notice >}}

### When is Authentication Needed?

Authentication is configured per-tool within the MCP Toolbox service itself. If a tool you intend to use is marked as requiring authentication in the service, you must configure the SDK client to provide the necessary credentials (currently OAuth2 tokens) when invoking that specific tool.

### Supported Authentication Mechanisms

The MCP Toolbox service enables secure tool usage through Authenticated Parameters. For detailed information on how these mechanisms work within the MCP Toolbox service and how to configure them, please refer to [MCP Toolbox Service Documentation \- Authenticated Parameters](https://googleapis.github.io/genai-toolbox/resources/tools/#authenticated-parameters)

### Step 1: Configure Tools in MCP Toolbox Service

First, ensure the target tool(s) are configured correctly in the MCP Toolbox service to require authentication. Refer to the [MCP Toolbox Service Documentation \- Authenticated Parameters](https://googleapis.github.io/genai-toolbox/resources/tools/#authenticated-parameters) for instructions.

### Step 2: Configure SDK Client

Your application needs a way to obtain the required token for the authenticated user. The SDK requires you to provide a function capable of retrieving this token *when the tool is invoked*.

### Provide a Token Retriever Function

You must provide the SDK with an `AuthTokenGetter` (a function that returns a `CompletableFuture<String>`). This implementation depends on your application's authentication flow (e.g., retrieving a stored token, initiating an OAuth flow).

{{< notice info >}}
The **Service Name** (or Auth Source) used when adding the getter (e.g., `"salesforce_auth"`) must exactly match the name of the corresponding auth source defined in the tool's configuration.
{{< /notice >}}

```java
import com.google.cloud.mcp.AuthTokenGetter;

// Define your token retrieval logic
AuthTokenGetter salesforceTokenGetter = () -> {
    return CompletableFuture.supplyAsync(() -> fetchTokenFromVault()); 
};
//example tool: search-salesforce and related sample params
client.loadTool("search-salesforce").thenCompose(tool -> {
    // Register the getter. It will be called every time 'execute' is run.
    tool.addAuthTokenGetter("salesforce_auth", salesforceTokenGetter);
    
    return tool.execute(Map.of("query", "recent leads"));
});
```

{{< notice tip >}}
Your token retriever function is invoked every time an authenticated parameter requires a token for a tool call. Consider implementing caching logic within this function to avoid redundant token fetching or generation, especially for tokens with longer validity periods or if the retrieval process is resource-intensive.
{{< /notice >}}


### Complete Authentication Example

Here is a complete example of how to configure and invoke an authenticated tool.

```java
import com.google.cloud.mcp.McpToolboxClient;
import com.google.cloud.mcp.AuthTokenGetter;
import java.util.Map;
import java.util.concurrent.CompletableFuture;

public class AuthExample {
    public static void main(String[] args) {
        // 1. Define your token retrieval logic
        AuthTokenGetter tokenGetter = () -> {
            // Logic to retrieve ID token (e.g., from local storage, OAuth flow)
            return CompletableFuture.completedFuture("YOUR_ID_TOKEN"); 
        };

        // 2. Initialize the client
        McpToolboxClient client = McpToolboxClient.builder()
            .baseUrl("http://127.0.0.1:5000/mcp")
            .build();

        // 3. Load tool, attach auth, and execute
        client.loadTool("my-tool")
            .thenCompose(tool -> {
                // "my_auth" must match the name in the tool's authSources config
                tool.addAuthTokenGetter("my_auth", tokenGetter);
                
                return tool.execute(Map.of("input", "some input"));
            })
            .thenAccept(result -> {
                // Pick the first item from the response.
                System.out.println(result.content().get(0).text());
            })
            .join();
    }
}
```

## Binding Parameter Values

The SDK allows you to pre-set, or "bind", values for specific tool parameters before the tool is invoked or even passed to an LLM. These bound values are fixed and will not be requested or modified by the LLM during tool use.

### Why Bind Parameters?

* Protecting sensitive information: API keys, secrets, etc.  
* Enforcing consistency: Ensuring specific values for certain parameters.  
* Pre-filling known data: Providing defaults or context.

{{< notice info >}}
The parameter names used for binding (e.g., `"api_key"`) must exactly match the parameter names defined in the tool's configuration within the MCP Toolbox service.
{{< /notice >}}

{{< notice tip >}}
You do not need to modify the tool's configuration in the MCP Toolbox service to bind parameter values using the SDK.
{{< /notice >}}

### Option A: Static Binding

Bind a fixed value to a tool object.

```java
client.loadTool("get-toy-price").thenCompose(tool -> {
    // Bind 'currency' to 'USD' permanently for this tool instance
    tool.bindParam("currency", "USD");
    
    // Now invoke without specifying currency
    return tool.execute(Map.of("description", "lego set")); 
});
```

### Option B: Dynamic Binding

Instead of a static value, you can bind a parameter to a synchronous or asynchronous function (`Supplier`). This function will be called **each time** the tool is invoked to dynamically determine the parameter's value at runtime.

```java
client.loadTool("check-order-status").thenCompose(tool -> {
    // Bind 'user_id' to a function that fetches the current user from context
    tool.bindParam("user_id", () -> SecurityContext.getCurrentUser().getId());
    
    // Invoke: The SDK will call the supplier to fill 'user_id'
    return tool.execute(Map.of("order_id", "12345"));
});
```

{{< notice tip >}}
You don't need to modify tool configurations to bind parameter values.
{{< /notice >}}

## Error Handling

The SDK uses Java's `CompletableFuture` API. Errors (Network issues, `4xx`/`5xx` responses) are propagated as exceptions wrapped in `CompletionException`.

```java

client.invokeTool("invalid-tool", Map.of())
    .handle((result, ex) -> {
        if (ex != null) {
            System.err.println("Invocation Failed: " + ex.getCause().getMessage());
            return null; // Handle error
        }
        return result; // Success path
    });

```
