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

1. In a new terminal, install the [SDK](https://www.npmjs.com/package/@toolbox-sdk/core).

    ```bash
    npm install @toolbox-sdk/core
    ```

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
{{< /tabpane >}}

1. Create a new file named `hotelAgent.js` and copy the following code to create an agent:

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
import { GoogleGenAI } from "@google/genai";
import { ToolboxClient } from "@toolbox-sdk/core";


const TOOLBOX_URL = "http://127.0.0.1:5000"; // Update if needed
const GOOGLE_API_KEY = 'enter your api here'; // Replace it with your API key

const prompt = `
You're a helpful hotel assistant. You handle hotel searching, booking, and
cancellations. When the user searches for a hotel, you MUST use the available tools to find information. Mention its name, id,
location and price tier. Always mention hotel id while performing any
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

function mapZodTypeToOpenAPIType(zodTypeName) {

    console.log(zodTypeName)
    const typeMap = {
        'ZodString': 'string',
        'ZodNumber': 'number',
        'ZodBoolean': 'boolean',
        'ZodArray': 'array',
        'ZodObject': 'object',
    };
    return typeMap[zodTypeName] || 'string';
}

async function runApplication() {
   
    const toolboxClient = new ToolboxClient(TOOLBOX_URL); 
    const toolboxTools = await toolboxClient.loadToolset("my-toolset");
    
    const geminiTools = [{
        functionDeclarations: toolboxTools.map(tool => {
            
            const schema = tool.getParamSchema();
            const properties = {};
            const required = [];

         
            for (const [key, param] of Object.entries(schema.shape)) {
                properties[key] = {
                        type: mapZodTypeToOpenAPIType(param.constructor.name),
                        description: param.description || '',
                    };
                required.push(key)
                }
            
            return {
                name: tool.getName(),
                description: tool.getDescription(),
                parameters: { type: 'object', properties, required },
            };
        })
    }];


    const genAI = new GoogleGenAI({ apiKey: GOOGLE_API_KEY });
    
    const chat = genAI.chats.create({
        model: "gemini-2.5-flash",
        config: {
            systemInstruction: prompt,
            tools: geminiTools,
        }
    });

    for (const query of queries) {
        
        let currentResult = await chat.sendMessage({ message: query });
        
        let finalResponseGiven = false
        while (!finalResponseGiven) {
            
            const response = currentResult;
            const functionCalls = response.functionCalls || [];

            if (functionCalls.length === 0) {
                console.log(response.text)
                finalResponseGiven = true;
            } else {
                const toolResponses = [];
                for (const call of functionCalls) {
                    const toolName = call.name
                    const toolToExecute = toolboxTools.find(t => t.getName() === toolName);
                    
                    if (toolToExecute) {
                        try {
                            const functionResult = await toolToExecute(call.args);
                            toolResponses.push({
                                functionResponse: { name: call.name, response: { result: functionResult } }
                            });
                        } catch (e) {
                            console.error(`Error executing tool '${toolName}':`, e);
                            toolResponses.push({
                                functionResponse: { name: call.name, response: { error: e.message } }
                            });
                        }
                    }
                }
                
                currentResult = await chat.sendMessage({ message: toolResponses });
            }
        }
        
    }
}

runApplication()
  .catch(console.error)
  .finally(() => console.log("\nApplication finished."));
{{< /tab >}}

{{< /tabpane >}}

1. Run your agent, and observe the results:

    ```sh
    node hotelAgent.js
    ```

{{< notice info >}}
For more information, visit the [JS SDK repo](https://github.com/googleapis/mcp-toolbox-sdk-js).
{{</ notice >}}
