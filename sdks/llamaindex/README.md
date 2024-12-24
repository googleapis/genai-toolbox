# GenAI Toolbox SDK

This SDK allows you to seamlessly integrate the functionalities of
[Toolbox](https://github.com/googleapis/genai-toolbox) into your LLM
applications, enabling advanced orchestration and interaction with GenAI models.

<!-- TOC ignore:true -->
## Table of Contents
<!-- TOC -->

- [Installation](#installation)
- [Usage](#usage)
- [Load a toolset](#load-a-toolset)
- [Load a single tool](#load-a-single-tool)
- [Use with LlamaIndex](#use-with-llamaindex)
- [Manual usage](#manual-usage)
- [Authenticating Tools](#authenticating-tools)
    - [Supported Authentication Mechanisms](#supported-authentication-mechanisms)
    - [Configuring Tools for Authentication](#configuring-tools-for-authentication)
    - [Configure SDK for Authentication](#configure-sdk-for-authentication)
    - [Complete Example](#complete-example)

<!-- /TOC -->

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
> [running an async program in python](https://docs.python.org/3/library/asyncio-runner.html#running-an-asyncio-program).

> [!TIP]
> You can also pass your own `ClientSession` so that the `ToolboxClient` can
> reuse the same session.
> ```py
> async with ClientSession() as session:
>   toolbox = ToolboxClient("http://localhost:5000", session)
> ```

## Load a toolset

You can load a toolset, a collection of related tools.

```py
# Load all tools
tools = await toolbox.load_toolset()

# Load a specific toolset
tools = await toolbox.load_toolset("my-toolset")
```

## Load a single tool

You can also load a single tool.

```py
tool = await toolbox.load_tool("my-tool")
```

## Use with LlamaIndex

LlamaIndex agents can dynamically choose and execute tools based on the user
input. The user can include the tools loaded from the Toolbox SDK in the agent's
toolkit.

```py
from llama_index.llms.vertex import Vertex
from llama_index.core.agent import ReActAgent

model = Vertex(model="gemini-1.5-flash")

# Initialize agent with tools
agent = ReActAgent.from_tools(tools, llm=model, verbose=True)

# Query the agent
response = agent.query("Get some response from the agent.")
```

## Manual usage

You can also execute a tool manually using the `acall` method.

```py
result = await tools[0].acall({ "name": "Alice", "age": 30 })
```

## Authenticating Tools

> [!WARNING]
> Always use HTTPS to connect your application with the Toolbox service,
> especially when using tools with authentication configured. Using HTTP exposes
> your application to serious security risks, including unauthorized access to
> user information and man-in-the-middle attacks, where sensitive data can be
> intercepted.

Some tools in your Toolbox configuration might require user authentication to
access sensitive data. This section guides you on how to configure tools for
authentication and use them with the SDK.

### Supported Authentication Mechanisms
The Toolbox SDK currently supports authentication using [OIDC
protocol](https://openid.net/specs/openid-connect-core-1_0.html). Specifically,
it uses [ID
tokens](https://openid.net/specs/openid-connect-core-1_0.html#IDToken) and *not*
access tokens for [Google OAuth
2.0](https://cloud.google.com/apigee/docs/api-platform/security/oauth/oauth-home).

### Configuring Tools for Authentication

Refer to [these
instructions](../../docs/tools/README.md#authenticated-parameters) on
configuring tools for authenticated parameters.

### Configure SDK for Authentication

Provide the `auth_headers` parameter to the `load_tool` or `load_toolset` calls
with a dictionary. The keys of this dictionary should match the names of the
authentication sources configured in your tools file (e.g., `my_auth_service`),
and the values should be callable functions (e.g., lambdas or regular functions)
that return the ID token of the logged-in user.

Here's an example:

```py
def get_auth_header():
    # ... Logic to retrieve ID token (e.g., from local storage, OAuth flow)
    # This example just returns a placeholder. Replace with your actual token retrieval.
    return "YOUR_ID_TOKEN"

toolbox = ToolboxClient("http://localhost:5000")

tools = toolbox.load_toolset(auth_headers={ "my_auth_service": get_auth_header })

# OR

tool = toolbox.load_tool("my_tool", auth_headers={ "my_auth_service": get_auth_header })
```

Alternatively, you can call the `add_auth_header` method to configure
authentication separately.

```py
toolbox.add_auth_header("my_auth_service", get_auth_header)
```

> [!NOTE]
> Authentication headers added via `load_tool`, `load_toolset`, or
> `add_auth_header` apply to all subsequent tool invocations, regardless of when
> the tool was loaded. This ensures a consistent authentication context.

### Complete Example

```py
import asyncio
from toolbox_llamaindex_sdk import ToolboxClient

async def get_auth_header():
    # Replace with your actual ID token retrieval logic.
    # For example, using a library like google-auth
    # from google.oauth2 import id_token
    # from google.auth.transport import requests
    # request = requests.Request()
    # id_token_string = id_token.fetch_id_token(request, "YOUR_AUDIENCE")# Replace with your audience
    # return id_token_string
    return "YOUR_ACTUAL_ID_TOKEN" # placeholder

async def main():
    toolbox = ToolboxClient("http://localhost:5000")
    toolbox.add_auth_header("my_auth_service", get_auth_header)
    tools = await toolbox.load_toolset()
    result = await tools[0].acall({"input": "some input"})
    print(result)

if __name__ == "__main__":
    asyncio.run(main())
```