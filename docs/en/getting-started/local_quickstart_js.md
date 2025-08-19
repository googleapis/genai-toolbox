---
title: "JS Quickstart (Local)"
type: docs
weight: 3
description: >
  How to get started running Toolbox locally with [JavaScript](https://github.com/googleapis/mcp-toolbox-sdk-js), PostgreSQL, and orchestration frameworks such as [LangChain](https://js.langchain.com/docs/introduction/), [GenkitJS](https://genkit.dev/docs/get-started/), and [LlamaIndex](https://ts.llamaindex.ai/).
---

## Before you begin

This guide assumes you have already done the following:

1. Installed [Node.js (v18 or higher)].
1. Installed [PostgreSQL 16+ and the `psql` client][install-postgres].

[Node.js (v18 or higher)]: https://nodejs.org/
[install-postgres]: https://www.postgresql.org/download/

### Cloud Setup (Optional)
{{< snippet "local_quickstart.md" "cloud_setup" >}}

{{< snippet "test.py" "demo" "py" >}}

## Step 1: Set up your database
{{< snippet "local_quickstart.md" "database_setup" >}}

## Step 2: Install and configure Toolbox
{{< snippet "local_quickstart.md" "configure_toolbox" >}}

## Step 3: Connect your agent to Toolbox

In this section, we will write and run an agent that will load the Tools
from Toolbox.

1. (Optional) Initialize a Node.js project:

    ```bash
    npm init -y
    ```

1. In a new terminal, install the [SDK](https://www.npmjs.com/package/@toolbox-sdk/core).

    ```bash
    npm install @toolbox-sdk/core
    ```

1. Install other required dependencies

   {{< tabpane persist=header >}}
{{< tab header="LangChain" lang="bash" >}}
npm install langchain @langchain/google-vertexai
{{< /tab >}}
{{< tab header="GenkitJS" lang="bash" >}}
npm install genkit @genkit-ai/vertexai
{{< /tab >}}
{{< tab header="LlamaIndex" lang="bash" >}}
npm install llamaindex @llamaindex/google @llamaindex/workflow
{{< /tab >}}
{{< /tabpane >}}

1. Create a new file named `hotelAgent.js` and copy the following code to create an agent:

    {{< tabpane persist=header >}}
{{< tab header="LangChain" lang="js" >}}

import { ChatVertexAI } from "@langchain/google-vertexai";
import { ToolboxClient } from "@toolbox-sdk/core";
import { tool } from "@langchain/core/tools";
import { createReactAgent } from "@langchain/langgraph/prebuilt";
import { MemorySaver } from "@langchain/langgraph";

// Replace it with your API key
process.env.GOOGLE_API_KEY = 'your-api-key';

const prompt = `
You're a helpful hotel assistant. You handle hotel searching, booking, and
cancellations. When the user searches for a hotel, mention its name, id,
location and price tier. Always mention hotel ids while performing any
searches. This is very important for any operations. For any bookings or
cancellations, please provide the appropriate confirmation. Be sure to
update checkin or checkout dates if mentioned by the user.
Don't ask for confirmations from the user.
`;

const queries = [
  "Find hotels in Basel with Basel in its name.",
  "Can you book the Hilton Basel for me?",
  "Oh wait, this is too expensive. Please cancel it and book the Hyatt Regency instead.",
  "My check in dates would be from April 10, 2024 to April 19, 2024.",
];

async function runApplication() {
  const model = new ChatVertexAI({
    model: "gemini-2.0-flash",
  });

  const client = new ToolboxClient("http://127.0.0.1:5000");
  const toolboxTools = await client.loadToolset("my-toolset");

  // Define the basics of the tool: name, description, schema and core logic
  const getTool = (toolboxTool) => tool(toolboxTool, {
    name: toolboxTool.getName(),
    description: toolboxTool.getDescription(),
    schema: toolboxTool.getParamSchema()
  });
  const tools = toolboxTools.map(getTool);

  const agent = createReactAgent({
    llm: model,
    tools: tools,
    checkpointer: new MemorySaver(),
    systemPrompt: prompt,
  });

  const langGraphConfig = {
    configurable: {
        thread_id: "test-thread",
    },
  };

  for (const query of queries) {
    const agentOutput = await agent.invoke(
    {
        messages: [
        {
            role: "user",
            content: query,
        },
        ],
        verbose: true,
    },
    langGraphConfig
    );
    const response = agentOutput.messages[agentOutput.messages.length - 1].content;
    console.log(response);
  }
}

runApplication()
  .catch(console.error)
  .finally(() => console.log("\nApplication finished."));

{{< /tab >}}

{{< tab header="GenkitJS" lang="js" >}}

import { ToolboxClient } from "@toolbox-sdk/core";
import { genkit } from "genkit";
import { googleAI } from '@genkit-ai/googleai';

// Replace it with your API key
process.env.GOOGLE_API_KEY = 'your-api-key';

const systemPrompt = `
You're a helpful hotel assistant. You handle hotel searching, booking, and
cancellations. When the user searches for a hotel, mention its name, id,
location and price tier. Always mention hotel ids while performing any
searches. This is very important for any operations. For any bookings or
cancellations, please provide the appropriate confirmation. Be sure to
update checkin or checkout dates if mentioned by the user.
Don't ask for confirmations from the user.
`;

const queries = [
  "Find hotels in Basel with Basel in its name.",
  "Can you book the Hilton Basel for me?",
  "Oh wait, this is too expensive. Please cancel it and book the Hyatt Regency instead.",
  "My check in dates would be from April 10, 2024 to April 19, 2024.",
];

async function run() {
  const toolboxClient = new ToolboxClient("http://127.0.0.1:5000");

  const ai = genkit({
    plugins: [
      googleAI({
        apiKey: process.env.GEMINI_API_KEY || process.env.GOOGLE_API_KEY
      })
    ],
    model: googleAI.model('gemini-2.0-flash'),
  });

  const toolboxTools = await toolboxClient.loadToolset("my-toolset");
  const toolMap = Object.fromEntries(
    toolboxTools.map((tool) => {
      const definedTool = ai.defineTool(
        {
          name: tool.getName(),
          description: tool.getDescription(),
          inputSchema: tool.getParamSchema(),
        },
        tool
      );
      return [tool.getName(), definedTool];
    })
  );
  const tools = Object.values(toolMap);

  let conversationHistory = [{ role: "system", content: [{ text: systemPrompt }] }];

  for (const query of queries) {
    conversationHistory.push({ role: "user", content: [{ text: query }] });
    const response = await ai.generate({
      messages: conversationHistory,
      tools: tools,
    });
    conversationHistory.push(response.message);

    const toolRequests = response.toolRequests;
    if (toolRequests?.length > 0) {
      // Execute tools concurrently and collect their responses.
      const toolResponses = await Promise.all(
        toolRequests.map(async (call) => {
          try {
            const toolOutput = await toolMap[call.name].invoke(call.input);
            return { role: "tool", content: [{ toolResponse: { name: call.name, output: toolOutput } }] };
          } catch (e) {
            console.error(`Error executing tool ${call.name}:`, e);
            return { role: "tool", content: [{ toolResponse: { name: call.name, output: { error: e.message } } }] };
          }
        })
      );
      
      conversationHistory.push(...toolResponses);
      
      // Call the AI again with the tool results.
      response = await ai.generate({ messages: conversationHistory, tools });
      conversationHistory.push(response.message);
    }
    
    console.log(response.text);
  }
}

run();
{{< /tab >}}

{{< tab header="LlamaIndex" lang="js" >}}

import { gemini, GEMINI_MODEL } from "@llamaindex/google";
import { agent } from "@llamaindex/workflow";
import { createMemory, staticBlock, tool } from "llamaindex";
import { ToolboxClient } from "@toolbox-sdk/core";

const TOOLBOX_URL = "http://127.0.0.1:5000"; // Update if needed
process.env.GOOGLE_API_KEY = 'your-api-key'; // Replace it with your API key

const prompt = `

You're a helpful hotel assistant. You handle hotel searching, booking and cancellations.
When the user searches for a hotel, mention its name, id, location and price tier.
Always mention hotel ids while performing any searches â€” this is very important for operations.
For any bookings or cancellations, please provide the appropriate confirmation.
Update check-in or check-out dates if mentioned by the user.
Don't ask for confirmations from the user.

`;

const queries = [
  "Find hotels in Basel with Basel in its name.",
  "Can you book the Hilton Basel for me?",
  "Oh wait, this is too expensive. Please cancel it and book the Hyatt Regency instead.",
  "My check in dates would be from April 10, 2024 to April 19, 2024.",
];

async function main() {
  // Connect to MCP Toolbox
  const client = new ToolboxClient(TOOLBOX_URL);
  const toolboxTools = await client.loadToolset("my-toolset");
  const tools = toolboxTools.map((toolboxTool) => {
    return tool({
      name: toolboxTool.getName(),
      description: toolboxTool.getDescription(),
      parameters: toolboxTool.getParamSchema(),
      execute: toolboxTool,
    });
  });

  // Initialize LLM
  const llm = gemini({
    model: GEMINI_MODEL.GEMINI_2_0_FLASH,
    apiKey: process.env.GOOGLE_API_KEY,
  });

  const memory = createMemory({
    memoryBlocks: [
      staticBlock({
        content: prompt,
      }),
    ],
  });

  // Create the Agent
  const myAgent = agent({
    tools: tools,
    llm,
    memory,
    systemPrompt: prompt,
  });

  for (const query of queries) {
    const result = await myAgent.run(query);
    const output = result.data.result;

    console.log(`\nUser: ${query}`);
    if (typeof output === "string") {
      console.log(output.trim());
    } else if (typeof output === "object" && "text" in output) {
      console.log(output.text.trim());
    } else {
      console.log(JSON.stringify(output));
    }
  }
  //You may observe some extra logs during execution due to the run method provided by Llama.
  console.log("Agent run finished.");
}

main();

{{< /tab >}}

{{< /tabpane >}}

1. Run your agent, and observe the results:

    ```sh
    node hotelAgent.js
    ```

{{< notice info >}}
For more information, visit the [JS SDK repo](https://github.com/googleapis/mcp-toolbox-sdk-js).
{{</ notice >}}
