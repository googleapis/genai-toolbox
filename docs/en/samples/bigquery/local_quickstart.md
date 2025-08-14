---
title: "Quickstart (Local with BigQuery)"
type: docs
weight: 1
description: >
  How to get started running Toolbox locally with Python, BigQuery, and
  LangGraph, LlamaIndex, or ADK.
---

[![Open In
Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/googleapis/genai-toolbox/blob/main/docs/en/samples/bigquery/colab_quickstart_bigquery.ipynb)

## Before you begin

This guide assumes you have already done the following:

1. Installed [Python 3.9+][install-python] (including [pip][install-pip] and
    your preferred virtual environment tool for managing dependencies e.g.
    [venv][install-venv]).
1. Installed and configured the [Google Cloud SDK (gcloud CLI)][install-gcloud].
1. Authenticated with Google Cloud for Application Default Credentials (ADC):

    ```bash
    gcloud auth login --update-adc
    ```

1. Set your default Google Cloud project (replace `YOUR_PROJECT_ID` with your
   actual project ID):

    ```bash
    gcloud config set project YOUR_PROJECT_ID
    export GOOGLE_CLOUD_PROJECT=YOUR_PROJECT_ID
    ```

    Toolbox and the client libraries will use this project for BigQuery, unless
    overridden in configurations.
1. [Enabled the BigQuery API][enable-bq-api] in your Google Cloud project.
1. Installed the BigQuery client library for Python:

    ```bash
    pip install google-cloud-bigquery
    ```

1. Completed setup for usage with an LLM model such as
{{< tabpane text=true persist=header >}}
{{% tab header="Core" lang="en" %}}

- [langchain-vertexai](https://python.langchain.com/docs/integrations/llms/google_vertex_ai_palm/#setup)
  package.

- [langchain-google-genai](https://python.langchain.com/docs/integrations/chat/google_generative_ai/#setup)
  package.

- [langchain-anthropic](https://python.langchain.com/docs/integrations/chat/anthropic/#setup)
  package.
{{% /tab %}}
{{% tab header="LangChain" lang="en" %}}
- [langchain-vertexai](https://python.langchain.com/docs/integrations/llms/google_vertex_ai_palm/#setup)
  package.

- [langchain-google-genai](https://python.langchain.com/docs/integrations/chat/google_generative_ai/#setup)
  package.

- [langchain-anthropic](https://python.langchain.com/docs/integrations/chat/anthropic/#setup)
  package.
{{% /tab %}}
{{% tab header="LlamaIndex" lang="en" %}}
- [llama-index-llms-google-genai](https://pypi.org/project/llama-index-llms-google-genai/)
  package.

- [llama-index-llms-anthropic](https://docs.llamaindex.ai/en/stable/examples/llm/anthropic)
  package.
{{% /tab %}}
{{% tab header="ADK" lang="en" %}}
- [google-adk](https://pypi.org/project/google-adk/) package.
{{% /tab %}}
{{< /tabpane >}}

[install-python]: https://wiki.python.org/moin/BeginnersGuide/Download
[install-pip]: https://pip.pypa.io/en/stable/installation/
[install-venv]:
    https://packaging.python.org/en/latest/tutorials/installing-packages/#creating-virtual-environments
[install-gcloud]: https://cloud.google.com/sdk/docs/install
[enable-bq-api]:
    https://cloud.google.com/bigquery/docs/quickstarts/query-public-dataset-console#before-you-begin

## Step 1: Set up your BigQuery Dataset and Table

In this section, we will create a BigQuery dataset and a table, then insert some
data that needs to be accessed by our agent. BigQuery operations are performed
against your configured Google Cloud project.

1. Create a new BigQuery dataset (replace `YOUR_DATASET_NAME` with your desired
   dataset name, e.g., `toolbox_ds`, and optionally specify a location like `US`
   or `EU`):

    ```bash
    export BQ_DATASET_NAME="YOUR_DATASET_NAME" # e.g., toolbox_ds
    export BQ_LOCATION="US" # e.g., US, EU, asia-northeast1

    bq --location=$BQ_LOCATION mk $BQ_DATASET_NAME
    ```

    You can also do this through the [Google Cloud
    Console](https://console.cloud.google.com/bigquery).

    {{< notice tip >}}
 For a real application, ensure that the service account or user running Toolbox
 has the necessary IAM permissions (e.g., BigQuery Data Editor, BigQuery User)
 on the dataset or project. For this local quickstart with user credentials,
 your own permissions will apply.
    {{< /notice >}}

1. The hotels table needs to be defined in your new dataset for use with the bq
   query command. First, create a file named `create_hotels_table.sql` with the
   following content:

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

    > **Note:** Replace `YOUR_PROJECT_ID` and `YOUR_DATASET_NAME` in the SQL
    > with your actual project ID and dataset name.

    Then run the command below to execute the sql query:

    ```bash
    bq query --project_id=$GOOGLE_CLOUD_PROJECT --dataset_id=$BQ_DATASET_NAME --use_legacy_sql=false < create_hotels_table.sql
    ```

1. Next, populate the hotels table with some initial data. To do this, create a
   file named `insert_hotels_data.sql` and add the following SQL INSERT
   statement to it.

    ```sql
    INSERT INTO `YOUR_PROJECT_ID.YOUR_DATASET_NAME.hotels` (id, name, location, price_tier, checkin_date, checkout_date, booked)
    VALUES
      (1, 'Hilton Basel', 'Basel', 'Luxury', '2024-04-20', '2024-04-22', FALSE),
      (2, 'Marriott Zurich', 'Zurich', 'Upscale', '2024-04-14', '2024-04-21', FALSE),
      (3, 'Hyatt Regency Basel', 'Basel', 'Upper Upscale', '2024-04-02', '2024-04-20', FALSE),
      (4, 'Radisson Blu Lucerne', 'Lucerne', 'Midscale', '2024-04-05', '2024-04-24', FALSE),
      (5, 'Best Western Bern', 'Bern', 'Upper Midscale', '2024-04-01', '2024-04-23', FALSE),
      (6, 'InterContinental Geneva', 'Geneva', 'Luxury', '2024-04-23', '2024-04-28', FALSE),
      (7, 'Sheraton Zurich', 'Zurich', 'Upper Upscale', '2024-04-02', '2024-04-27', FALSE),
      (8, 'Holiday Inn Basel', 'Basel', 'Upper Midscale', '2024-04-09', '2024-04-24', FALSE),
      (9, 'Courtyard Zurich', 'Zurich', 'Upscale', '2024-04-03', '2024-04-13', FALSE),
      (10, 'Comfort Inn Bern', 'Bern', 'Midscale', '2024-04-04', '2024-04-16', FALSE);
    ```

    > **Note:** Replace `YOUR_PROJECT_ID` and `YOUR_DATASET_NAME` in the SQL
    > with your actual project ID and dataset name.

    Then run the command below to execute the sql query:

    ```bash
    bq query --project_id=$GOOGLE_CLOUD_PROJECT --dataset_id=$BQ_DATASET_NAME --use_legacy_sql=false < insert_hotels_data.sql
    ```

## Step 2: Install and configure Toolbox

In this section, we will download Toolbox, configure our tools in a `tools.yaml`
to use BigQuery, and then run the Toolbox server.

1. Download the latest version of Toolbox as a binary:

    {{< notice tip >}}
 Select the
 [correct binary](https://github.com/googleapis/genai-toolbox/releases)
 corresponding to your OS and CPU architecture.
    {{< /notice >}}
    <!-- {x-release-please-start-version} -->
    ```bash
    export OS="linux/amd64" # one of linux/amd64, darwin/arm64, darwin/amd64, or windows/amd64
    curl -O https://storage.googleapis.com/genai-toolbox/v0.11.0/$OS/toolbox
    ```
    <!-- {x-release-please-end} -->

1. Make the binary executable:

    ```bash
    chmod +x toolbox
    ```

1. Write the following into a `tools.yaml` file. You must replace the
   `YOUR_PROJECT_ID` and `YOUR_DATASET_NAME` placeholder in the config with your
   actual BigQuery project and dataset name. The `location` field is optional;
   if not specified, it defaults to 'us'. The table name `hotels` is used
   directly in the statements.

    {{< notice tip >}}
 Authentication with BigQuery is handled via Application Default Credentials
 (ADC). Ensure you have run `gcloud auth application-default login`.
    {{< /notice >}}

    ```yaml
    sources:
      my-bigquery-source:
        kind: bigquery
        project: YOUR_PROJECT_ID
        location: us
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

    **Important Note on `toolsets`**: The `tools.yaml` content above does not
    include a `toolsets` section. The Python agent examples in Step 3 (e.g.,
    `await toolbox_client.load_toolset("my-toolset")`) rely on a toolset named
    `my-toolset`. To make those examples work, you will need to add a `toolsets`
    section to your `tools.yaml` file, for example:

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

    Alternatively, you can modify the agent code to load tools individually
    (e.g., using `await toolbox_client.load_tool("search-hotels-by-name")`).

    For more info on tools, check out the [Resources](../../resources/) section
    of the docs.

1. Run the Toolbox server, pointing to the `tools.yaml` file created earlier:

    ```bash
    ./toolbox --tools-file "tools.yaml"
    ```

    {{< notice note >}}
    Toolbox enables dynamic reloading by default. To disable, use the
    `--disable-reload` flag.
    {{< /notice >}}

## Step 3: Connect your agent to Toolbox

In this section, we will write and run an agent that will load the Tools
from Toolbox.

{{< notice tip>}} If you prefer to experiment within a Google Colab environment,
you can connect to a
[local runtime](https://research.google.com/colaboratory/local-runtimes.html).
{{< /notice >}}

1. In a new terminal, install the SDK package.

    {{< tabpane persist=header >}}
{{< tab header="Core" lang="bash" >}}

pip install toolbox-core
{{< /tab >}}
{{< tab header="Langchain" lang="bash" >}}

pip install toolbox-langchain
{{< /tab >}}
{{< tab header="LlamaIndex" lang="bash" >}}

pip install toolbox-llamaindex
{{< /tab >}}
{{< tab header="ADK" lang="bash" >}}

pip install google-adk
{{< /tab >}}

{{< /tabpane >}}

1. Install other required dependencies:

    {{< tabpane persist=header >}}
{{< tab header="Core" lang="bash" >}}

# TODO(developer): replace with correct package if needed

pip install langgraph langchain-google-vertexai

# pip install langchain-google-genai

# pip install langchain-anthropic

{{< /tab >}}
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
{{< tab header="ADK" lang="bash" >}}
pip install toolbox-core
{{< /tab >}}
{{< /tabpane >}}

1. Create a new file named `hotel_agent.py` and copy the following
   code to create an agent:
    {{< tabpane persist=header >}}
{{< tab header="Core" lang="python" >}}

import asyncio

from google import genai
from google.genai.types import (
    Content,
    FunctionDeclaration,
    GenerateContentConfig,
    Part,
    Tool,
)

from toolbox_core import ToolboxClient

prompt = """
  You're a helpful hotel assistant. You handle hotel searching, booking and
  cancellations. When the user searches for a hotel, mention it's name, id,
  location and price tier. Always mention hotel id while performing any
  searches. This is very important for any operations. For any bookings or
  cancellations, please provide the appropriate confirmation. Be sure to
  update checkin or checkout dates if mentioned by the user.
  Don't ask for confirmations from the user.
"""

queries = [
    "Find hotels in Basel with Basel in it's name.",
    "Please book the hotel Hilton Basel for me.",
    "This is too expensive. Please cancel it.",
    "Please book Hyatt Regency for me",
    "My check in dates for my booking would be from April 10, 2024 to April 19, 2024.",
]

async def run_application():
    async with ToolboxClient("<http://127.0.0.1:5000>") as toolbox_client:

        # The toolbox_tools list contains Python callables (functions/methods) designed for LLM tool-use
        # integration. While this example uses Google's genai client, these callables can be adapted for
        # various function-calling or agent frameworks. For easier integration with supported frameworks
        # (https://github.com/googleapis/mcp-toolbox-python-sdk/tree/main/packages), use the
        # provided wrapper packages, which handle framework-specific boilerplate.
        toolbox_tools = await toolbox_client.load_toolset("my-toolset")
        genai_client = genai.Client(
            vertexai=True, project="project-id", location="us-central1"
        )

        genai_tools = [
            Tool(
                function_declarations=[
                    FunctionDeclaration.from_callable_with_api_option(callable=tool)
                ]
            )
            for tool in toolbox_tools
        ]
        history = []
        for query in queries:
            user_prompt_content = Content(
                role="user",
                parts=[Part.from_text(text=query)],
            )
            history.append(user_prompt_content)

            response = genai_client.models.generate_content(
                model="gemini-2.0-flash-001",
                contents=history,
                config=GenerateContentConfig(
                    system_instruction=prompt,
                    tools=genai_tools,
                ),
            )
            history.append(response.candidates[0].content)
            function_response_parts = []
            for function_call in response.function_calls:
                fn_name = function_call.name
                # The tools are sorted alphabetically
                if fn_name == "search-hotels-by-name":
                    function_result = await toolbox_tools[3](**function_call.args)
                elif fn_name == "search-hotels-by-location":
                    function_result = await toolbox_tools[2](**function_call.args)
                elif fn_name == "book-hotel":
                    function_result = await toolbox_tools[0](**function_call.args)
                elif fn_name == "update-hotel":
                    function_result = await toolbox_tools[4](**function_call.args)
                elif fn_name == "cancel-hotel":
                    function_result = await toolbox_tools[1](**function_call.args)
                else:
                    raise ValueError("Function name not present.")
                function_response = {"result": function_result}
                function_response_part = Part.from_function_response(
                    name=function_call.name,
                    response=function_response,
                )
                function_response_parts.append(function_response_part)

            if function_response_parts:
                tool_response_content = Content(role="tool", parts=function_response_parts)
                history.append(tool_response_content)

            response2 = genai_client.models.generate_content(
                model="gemini-2.0-flash-001",
                contents=history,
                config=GenerateContentConfig(
                    tools=genai_tools,
                ),
            )
            final_model_response_content = response2.candidates[0].content
            history.append(final_model_response_content)
            print(response2.text)

asyncio.run(run_application())
{{< /tab >}}
{{< tab header="LangChain" lang="python" >}}

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
    "Find hotels in Basel with Basel in its name.",
    "Can you book the Hilton Basel for me?",
    "Oh wait, this is too expensive. Please cancel it and book the Hyatt Regency instead.",
    "My check in dates would be from April 10, 2024 to April 19, 2024.",
]

async def main():
    # TODO(developer): replace this with another model if needed
    model = ChatVertexAI(model_name="gemini-2.0-flash-001")
    # model = ChatGoogleGenerativeAI(model="gemini-2.0-flash-001")
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
        model="gemini-2.0-flash-001",
        vertexai_config={"location": "us-central1"},
    )
    # llm = GoogleGenAI(
    #     api_key=os.getenv("GOOGLE_API_KEY"),
    #     model="gemini-2.0-flash-001",
    # )
    # llm = Anthropic(
    #   model="claude-3-7-sonnet-latest",
    #   api_key=os.getenv("ANTHROPIC_API_KEY")
    # )

    # Load the tools from the Toolbox server
    client = ToolboxClient("http://127.0.0.1:5000")
    tools = await client.aload_toolset()

    agent = AgentWorkflow.from_tools_or_functions(
        tools,
        llm=llm,
        system_prompt=prompt,
    )
    ctx = Context(agent)
    for query in queries:
        response = await agent.arun(user_msg=query, ctx=ctx)
        print(f"---- {query} ----")
        print(str(response))

asyncio.run(main())
{{< /tab >}}
{{< tab header="ADK" lang="python" >}}
from google.adk.agents import Agent
from google.adk.runners import Runner
from google.adk.sessions import InMemorySessionService
from google.adk.artifacts.in_memory_artifact_service import InMemoryArtifactService
from google.genai import types # For constructing message content
from toolbox_core import ToolboxSyncClient

import os
os.environ['GOOGLE_GENAI_USE_VERTEXAI'] = 'True'

# TODO(developer): Replace 'YOUR_PROJECT_ID' with your Google Cloud Project ID

os.environ['GOOGLE_CLOUD_PROJECT'] = 'YOUR_PROJECT_ID'

# TODO(developer): Replace 'us-central1' with your Google Cloud Location (region)

os.environ['GOOGLE_CLOUD_LOCATION'] = 'us-central1'

# --- Load Tools from Toolbox ---

# TODO(developer): Ensure the Toolbox server is running at <http://127.0.0.1:5000>

with ToolboxSyncClient("<http://127.0.0.1:5000>") as toolbox_client:
    # TODO(developer): Replace "my-toolset" with the actual ID of your toolset as configured in your MCP Toolbox server.
    agent_toolset = toolbox_client.load_toolset("my-toolset")

    # --- Define the Agent's Prompt ---
    prompt = """
      You're a helpful hotel assistant. You handle hotel searching, booking and
      cancellations. When the user searches for a hotel, mention it's name, id,
      location and price tier. Always mention hotel ids while performing any
      searches. This is very important for any operations. For any bookings or
      cancellations, please provide the appropriate confirmation. Be sure to
      update checkin or checkout dates if mentioned by the user.
      Don't ask for confirmations from the user.
    """

    # --- Configure the Agent ---

    root_agent = Agent(
        model='gemini-2.0-flash-001',
        name='hotel_agent',
        description='A helpful AI assistant that can search and book hotels.',
        instruction=prompt,
        tools=agent_toolset, # Pass the loaded toolset
    )

    # --- Initialize Services for Running the Agent ---
    session_service = InMemorySessionService()
    artifacts_service = InMemoryArtifactService()
    # Create a new session for the interaction.
    session = session_service.create_session(
        state={}, app_name='hotel_agent', user_id='123'
    )

    runner = Runner(
        app_name='hotel_agent',
        agent=root_agent,
        artifact_service=artifacts_service,
        session_service=session_service,
    )

    # --- Define Queries and Run the Agent ---
    queries = [
        "Find hotels in Basel with Basel in it's name.",
        "Can you book the Hilton Basel for me?",
        "Oh wait, this is too expensive. Please cancel it and book the Hyatt Regency instead.",
        "My check in dates would be from April 10, 2024 to April 19, 2024.",
    ]

    for query in queries:
        content = types.Content(role='user', parts=[types.Part(text=query)])
        events = runner.run(session_id=session.id,
                            user_id='123', new_message=content)

        responses = (
          part.text
          for event in events
          for part in event.content.parts
          if part.text is not None
        )

        for text in responses:
          print(text)
{{< /tab >}}
{{< /tabpane >}}

    {{< tabpane text=true persist=header >}}
{{% tab header="Core" lang="en" %}}
To learn more about the Core SDK, check out the [Toolbox Core SDK
documentation.](https://github.com/googleapis/mcp-toolbox-sdk-python/blob/main/packages/toolbox-core/README.md)
{{% /tab %}}
{{% tab header="Langchain" lang="en" %}}
To learn more about Agents in LangChain, check out the [LangGraph Agent
documentation.](https://langchain-ai.github.io/langgraph/reference/prebuilt/#langgraph.prebuilt.chat_agent_executor.create_react_agent)
{{% /tab %}}
{{% tab header="LlamaIndex" lang="en" %}}
To learn more about Agents in LlamaIndex, check out the [LlamaIndex
AgentWorkflow
documentation.](https://docs.llamaindex.ai/en/stable/examples/agent/agent_workflow_basic/)
{{% /tab %}}
{{% tab header="ADK" lang="en" %}}
To learn more about Agents in ADK, check out the [ADK
documentation.](https://google.github.io/adk-docs/)
{{% /tab %}}
{{< /tabpane >}}

1. Run your agent, and observe the results:

    ```sh
    python hotel_agent.py
    ```
