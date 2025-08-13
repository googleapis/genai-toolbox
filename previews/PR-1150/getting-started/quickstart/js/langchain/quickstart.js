import { ChatGoogleGenerativeAI } from "@langchain/google-genai";
import { ToolboxClient } from "@toolbox-sdk/core";
import { tool } from "@langchain/core/tools";
import { createReactAgent } from "@langchain/langgraph/prebuilt";
import { MemorySaver } from "@langchain/langgraph";

// // Replace it with your API key
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
  const model = new ChatGoogleGenerativeAI({
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
