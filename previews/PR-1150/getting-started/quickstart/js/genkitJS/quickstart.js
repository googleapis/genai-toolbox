// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.


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
      console.log("Tool call request:", JSON.stringify(toolRequests, null, 2));
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
