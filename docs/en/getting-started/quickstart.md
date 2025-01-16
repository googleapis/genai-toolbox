---
title: "Toolbox Quickstart"
type: docs
weight: 1
description: This guide would help you set up a basic agentic application using toolbox.

---

## Step 1: Set up a Cloud SQL database

1. [Create a Cloud SQL instance](https://cloud.google.com/sql/docs/postgres/connect-instance-local-computer#create-instance).
1. [Create a
   database](https://cloud.google.com/sql/docs/postgres/connect-instance-local-computer#create-database).
1. [Create a database
   user](https://cloud.google.com/sql/docs/postgres/connect-instance-local-computer#create_a_user).
1. [Set up a service
   account](https://cloud.google.com/sql/docs/postgres/connect-instance-local-computer#set_up_a_service_account).

## Step 2: Import data into the database

Run the command.

```bash
gcloud sql import sql {INSTANCE_NAME} gs://toolbox-quickstart/hotels.gz --database={DATABASE_NAME}
```

### (Optional) Explore your data

You can explore the data by logging into [Cloud SQL
Studio](https://cloud.google.com/sql/docs/mysql/manage-data-using-studio#explore-data)
and running SQL queries.

## Step 3: Create a tools config file

Create a `tools.yaml` file with the following content, updating the `project`,
`database`, `instance`, `user`, and `password` fields:

```yaml
sources:
    my-cloud-sql-pg-source:
        kind: cloud-sql-postgres
        project: {GCP_PROJECT}
        region: us-central1
        instance: {INSTANCE_NAME}
        database: {DATABASE_NAME}
        user: {DATABASE_USER}
        password: {PASSWORD}
tools:
  search-hotels:
    kind: postgres-sql
    source: my-cloud-sql-pg-source
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
    source: my-cloud-sql-pg-source
    description: Book a hotel by its ID. Returns a message indicating 
       whether the hotel was successfully booked or not.
    parameters:
      - name: hotel_id
        type: string
        description: The ID of the hotel to book.
    statement: UPDATE hotels SET booked = B'1' WHERE id = $1;
  update-hotel:
    kind: postgres-sql
    source: my-cloud-sql-pg-source
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
    source: my-cloud-sql-pg-source
    description: Cancel a hotel by its ID.
    parameters:
      - name: hotel_id
        type: string
        description: The ID of the hotel to cancel.
    statement: UPDATE hotels SET booked = B'0' WHERE id = $1;
```

> **_NOTE:_**  If your instance belongs to a different region, update the `region` field.

The config file defines four tools:
`search-hotels`, `book-hotel`, `update-hotel` and `cancel-hotel`.

Each tool specifies its description,
[kind](https://github.com/googleapis/genai-toolbox/blob/main/docs/sources/cloud-sql-pg.md#requirements),
[source](./docs/sources/README.md), required parameters
and the corresponding SQL statements to execute upon tool invocation.

## Step 4: Start a toolbox server locally

1. Download the latest toolbox binary.

    ```bash
    curl -O https://storage.googleapis.com/genai-toolbox/v0.0.5/linux/amd64/toolbox
    ```

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

1. Install required modules.

    ```bash
    pip install langgraph langchain-google-vertexai
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

    To verify the agent's actions, you can [examine the hotels table in Cloud
    SQL Studio](#optional-explore-your-data). You should observe that the
    `booked` column for the `Hyatt Regency Basel` has changed from `0` to `1`,
    indicating that the hotel has been successfully booked. Additionally, the
    `checkin_date` and `checkout_date` have been updated to `2024-04-10` and
    `2024-04-19` from `2024-04-02` and `2024-04-20` respectively.