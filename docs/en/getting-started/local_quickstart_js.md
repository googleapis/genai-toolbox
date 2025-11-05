---
title: "JS Quickstart (Local)"
type: docs
weight: 3
description: >
  How to get started running Toolbox locally with [JavaScript](https://github.com/googleapis/mcp-toolbox-sdk-js), PostgreSQL, and orchestration frameworks such as [LangChain](https://js.langchain.com/docs/introduction/), [GenkitJS](https://genkit.dev/docs/get-started/),  [LlamaIndex](https://ts.llamaindex.ai/) and [GoogleGenAI](https://github.com/googleapis/js-genai).
---

## Before you begin

This guide assumes you have already done the following:

1. Installed [Node.js (v18 or higher)].
1. Installed [PostgreSQL 16+ and the `psql` client][install-postgres].

[Node.js (v18 or higher)]: https://nodejs.org/
[install-postgres]: https://www.postgresql.org/download/

### Cloud Setup (Optional)

{{< regionInclude "quickstart/shared/cloud_setup.md" "cloud_setup" >}}

## Step 1: Set up your database

{{< regionInclude "quickstart/shared/database_setup.md" "database_setup" >}}

## Step 2: Install and configure Toolbox

{{< regionInclude "quickstart/shared/configure_toolbox.md" "configure_toolbox" >}}

## Step 3: Connect your agent to Toolbox

In this section, we will write and run an agent that will load the Tools
from Toolbox.

1. (Optional) Initialize a Node.js project:

    ```bash
    npm init -y
    ```

1. In a new terminal, install the
   SDK package.
   {{< tabpane persist=header >}}
{{< tab header="LangChain" lang="bash" >}}
npm install @toolbox-sdk/core
{{< tab header="GenkitJS" lang="bash" >}}
npm install @toolbox-sdk/core
{{< /tab >}}
{{< tab header="LlamaIndex" lang="bash" >}}
npm install @toolbox-sdk/core
{{< /tab >}}
{{< tab header="GoogleGenAI" lang="bash" >}}
npm install @toolbox-sdk/core
{{< /tab >}}
{{< tab header="adk" lang="bash" >}}
npm install @toolbox-sdk/adk
{{< /tab >}}
{{< /tabpane >}}

1. Install other required dependencies

   {{< tabpane persist=header >}}
{{< tab header="LangChain" lang="bash" >}}
npm install langchain @langchain/google-genai
{{< /tab >}}
{{< tab header="GenkitJS" lang="bash" >}}
npm install genkit @genkit-ai/googleai
{{< /tab >}}
{{< tab header="LlamaIndex" lang="bash" >}}
npm install llamaindex @llamaindex/google @llamaindex/workflow
{{< /tab >}}
{{< tab header="GoogleGenAI" lang="bash" >}}
npm install @google/genai
{{< /tab >}}
{{< tab header="adk" lang="bash" >}}
npm install @google/adk
{{< /tab >}}
{{< /tabpane >}}

1. Create a new file named `hotelAgent.js` and copy the following code to create
   an agent:

    {{< tabpane persist=header >}}
{{< tab header="LangChain" lang="js" >}}

{{< include "quickstart/js/langchain/quickstart.js" >}}

{{< /tab >}}

{{< tab header="GenkitJS" lang="js" >}}

{{< include "quickstart/js/genkit/quickstart.js" >}}

{{< /tab >}}

{{< tab header="LlamaIndex" lang="js" >}}

{{< include "quickstart/js/llamaindex/quickstart.js" >}}

{{< /tab >}}

{{< tab header="GoogleGenAI" lang="js" >}}

{{< include "quickstart/js/genAI/quickstart.js" >}}

{{< /tab >}}

{{< tab header="adk" lang="js" >}}

{{< include "quickstart/js/adk/quickstart.js" >}}

{{< /tab >}}

{{< /tabpane >}}

1. Run your agent, and observe the results:

    ```sh
    node hotelAgent.js
    ```

{{< notice info >}}
For more information, visit the [JS SDK
repo](https://github.com/googleapis/mcp-toolbox-sdk-js).
{{</ notice >}}
