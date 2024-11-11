# GenAI Toolbox SDK

This SDK allows you to seamlessly integrate the functionalities of
[Toolbox](https://github.com/googleapis/genai-toolbox) into your LLM
applications, enabling advanced orchestration and interaction with GenAI
models.

<!-- TOC ignore:true -->

## Table of Contents

<!-- TOC -->

- [Installation](#installation)
- [Usage](#usage)
- [Load a toolset](#load-a-toolset)
- [Use with LlamaIndex](#use-with-llamaindex)
- [Manual usage](#manual-usage)

<!-- /TOC -->

## Installation

You can install the Toolbox SDK for LlamaIndex using `pip`.

```bash
pip install toolbox-llamaindex-sdk
```

> [!IMPORTANT]
> This SDK is not yet available on PyPI. For now, install it from source by
following these [instructions](sdks/DEVELOPER.md#setting-up-a-development-environment).

## Usage

Import and initialize the toolbox client.

```python
from toolbox_llamaindex_sdk import ToolboxClient

# Replace with your Toolbox service's URL
toolbox = ToolboxClient("http://127.0.0.1:5000")
```

## Load a toolset

You can load a toolsets, that are collections of related tools.

```python
# Load all tools
tools = await toolbox.load_toolset()

# Load a specific toolset
tools = await toolbox.load_toolset("my-toolset")
```

## Use with LlamaIndex

LlamaIndex agents can dynamically choose and execute tools based on the user
input. The user can include the tools loaded from the Toolbox SDK in the
agent's toolkit.

```python
from llama_index.llms.vertex import Vertex
from llama_index.core.agent import ReActAgent

model = Vertex(model="gemini-1.5-pro")

# Initialize agent with tools
agent = ReActAgent.from_tools(tools, llm=model, verbose=True)

# Query the agent
response = agent.query("Get some response from the agent.")
```

## Manual usage

You can also execute a tool manually using the `acall` method.

```python
result = await tools[0].acall({"param1": "value1", "param2": "value2"})
```
