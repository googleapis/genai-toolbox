---
title: "JS Quickstart (Local)"
type: docs
weight: 2
description: >
  How to get started running Toolbox locally with JavaScript, PostgreSQL, and orchestration frameworks such as [LangChain](https://js.langchain.com/docs/introduction/), [LlamaIndex](https://ts.llamaindex.ai/), or [GenkitJS](https://genkit.dev/docs/get-started/).
---

## Before you begin

This guide assumes you have already done the following:

1. Installed [Node.js (v18 or higher)].
2. Installed [PostgreSQL 16+ and the `psql` client][install-postgres].

### Cloud Setup (Optional)

If you plan to use **Google Cloud’s Vertex AI** with your agent (e.g., using Gemini or PaLM models), follow these one-time setup steps:

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
    curl -O https://storage.googleapis.com/genai-toolbox/v0.9.0/$OS/toolbox
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
  {{< notice note >}}
    Toolbox enables dynamic reloading by default. To disable, use the `--disable-reload` flag.
  {{< /notice >}}

## Step 3: Connect your agent to Toolbox

In this section, we will write and run an agent that will load the Tools
from Toolbox.

First let's create a new folder for your project, initialize it with `npm`, and install the required dependencies.

1. Create a new folder for your project and navigate into it:

    ```bash
    mkdir my-agent-app
    cd my-agent-app
    ```

1. (Optional) Initialize a Node.js project:

    ```bash
    npm init -y
    ```

1. Create a new file in the root directory:

    ```bash
    touch index.js
    ```

4. Install required dependencies

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

5. Create a new file named `hotelAgent.js` and copy the following code to create an agent:

{{< tabpane persist=header >}}
{{< tab header="LangChain" lang="js" >}}

import { ChatVertexAI } from "@langchain/google-vertexai";
import { ToolboxClient } from "@toolbox-sdk/core";
import { tool } from "@langchain/core/tools";
import { createReactAgent } from "@langchain/langgraph/prebuilt";
import { MemorySaver } from "@langchain/langgraph";

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

{{< tab header="LlamaIndex" lang="js" >}}

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
import { googleAI } from '@genkit-ai/googleai';

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

run();
{{< /tab >}}
{{< /tabpane >}}

1. Run your agent, and observe the results:

    ```sh
    node hotelAgent.js
    ```
