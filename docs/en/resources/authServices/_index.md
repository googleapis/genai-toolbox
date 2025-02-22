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
  to be validate by the auth service before the call can be invoked. Toolbox
  will rejected an calls that fail to validate or have an invalid token.
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
    clientId: YOUR_CLIENT_ID_1
  my_auth_app_2:
    kind: google
    clientId: YOUR_CLIENT_ID_2
```

After you've configured an `authService` you'll, need to reference it in the
configuration for each tool that should use it:
- **Authorized Invocations** for authorizing a tool call, [use the
  `requiredAuth` field in a tool config][auth-invoke]
- **Authenticated Parameters** for using the value from a ODIC claim, [use the
  `authServices` field in a parameter config][auth-params]


## Specifying ID Tokens from Clients

After [configuring](#example) your `authServices` section, use a Toolbox SDK to
add your ID tokens to the header of a Tool invocation request. When specifying a
token you will provide a function (that returns an id). This function is called
when the tool is invoked. This allows you to cache and refresh the ID token as
needed. 

### Specifying tokens during load
{{< tabpane >}}
{{< tab header="LangChain" lang="Python" >}}
async def get_auth_token():
    # ... Logic to retrieve ID token (e.g., from local storage, OAuth flow)
    # This example just returns a placeholder. Replace with your actual token retrieval.
    return "YOUR_ID_TOKEN" # Placeholder

# for a single tool use:
authorized_tool = await toolbox.aload_tool("my-tool-name", auth_tokens={"my_auth": get_auth_token})

# for a toolset use: 
authorized_tools = await toolbox.aload_toolset("my-toolset-name", auth_tokens={"my_auth": get_auth_token})
{{< /tab >}}
{{< /tabpane >}}


### Specifying tokens for existing tools

{{< tabpane >}}
{{< tab header="LangChain" lang="Python" >}}
tools = await toolbox.aload_toolset()
# for a single token
auth_tools = [tool.add_auth_token("my_auth", get_auth_token) for tool in tools]
# OR, if multiple tokens are needed
authorized_tool = tools[0].add_auth_tokens({
  "my_auth1": get_auth1_token,
  "my_auth2": get_auth2_token,
}) 
{{< /tab >}}
{{< /tabpane >}}

## Kinds of Auth Services
