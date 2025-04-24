---
title: "Quickstart (Local with BigQuery)"
type: docs
weight: 2
description: >
  How to get started running Toolbox locally with Python, BigQuery, and 
  LangGraph or LlamaIndex. 
---

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/googleapis/genai-toolbox/blob/main/docs/en/getting-started/colab_quickstart_bigquery.ipynb)
## Before you begin

This guide assumes you have already done the following:

1.  Installed [Python 3.9+][install-python] (including [pip][install-pip] and
    your preferred virtual environment tool for managing dependencies e.g. [venv][install-venv]).
1.  Installed and configured the [Google Cloud SDK (`gcloud` CLI)][install-gcloud].
1.  Authenticated with Google Cloud for application-default credentials:
    ```bash
    gcloud auth login
    gcloud auth application-default login
    ```
1.  Set your default Google Cloud project (replace `YOUR_PROJECT_ID` with your actual project ID):
    ```bash
    gcloud config set project YOUR_PROJECT_ID
    export GOOGLE_CLOUD_PROJECT=YOUR_PROJECT_ID
    ```
    Toolbox and the client libraries will use this project for BigQuery, unless overridden in configurations.
1.  Enabled the BigQuery API in your Google Cloud project.
1.  Installed the BigQuery client library for Python:
    ```bash
    pip install google-cloud-bigquery
    ```
