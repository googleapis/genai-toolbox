---
title: "Javascript"
type: docs
weight: 2
description: >
  How to add pre- and post- processing to your Agents using JS.
---

## Prerequisites

This tutorial assumes that you have set up MCP Toolbox with a basic agent as described in the [local quickstart](../../getting-started/local_quickstart_js.md).

This guide demonstrates how to implement these patterns in your Toolbox applications.

## Implementation

{{< tabpane persist=header >}}
{{% tab header="ADK" text=true %}}
The following example demonstrates how to use the `beforeToolCallback` and `afterToolCallback` hooks in the ADK `LlmAgent` to implement pre and post processing logic.

{{< include "js/adk/agent.js" "js" >}}

You can also add model-level (`beforeModelCallback`, `afterModelCallback`) and agent-level (`beforeAgentCallback`, `afterAgentCallback`) hooks to intercept messages at different stages of the execution loop. 

For more information, see the [ADK Callbacks documentation](https://google.github.io/adk-docs/callbacks/types-of-callbacks/).
{{% /tab %}}
{{% tab header="Langchain" text=true %}}
The following example demonstrates how to use `ToolboxClient` with LangChain's middleware to implement pre and post processing for tool calls.

{{< include "js/langchain/agent.js" "js" >}}

You can also use the `wrapModelCall` hook to intercept messages before and after model calls.
You can also use [node-style hooks](https://docs.langchain.com/oss/javascript/langchain/middleware/custom#node-style-hooks) to intercept messages at the agent and model level.
See the [LangChain Middleware documentation](https://docs.langchain.com/oss/javascript/langchain/middleware/custom#tool-call-monitoring) for details on these additional hook types.

{{% /tab %}}
{{< /tabpane >}}

## Results

The output should look similar to the following.

{{< notice note >}}
The exact responses may vary due to the non-deterministic nature of LLMs and differences between orchestration frameworks.
{{< /notice >}}

```
AI: Booking Confirmed! You earned 500 Loyalty Points with this stay.

AI: Error: Maximum stay duration is 14 days.
```
