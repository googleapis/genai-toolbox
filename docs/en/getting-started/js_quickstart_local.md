---
title: "Quickstart (Local) using JS SDK"
type: docs
weight: 2
description: >
  How to get started running Toolbox locally with JavaScript, PostgreSQL, and orchestration frameworks such as [LangChain](https://js.langchain.com/docs/introduction/), [LlamaIndex](https://www.llamaindex.ai/), or [GenkitJS](https://github.com/genkit-ai/genkit). This guide covers setup and integration for each framework using the JS SDK.
---

## Before you begin

This guide assumes you have already done the following:

1. Installed [Node.js (v18 or higher)]
2. Installed [PostgreSQL 16+ and the `psql` client][install-postgres]

### Cloud Setup (Optional)

If you plan to use **Google Cloud’s Vertex AI** with your agent (e.g., using Gemini or PaLM models), follow these one-time setup steps:

> 📚 Before you begin:
> - [Install the Google Cloud CLI]
> - [Set up Application Default Credentials (ADC)]

#### Set your project and enable Vertex AI

```bash
gcloud config set project YOUR_PROJECT_ID
gcloud services enable aiplatform.googleapis.com
```

[Node.js (v18 or higher)]: https://nodejs.org/
[install-postgres]: https://www.postgresql.org/download/
[Install the Google Cloud CLI]: https://cloud.google.com/sdk/docs/install
[Set up Application Default Credentials (ADC)]: https://cloud.google.com/docs/authentication/set-up-adc-local-dev-environment

---

## Step 1: Set up your database

In this section, we will create a database, insert some data that needs to be
accessed by our agent, and create a database user for Toolbox to connect with.

1. Connect to postgres using the `psql` command:

    ```bash
    psql -h 127.0.0.1 -U postgres
    ```

    Here, `postgres` denotes the default postgres superuser.

    {{< notice info >}}

#### **Having trouble connecting?**

* **Password Prompt:** If you are prompted for a password for the `postgres`
  user and do not know it (or a blank password doesn't work), your PostgreSQL
  installation might require a password or a different authentication method.
* **`FATAL: role "postgres" does not exist`:** This error means the default
  `postgres` superuser role isn't available under that name on your system.
* **`Connection refused`:** Ensure your PostgreSQL server is actually running.
  You can typically check with `sudo systemctl status postgresql` and start it
  with `sudo systemctl start postgresql` on Linux systems.

<br/>

#### **Common Solution**

For password issues or if the `postgres` role seems inaccessible directly, try
switching to the `postgres` operating system user first. This user often has
permission to connect without a password for local connections (this is called
peer authentication).

```bash
sudo -i -u postgres
psql -h 127.0.0.1
```

Once you are in the `psql` shell using this method, you can proceed with the
database creation steps below. Afterwards, type `\q` to exit `psql`, and then
`exit` to return to your normal user shell.

If desired, once connected to `psql` as the `postgres` OS user, you can set a
password for the `postgres` *database* user using: `ALTER USER postgres WITH
PASSWORD 'your_chosen_password';`. This would allow direct connection with `-U
postgres` and a password next time.
    {{< /notice >}}

1. Create a new database and a new user:

    {{< notice tip >}}
  For a real application, it's best to follow the principle of least permission
  and only grant the privileges your application needs.
    {{< /notice >}}

    ```sql
      CREATE USER toolbox_user WITH PASSWORD 'my-password';

      CREATE DATABASE toolbox_db;
      GRANT ALL PRIVILEGES ON DATABASE toolbox_db TO toolbox_user;

      ALTER DATABASE toolbox_db OWNER TO toolbox_user;
    ```

1. End the database session:

    ```bash
    \q
    ```

    (If you used `sudo -i -u postgres` and then `psql`, remember you might also
    need to type `exit` after `\q` to leave the `postgres` user's shell
    session.)

1. Connect to your database with your new user:

    ```bash
    psql -h 127.0.0.1 -U toolbox_user -d toolbox_db
    ```

1. Create a table using the following command:

    ```sql
    CREATE TABLE hotels(
      id            INTEGER NOT NULL PRIMARY KEY,
      name          VARCHAR NOT NULL,
      location      VARCHAR NOT NULL,
      price_tier    VARCHAR NOT NULL,
      checkin_date  DATE    NOT NULL,
      checkout_date DATE    NOT NULL,
      booked        BIT     NOT NULL
    );
    ```

1. Insert data into the table.

    ```sql
    INSERT INTO hotels(id, name, location, price_tier, checkin_date, checkout_date, booked)
    VALUES 
      (1, 'Hilton Basel', 'Basel', 'Luxury', '2024-04-22', '2024-04-20', B'0'),
      (2, 'Marriott Zurich', 'Zurich', 'Upscale', '2024-04-14', '2024-04-21', B'0'),
      (3, 'Hyatt Regency Basel', 'Basel', 'Upper Upscale', '2024-04-02', '2024-04-20', B'0'),
      (4, 'Radisson Blu Lucerne', 'Lucerne', 'Midscale', '2024-04-24', '2024-04-05', B'0'),
      (5, 'Best Western Bern', 'Bern', 'Upper Midscale', '2024-04-23', '2024-04-01', B'0'),
      (6, 'InterContinental Geneva', 'Geneva', 'Luxury', '2024-04-23', '2024-04-28', B'0'),
      (7, 'Sheraton Zurich', 'Zurich', 'Upper Upscale', '2024-04-27', '2024-04-02', B'0'),
      (8, 'Holiday Inn Basel', 'Basel', 'Upper Midscale', '2024-04-24', '2024-04-09', B'0'),
      (9, 'Courtyard Zurich', 'Zurich', 'Upscale', '2024-04-03', '2024-04-13', B'0'),
      (10, 'Comfort Inn Bern', 'Bern', 'Midscale', '2024-04-04', '2024-04-16', B'0');
    ```

1. End the database session:

    ```bash
    \q
    ```
---

## Step 2: Install and configure Toolbox

In this section, we will download Toolbox, configure our tools in a
`tools.yaml`, and then run the Toolbox server.

1. Download the latest version of Toolbox as a binary:

    {{< notice tip >}}
  Select the
  [correct binary](https://github.com/googleapis/genai-toolbox/releases)
  corresponding to your OS and CPU architecture.
    {{< /notice >}}
    <!-- {x-release-please-start-version} -->
    ```bash
    export OS="linux/amd64" # one of linux/amd64, darwin/arm64, darwin/amd64, or windows/amd64
    curl -O https://storage.googleapis.com/genai-toolbox/v0.8.0/$OS/toolbox
    ```
    <!-- {x-release-please-end} -->

1. Make the binary executable:

    ```bash
    chmod +x toolbox
    ```

1. Write the following into a `tools.yaml` file. Be sure to update any fields
   such as `user`, `password`, or `database` that you may have customized in the
   previous step.

    {{< notice tip >}}
  In practice, use environment variable replacement with the format ${ENV_NAME}
  instead of hardcoding your secrets into the configuration file.
    {{< /notice >}}

    ```yaml
    sources:
      my-pg-source:
        kind: postgres
        host: 127.0.0.1
        port: 5432
        database: toolbox_db
        user: ${USER_NAME}
        password: ${PASSWORD}
    tools:
      search-hotels-by-name:
        kind: postgres-sql
        source: my-pg-source
        description: Search for hotels based on name.
        parameters:
          - name: name
            type: string
            description: The name of the hotel.
        statement: SELECT * FROM hotels WHERE name ILIKE '%' || $1 || '%';
      search-hotels-by-location:
        kind: postgres-sql
        source: my-pg-source
        description: Search for hotels based on location.
        parameters:
          - name: location
            type: string
            description: The location of the hotel.
        statement: SELECT * FROM hotels WHERE location ILIKE '%' || $1 || '%';
      book-hotel:
        kind: postgres-sql
        source: my-pg-source
        description: >-
           Book a hotel by its ID. If the hotel is successfully booked, returns a NULL, raises an error if not.
        parameters:
          - name: hotel_id
            type: string
            description: The ID of the hotel to book.
        statement: UPDATE hotels SET booked = B'1' WHERE id = $1;
      update-hotel:
        kind: postgres-sql
        source: my-pg-source
        description: >-
          Update a hotel's check-in and check-out dates by its ID. Returns a message
          indicating  whether the hotel was successfully updated or not.
        parameters:
          - name: hotel_id
            type: string
            description: The ID of the hotel to update.
          - name: checkin_date
            type: string
            description: The new check-in date of the hotel.
          - name: checkout_date
            type: string
            description: The new check-out date of the hotel.
        statement: >-
          UPDATE hotels SET checkin_date = CAST($2 as date), checkout_date = CAST($3
          as date) WHERE id = $1;
      cancel-hotel:
        kind: postgres-sql
        source: my-pg-source
        description: Cancel a hotel by its ID.
        parameters:
          - name: hotel_id
            type: string
            description: The ID of the hotel to cancel.
        statement: UPDATE hotels SET booked = B'0' WHERE id = $1;
   toolsets:
      my-toolset:
        - search-hotels-by-name
        - search-hotels-by-location
        - book-hotel
        - update-hotel
        - cancel-hotel
    ```

    For more info on tools, check out the `Resources` section of the docs.

1. Run the Toolbox server, pointing to the `tools.yaml` file created earlier:

    ```bash
    ./toolbox --tools-file "tools.yaml"
    ```

---

## Step 3: Set up your Node.js project

In this step, you'll create a new folder for your project, initialize it with `npm`, and install the required dependencies.

1. Create a new folder for your project and navigate into it:

    ```bash
    mkdir my-agent-app
    cd my-agent-app
    ```

1. Initialize a new Node.js project:

    ```bash
    npm init -y
    ```

1. Create a new file in the root directory:
   
    ```bash
    touch index.js
    ```

**Suggestion:**  
> We recommend the following folder structure for your project:
>
> ```
> my-agent-app/
> ├── .env                  # For environment variables (e.g., API keys)
> ├── index.js              # Main application file
> ├── package.json
> └── node_modules/
> ```
>
> - Place your main code in `index.js`.
> - Store sensitive information like API keys in a `.env` file (never commit this to version control).

---

## Step 4: Install dependencies for your orchestration framework

Depending on which orchestration framework you want to use, install the relevant dependencies:

{{< tabpane persist=header >}}
{{< tab header="LangChain" lang="bash" >}}
npm install langchain @genai-toolbox/sdk @langchain/google-vertexai dotenv
{{< /tab >}}
{{< tab header="LlamaIndex" lang="bash" >}}
npm install @llamaindex/core @llamaindex/llms-google-genai @genai-toolbox/sdk dotenv
{{< /tab >}}
{{< tab header="GenkitJS" lang="bash" >}}
npm install @toolbox-sdk/core genkit @genkit-ai/vertexai dotenv
{{< /tab >}}
{{< /tabpane >}}

---

## Step 5: Add your application code

Below are sample code templates for each framework. Replace the sample code with your actual implementation as needed.

{{< tabpane persist=header >}}
{{< tab header="LangChain" lang="js" >}}
// index.js

import "dotenv/config";
import { ChatVertexAI } from "@langchain/google-vertexai";
import { ToolboxClient } from "@toolbox-sdk/core";
import { tool } from "@langchain/core/tools";
import { HumanMessage, ToolMessage } from "@langchain/core/messages";

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
  console.log("Starting hotel agent...");

  const model = new ChatVertexAI({
    model: "gemini-2.0-flash-001",
    temperature: 0,
  });

  const client = new ToolboxClient("http://127.0.0.1:5000");
  const toolboxTools = await client.loadToolset("my-toolset");

  console.log(`Loaded ${toolboxTools.length} tools from Toolbox`);

  const tools = toolboxTools
    .map((t) => {
      return tool(t, {
        name: t.toolName,
        description: t.description,
        schema: t.params,
      });
    })
    .filter(Boolean);

  const modelWithTools = model.bindTools(tools);

  let messages = [new HumanMessage(prompt)];

  for (const query of queries) {
    console.log(`\nUser: ${query}`);

    messages.push(new HumanMessage(query));

    for (let step = 0; step < 5; step++) {
      const response = await modelWithTools.invoke(messages);

      if (!response.tool_calls || response.tool_calls.length === 0) {
        console.log("Agent:", response.content);
        messages.push(response);
        break;
      }

      console.log("Agent decided to use tools:", response.tool_calls);
      messages.push(response);

      const toolMessages = await Promise.all(
        response.tool_calls.map(async (call) => {
          const toolToCall = tools.find((t) => t.name === call.name);
          if (!toolToCall) {
            return new ToolMessage({
              content: `Error: Tool ${call.name} not found`,
              tool_call_id: call.id,
            });
          }
          try {
            const result = await toolToCall.invoke(call.args);
            return new ToolMessage({
              content: JSON.stringify(result ?? "No result returned."),
              tool_call_id: call.id,
            });
          } catch (e) {
            return new ToolMessage({
              content: `Error: ${e.message}`,
              tool_call_id: call.id,
            });
          }
        })
      );

      messages.push(...toolMessages);
    }
  }
  if (client.close) {
    await client.close();
  }
}

runApplication()
  .catch(console.error)
  .finally(() => console.log("\nApplication finished."));

{{< /tab >}}

{{< tab header="LlamaIndex" lang="js" >}}
// index.js

import "dotenv/config";
import { LlamaIndexAgent } from "@llamaindex/core";
import { GoogleGenAI } from "@llamaindex/llms-google-genai";
import { ToolboxClient } from "@genai-toolbox/sdk";

// Sample prompt and queries
const prompt = `
You're a helpful hotel assistant. You handle hotel searching, booking, and cancellations.
... (same as above) ...
`;

const queries = [
  "Find hotels in Basel with Basel in its name.",
  // ...more queries...
];

async function runApplication() {
  const llm = new GoogleGenAI({
    model: "gemini-2.0-flash-001",
    // Add any required config here
  });

  const client = new ToolboxClient("http://127.0.0.1:5000");
  const tools = await client.loadToolset("my-toolset");

  const agent = new LlamaIndexAgent({
    llm,
    tools,
    systemPrompt: prompt,
  });

  for (const query of queries) {
    const response = await agent.run(query);
    console.log(response);
  }

  if (client.close) await client.close();
}

runApplication().catch(console.error);
{{< /tab >}}

{{< tab header="GenkitJS" lang="js" >}}
import { ToolboxClient } from "@toolbox-sdk/core";
import { genkit } from "genkit";
import { vertexAI } from "@genkit-ai/vertexai";
import { z } from "zod";

const toolboxClient = new ToolboxClient("http://127.0.0.1:5000");

const ai = genkit({
  plugins: [vertexAI({ location: "us-central1", projectId: process.env.PROJECT_ID })],
});

const systemPrompt = `
You're a helpful hotel assistant. You handle hotel searching, booking and cancellations.
When the user searches for a hotel, mention its name, ID, location and price tier.
Always mention hotel ID while performing any operations. This is very important for any operations.
For any bookings or cancellations, please provide the appropriate confirmation.
Be sure to update checkin or checkout dates if mentioned by the user.
Don't ask for confirmations from the user.
`;

const queries = [
  "Find hotels in Bern with Bern in it's name.",
  "Please book the hotel  Best Western Bern for me.",
  "This is too expensive. Please cancel it.",
  "Please book Comfort Inn Bern for me",
  "My check in dates for my booking would be from April 10, 2024 to April 19, 2024.",
];

async function run() {
  let tools;
  try {
    tools = await toolboxClient.loadToolset("my-toolset");
  } catch {
    return;
  }

  const toolboxTools = await toolboxClient.loadToolset("my-toolset");

  const toolMap = {};
  for (const tool of toolboxTools) {
    let inputSchema;
    switch (tool.getName()) {
      case "search-hotels-by-name":
        inputSchema = z.object({ name: z.string() });
        break;
      case "search-hotels-by-location":
        inputSchema = z.object({ location: z.string() });
        break;
      case "book-hotel":
        inputSchema = z.object({ hotel_id: z.string() });
        break;
      case "update-hotel":
        inputSchema = z.object({
          hotel_id: z.string(),
          checkin_date: z.string(),
          checkout_date: z.string(),
        });
        break;
      case "cancel-hotel":
        inputSchema = z.object({ hotel_id: z.string() });
        break;
      default:
        inputSchema = z.object({});
    }

    const definedTool = ai.defineTool(
      {
        name: tool.getName(),
        description: tool.getDescription(),
        inputSchema,
      },
      tool
    );

    toolMap[tool.getName()] = definedTool;
  }

  let conversationHistory = [
    {
      role: "system",
      content: [{ text: systemPrompt }],
    },
  ];

  for (const userQuery of queries) {
    console.log(`\n👤 User: "${userQuery}"`);
    conversationHistory.push({
      role: "user",
      content: [{ text: userQuery }],
    });

    const response = await ai.generate({
      model: vertexAI.model("gemini-2.5-flash"),
      messages: conversationHistory,
      tools: Object.values(toolMap),
    });

    let content = [],
      functionCalls = [];

    if (response.toolRequests?.length) {
      functionCalls = response.toolRequests;
      content = response.content || [{ text: response.text || "" }];
    } else if (response.candidates?.length) {
      content = response.candidates[0].content;
      functionCalls = content.filter((part) => part.functionCall);
    } else {
      content = [{ text: response.text || response.output || "No response text found" }];
    }

    conversationHistory.push({
      role: "model",
      content,
    });

    if (functionCalls.length > 0) {
      for (const call of functionCalls) {
        const toolName =
          call.functionCall?.name || call.toolRequest?.name || call.name;
        const toolArgs =
          call.functionCall?.args || call.toolRequest?.input || call.input;

        const tool = toolMap[toolName];
        if (!tool) continue;

        try {
          const toolResponse = await tool.invoke(toolArgs);

          conversationHistory.push({
            role: "function",
            content: [
              {
                functionResponse: {
                  name: toolName,
                  response: toolResponse,
                },
              },
            ],
          });

          if (toolName.includes("search-hotels")) {
            if (Array.isArray(toolResponse) && toolResponse.length > 0) {
              const hotelList = toolResponse
                .map(
                  (h) =>
                    `*   Hotel Name: ${h.name}, ID: ${h.hotel_id}, Location: ${h.location}, Price Tier: ${h.price_tier}`
                )
                .join("\n");
              console.log(`🤖 Hotel Agent: I found these hotels in Bern:\n\n${hotelList}`);
            } else {
              console.log("🤖 Hotel Agent: No hotels found.");
            }
          }
        } catch {}
      }

      const finalResponse = await ai.generate({
        model: vertexAI.model("gemini-2.5-flash"),
        messages: conversationHistory,
        tools: Object.values(toolMap),
      });

      const finalMessage =
        finalResponse.text || finalResponse.output || "No final response";

      conversationHistory.push({
        role: "model",
        content: [{ text: finalMessage }],
      });

      console.log(`🤖 Hotel Agent: ${finalMessage}`);
    } else {
      const message =
        response.text || response.output || content[0]?.text || "No response";
      console.log(`🤖 Hotel Agent: ${message}`);
    }
  }
}

run();
{{< /tab >}}
{{< /tabpane >}}

---

## Step 6: Run your application

Make sure your Toolbox server is running (`./toolbox --tools-file "tools.yaml"`), then run your script [make sure you are in the root directory]:

```bash
node index.js
```