---
title: "(JS) Pre and post processing"
type: docs
weight: 5
description: >
  How to add pre and post processing to your JS toolbox applications.
---

## Prerequisites

This tutorial assumes that you have set up a basic toolbox application as described in the [local quickstart](../../getting-started/local_quickstart_js).

This guide demonstrates how to implement these patterns in your Toolbox applications.

## Implementation

{{< tabpane persist=header >}}
{{% tab header="ADK" text=true %}}
The following example demonstrates how to use the `beforeToolCallback` and `afterToolCallback` hooks in the ADK `LlmAgent` to implement pre and post processing logic.

```js
{{< include "js/adk/agent.js" >}}
```

You can also add model-level (`beforeModelCallback`, `afterModelCallback`) and agent-level (`beforeAgentCallback`, `afterAgentCallback`) hooks to intercept messages at different stages of the execution loop. 

For more information, see the [ADK Callbacks documentation](https://google.github.io/adk-docs/callbacks/types-of-callbacks/).
{{% /tab %}}
{{% tab header="Langchain" text=true %}}
The following example demonstrates how to use `ToolboxClient` with LangChain's middleware to implement pre and post processing for tool calls.

```js
{{< include "js/langchain/agent.js" >}}
```

For more information, see the [LangChain Middleware documentation](https://js.langchain.com/docs/introduction/middleware).
You can also add model-level (`wrap_model`) and agent-level (`before_agent`, `after_agent`) hooks to intercept messages at different stages of the execution loop. See the [LangChain Middleware documentation](https://js.langchain.com/docs/introduction/middleware) for details on these additional hook types.
{{% /tab %}}
{{< /tabpane >}}
