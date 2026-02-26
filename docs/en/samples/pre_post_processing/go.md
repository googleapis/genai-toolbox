---
title: "Go"
type: docs
weight: 3
description: >
  How to add pre- and post- processing to your Agents using Go.
---

## Prerequisites

This tutorial assumes that you have set up MCP Toolbox with a basic agent as described in the [local quickstart](../../getting-started/local_quickstart_go.md).

This guide demonstrates how to implement these patterns in your Toolbox applications.

## Implementation

{{< tabpane persist=header >}}
{{% tab header="ADK" text=true %}}
The following example demonstrates how to use the `beforeToolCallback` and `afterToolCallback` hooks in the ADK `LlmAgent` to implement pre and post processing logic.

{{< include "go/adk/agent.go" "go" >}}

You can also add model-level (`beforeModelCallback`, `afterModelCallback`) and agent-level (`beforeAgentCallback`, `afterAgentCallback`) hooks to intercept messages at different stages of the execution loop.

For more information, see the [ADK Callbacks documentation](https://google.github.io/adk-docs/callbacks/types-of-callbacks/).{{% /tab %}}
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
