---
title: "Local Quickstart"
type: docs
weight: 1
description: This guide would help you set up a basic agentic application using toolbox.

---

## Step 1: Set up a database

[Install postgres and configure a
database](https://neon.tech/postgresql/postgresql-getting-started) for your
system.

This process creates a database `postgres` with superuser `postgres`.

## Step 2: Import data into the database

1. Connect to the database using command line.

    ```bash
    psql -U postgres -d postgres
    ```

    Here, the first postgres denotes the user and the second one denotes the
    database name.

1. Create a table using the following command.

    ```sql
    CREATE TABLE hotels(
      id            INTEGER  NOT NULL PRIMARY KEY,
      name          VARCHAR NOT NULL,
      location      VARCHAR NOT NULL,
      price_tier    VARCHAR NOT NULL,
      checkin_date  DATE  NOT NULL,
      checkout_date DATE  NOT NULL,
      booked        BIT  NOT NULL
    );
    ```
    <!-- 1. Download the data csv file (TODO: Add csv filepath) -->

1. Insert data into the table.

    ```sql
    INSERT INTO hotels(id, name, location, price_tier, checkin_date, checkout_date, booked) VALUES 
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

## Step 3: Create a tools config file

Create a `tools.yaml` file with the following content, updating the `password` field:

```yaml
sources:
    my-pg-source:
        kind: postgres
        host: 127.0.0.1
        port: 5432
        database: postgres
        user: postgres
        password: {password}
tools:
  search-hotels:
    kind: postgres-sql
    source: my-pg-source
    description: Search for hotels based on location and name. 
      Returns a list of hotel dictionaries matching the search criteria.
    parameters:
      - name: location
        type: string
        description: The location of the hotel.
      - name: name
        type: string
        description: The name of the hotel.
    statement: SELECT * FROM hotels WHERE location ILIKE '%' || $1 || '%' AND
        name ILIKE '%' || $2 || '%';
  book-hotel:
    kind: postgres-sql
    source: my-pg-source
    description: Book a hotel by its ID. Returns a message indicating 
       whether the hotel was successfully booked or not.
    parameters:
      - name: hotel_id
        type: string
        description: The ID of the hotel to book.
    statement: UPDATE hotels SET booked = B'1' WHERE id = $1;
  update-hotel:
    kind: postgres-sql
    source: my-pg-source
    description: Update a hotel's check-in and check-out dates by its ID. Returns a 
        message indicating  whether the hotel was successfully updated or not.
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
    statement: UPDATE hotels SET checkin_date = CAST($2 as date),
        checkout_date = CAST($3 as date) WHERE id = $1;
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

The config file defines four tools:
`search-hotels`, `book-hotel`, `update-hotel` and `cancel-hotel`.

Each tool specifies its description,
[kind](https://github.com/googleapis/genai-toolbox/tree/main/docs/sources#kinds-of-sources),
[source](https://github.com/googleapis/genai-toolbox/blob/main/docs/sources/README.md), required parameters
and the corresponding SQL statements to execute upon tool invocation.

## Step 4: Start a toolbox server locally

1. Download the latest toolbox binary.

    ```bash
    curl -O https://storage.googleapis.com/genai-toolbox/v0.0.5/linux/amd64/toolbox
    ```

  > **_NOTE:_**  Use the [correct binary](https://github.com/googleapis/genai-toolbox/releases) corresponding to your OS and CPU architecture.

1. Provide binary execution permissions.

    ```bash
    chmod +x toolbox
    ```

1. Run the Toolbox server.

    ```bash
    ./toolbox --tools_file "tools.yaml"
    ```

## Step 5: Start building using Toolbox

1. Install the `toolbox_langchain_sdk` package.

    ```bash
    pip install toolbox_langchain_sdk
    ```

    > **_NOTE:_**
    > Right now, the toolbox_langchain_sdk package is not available on pip. To
    > use the sdk, download the source code from [git](https://github.com/googleapis/genai-toolbox/tree/main/sdks/langchain) and install it locally
    > using the command:
    >
    > ```bash
    > pip install .
    > ```

1. Install other required modules.

    ```bash
    pip install langgraph langchain-google-vertexai
    ```

1. Create a python script to connect to the toolbox SDK.

    ```python
    import asyncio
    from toolbox_langchain_sdk import ToolboxClient

    async def main():
      client = ToolboxClient("http://127.0.0.1:5000")
      # All our code will be added here
      pass

    asyncio.run(main())
    ```

1. Try out the `search-hotels` tool.

    ```python
    search_tool = await client.load_tool('search-hotels')
    response = await search_tool.ainvoke({"location": "Zurich", "name": ""})
    print(response)
    ```

1. Load all tools.

    ```python
    tools = await client.load_toolset()
    ```

1. Create a LangGraph [ReAct
   agent](https://langchain-ai.github.io/langgraph/reference/prebuilt/#langgraph.prebuilt.chat_agent_executor.create_react_agent).

    ```python
    from langgraph.prebuilt import create_react_agent
    from langchain_google_vertexai import ChatVertexAI
    from langgraph.checkpoint.memory import MemorySaver

    model = ChatVertexAI(model_name="gemini-pro", project="my-project") # Change the GCP project here
    agent = create_react_agent(model, tools, checkpointer=MemorySaver())
    ```

1. Define the initial prompt and user queries.

    ```python
    prompt = """
      You're a helpful hotel assistant. You handle hotel searching, booking and
      cancellations. When the user searches for a hotel, mention it's name, id, 
      location and price tier. Always mention hotel ids while performing any 
      searches. This is very important for any operations. For any bookings or 
      cancellations, please provide the appropriate confirmation.
      Don't ask for confirmations from the user.
    """
    
    queries = [
        "Find hotels in Basel with Basel in it's name.",
        "Can you book the Hilton Basel for me?",
        "Oh wait, this is too expensive. Please cancel it and book the Hyatt Regency instead.",
        "My check in dates would be from April 10, 2024 to April 19, 2024.",
    ]
    ```

1. Run the queries and observe the output!

    ```python
    config = {"configurable": {"thread_id": "thread-1"}}

    for query in queries:
        inputs = {"messages": [("user", prompt + query)]}
        response = await agent.ainvoke(inputs, stream_mode="values", config=config)
        print(response["messages"][-1].content)
    ```

    To verify the agent's actions, you can examine the hotels table. You should
    observe that the `booked` column for the `Hyatt Regency Basel` has changed
    from `0` to `1`, indicating that the hotel has been successfully booked.
    Additionally, the `checkin_date` and `checkout_date` have been updated to
    `2024-04-10` and `2024-04-19` from `2024-04-02` and `2024-04-20`
    respectively.
