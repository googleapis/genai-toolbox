---
title: "Introduction"
type: docs
weight: 1
description: An introduction to MCP Toolbox for Databases.
---

MCP Toolbox for Databases is an open source MCP server for databases. It was
designed with enterprise-grade and production-quality in mind. It enables you to
develop tools easier, faster, and more securely by handling the complexities
such as connection pooling, authentication, and more.


{{< notice note >}} 
This product was originally named “Gen AI Toolbox for
Databases” as its initial development predated MCP, but was renamed to align
with recently added MCP compatibility. 
{{< /notice >}}

##  Why Toolbox?

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


## General Architecture

Toolbox sits between your application's orchestration framework and your
database, providing a control plane that is used to modify, distribute, or
invoke tools. It simplifies the management of your tools by providing you with a
centralized location to store and update tools, allowing you to share tools
between agents and applications and update those tools without necessarily
redeploying your application.

![architecture](./architecture.png)

## Getting Started

### Installing the server
For the latest version, check the [releases page][releases] and use the
following instructions for your OS and CPU architecture.

[releases]: https://github.com/googleapis/genai-toolbox/releases

<!-- {x-release-please-start-version} -->
{{< tabpane text=true >}}
{{% tab header="Binary" lang="en" %}}

To install Toolbox as a binary:

```sh
# see releases page for other versions
export VERSION=0.3.0
curl -O https://storage.googleapis.com/genai-toolbox/v$VERSION/linux/amd64/toolbox
chmod +x toolbox
```

{{% /tab %}}
{{% tab header="Container image" lang="en" %}}
You can also install Toolbox as a container:

```sh
# see releases page for other versions
export VERSION=0.3.0
docker pull us-central1-docker.pkg.dev/database-toolbox/toolbox/toolbox:$VERSION
```

{{% /tab %}}
{{% tab header="Compile from source" lang="en" %}}

To install from source, ensure you have the latest version of
[Go installed](https://go.dev/doc/install), and then run the following command:

```sh
go install github.com/googleapis/genai-toolbox@v0.3.0
```

{{% /tab %}}
{{< /tabpane >}}
<!-- {x-release-please-end} -->

### Running the server

[Configure](../configure.md) a `tools.yaml` to define your tools, and then
execute `toolbox` to start the server:

```sh
./toolbox --tools_file "tools.yaml"
```

You can use `toolbox help` for a full list of flags! To stop the server, send a
terminate signal (`ctrl+c` on most platforms).

For more detailed documentation on deploying to different environments, check
out the resources in the [How-to section](../../how-to/_index.md)

### Integrating your application

Once your server is up and running, you can load the tools into your
application. See below the list of Client SDKs for using various frameworks:

{{< tabpane text=true persist=header >}}
{{% tab header="Core" lang="en" %}}

Once you've installed the [Toolbox Core
SDK](https://pypi.org/project/toolbox-core/), you can load
tools:

{{< highlight python >}}
from toolbox_core import ToolboxClient

# update the url to point to your server
client = ToolboxClient("http://127.0.0.1:5000")

# these tools can be passed to your application! 
tools = await client.aload_toolset("toolset_name")
{{< /highlight >}}

For more detailed instructions on using the Toolbox Core SDK, see the
[project's README](https://github.com/googleapis/mcp-toolbox-sdk-python/blob/main/packages/toolbox-core/README.md).

{{% /tab %}}
{{% tab header="LangChain" lang="en" %}}

Once you've installed the [Toolbox LangChain
SDK](https://pypi.org/project/toolbox-langchain/), you can load
tools:

{{< highlight python >}}
from toolbox_langchain import ToolboxClient

# update the url to point to your server
client = ToolboxClient("http://127.0.0.1:5000")

# these tools can be passed to your application! 
tools = client.load_toolset()
{{< /highlight >}}

For more detailed instructions on using the Toolbox LangChain SDK, see the
[project's README](https://github.com/googleapis/genai-toolbox-langchain-python/blob/main/README.md).

{{% /tab %}}
{{% tab header="Llamaindex" lang="en" %}}

Once you've installed the [Toolbox Llamaindex
SDK](https://github.com/googleapis/genai-toolbox-llamaindex-python), you can load
tools:

{{< highlight python >}}
from toolbox_llamaindex import ToolboxClient

# update the url to point to your server
client = ToolboxClient("http://127.0.0.1:5000")

# these tools can be passed to your application! 
tools = client.load_toolset()
{{< /highlight >}}

For more detailed instructions on using the Toolbox Llamaindex SDK, see the
[project's README](https://github.com/googleapis/genai-toolbox-llamaindex-python/blob/main/README.md).

{{% /tab %}}
{{< /tabpane >}}
