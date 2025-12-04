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

1. Installed [Python 3.10+][install-python] (including [pip][install-pip] and
   your preferred virtual environment tool for managing dependencies e.g.
   [venv][install-venv]).
1. Installed [PostgreSQL 16+ and the `psql` client][install-postgres].

[install-python]: https://wiki.python.org/moin/BeginnersGuide/Download
[install-pip]: https://pip.pypa.io/en/stable/installation/
[install-venv]:
    https://packaging.python.org/en/latest/tutorials/installing-packages/#creating-virtual-environments
[install-postgres]: https://www.postgresql.org/download/

### Cloud Setup (Optional)

{{< regionInclude "quickstart/shared/cloud_setup.md" "cloud_setup" >}}

## Step 1: Set up your database

{{< regionInclude "quickstart/shared/database_setup.md" "database_setup" >}}

## Step 2: Install and configure Toolbox

{{< regionInclude "quickstart/shared/configure_toolbox.md" "configure_toolbox" >}}

## Step 3: Connect your agent to Toolbox

In this section, we will write and run an agent that will load the Tools
from Toolbox.

{{< notice tip>}}
If you prefer to experiment within a Google Colab environment, you can connect
to a [local
runtime](https://research.google.com/colaboratory/local-runtimes.html).
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

1. Create the agent:
{{< tabpane persist=header >}}
{{% tab header="ADK" text=true %}}

1. Create a new agent project. This will create a new directory named `my_agent`
   with a file `agent.py`.

    ```bash
    adk create my_agent
    ```
    <br/>

1. Update `my_agent/agent.py` with the following content to connect to Toolbox:
    ```py
    {{< include "quickstart/python/adk/quickstart.py" >}}
    ```
    <br/>

1. Create a `.env` file with your Google API key:
    ```bash
    echo 'GOOGLE_API_KEY="YOUR_API_KEY"' > my_agent/.env
    ```
{{% /tab %}}
{{% tab header="LangChain" text=true %}}
Create a new file named `agent.py` and copy the following code:
```py
{{< include "quickstart/python/langchain/quickstart.py" >}}
```
{{% /tab %}}
{{% tab header="LlamaIndex" text=true %}}
Create a new file named `agent.py` and copy the following code:
```py
{{< include "quickstart/python/llamaindex/quickstart.py" >}}
```
{{% /tab %}}
{{% tab header="Core" text=true %}}
Create a new file named `agent.py` and copy the following code:
```py
{{< include "quickstart/python/core/quickstart.py" >}}
```
{{% /tab %}}
{{< /tabpane >}}

    {{< tabpane text=true persist=header >}}
{{% tab header="ADK" lang="en" %}}
To learn more about Agent Development Kit, check out the [ADK
Documentation](https://google.github.io/adk-docs/get-started/python/).
{{% /tab %}}
{{% tab header="Langchain" lang="en" %}}
To learn more about Agents in LangChain, check out the [LangGraph Agent
Documentation](https://langchain-ai.github.io/langgraph/reference/prebuilt/#langgraph.prebuilt.chat_agent_executor.create_react_agent).
{{% /tab %}}
{{% tab header="LlamaIndex" lang="en" %}}
To learn more about Agents in LlamaIndex, check out the [LlamaIndex
AgentWorkflow
Documentation](https://docs.llamaindex.ai/en/stable/examples/agent/agent_workflow_basic/).
{{% /tab %}}
{{% tab header="Core" lang="en" %}}
To learn more about tool calling with Google GenAI, check out the
[Google GenAI
Documentation](https://github.com/googleapis/python-genai?tab=readme-ov-file#manually-declare-and-invoke-a-function-for-function-calling).
{{% /tab %}}
{{< /tabpane >}}

4. Run your agent, and observe the results:

    {{< tabpane persist=header >}}
{{% tab header="ADK" text=true %}}
Run your agent locally for testing:
```sh
adk run my_agent
```
<br/>

Alternatively, serve it via a web interface:
```sh
adk web --port 8000
```
<br/>

For more information, refer to the ADK documentation on [Running
Agents](https://google.github.io/adk-docs/get-started/python/#run-your-agent)
and [Deploying to Cloud](https://google.github.io/adk-docs/deploy/).

{{% /tab %}}
{{< tab header="Langchain" lang="bash" >}}
python agent.py
{{< /tab >}}
{{< tab header="LlamaIndex" lang="bash" >}}
python agent.py
{{< /tab >}}
{{< tab header="Core" lang="bash" >}}
python agent.py
{{< /tab >}}
    {{< /tabpane >}}

{{< notice info >}}
For more information, visit the [Python SDK
repo](https://github.com/googleapis/mcp-toolbox-sdk-python).
{{</ notice >}}
