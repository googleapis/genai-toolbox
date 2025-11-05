import { InMemoryRunner, LlmAgent } from '@google/adk';
import { ToolboxClient } from '@toolbox-sdk/adk';

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

async function main() {
  const userId = 'test_user';
  const client = new ToolboxClient('http://127.0.0.1:5000');
  const tools = await client.loadToolset("my-toolset");

  const rootAgent = new LlmAgent({
    name: 'hotel_agent',
    model: 'gemini-2.5-flash',
    description: 'Agent for hotel bookings and administration.',
    instruction: prompt,
    tools: tools,
  });

  const appName = rootAgent.name;
  const runner = new InMemoryRunner({ agent: rootAgent, appName });
  const session = await runner.sessionService.createSession({ appName, userId });

  for (query in queries) {
    await runPrompt(runner, userId, session.id, query);
  }
}

async function runPrompt(runner, userId, sessionId, prompt) {
  const content = { role: 'user', parts: [{ text: prompt }] };
  const stream = runner.runAsync({ userId, sessionId, newMessage: content });
  const responses = await Array.fromAsync(stream);
  const accumulatedResponse = responses
      .flatMap((e) => e.content?.parts?.map((p) => p.text) ?? [])
      .join('');

  console.log(accumulatedResponse);
}

main().catch(console.error);