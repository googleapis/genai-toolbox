# GenAI Toolbox SDK

This SDK allows you to seamlessly integrate the functionalities of
[Toolbox](https://github.com/googleapis/genai-toolbox) into your LLM
applications, enabling advanced orchestration and interaction with GenAI models.

<!-- TOC ignore:true -->
## Table of Contents
<!-- TOC -->

- [Quickstart](#quickstart)
- [Installation](#installation)
- [Usage](#usage)
- [Loading Tools](#loading-tools)
    - [Load a toolset](#load-a-toolset)
    - [Load a single tool](#load-a-single-tool)
- [Use with LlamaIndex](#use-with-llamaindex)
- [Manual usage](#manual-usage)
- [Authenticating Tools](#authenticating-tools)
    - [Supported Authentication Mechanisms](#supported-authentication-mechanisms)
    - [Configure Tools](#configure-tools)
    - [Configure SDK](#configure-sdk)
        - [Add Authentication to a Tool](#add-authentication-to-a-tool)
        - [Add Authentication While Loading](#add-authentication-while-loading)
    - [Complete Example](#complete-example)
- [Binding Parameter Values](#binding-parameter-values)
    - [Binding Parameters to a Tool](#binding-parameters-to-a-tool)
    - [Binding Parameters While Loading](#binding-parameters-while-loading)
    - [Binding Dynamic Values](#binding-dynamic-values)
- [Error Handling](#error-handling)

<!-- /TOC -->

## Quickstart

Here's a minimal example to get you started:

```py
import asyncio
from llama_index.llms.vertex import Vertex
from llama_index.core.agent import ReActAgent
from toolbox_llamaindex_sdk import ToolboxClient

async def main():
    toolbox = ToolboxClient("http://127.0.0.1:5000")
    tools = await toolbox.load_toolset()
    
    model = Vertex(model="gemini-1.5-pro-002")
    agent = ReActAgent.from_tools(tools, llm=model, verbose=True)
    response = agent.query("Get some response from the agent.")
    print(result)

if __name__ == "__main__":
    asyncio.run(main())
```

## Installation

> [!IMPORTANT]
> This SDK is not yet available on PyPI. For now, install it from source by
> following these [installation instructions](DEVELOPER.md).

You can install the Toolbox SDK for LlamaIndex using `pip`.

```bash
pip install toolbox-llamaindex-sdk
```

## Usage

Import and initialize the toolbox client.

```py
from toolbox_llamaindex_sdk import ToolboxClient

# Replace with your Toolbox service's URL
toolbox = ToolboxClient("http://127.0.0.1:5000")
```

> [!IMPORTANT]
> The toolbox client requires an asynchronous environment.
> For guidance on running asynchronous Python programs, see
> [asyncio documentation](https://docs.python.org/3/library/asyncio-runner.html#running-an-asyncio-program).

> [!TIP]
> You can also pass your own `ClientSession` to reuse the same session:
> ```py
> async with ClientSession() as session:
>   toolbox = ToolboxClient("http://localhost:5000", session)
> ```

## Loading Tools

### Load a toolset

A toolset is a collection of related tools. You can load all tools in a toolset
or a specific one:

```py
# Load all tools
tools = await toolbox.load_toolset()

# Load a specific toolset
tools = await toolbox.load_toolset("my-toolset")
```

### Load a single tool

```py
tool = await toolbox.load_tool("my-tool")
```

Loading individual tools gives you finer-grained control over which tools are
available to your LLM agent.

## Use with LlamaIndex

LlamaIndex's agents can dynamically choose and execute tools based on the user
input. Include tools loaded from the Toolbox SDK in the agent's toolkit:

```py
from llama_index.llms.vertex import Vertex
from llama_index.core.agent import ReActAgent

model = Vertex(model="gemini-1.5-pro-002")

# Initialize agent with tools
agent = ReActAgent.from_tools(tools, llm=model, verbose=True)

# Query the agent
response = agent.query("Get some response from the agent.")
```

## Manual usage

Execute a tool manually using the `acall` method:

```py
result = await tools[0].acall({"name": "Alice", "age": 30})
```

This is useful for testing tools or when you need precise control over tool
execution outside of an agent framework.

## Authenticating Tools

> [!WARNING]
> Always use HTTPS to connect your application with the Toolbox service,
> especially when using tools with authentication configured. Using HTTP exposes
> your application to serious security risks.

Some tools require user authentication to access sensitive data.

### Supported Authentication Mechanisms
Toolbox currently supports authentication using the [OIDC
protocol](https://openid.net/specs/openid-connect-core-1_0.html) with [ID
tokens](https://openid.net/specs/openid-connect-core-1_0.html#IDToken) (not
access tokens) for [Google OAuth
2.0](https://cloud.google.com/apigee/docs/api-platform/security/oauth/oauth-home).

### Configure Tools

Refer to [these
instructions](../../docs/tools/README.md#authenticated-parameters) on
configuring tools for authenticated parameters.

### Configure SDK

You need a method to retrieve an ID token from your authentication service:

```py
async def get_auth_token():
    # ... Logic to retrieve ID token (e.g., from local storage, OAuth flow)
    return "YOUR_ID_TOKEN" # Placeholder
```

#### Add Authentication to a Tool

```py
toolbox = ToolboxClient("http://localhost:5000")
tools = await toolbox.load_toolset()

tools[0].add_auth_token("my_auth", get_auth_token)

# OR

for tool in tools:
    tool.add_auth_token("my_auth", get_auth_token)
```

#### Add Authentication While Loading

```py
toolbox = ToolboxClient("http://localhost:5000")

tool = await toolbox.load_tool(auth_tokens={"my_auth": get_auth_token})

# OR

tools = await toolbox.load_toolset(auth_tokens={"my_auth": get_auth_token})
```

> [!NOTE]
> Authentication tokens added during loading only affect the tools loaded within
> that specific call.

### Complete Example

```py
import asyncio
from toolbox_llamaindex_sdk import ToolboxClient

async def get_auth_token():
    # ... Logic to retrieve ID token
    return "YOUR_ID_TOKEN" # Placeholder

async def main():
    toolbox = ToolboxClient("http://localhost:5000")
    tool = await toolbox.load_tool("my-tool")

    tool.add_auth_token({"my_auth": get_auth_token})
    result = await tool.acall({"input": "some input"})
    print(result)

if __name__ == "__main__":
    asyncio.run(main())
```

## Binding Parameter Values

Predetermine values for tool parameters using the SDK. These values won't be
modified by the LLM. This is useful for:

* **Protecting sensitive information:**  API keys, secrets, etc.
* **Enforcing consistency:** Ensuring specific values for certain parameters.
* **Pre-filling known data:**  Providing defaults or context.

### Binding Parameters to a Tool

```py
tool.bind_param({"param": "value"}) 
```

### Binding Parameters While Loading

```py
tool = await toolbox.load_tool(bound_params={"param": "value"})
tools = await toolbox.load_toolset(bound_params={"param": "value"})
```

> [!NOTE]
> Bound values during loading only affect the tools loaded in that call.

### Binding Dynamic Values

Use a function to bind dynamic values:

```py
def get_dynamic_value():
  # Logic to determine the value
  return "dynamic_value"

tool.bind_param({"param": get_dynamic_value})

# OR

tools = await toolbox.load_toolset(bound_params={"param": get_dynamic_value})
```

> [!IMPORTANT]
> You don't need to modify tool configurations to bind parameter values.

## Error Handling

When interacting with the Toolbox service or executing tools, you might
encounter errors. Handle potential exceptions gracefully:

```py
try:
    result = await tool.acall({"input": "some input"})
except Exception as e:
    print(f"An error occurred: {e}")
    # Implement error recovery logic, e.g., retrying the request or logging the error
```