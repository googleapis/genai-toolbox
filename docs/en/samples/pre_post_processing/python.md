---
title: "(Python) Pre and post processing"
type: docs
weight: 4
description: >
  How to add pre and post processing to your Python toolbox applications.
---

## Prerequisites

This tutorial assumes that you have set up a basic toolbox application as described in the [local quickstart](../../getting-started/local_quickstart).

This guide demonstrates how to implement these patterns in your Toolbox applications.

## Implementation

{{< tabpane persist=header >}}
{{% tab header="ADK" text=true %}}
The following example demonstrates how to use `ToolboxToolset` with ADK's pre and post processing hooks to implement pre and post processing for tool calls.

```py
{{< include "python/adk/agent.py" >}}
```
You can also add model-level (`before_model_callback`, `after_model_callback`) and agent-level (`before_agent_callback`, `after_agent_callback`) hooks to intercept messages at different stages of the execution loop. 

For more information, see the [ADK Callbacks documentation](https://google.github.io/adk-docs/callbacks/types-of-callbacks/).
{{% /tab %}}
{{% tab header="Langchain" text=true %}}
The following example demonstrates how to use `ToolboxClient` with LangChain's middleware to implement pre and post processing for tool calls.

```py
{{< include "python/langchain/agent.py" >}}
```

For more information, see the [LangChain Middleware documentation](https://docs.langchain.com/oss/python/langchain/middleware/custom#wrap-style-hooks).
You can also add model-level (`wrap_model`) and agent-level (`before_agent`, `after_agent`) hooks to intercept messages at different stages of the execution loop. See the [LangChain Middleware documentation](https://docs.langchain.com/oss/python/langchain/middleware/custom#wrap-style-hooks) for details on these additional hook types.
{{% /tab %}}
{{< /tabpane >}}
