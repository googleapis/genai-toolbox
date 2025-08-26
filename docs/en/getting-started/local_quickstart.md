---
title: "Python Quickstart (Local)"
type: docs
weight: 2
description: >
  How to get started running Toolbox locally with [Python](https://github.com/googleapis/mcp-toolbox-sdk-python), PostgreSQL, and  [Agent Development Kit](https://google.github.io/adk-docs/),
  [LangGraph](https://www.langchain.com/langgraph), [LlamaIndex](https://www.llamaindex.ai/) or [GoogleGenAI](https://pypi.org/project/google-genai/).
---

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/googleapis/genai-toolbox/blob/main/docs/en/getting-started/colab_quickstart.ipynb)

## Before you begin

This guide assumes you have already done the following:

1. Installed [Python 3.9+][install-python] (including [pip][install-pip] and
   your preferred virtual environment tool for managing dependencies e.g. [venv][install-venv]).
1. Installed [PostgreSQL 16+ and the `psql` client][install-postgres].

### Cloud Setup (Optional)
{{< regionInclude "quickstart/shared/cloud_setup.md" "cloud_setup" >}}

## Step 1: Set up your database
{{< regionInclude "quickstart/shared/database_setup.md" "database_setup" >}}

## Step 2: Install and configure Toolbox
{{< regionInclude "quickstart/shared/configure_toolbox.md" "configure_toolbox" >}}

## Step 3: Connect your agent to Toolbox

In this section, we will write and run an agent that will load the Tools
from Toolbox.

{{< notice tip>}} If you prefer to experiment within a Google Colab environment,
you can connect to a
[local runtime](https://research.google.com/colaboratory/local-runtimes.html).
{{< /notice >}}

1. In a new terminal, install the SDK package.

    {{< tabpane persist=header >}}
{{< tab header="ADK" lang="bash" >}}

pip install toolbox-core
{{< /tab >}}
{{< tab header="Langchain" lang="bash" >}}

pip install toolbox-langchain
{{< /tab >}}
{{< tab header="LlamaIndex" lang="bash" >}}

pip install toolbox-llamaindex
{{< /tab >}}
{{< tab header="Core" lang="bash" >}}

pip install toolbox-core
{{< /tab >}}
{{< /tabpane >}}

1. Install other required dependencies:

    {{< tabpane persist=header >}}
{{< tab header="ADK" lang="bash" >}}

pip install google-adk
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
{{< tab header="Core" lang="bash" >}}

pip install google-genai
{{< /tab >}}
{{< /tabpane >}}

1. Create a new file named `hotel_agent.py` and copy the following
   code to create an agent:
    {{< tabpane persist=header >}}
{{< tab header="ADK" lang="python" >}}
from google.adk.agents import Agent
from google.adk.runners import Runner
from google.adk.sessions import InMemorySessionService
from google.adk.artifacts.in_memory_artifact_service import InMemoryArtifactService
from google.genai import types
from toolbox_core import ToolboxSyncClient

import asyncio
import os

# TODO(developer): replace this with your Google API key

os.environ['GOOGLE_API_KEY'] = 'your-api-key'

async def main():
  with ToolboxSyncClient("<http://127.0.0.1:5000>") as toolbox_client:

      prompt = """
        You're a helpful hotel assistant. You handle hotel searching, booking and
        cancellations. When the user searches for a hotel, mention it's name, id,
        location and price tier. Always mention hotel ids while performing any
        searches. This is very important for any operations. For any bookings or
        cancellations, please provide the appropriate confirmation. Be sure to
        update checkin or checkout dates if mentioned by the user.
        Don't ask for confirmations from the user.
      """

      root_agent = Agent(
          model='gemini-2.0-flash-001',
          name='hotel_agent',
          description='A helpful AI assistant.',
          instruction=prompt,
          tools=toolbox_client.load_toolset("my-toolset"),
      )

      session_service = InMemorySessionService()
      artifacts_service = InMemoryArtifactService()
      session = await session_service.create_session(
          state={}, app_name='hotel_agent', user_id='123'
      )
      runner = Runner(
          app_name='hotel_agent',
          agent=root_agent,
          artifact_service=artifacts_service,
          session_service=session_service,
      )

      queries = [
          "Find hotels in Basel with Basel in its name.",
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

asyncio.run(main())
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
    async with ToolboxClient("http://127.0.0.1:5000") as client:
        tools = await client.aload_toolset()

        agent = create_react_agent(model, tools, checkpointer=MemorySaver())

        config = {"configurable": {"thread_id": "thread-1"}}
        for query in queries:
            inputs = {"messages": [("user", prompt + query)]}
            response = agent.invoke(inputs, stream_mode="values", config=config)
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
    "Find hotels in Basel with Basel in its name.",
    "Can you book the Hilton Basel for me?",
    "Oh wait, this is too expensive. Please cancel it and book the Hyatt Regency instead.",
    "My check in dates would be from April 10, 2024 to April 19, 2024.",
]

async def main():
    # TODO(developer): replace this with another model if needed
    llm = GoogleGenAI(
        model="gemini-2.0-flash-001",
        vertexai_config={"project": "project-id", "location": "us-central1"},
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
    async with ToolboxClient("http://127.0.0.1:5000") as client:
        tools = await client.aload_toolset()

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
    "Find hotels in Basel with Basel in its name.",
    "Please book the hotel Hilton Basel for me.",
    "This is too expensive. Please cancel it.",
    "Please book Hyatt Regency for me",
    "My check in dates for my booking would be from April 10, 2024 to April 19, 2024.",
]

async def main():
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

asyncio.run(main())

{{< /tab >}}
{{< /tabpane >}}

    {{< tabpane text=true persist=header >}}
{{% tab header="ADK" lang="en" %}}
To learn more about Agent Development Kit, check out the [ADK
documentation.](https://google.github.io/adk-docs/)
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
{{% tab header="Core" lang="en" %}}
To learn more about tool calling with Google GenAI, check out the
[Google GenAI
Documentation](https://github.com/googleapis/python-genai?tab=readme-ov-file#manually-declare-and-invoke-a-function-for-function-calling).
{{% /tab %}}
{{< /tabpane >}}

1. Run your agent, and observe the results:

    ```sh
    python hotel_agent.py
    ```

{{< notice info >}}
For more information, visit the [Python SDK repo](https://github.com/googleapis/mcp-toolbox-sdk-python).
{{</ notice >}}