1. Completed setup for usage with an LLM model such as
{{< tabpane text=true persist=header >}}
{{% tab header="LangChain" lang="en" %}}
- [langchain-vertexai](https://python.langchain.com/docs/integrations/llms/google_vertex_ai_palm/#setup) package.

- [langchain-google-genai](https://python.langchain.com/docs/integrations/chat/google_generative_ai/#setup) package.

- [langchain-anthropic](https://python.langchain.com/docs/integrations/chat/anthropic/#setup) package.
{{% /tab %}}
{{% tab header="LlamaIndex" lang="en" %}}
- [llama-index-llms-google-genai](https://pypi.org/project/llama-index-llms-google-genai/) package.

- [llama-index-llms-anthropic](https://docs.llamaindex.ai/en/stable/examples/llm/anthropic) package.
{{% /tab %}}
{{< /tabpane >}}

[install-python]: https://wiki.python.org/moin/BeginnersGuide/Download
[install-pip]: https://pip.pypa.io/en/stable/installation/
[install-venv]: https://packaging.python.org/en/latest/tutorials/installing-packages/#creating-virtual-environments
[install-gcloud]: https://cloud.google.com/sdk/docs/install

## Step 1: Set up your BigQuery Dataset and Table

In this section, we will create a BigQuery dataset and a table, then insert some data that needs to be accessed by our agent. BigQuery operations are performed against your configured Google Cloud project.

1.  Create a new BigQuery dataset (replace `YOUR_DATASET_NAME` with your desired dataset name, e.g., `toolbox_ds`, and optionally specify a location like `US` or `EU`):

    ```bash
    export BQ_DATASET_NAME="YOUR_DATASET_NAME" # e.g., toolbox_ds
    export BQ_LOCATION="US" # e.g., US, EU, asia-northeast1

    bq --location=$BQ_LOCATION mk $BQ_DATASET_NAME
    ```
    You can also do this through the [Google Cloud Console](https://console.cloud.google.com/bigquery).

    {{< notice tip >}}
 For a real application, ensure that the service account or user running Toolbox has the necessary IAM permissions (e.g., BigQuery Data Editor, BigQuery User) on the dataset or project. For this local quickstart with user credentials, your own permissions will apply.
    {{< /notice >}}

2.  Create a `hotels` table within your new dataset using the `bq query` command. The schema is adapted for BigQuery data types:

    ```sql
    CREATE TABLE IF NOT EXISTS `YOUR_PROJECT_ID.YOUR_DATASET_NAME.hotels` (
      id            INT64 NOT NULL,
      name          STRING NOT NULL,
      location      STRING NOT NULL,
      price_tier    STRING NOT NULL,
      checkin_date  DATE NOT NULL,
      checkout_date DATE NOT NULL,
      booked        BOOLEAN NOT NULL
    );
    ```
    Save this as `create_hotels_table.sql` and run:
    ```bash
    bq query --project_id=$GOOGLE_CLOUD_PROJECT --dataset_id=$BQ_DATASET_NAME --use_legacy_sql=false < create_hotels_table.sql
    ```
    Replace `YOUR_PROJECT_ID` and `YOUR_DATASET_NAME` in the SQL with your actual project ID and dataset name if you are not using environment variables for them directly in the SQL. Or, ensure your `bq` CLI is configured with the default project and dataset if you omit them from the table name like `hotels`. For clarity, fully qualifying is safer: `\`$GOOGLE_CLOUD_PROJECT.$BQ_DATASET_NAME.hotels\``.

3.  Insert data into the `hotels` table. Note that `BOOLEAN` values are `TRUE` or `FALSE`.

    ```sql
    INSERT INTO `YOUR_PROJECT_ID.YOUR_DATASET_NAME.hotels` (id, name, location, price_tier, checkin_date, checkout_date, booked)
    VALUES
      (1, 'Hilton Basel', 'Basel', 'Luxury', DATE('2024-04-22'), DATE('2024-04-20'), FALSE),
      (2, 'Marriott Zurich', 'Zurich', 'Upscale', DATE('2024-04-14'), DATE('2024-04-21'), FALSE),
      (3, 'Hyatt Regency Basel', 'Basel', 'Upper Upscale', DATE('2024-04-02'), DATE('2024-04-20'), FALSE),
      (4, 'Radisson Blu Lucerne', 'Lucerne', 'Midscale', DATE('2024-04-24'), DATE('2024-04-05'), FALSE),
      (5, 'Best Western Bern', 'Bern', 'Upper Midscale', DATE('2024-04-23'), DATE('2024-04-01'), FALSE),
      (6, 'InterContinental Geneva', 'Geneva', 'Luxury', DATE('2024-04-23'), DATE('2024-04-28'), FALSE),
      (7, 'Sheraton Zurich', 'Zurich', 'Upper Upscale', DATE('2024-04-27'), DATE('2024-04-02'), FALSE),
      (8, 'Holiday Inn Basel', 'Basel', 'Upper Midscale', DATE('2024-04-24'), DATE('2024-04-09'), FALSE),
      (9, 'Courtyard Zurich', 'Zurich', 'Upscale', DATE('2024-04-03'), DATE('2024-04-13'), FALSE),
      (10, 'Comfort Inn Bern', 'Bern', 'Midscale', DATE('2024-04-04'), DATE('2024-04-16'), FALSE);
    ```
    Save this as `insert_hotels_data.sql` and run:
    ```bash
    bq query --project_id=$GOOGLE_CLOUD_PROJECT --dataset_id=$BQ_DATASET_NAME --use_legacy_sql=false < insert_hotels_data.sql
    ```

## Step 2: Install and configure Toolbox

In this section, we will download Toolbox, configure our tools in a `tools.yaml` to use BigQuery, and then run the Toolbox server.

1.  Download the latest version of Toolbox as a binary:

    {{< notice tip >}}
 Select the
 [correct binary](https://github.com/googleapis/genai-toolbox/releases)
 corresponding to your OS and CPU architecture.
    {{< /notice >}}
    ```bash
    export OS="linux/amd64" # one of linux/amd64, darwin/arm64, darwin/amd64, or windows/amd64
    curl -O [https://storage.googleapis.com/genai-toolbox/v0.4.0/$OS/toolbox](https://storage.googleapis.com/genai-toolbox/v0.4.0/$OS/toolbox)
    ```
    2.  Make the binary executable:

    ```bash
    chmod +x toolbox
    ```

3.  Write the following into a `tools.yaml` file. The `project` in the `sources` section will use the `GOOGLE_CLOUD_PROJECT` environment variable you set earlier. You must replace the `YOUR_DATASET_NAME` placeholder in the SQL `statement` fields with your actual BigQuery dataset name (e.g., the value of `$BQ_DATASET_NAME` from Step 1). The table name `hotels` is used directly in the statements.

    {{< notice tip >}}
 Authentication with BigQuery is handled via Application Default Credentials (ADC). Ensure you have run `gcloud auth application-default login`. Avoid hardcoding secrets.
    {{< /notice >}}

    ```yaml
    sources:
      my-bigquery-source:
        kind: bigquery
        project: YOUR_PROJECT_ID
    tools:
      search-hotels-by-name:
        kind: bigquery-sql
        source: my-bigquery-source
        description: Search for hotels based on name.
        parameters:
          - name: name
            type: string
            description: The name of the hotel.
        statement: SELECT * FROM `YOUR_DATASET_NAME.hotels` WHERE LOWER(name) LIKE LOWER(CONCAT('%', @name, '%'));
      search-hotels-by-location:
        kind: bigquery-sql
        source: my-bigquery-source
        description: Search for hotels based on location.
        parameters:
          - name: location
            type: string
            description: The location of the hotel.
        statement: SELECT * FROM `YOUR_DATASET_NAME.hotels` WHERE LOWER(location) LIKE LOWER(CONCAT('%', @location, '%'));
      book-hotel:
        kind: bigquery-sql
        source: my-bigquery-source
        description: >-
           Book a hotel by its ID. If the hotel is successfully booked, returns a NULL, raises an error if not.
        parameters:
          - name: hotel_id
            type: integer
            description: The ID of the hotel to book.
        statement: UPDATE `YOUR_DATASET_NAME.hotels` SET booked = TRUE WHERE id = @hotel_id;
      update-hotel:
        kind: bigquery-sql
        source: my-bigquery-source
        description: >-
          Update a hotel's check-in and check-out dates by its ID. Returns a message indicating whether the hotel was successfully updated or not.
        parameters:
          - name: checkin_date
            type: string
            description: The new check-in date of the hotel.
          - name: checkout_date
            type: string
            description: The new check-out date of the hotel.
          - name: hotel_id
            type: integer
            description: The ID of the hotel to update.
        statement: >-
          UPDATE `YOUR_DATASET_NAME.hotels` SET checkin_date = PARSE_DATE('%Y-%m-%d', @checkin_date), checkout_date = PARSE_DATE('%Y-%m-%d', @checkout_date) WHERE id = @hotel_id;
      cancel-hotel:
        kind: bigquery-sql
        source: my-bigquery-source
        description: Cancel a hotel by its ID.
        parameters:
          - name: hotel_id
            type: integer
            description: The ID of the hotel to cancel.
        statement: UPDATE `YOUR_DATASET_NAME.hotels` SET booked = FALSE WHERE id = @hotel_id;
    ```

    **Important Note on `toolsets`**: The `tools.yaml` content above does not include a `toolsets` section. The Python agent examples in Step 3 (e.g., `await toolbox_client.load_toolset("my-toolset")`) rely on a toolset named `my-toolset`. To make those examples work, you will need to add a `toolsets` section to your `tools.yaml` file, for example:
    ```yaml
    # Add this to your tools.yaml if using load_toolset("my-toolset")
    # Ensure it's at the same indentation level as 'sources:' and 'tools:'
    toolsets:
      my-toolset:
        - search-hotels-by-name
        - search-hotels-by-location
        - book-hotel
        - update-hotel
        - cancel-hotel
    ```
    Alternatively, you can modify the agent code to load tools individually (e.g., using `await toolbox_client.load_tool("search-hotels-by-name")`).

    For more info on tools, check out the `Resources` section of the docs.

4.  Run the Toolbox server, pointing to the `tools.yaml` file created earlier:

    ```bash
    ./toolbox --tools_file "tools.yaml"
    ```

## Step 3: Connect your agent to Toolbox

In this section, we will write and run an agent that will load the Tools
from Toolbox.

{{< notice tip>}} If you prefer to experiment within a Google Colab environment, 
you can connect to a 
[local runtime](https://research.google.com/colaboratory/local-runtimes.html). 
{{< /notice >}}


1. In a new terminal, install the SDK package.
    
    {{< tabpane persist=header >}}
{{< tab header="Langchain" lang="bash" >}}

pip install toolbox-langchain
{{< /tab >}}
{{< tab header="LlamaIndex" lang="bash" >}}

pip install toolbox-llamaindex
{{< /tab >}}
{{< /tabpane >}}

1. Install other required dependencies:
    
    {{< tabpane persist=header >}}
{{< tab header="Langchain" lang="bash" >}}

# TODO(developer): replace with correct package if needed
pip install langgraph langchain-google-vertexai
# pip install langchain-google-genai
# pip install langchain-anthropic
{{< /tab >}}
{{< tab header="LlamaIndex" lang="bash" >}}

# TODO(developer): replace with correct package if needed
pip install llama-index-llms-google-genai
# pip install llama-index-llms-anthropic
{{< /tab >}}
{{< /tabpane >}}
    
1. Create a new file named `hotel_agent.py` and copy the following
   code to create an agent:
    {{< tabpane persist=header >}}
{{< tab header="LangChain" lang="python" >}}

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

def main():
    # TODO(developer): replace this with another model if needed
    model = ChatVertexAI(model_name="gemini-1.5-pro")
    # model = ChatGoogleGenerativeAI(model="gemini-1.5-pro")
    # model = ChatAnthropic(model="claude-3-5-sonnet-20240620")
    
    # Load the tools from the Toolbox server
    client = ToolboxClient("http://127.0.0.1:5000")
    tools = client.load_toolset()

    agent = create_react_agent(model, tools, checkpointer=MemorySaver())

    config = {"configurable": {"thread_id": "thread-1"}}
    for query in queries:
        inputs = {"messages": [("user", prompt + query)]}
        response = agent.invoke(inputs, stream_mode="values", config=config)
        print(response["messages"][-1].content)

main()
{{< /tab >}}
{{< tab header="LlamaIndex" lang="python" >}}
import asyncio
import os

from llama_index.core.agent.workflow import AgentWorkflow

from llama_index.core.workflow import Context

# TODO(developer): replace this with another import if needed 
from llama_index.llms.google_genai import GoogleGenAI
# from llama_index.llms.anthropic import Anthropic

from toolbox_llamaindex import ToolboxClient

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
    llm = GoogleGenAI(
        model="gemini-1.5-pro",
        vertexai_config={"project": "project-id", "location": "us-central1"},
    )
    # llm = GoogleGenAI(
    #     api_key=os.getenv("GOOGLE_API_KEY"),
    #     model="gemini-1.5-pro",
    # )
    # llm = Anthropic(
    #   model="claude-3-7-sonnet-latest",
    #   api_key=os.getenv("ANTHROPIC_API_KEY")
    # )
    
    # Load the tools from the Toolbox server
    client = ToolboxClient("http://127.0.0.1:5000")
    tools = client.load_toolset()

    agent = AgentWorkflow.from_tools_or_functions(
        tools,
        llm=llm,
        system_prompt=prompt,
    )
    ctx = Context(agent)
    for query in queries:
         response = await agent.run(user_msg=query, ctx=ctx)
         print(f"---- {query} ----")
         print(str(response))

asyncio.run(main())
{{< /tab >}}
{{< /tabpane >}}
    
    {{< tabpane text=true persist=header >}}
{{% tab header="Langchain" lang="en" %}}
To learn more about Agents in LangChain, check out the [LangGraph Agent documentation.](https://langchain-ai.github.io/langgraph/reference/prebuilt/#langgraph.prebuilt.chat_agent_executor.create_react_agent)
{{% /tab %}}
{{% tab header="LlamaIndex" lang="en" %}}
To learn more about Agents in LlamaIndex, check out the [LlamaIndex AgentWorkflow documentation.](https://docs.llamaindex.ai/en/stable/examples/agent/agent_workflow_basic/)
{{% /tab %}}
{{< /tabpane >}}
1. Run your agent, and observe the results:

    ```sh
    python hotel_agent.py
    ```