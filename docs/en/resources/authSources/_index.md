---
title: "AuthSources"
type: docs
weight: 1
description: >
  AuthSources represent services that handle authentication and authorization. 
---

AuthSources represent services that handle authentication and authorization. It
can primarily be used by [Tools](../tools) for two different features: 

- [**Authenticated Parameters**](../tools/#authenticated-parameters) replace the
  value of a parameter with a field from an [OIDC][openid-claims] claim. Toolbox
  will automatically resolve the ID token provided by the client and replace the
  parameter in the tool call.
- [**Authorized Invocation**](../tools/#authorized-invocations) is when a
  tool requires the client to have a valid Oauth2.0 token attached
  before the call can be invoked. Toolbox will rejected an calls without a valid
  token.

## Specifying ID Tokens

After configuring your `authSources` section, use a Toolbox SDK to add your `ID
tokens` to the header of a Tool invocation request. When specifying a token you
will provide a function (that returns an id). This function is called when the
tool is invoked. This allows you to cache and refresh the ID token as needed. 

### Specify token during load
{{< tabpane >}}
{{< tab header="LangChain" lang="Python" >}}
async def get_auth_token():
    # ... Logic to retrieve ID token (e.g., from local storage, OAuth flow)
    # This example just returns a placeholder. Replace with your actual token retrieval.
    return "YOUR_ID_TOKEN" # Placeholder

# for a single tool use:
authorized_tool = await toolbox.load_tool("my-tool-name", auth_tokens={"my_auth": get_auth_token})

# for a toolset use: 
authorized_tools = await toolbox.load_toolset("my-toolset-name", auth_tokens={"my_auth": get_auth_token})
{{< /tab >}}
{{< /tabpane >}}


### Specify token to existing tool

{{< tabpane >}}
{{< tab header="LangChain" lang="Python" >}}
tools = await toolbox.load_toolset()
# for a single token
auth_tools = [tool.add_auth_token("my_auth", get_auth_token) for tool in tools]
# OR, if multiple tokens are needed
authorized_tool = tools[0].add_auth_tokens({
  "my_auth1": get_auth1_token,
  "my_auth2": get_auth2_token,
}) 
{{< /tab >}}
{{< /tabpane >}}

## Example

The following configuration is placed at the top level of your `tools.yaml`
file: 

```yaml
authSources:
  my-google-auth:
    kind: google
    clientId: YOUR_GOOGLE_CLIENT_ID
```

> [!TIP] If you are accessing Toolbox with multiple applications, each
> application should register their own Client ID even if they use the same
> `kind` of auth provider.
>
> Here's an example:
>
> ```yaml
> authSources:
>     my_auth_app_1:
>         kind: google
>         clientId: YOUR_CLIENT_ID_1
>     my_auth_app_2:
>         kind: google
>         clientId: YOUR_CLIENT_ID_2
>
> tools:
>     my_tool:
>         parameters:
>             - name: user_id
>               type: string
>               authSources:
>                   - name: my_auth_app_1
>                     field: sub
>                   - name: my_auth_app_2
>                     field: sub
>         ...
>
>     my_tool_no_param:
>         authRequired:
>             - my_auth_app_1
>             - my_auth_app_2
>         ...
> ```


## Kinds of Auth Sources
