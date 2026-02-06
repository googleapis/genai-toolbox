---
title: "Python"
type: docs
weight: 1
description: >
  How to add pre and post processing to your Python toolbox applications.
---

## Prerequisites

This tutorial assumes that you have set up a basic toolbox application as described in the [local quickstart](../../getting-started/local_quickstart).

This guide demonstrates how to implement these patterns in your Toolbox applications.

## Implementation

{{< tabpane persist=header >}}
{{% tab header="ADK" text=true %}}
Coming soon.
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

## Results

The output should look similar to the following. Note that exact responses may vary due to the non-deterministic nature of LLMs and differences between orchestration frameworks.

```
AI: Booking Confirmed! You earned 500 Loyalty Points with this stay.

AI: Error: Maximum stay duration is 14 days.
```