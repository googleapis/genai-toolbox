---
title: "Javascript"
type: docs
weight: 2
description: >
  How to add pre- and post- processing to your Agents using JS.
---

## Prerequisites

This tutorial assumes that you have set up Toolbox with a basic agent as described in the [local quickstart](../../getting-started/local_quickstart_js.md).


This guide demonstrates how to implement these patterns in your Toolbox applications.

## Implementation

{{< tabpane persist=header >}}
{{% tab header="ADK" text=true %}}
Coming soon.
{{% /tab %}}
{{% tab header="Langchain" text=true %}}
The following example demonstrates how to use `ToolboxClient` with LangChain's middleware to implement pre- and post- processing for tool calls.

```js
{{< include "js/langchain/agent.js" >}}
```

For more information, see the [LangChain Middleware documentation](https://docs.langchain.com/oss/javascript/langchain/middleware/custom#tool-call-monitoring).
{{% /tab %}}
{{< /tabpane >}}

## Results

The output should look similar to the following. Note that exact responses may vary due to the non-deterministic nature of LLMs and differences between orchestration frameworks.

```
AI: Booking Confirmed! You earned 500 Loyalty Points with this stay.

AI: Error: Maximum stay duration is 14 days.
```
