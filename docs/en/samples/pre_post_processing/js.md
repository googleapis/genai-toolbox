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
Coming soon.
{{% /tab %}}
{{% tab header="Langchain" text=true %}}
The following example demonstrates how to use `ToolboxClient` with LangChain's middleware to implement pre and post processing for tool calls.

```js
{{< include "js/langchain/agent.js" >}}
```

For more information, see the [LangChain Middleware documentation](https://docs.langchain.com/oss/javascript/langchain/middleware/custom#tool-call-monitoring).
{{% /tab %}}
{{< /tabpane >}}
