// Copyright 2026 Google LLC
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
import { ChatVertexAI } from "@langchain/google-vertexai";
import { AgentExecutor, createToolCallingAgent } from "langchain/agents";
import { ChatPromptTemplate } from "@langchain/core/prompts";
import { tool } from "@langchain/core/tools";
import { fileURLToPath } from "url";
import process from "process";

const systemPrompt = `
  You're a helpful hotel assistant. You handle hotel searching, booking and
  cancellations. When the user searches for a hotel, mention it's name, id,
  location and price tier. Always mention hotel ids while performing any
  searches. This is very important for any operations. For any bookings or
  cancellations, please provide the appropriate confirmation. Be sure to
  update checkin or checkout dates if mentioned by the user.
  Don't ask for confirmations from the user.
`;

/**
 * Pre-processing: Enforce Business Rules
 * @param {string} name 
 * @param {object} args 
 * @returns {string|null} Error message if blocked, null otherwise
 */
function checkBusinessRules(name, args) {
  console.log(`POLICY CHECK: Intercepting '${name}'`);

  if (name === "update-hotel" && args.checkin_date && args.checkout_date) {
    try {
      const start = new Date(args.checkin_date);
      const end = new Date(args.checkout_date);
      const duration = (end - start) / (1000 * 60 * 60 * 24); // days

      if (duration > 14) {
        console.log("BLOCKED: Stay too long");
        return "Error: Maximum stay duration is 14 days."; 
      }
    } catch (e) {
      // Ignore invalid dates
    }
  }
  return null;
}

/**
 * Post-processing: Enrich Response
 * @param {string} name 
 * @param {any} result 
 * @returns {any} Enriched result
 */
function enrichResponse(name, result) {
  let content = result;
  if (typeof result === 'object' && result !== null && result.content) {
      content = result.content;
  }

  if (name === "book-hotel" && typeof content === 'string' && !content.includes("Error")) {
      const loyaltyBonus = 500;
      const enrichedContent = `Booking Confirmed!\n You earned ${loyaltyBonus} Loyalty Points with this stay.\n\nSystem Details: ${content}`;
      
      if (typeof result === 'object' && result !== null) {
          result.content = enrichedContent;
          return result;
      }
      return enrichedContent;
  }
  return result;
}

/**
 * Wraps a tool to add pre- and post-processing logic.
 * @param {import("@langchain/core/tools").StructuredTool} toolInstance
 */
function wrapToolWithBusinessLogic(toolInstance) {
  const originalInvoke = toolInstance.invoke.bind(toolInstance);

  toolInstance.invoke = async (input, config) => {
    // 1. Pre-processing
    const validationError = checkBusinessRules(toolInstance.name, input);
    if (validationError) {
      return validationError;
    }

    // 2. Execution
    const result = await originalInvoke(input, config);

    // 3. Post-processing
    return enrichResponse(toolInstance.name, result);
  };

  return toolInstance;
}

/**
 * Helper to run a single turn.
 */
async function runTurn(agentExecutor, input) {
  console.log(`\nUSER: '${input}'`);
  const result = await agentExecutor.invoke({ input });
  console.log("-".repeat(50));
  console.log("Final Client Response:");
  console.log(`AI: ${result.output}`);
}

async function main() {
  const client = new ToolboxClient("http://127.0.0.1:5000");
  
  try {
    const rawTools = await client.loadToolset("my-toolset");
    
    // Convert & Wrap tools
    const tools = rawTools
      .map(t => tool(t, {
          name: t.getName(),
          description: t.getDescription(),
          schema: t.getParamSchema()
      }))
      .map(wrapToolWithBusinessLogic);

    const model = new ChatVertexAI({
      model: "gemini-2.5-flash",
      temperature: 0,
    });

    const prompt = ChatPromptTemplate.fromMessages([
      ["system", systemPrompt],
      ["placeholder", "{chat_history}"],
      ["human", "{input}"],
      ["placeholder", "{agent_scratchpad}"],
    ]);

    const agent = createToolCallingAgent({ llm: model, tools, prompt });
    const agentExecutor = new AgentExecutor({ agent, tools, verbose: false });

    // Turn 1: Booking
    await runTurn(agentExecutor, "Book hotel with id 3.");
    
    // Turn 2: Policy Violation
    await runTurn(agentExecutor, "Update my hotel with id 3 with checkin date 2025-01-18 and checkout date 2025-02-10");

  } catch (error) {
    console.error("Error running agent:", error);
  }
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  main();
}

export { main };