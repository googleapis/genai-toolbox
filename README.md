![logo](./logo.png)

# MCP Toolbox for Databases

[![Docs](https://img.shields.io/badge/docs-MCP_Toolbox-blue)](https://googleapis.github.io/genai-toolbox/)
[![Discord](https://img.shields.io/badge/Discord-%235865F2.svg?style=flat&logo=discord&logoColor=white)](https://discord.gg/Dmm69peqjh)
[![Medium](https://img.shields.io/badge/Medium-12100E?style=flat&logo=medium&logoColor=white)](https://medium.com/@mcp_toolbox)
[![Go Report Card](https://goreportcard.com/badge/github.com/googleapis/genai-toolbox)](https://goreportcard.com/report/github.com/googleapis/genai-toolbox)

> [!NOTE]
> MCP Toolbox for Databases is currently in beta, and may see breaking
> changes until the first stable release (v1.0).

MCP Toolbox for Databases is an open source MCP server for databases. It enables
you to develop tools easier, faster, and more securely by handling the complexities
such as connection pooling, authentication, and more.

This README provides a brief overview. For comprehensive details, see the [full
documentation](https://googleapis.github.io/genai-toolbox/).

> [!NOTE]
> This solution was originally named “Gen AI Toolbox for Databases” as
> its initial development predated MCP, but was renamed to align with recently
> added MCP compatibility.

<!-- TOC ignore:true -->
## Table of Contents

<!-- TOC -->

- [Why Toolbox?](#why-toolbox)
- [General Architecture](#general-architecture)
- [Getting Started](#getting-started)
  - [Installing the server](#installing-the-server)
  - [Running the server](#running-the-server)
  - [Integrating your application](#integrating-your-application)
- [Configuration](#configuration)
  - [Sources](#sources)
  - [Tools](#tools)
  - [Toolsets](#toolsets)
  - [Prompts](#prompts)
- [Versioning](#versioning)
  - [Pre-1.0.0 Versioning](#pre-100-versioning)
  - [Post-1.0.0 Versioning](#post-100-versioning)
- [Contributing](#contributing)
- [Community](#community)

<!-- /TOC -->

## Why Toolbox?

Toolbox helps you build Gen AI tools that let your agents access data in your
database. Toolbox provides:

- **Simplified development**: Integrate tools to your agent in less than 10
  lines of code, reuse tools between multiple agents or frameworks, and deploy
  new versions of tools more easily.
- **Better performance**: Best practices such as connection pooling,
  authentication, and more.
- **Enhanced security**: Integrated auth for more secure access to your data
- **End-to-end observability**: Out of the box metrics and tracing with built-in
  support for OpenTelemetry.

**⚡ Supercharge Your Workflow with an AI Database Assistant ⚡**

Stop context-switching and let your AI assistant become a true co-developer. By
[connecting your IDE to your databases with MCP Toolbox][connect-ide], you can
delegate complex and time-consuming database tasks, allowing you to build faster
and focus on what matters. This isn't just about code completion; it's about
giving your AI the context it needs to handle the entire development lifecycle.

Here’s how it will save you time:

- **Query in Plain English**: Interact with your data using natural language
  right from your IDE. Ask complex questions like, *"How many orders were
  delivered in 2024, and what items were in them?"* without writing any SQL.
- **Automate Database Management**: Simply describe your data needs, and let the
  AI assistant manage your database for you. It can handle generating queries,
  creating tables, adding indexes, and more.
- **Generate Context-Aware Code**: Empower your AI assistant to generate
  application code and tests with a deep understanding of your real-time
  database schema.  This accelerates the development cycle by ensuring the
  generated code is directly usable.
- **Slash Development Overhead**: Radically reduce the time spent on manual
  setup and boilerplate. MCP Toolbox helps streamline lengthy database
  configurations, repetitive code, and error-prone schema migrations.

Learn [how to connect your AI tools (IDEs) to Toolbox using MCP][connect-ide].

[connect-ide]: https://googleapis.github.io/genai-toolbox/how-to/connect-ide/

## General Architecture

Toolbox sits between your application's orchestration framework and your
database, providing a control plane that is used to modify, distribute, or
invoke tools. It simplifies the management of your tools by providing you with a
centralized location to store and update tools, allowing you to share tools
between agents and applications and update those tools without necessarily
redeploying your application.

![architecture](./docs/en/getting-started/introduction/architecture.png)

## Getting Started

### Installing the server

For the latest version, check the [releases page][releases] and use the
following instructions for your OS and CPU architecture.

[releases]: https://github.com/googleapis/genai-toolbox/releases

<details open>
<summary>Binary</summary>

To install Toolbox as a binary:

<!-- {x-release-please-start-version} -->
> <details>
> <summary>Linux (AMD64)</summary>
>
> To install Toolbox as a binary on Linux (AMD64):
>
> ```sh
> # see releases page for other versions
> export VERSION=0.17.0
> curl -L -o toolbox https://storage.googleapis.com/genai-toolbox/v$VERSION/linux/amd64/toolbox
> chmod +x toolbox
> ```
>
> </details>
> <details>
> <summary>macOS (Apple Silicon)</summary>
>
> To install Toolbox as a binary on macOS (Apple Silicon):
>
> ```sh
> # see releases page for other versions
> export VERSION=0.17.0
> curl -L -o toolbox https://storage.googleapis.com/genai-toolbox/v$VERSION/darwin/arm64/toolbox
> chmod +x toolbox
> ```
>
> </details>
> <details>
> <summary>macOS (Intel)</summary>
>
> To install Toolbox as a binary on macOS (Intel):
>
> ```sh
> # see releases page for other versions
> export VERSION=0.17.0
> curl -L -o toolbox https://storage.googleapis.com/genai-toolbox/v$VERSION/darwin/amd64/toolbox
> chmod +x toolbox
> ```
>
> </details>
> <details>
> <summary>Windows (AMD64)</summary>
>
> To install Toolbox as a binary on Windows (AMD64):
>
> ```powershell
> # see releases page for other versions
> $VERSION = "0.17.0"
> Invoke-WebRequest -Uri "https://storage.googleapis.com/genai-toolbox/v$VERSION/windows/amd64/toolbox.exe" -OutFile "toolbox.exe"
> ```
>
> </details>
</details>

<details>
<summary>Container image</summary>
You can also install Toolbox as a container:

```sh
# see releases page for other versions
export VERSION=0.17.0
docker pull us-central1-docker.pkg.dev/database-toolbox/toolbox/toolbox:$VERSION
```

</details>

<details>
<summary>Homebrew</summary>

To install Toolbox using Homebrew on macOS or Linux:

```sh
brew install mcp-toolbox
```

</details>

<details>
<summary>Compile from source</summary>

To install from source, ensure you have the latest version of
[Go installed](https://go.dev/doc/install), and then run the following command:

```sh
go install github.com/googleapis/genai-toolbox@v0.17.0
```
<!-- {x-release-please-end} -->

</details>

<details>
<summary>Gemini CLI Extensions</summary>

To install Gemini CLI Extensions for MCP Toolbox, run the following command:

```sh
gemini extensions install https://github.com/gemini-cli-extensions/mcp-toolbox
```

</details>

### Running the server

[Configure](#configuration) a `tools.yaml` to define your tools, and then
execute `toolbox` to start the server:

<details open>
<summary>Binary</summary>

To run Toolbox from binary:

```sh
./toolbox --tools-file "tools.yaml"
```

ⓘ **NOTE:**  
Toolbox enables dynamic reloading by default. To disable, use the
`--disable-reload` flag.

</details>

<details>

<summary>Container image</summary>

To run the server after pulling the [container image](#installing-the-server):

```sh
export VERSION=0.11.0 # Use the version you pulled
docker run -p 5000:5000 \
-v $(pwd)/tools.yaml:/app/tools.yaml \
us-central1-docker.pkg.dev/database-toolbox/toolbox/toolbox:$VERSION \
--tools-file "/app/tools.yaml"
```

ⓘ **NOTE:**  
The `-v` flag mounts your local `tools.yaml` into the container, and `-p` maps
the container's port `5000` to your host's port `5000`.

</details>

<details>

<summary>Source</summary>

To run the server directly from source, navigate to the project root directory
and run:

```sh
go run .
```

ⓘ **NOTE:**  
This command runs the project from source, and is more suitable for development
and testing. It does **not** compile a binary into your `$GOPATH`. If you want
to compile a binary instead, refer the [Developer
Documentation](./DEVELOPER.md#building-the-binary).

</details>

<details>

<summary>Homebrew</summary>

If you installed Toolbox using [Homebrew](https://brew.sh/), the `toolbox`
binary is available in your system path. You can start the server with the same
command:

```sh
toolbox --tools-file "tools.yaml"
```

</details>

<details>

<summary>Gemini CLI</summary>

Interact with your custom tools using natural language. Check
[gemini-cli-extensions/mcp-toolbox](https://github.com/gemini-cli-extensions/mcp-toolbox)
for more information.

</details>

You can use `toolbox help` for a full list of flags! To stop the server, send a
terminate signal (`ctrl+c` on most platforms).

For more detailed documentation on deploying to different environments, check
out the resources in the [How-to
section](https://googleapis.github.io/genai-toolbox/how-to/)

### Integrating your application

Once your server is up and running, you can load the tools into your
application. See below the list of Client SDKs for using various frameworks:

<details open>
  <summary>Python (<a href="https://github.com/googleapis/mcp-toolbox-sdk-python">Github</a>)</summary>
  <br>
  <blockquote>

  <details open>
    <summary>Core</summary>

1. Install [Toolbox Core SDK][toolbox-core]:

    ```bash
    pip install toolbox-core
    ```

1. Load tools:

    ```python
    from toolbox_core import ToolboxClient

    # update the url to point to your server
    async with ToolboxClient("http://127.0.0.1:5000") as client:

        # these tools can be passed to your application!
        tools = await client.load_toolset("toolset_name")
    ```

For more detailed instructions on using the Toolbox Core SDK, see the
[project's README][toolbox-core-readme].

[toolbox-core]: https://pypi.org/project/toolbox-core/
[toolbox-core-readme]: https://github.com/googleapis/mcp-toolbox-sdk-python/tree/main/packages/toolbox-core/README.md

  </details>
  <details>
    <summary>LangChain / LangGraph</summary>

1. Install [Toolbox LangChain SDK][toolbox-langchain]:

    ```bash
    pip install toolbox-langchain
    ```

1. Load tools:

    ```python
    from toolbox_langchain import ToolboxClient

    # update the url to point to your server
    async with ToolboxClient("http://127.0.0.1:5000") as client:

        # these tools can be passed to your application!
        tools = client.load_toolset()
    ```

    For more detailed instructions on using the Toolbox LangChain SDK, see the
    [project's README][toolbox-langchain-readme].

    [toolbox-langchain]: https://pypi.org/project/toolbox-langchain/
    [toolbox-langchain-readme]: https://github.com/googleapis/mcp-toolbox-sdk-python/blob/main/packages/toolbox-langchain/README.md

  </details>
  <details>
    <summary>LlamaIndex</summary>

1. Install [Toolbox Llamaindex SDK][toolbox-llamaindex]:

    ```bash
    pip install toolbox-llamaindex
    ```

1. Load tools:

    ```python
    from toolbox_llamaindex import ToolboxClient

    # update the url to point to your server
    async with ToolboxClient("http://127.0.0.1:5000") as client:

        # these tools can be passed to your application!
        tools = client.load_toolset()
    ```

    For more detailed instructions on using the Toolbox Llamaindex SDK, see the
    [project's README][toolbox-llamaindex-readme].

    [toolbox-llamaindex]: https://pypi.org/project/toolbox-llamaindex/
    [toolbox-llamaindex-readme]: https://github.com/googleapis/genai-toolbox-llamaindex-python/blob/main/README.md

  </details>
</details>
</blockquote>
<details>
  <summary>Javascript/Typescript (<a href="https://github.com/googleapis/mcp-toolbox-sdk-js">Github</a>)</summary>
  <br>
  <blockquote>

  <details open>
    <summary>Core</summary>

1. Install [Toolbox Core SDK][toolbox-core-js]:

    ```bash
    npm install @toolbox-sdk/core
    ```

1. Load tools:

    ```javascript
    import { ToolboxClient } from '@toolbox-sdk/core';

    // update the url to point to your server
    const URL = 'http://127.0.0.1:5000';
    let client = new ToolboxClient(URL);

    // these tools can be passed to your application!
    const tools = await client.loadToolset('toolsetName');
    ```

    For more detailed instructions on using the Toolbox Core SDK, see the
    [project's README][toolbox-core-js-readme].

    [toolbox-core-js]: https://www.npmjs.com/package/@toolbox-sdk/core
    [toolbox-core-js-readme]: https://github.com/googleapis/mcp-toolbox-sdk-js/blob/main/packages/toolbox-core/README.md

  </details>
  <details>
    <summary>LangChain / LangGraph</summary>

1. Install [Toolbox Core SDK][toolbox-core-js]:

    ```bash
    npm install @toolbox-sdk/core
    ```

2. Load tools:

    ```javascript
    import { ToolboxClient } from '@toolbox-sdk/core';

    // update the url to point to your server
    const URL = 'http://127.0.0.1:5000';
    let client = new ToolboxClient(URL);

    // these tools can be passed to your application!
    const toolboxTools = await client.loadToolset('toolsetName');

    // Define the basics of the tool: name, description, schema and core logic
    const getTool = (toolboxTool) => tool(currTool, {
        name: toolboxTool.getName(),
        description: toolboxTool.getDescription(),
        schema: toolboxTool.getParamSchema()
    });

    // Use these tools in your Langchain/Langraph applications
    const tools = toolboxTools.map(getTool);
    ```

  </details>
  <details>
    <summary>Genkit</summary>

1. Install [Toolbox Core SDK][toolbox-core-js]:

    ```bash
    npm install @toolbox-sdk/core
    ```

2. Load tools:

    ```javascript
    import { ToolboxClient } from '@toolbox-sdk/core';
    import { genkit } from 'genkit';

    // Initialise genkit
    const ai = genkit({
        plugins: [
            googleAI({
                apiKey: process.env.GEMINI_API_KEY || process.env.GOOGLE_API_KEY
            })
        ],
        model: googleAI.model('gemini-2.0-flash'),
    });

    // update the url to point to your server
    const URL = 'http://127.0.0.1:5000';
    let client = new ToolboxClient(URL);

    // these tools can be passed to your application!
    const toolboxTools = await client.loadToolset('toolsetName');

    // Define the basics of the tool: name, description, schema and core logic
    const getTool = (toolboxTool) => ai.defineTool({
        name: toolboxTool.getName(),
        description: toolboxTool.getDescription(),
        schema: toolboxTool.getParamSchema()
    }, toolboxTool)

    // Use these tools in your Genkit applications
    const tools = toolboxTools.map(getTool);
    ```

  </details>
</details>
</blockquote>
<details>
  <summary>Go (<a href="https://github.com/googleapis/mcp-toolbox-sdk-go">Github</a>)</summary>
  <br>
  <blockquote>

  <details open>
    <summary>Core</summary>

1. Install [Toolbox Go SDK][toolbox-go]:

    ```bash
    go get github.com/googleapis/mcp-toolbox-sdk-go
    ```

1. Load tools:

    ```go
    package main

    import (
      "github.com/googleapis/mcp-toolbox-sdk-go/core"
      "context"
    )

    func main() {
      // Make sure to add the error checks
      // update the url to point to your server
      URL := "http://127.0.0.1:5000";
      ctx := context.Background()

      client, err := core.NewToolboxClient(URL)

      // Framework agnostic tools
      tools, err := client.LoadToolset("toolsetName", ctx)
    }
    ```

    For more detailed instructions on using the Toolbox Go SDK, see the
    [project's README][toolbox-core-go-readme].

    [toolbox-go]: https://pkg.go.dev/github.com/googleapis/mcp-toolbox-sdk-go/core
    [toolbox-core-go-readme]: https://github.com/googleapis/mcp-toolbox-sdk-go/blob/main/core/README.md

  </details>
  <details>
    <summary>LangChain Go</summary>

1. Install [Toolbox Go SDK][toolbox-go]:

    ```bash
    go get github.com/googleapis/mcp-toolbox-sdk-go
    ```

2. Load tools:

    ```go
    package main

    import (
      "context"
      "encoding/json"

      "github.com/googleapis/mcp-toolbox-sdk-go/core"
      "github.com/tmc/langchaingo/llms"
    )

    func main() {
      // Make sure to add the error checks
      // update the url to point to your server
      URL := "http://127.0.0.1:5000"
      ctx := context.Background()

      client, err := core.NewToolboxClient(URL)

      // Framework agnostic tool
      tool, err := client.LoadTool("toolName", ctx)

      // Fetch the tool's input schema
      inputschema, err := tool.InputSchema()

      var paramsSchema map[string]any
      _ = json.Unmarshal(inputschema, &paramsSchema)

      // Use this tool with LangChainGo
      langChainTool := llms.Tool{
        Type: "function",
        Function: &llms.FunctionDefinition{
          Name:        tool.Name(),
          Description: tool.Description(),
          Parameters:  paramsSchema,
        },
      }
    }

    ```

  </details>
  <details>
    <summary>Genkit</summary>

1. Install [Toolbox Go SDK][toolbox-go]:

    ```bash
    go get github.com/googleapis/mcp-toolbox-sdk-go
    ```

2. Load tools:

    ```go
    package main
    import (
      "context"
      "encoding/json"

      "github.com/firebase/genkit/go/ai"
      "github.com/firebase/genkit/go/genkit"
      "github.com/googleapis/mcp-toolbox-sdk-go/core"
      "github.com/googleapis/mcp-toolbox-sdk-go/tbgenkit"
      "github.com/invopop/jsonschema"
    )

    func main() {
      // Make sure to add the error checks
      // Update the url to point to your server
      URL := "http://127.0.0.1:5000"
      ctx := context.Background()
      g, err := genkit.Init(ctx)

      client, err := core.NewToolboxClient(URL)

      // Framework agnostic tool
      tool, err := client.LoadTool("toolName", ctx)

      // Convert the tool using the tbgenkit package
      // Use this tool with Genkit Go
      genkitTool, err := tbgenkit.ToGenkitTool(tool, g)
      if err != nil {
        log.Fatalf("Failed to convert tool: %v\n", err)
      }
    }
    ```

  </details>
  <details>
    <summary>Go GenAI</summary>

1. Install [Toolbox Go SDK][toolbox-go]:

    ```bash
    go get github.com/googleapis/mcp-toolbox-sdk-go
    ```

2. Load tools:

    ```go
    package main

    import (
      "context"
      "encoding/json"

      "github.com/googleapis/mcp-toolbox-sdk-go/core"
      "google.golang.org/genai"
    )

    func main() {
      // Make sure to add the error checks
      // Update the url to point to your server
      URL := "http://127.0.0.1:5000"
      ctx := context.Background()

      client, err := core.NewToolboxClient(URL)

      // Framework agnostic tool
      tool, err := client.LoadTool("toolName", ctx)

      // Fetch the tool's input schema
      inputschema, err := tool.InputSchema()

      var schema *genai.Schema
      _ = json.Unmarshal(inputschema, &schema)

      funcDeclaration := &genai.FunctionDeclaration{
        Name:        tool.Name(),
        Description: tool.Description(),
        Parameters:  schema,
      }

      // Use this tool with Go GenAI
      genAITool := &genai.Tool{
        FunctionDeclarations: []*genai.FunctionDeclaration{funcDeclaration},
      }
    }
    ```

  </details>
  <details>
    <summary>OpenAI Go</summary>

1. Install [Toolbox Go SDK][toolbox-go]:

    ```bash
    go get github.com/googleapis/mcp-toolbox-sdk-go
    ```

2. Load tools:

    ```go
    package main

    import (
      "context"
      "encoding/json"

      "github.com/googleapis/mcp-toolbox-sdk-go/core"
      openai "github.com/openai/openai-go"
    )

    func main() {
      // Make sure to add the error checks
      // Update the url to point to your server
      URL := "http://127.0.0.1:5000"
      ctx := context.Background()

      client, err := core.NewToolboxClient(URL)

      // Framework agnostic tool
      tool, err := client.LoadTool("toolName", ctx)

      // Fetch the tool's input schema
      inputschema, err := tool.InputSchema()

      var paramsSchema openai.FunctionParameters
      _ = json.Unmarshal(inputschema, &paramsSchema)

      // Use this tool with OpenAI Go
      openAITool := openai.ChatCompletionToolParam{
        Function: openai.FunctionDefinitionParam{
          Name:        tool.Name(),
          Description: openai.String(tool.Description()),
          Parameters:  paramsSchema,
        },
      }

    }
    ```

  </details>
</details>
</blockquote>
</details>

## Configuration

The primary way to configure Toolbox is through the `tools.yaml` file. If you
have multiple files, you can tell toolbox which to load with the `--tools-file
tools.yaml` flag.

You can find more detailed reference documentation to all resource types in the
[Resources](https://googleapis.github.io/genai-toolbox/resources/).

### Sources

The `sources` section of your `tools.yaml` defines what data sources your
Toolbox should have access to. Most tools will have at least one source to
execute against.

```yaml
sources:
  my-pg-source:
    kind: postgres
    host: 127.0.0.1
    port: 5432
    database: toolbox_db
    user: toolbox_user
    password: my-password
```

For more details on configuring different types of sources, see the
[Sources](https://googleapis.github.io/genai-toolbox/resources/sources).

### Tools

The `tools` section of a `tools.yaml` define the actions an agent can take: what
kind of tool it is, which source(s) it affects, what parameters it uses, etc.

```yaml
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
```

For more details on configuring different types of tools, see the
[Tools](https://googleapis.github.io/genai-toolbox/resources/tools).

### Toolsets

The `toolsets` section of your `tools.yaml` allows you to define groups of tools
that you want to be able to load together. This can be useful for defining
different groups based on agent or application.

```yaml
toolsets:
    my_first_toolset:
        - my_first_tool
        - my_second_tool
    my_second_toolset:
        - my_second_tool
        - my_third_tool
```

You can load toolsets by name:

```python
# This will load all tools
all_tools = client.load_toolset()

# This will only load the tools listed in 'my_second_toolset'
my_second_toolset = client.load_toolset("my_second_toolset")
```

### Prompts

The `prompts` section of a `tools.yaml` defines prompts that can be used for
interactions with LLMs.

```yaml
prompts:
  code_review:
    description: "Asks the LLM to analyze code quality and suggest improvements."
    messages:
      - content: "Please review the following code for quality, correctness, and potential improvements: \n\n{{.code}}"
    arguments:
      - name: "code"
        description: "The code to review"
```

For more details on configuring prompts, see the
[Prompts](https://googleapis.github.io/genai-toolbox/resources/prompts).

## Versioning

This project uses [semantic versioning](https://semver.org/) (`MAJOR.MINOR.PATCH`).
Since the project is in a pre-release stage (version `0.x.y`), we follow the
standard conventions for initial  development:

### Pre-1.0.0 Versioning

While the major version is `0`, the public API should be considered unstable.
The version will be incremented  as follows:

- **`0.MINOR.PATCH`**: The **MINOR** version is incremented when we add
  new functionality or make breaking, incompatible API changes.
- **`0.MINOR.PATCH`**: The **PATCH** version is incremented for
  backward-compatible bug fixes.

### Post-1.0.0 Versioning

Once the project reaches a stable `1.0.0` release, the versioning will follow
the more common convention:

- **`MAJOR.MINOR.PATCH`**: Incremented for incompatible API changes.
- **`MAJOR.MINOR.PATCH`**: Incremented for new, backward-compatible functionality.
- **`MAJOR.MINOR.PATCH`**: Incremented for backward-compatible bug fixes.

The public API that this applies to is the CLI associated with Toolbox, the
interactions with official SDKs, and the definitions in the `tools.yaml` file.

## Contributing

Contributions are welcome. Please, see the [CONTRIBUTING](CONTRIBUTING.md)
to get started.

Please note that this project is released with a Contributor Code of Conduct.
By participating in this project you agree to abide by its terms. See
[Contributor Code of Conduct](CODE_OF_CONDUCT.md) for more information.

## Community

Join our [discord community](https://discord.gg/GQrFB3Ec3W) to connect with our developers!
