---
title: "Quickstart"
type: docs
weight: 2
description: >
  How to get started running Toolbox locally with Python, PostgreSQL, and 
  LangGraph. 
---

## Before you begin

This guide assumes you have already done the following: 

1. Installed [Python 3.9+][install-python]
1. Installed [PostgreSQL 16+ and the `psql` client][install-postgres]
1. Completed setup for usage with a [LangChain chat model][lc-chat], such as:
    - [`langchain-vertexai` package][install-vertexai]
    - [`langchain-google-genai` package][install-genai]
    - [`langchain-anthropic` package][install-anthropic] 


[install-python]: https://wiki.python.org/moin/BeginnersGuide/Download
[install-postgres]: https://www.postgresql.org/download/
[lc-chat]: https://python.langchain.com/docs/integrations/chat/
[install-vertexai]: https://python.langchain.com/docs/integrations/llms/google_vertex_ai_palm/#setup
[install-genai]: https://python.langchain.com/docs/integrations/chat/google_generative_ai/#setup
[install-anthropic]: https://python.langchain.com/docs/integrations/chat/anthropic/#setup

## Step 1: Set up your database

In this section, we will create a database, insert some data that needs to be
access by our agent, and create a database user for Toolbox to connect with. 

1. Connect to postgres using the `psql` command:

    ```bash
    psql -h 127.0.0.1 -U postgres
    ```

    Here, `postgres` denotes the default postgres superuser.

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

1. Connect to your database with your new user:

    ```bash
    psql -U toolbox_user -d toolbox_db
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
    ```bash
    export OS="linux/amd64" # one of linux/amd64, darwin/arm64, darwin/amd64, or windows/amd64
    curl -O https://storage.googleapis.com/genai-toolbox/v0.1.0/$OS/toolbox
    ```

1. Make the binary executable:

    ```bash
    chmod +x toolbox
    ```

1. Write the following into a `tools.yaml` file. Be sure to update any fields
   such as `user`, `password`, or `database` that you may have customized in the
   previous step.

    ```yaml
    sources:
      my-pg-source:
        kind: postgres
        host: 127.0.0.1
        port: 5432
        database: toolbox_db
        user: toolbox_user
        password: my-password
    tools:
      # Define the 5 tools we want our agent to have
      # for more info on tools check out the "Resources" section of the docs
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
          Book a hotel by its ID. Returns a message indicating whether the hotel was
          successfully booked or not.
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
    ```

1. Run the Toolbox server, pointing to the `tools.yaml` file created earlier:

    ```bash
    ./toolbox --tools_file "tools.yaml"
    ```

## Step 3: Connect your agent to Toolbox

In this section, we will write and run a LangGraph agent that will load the Tools
from Toolbox.

1. In a new terminal, install the `toolbox-langchain-sdk` package.

    ```bash
    pip install toolbox-langchain-sdk
    ```

1. Install other required dependencies:

    ```bash
    # TODO(developer): replace with correct package if needed
    pip install langgraph langchain-google-vertexai
    # pip install langchain_google_genai
    # pip install langchain_anthropic
    ```

1. Create a new file named `langgraph_hotel_agent.py` and copy the following
   code to create a [LangGraph agent][langgraph-agent], based on their [Hotels
   example][langchain-hotels]:

    ```python
    import asyncio

    from langgraph.prebuilt import create_react_agent
    # TODO(developer): replace this with another import if needed
    from langchain_google_vertexai import ChatVertexAI
    # from langchain_google_genai import ChatGoogleGenerativeAI
    # from langchain_anthropic import ChatAnthropic
    from langgraph.checkpoint.memory import MemorySaver
    
    from toolbox_langchain import ToolboxClient

    prompt = """
      You're a helpful hotel assistant. You handle hotel searching, booking and
      cancellations. When the user searches for a hotel, mention it's name, id, 
      location and price tier. Always mention hotel ids while performing any 
      searches. This is very important for any operations. For any bookings or 
      cancellations, please provide the appropriate confirmation. Be sure to 
      update checkin or checkout dates if mentioned by the user.
      Don't ask for confirmations from the user.
    """
    
    queries = [
        "Find hotels in Basel with Basel in it's name.",
        "Can you book the Hilton Basel for me?",
        "Oh wait, this is too expensive. Please cancel it and book the Hyatt Regency instead.",
        "My check in dates would be from April 10, 2024 to April 19, 2024.",
    ]

    async def main():
        # TODO(developer): replace this with another model if needed
        model = ChatVertexAI(model_name="gemini-1.5-pro")
        # model = ChatGoogleGenerativeAI(model="gemini-1.5-pro")
        # model = ChatAnthropic(model="claude-3-5-sonnet-20240620")
        
        # Load the tools from the Toolbox server
        client = ToolboxClient("http://127.0.0.1:5000")
        tools = await client.aload_toolset()

        agent = create_react_agent(model, tools, checkpointer=MemorySaver())

        config = {"configurable": {"thread_id": "thread-1"}}
        for query in queries:
            inputs = {"messages": [("user", prompt + query)]}
            response = await agent.ainvoke(inputs, stream_mode="values", config=config)
            print(response["messages"][-1].content)

    asyncio.run(main())
    ```

    [langgraph-agent]:https://langchain-ai.github.io/langgraph/reference/prebuilt/#langgraph.prebuilt.chat_agent_executor.create_react_agent
    [langchain-hotels]: https://langchain-ai.github.io/langgraph/tutorials/customer-support/customer-support/#hotels

1. Run your agent, and observe the results:

    ```sh
    python langgraph_hotel_agent.py
    ```
