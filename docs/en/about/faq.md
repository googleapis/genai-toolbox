---
title: "FAQ"
type: docs
weight: 2
description: Frequently asked questions about Toolbox. 
---

## How can I deploy or run Toolbox?

MCP Toolbox for Databases is open-source and can be run or deployed to a
multitude of environments. For convenience, we release [compiled binaries and
docker images][release-notes] (but you can always compile yourself as well!).

For detailed instructions, check out these resources:

- [Quickstart: How to Run Locally](../getting-started/local_quickstart.md)
- [Deploy to Cloud Run](../how-to/deploy_toolbox.md)

[release-notes]: https://github.com/googleapis/genai-toolbox/releases/

## Do I need a Google Cloud account/project to get started with Toolbox?

Nope! While some of the sources Toolbox connects to may require GCP credentials,
Toolbox doesn't require them and can connect to a bunch of different resources
that don't.

## Does Toolbox take contributions from external users?

Absolutely! Please check out our [DEVELOPER.md][] for instructions on how to get
started developing _on_ Toolbox instead of with it, and the [CONTRIBUTING.md][]
for instructions on completing the CLA and getting a PR accepted.

[DEVELOPER.md]: https://github.com/googleapis/genai-toolbox/blob/main/DEVELOPER.md
[CONTRIBUTING.MD]: https://github.com/googleapis/genai-toolbox/blob/main/CONTRIBUTING.md

## Can Toolbox support a feature to let me do _$FOO_?

Maybe? The best place to start is by [opening an issue][github-issue] for
discussion (or seeing if there is already one open), so we can better understand
your use case and the best way to solve it. Generally we aim to prioritize the
most popular issues, so make sure to +1 ones you are the most interested in.

[github-issue]: https://github.com/googleapis/genai-toolbox/issues

## Can Toolbox be used for non-database tools?

**Yes!** While Toolbox is primarily focused on databases, it also supports generic 
**HTTP tools** (`kind: http`). These allow you to connect your agents to REST APIs 
and other web services, enabling workflows that extend beyond database interactions.

For configuration details, see the [HTTP Tools documentation](../resources/tools/http/http.md).

## Can I use _$BAR_ orchestration framework to use tools from Toolbox?

Currently, Toolbox only supports a limited number of client SDKs at our initial
launch. We are investigating support for more frameworks as well as more general
approaches for users without a framework -- look forward to seeing an update
soon.

## Why does Toolbox use a server-client architecture pattern?

Toolbox's server-client architecture allows us to more easily support a wide
variety of languages and frameworks with a centralized implementation. It also
allows us to tackle problems like connection pooling, auth, or caching more
completely than entirely client-side solutions.

## Why was Toolbox written in Go?

While a large part of the Gen AI Ecosystem is predominately Python, we opted to
use Go. We chose Go because it's still easy and simple to use, but also easier
to write fast, efficient, and concurrent servers. Additionally, given the
server-client architecture, we can still meet many developers where they are
with clients in their preferred language. As Gen AI matures, we want developers
to be able to use Toolbox on the serving path of mission critical applications.
It's easier to build the needed robustness, performance and scalability in Go
than in Python.

## Is Toolbox compatible with Model Context Protocol (MCP)?

Yes! Toolbox is compatible with [Anthropic's Model Context Protocol
(MCP)](https://modelcontextprotocol.io/). Please checkout [Connect via
MCP](../how-to/connect_via_mcp.md) on how to connect to Toolbox with an MCP
client.
