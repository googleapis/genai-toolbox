---
title: "AuthServices"
type: docs
weight: 1
description: >
  AuthServices represent services that handle authentication and authorization. 
---

AuthServices represent services that handle authentication and authorization. It
can primarily be used by [Tools](../tools) in two different ways:

- [**Authorized Invocation**][auth-invoke] is when a tool
  is validated by the auth service before the call can be invoked. Toolbox
  will reject any calls that fail to validate or have an invalid token.
- [**Authenticated Parameters**][auth-params] replace the value of a parameter
  with a field from an [OIDC][openid-claims] claim. Toolbox will automatically
  resolve the ID token provided by the client and replace the parameter in the
  tool call.

[openid-claims]: https://openid.net/specs/openid-connect-core-1_0.html#StandardClaims
[auth-invoke]: ../tools/#authorized-invocations
[auth-params]: ../tools/#authenticated-parameters

## Example

The following configurations are placed at the top level of a `tools.yaml` file.

{{< notice tip >}}
If you are accessing Toolbox with multiple applications, each
 application should register their own Client ID even if they use the same
 "kind" of auth provider.
{{< /notice >}}

```yaml
authServices:
  my_auth_app_1:
    kind: google
    clientId: ${YOUR_CLIENT_ID_1}
  my_auth_app_2:
    kind: google
    clientId: ${YOUR_CLIENT_ID_2}
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

After you've configured an `authService` you'll, need to reference it in the
configuration for each tool that should use it:

- **Authorized Invocations** for authorizing a tool call, [use the
  `authRequired` field in a tool config][auth-invoke]
- **Authenticated Parameters** for using the value from a OIDC claim, [use the
  `authServices` field in a parameter config][auth-params]

## Specifying ID Tokens from Clients

After [configuring](#example) your `authServices` section, use a Toolbox SDK to
add your ID tokens to the header of a Tool invocation request. When specifying a
token you will provide a function (that returns an id). This function is called
when the tool is invoked. This allows you to cache and refresh the ID token as
needed.

The primary method for providing these getters is via the `auth_token_getters`
parameter when loading tools, or the `add_auth_token_getter`() /
`add_auth_token_getters()` methods on a loaded tool object.

### Specifying tokens during load

{{< tabpane persist=header >}}
{{< tab header="Core" lang="Python" >}}
import asyncio
from toolbox_core import ToolboxClient

async def get_auth_token():
    # ... Logic to retrieve ID token (e.g., from local storage, OAuth flow)
    # This example just returns a placeholder. Replace with your actual token retrieval.
    return "YOUR_ID_TOKEN" # Placeholder

async def main():
    async with ToolboxClient("<http://127.0.0.1:5000>") as toolbox:
        auth_tool = await toolbox.load_tool(
            "get_sensitive_data",
            auth_token_getters={"my_auth_app_1": get_auth_token}
        )
        result = await auth_tool(param="value")
        print(result)

if **name** == "**main**":
    asyncio.run(main())
{{< /tab >}}
{{< tab header="LangChain" lang="Python" >}}
import asyncio
from toolbox_langchain import ToolboxClient

async def get_auth_token():
    # ... Logic to retrieve ID token (e.g., from local storage, OAuth flow)
    # This example just returns a placeholder. Replace with your actual token retrieval.
    return "YOUR_ID_TOKEN" # Placeholder

async def main():
    toolbox = ToolboxClient("<http://127.0.0.1:5000>")

    auth_tool = await toolbox.aload_tool(
        "get_sensitive_data",
        auth_token_getters={"my_auth_app_1": get_auth_token}
    )
    result = await auth_tool.ainvoke({"param": "value"})
    print(result)

if **name** == "**main**":
    asyncio.run(main())
{{< /tab >}}
{{< tab header="Llamaindex" lang="Python" >}}
import asyncio
from toolbox_llamaindex import ToolboxClient

async def get_auth_token():
    # ... Logic to retrieve ID token (e.g., from local storage, OAuth flow)
    # This example just returns a placeholder. Replace with your actual token retrieval.
    return "YOUR_ID_TOKEN" # Placeholder

async def main():
    toolbox = ToolboxClient("<http://127.0.0.1:5000>")

    auth_tool = await toolbox.aload_tool(
        "get_sensitive_data",
        auth_token_getters={"my_auth_app_1": get_auth_token}
    )
    # result = await auth_tool.acall(param="value")
    # print(result.content)

if **name** == "**main**":
    asyncio.run(main()){{< /tab >}}
{{< /tabpane >}}

### Specifying tokens for existing tools

{{< tabpane persist=header >}}
{{< tab header="Core" lang="Python" >}}
tools = await toolbox.load_toolset()

# for a single token

authorized_tool = tools[0].add_auth_token_getter("my_auth", get_auth_token)

# OR, if multiple tokens are needed

authorized_tool = tools[0].add_auth_token_getters({
  "my_auth1": get_auth1_token,
  "my_auth2": get_auth2_token,
})
{{< /tab >}}
{{< tab header="LangChain" lang="Python" >}}
tools = toolbox.load_toolset()

# for a single token

authorized_tool = tools[0].add_auth_token_getter("my_auth", get_auth_token)

# OR, if multiple tokens are needed

authorized_tool = tools[0].add_auth_token_getters({
  "my_auth1": get_auth1_token,
  "my_auth2": get_auth2_token,
})
{{< /tab >}}
{{< tab header="Llamaindex" lang="Python" >}}
tools = toolbox.load_toolset()

# for a single token

authorized_tool = tools[0].add_auth_token_getter("my_auth", get_auth_token)

# OR, if multiple tokens are needed

authorized_tool = tools[0].add_auth_token_getters({
  "my_auth1": get_auth1_token,
  "my_auth2": get_auth2_token,
})
{{< /tab >}}
{{< /tabpane >}}

## Kinds of Auth Services
