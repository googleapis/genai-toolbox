---
title: "Quickstart (MCP with Neo4j)"
type: docs
weight: 1
description: >
  How to get started running Toolbox with MCP Inspector and Neo4j as the source.
---

## Overview

[Model Context Protocol](https://modelcontextprotocol.io) is an open protocol that standardizes how applications provide context to LLMs. Check out this page on how to [connect to Toolbox via MCP](../../../how-to/connect_via_mcp.md).


## Step 1: Set up your Neo4j Database and Data

In this section, you'll set up a database and populate it with sample data for a movies-related agent. This guide assumes you have a running Neo4j instance, either locally or in the cloud.

. **Populate the database with data.**
To make this quickstart straightforward, we'll use the built-in Movies dataset available in Neo4j.

. In your Neo4j Browser, run the following command to create and populate the database:
+
```cypher
:play movies
````

. Follow the instructions to load the data. This will create a graph with `Movie`, `Person`, and `Actor` nodes and their relationships.


## Step 2: Install and configure Toolbox

In this section, we will install the MCP Toolbox, configure our tools in a `tools.yaml` file, and then run the Toolbox server.

. **Install the Toolbox binary.**
The simplest way to get started is to download the latest binary for your operating system.

. Download the latest version of Toolbox as a binary:
\+

```bash
export OS="linux/amd64" # one of linux/amd64, darwin/arm64, darwin/amd64, or windows/amd64
curl -O [https://storage.googleapis.com/genai-toolbox/v0.16.0/$OS/toolbox](https://storage.googleapis.com/genai-toolbox/v0.16.0/$OS/toolbox)
```

  + 
. Make the binary executable:
\+

```bash
chmod +x toolbox
```

. **Create the `tools.yaml` file.**
This file defines your Neo4j source and the specific tools that will be exposed to your AI agent.
\+
{{\< notice tip \>}}
Authentication for the Neo4j source uses standard username and password fields. For production use, it is highly recommended to use environment variables for sensitive information like passwords.
{{\< /notice \>}}
\+
Write the following into a `tools.yaml` file:
\+

```yaml
sources:
  my-neo4j-source:
    kind: neo4j
    uri: bolt://localhost:7687
    user: neo4j
    password: my-password # Replace with your actual password

tools:
  search-movies-by-actor:
    kind: neo4j-cypher
    source: my-neo4j-source
    description: "Searches for movies an actor has appeared in based on their name. Useful for questions like 'What movies has Tom Hanks been in?'"
    parameters:
      - name: actor_name
        type: string
        description: The full name of the actor to search for.
    statement: |
      MATCH (p:Person {name: $actor_name}) -[:ACTED_IN]-> (m:Movie)
      RETURN m.title AS title, m.year AS year, m.genre AS genre
  
  get-actor-for-movie:
    kind: neo4j-cypher
    source: my-neo4j-source
    description: "Finds the actors who starred in a specific movie. Useful for questions like 'Who acted in Inception?'"
    parameters:
      - name: movie_title
        type: string
        description: The exact title of the movie.
    statement: |
      MATCH (p:Person) -[:ACTED_IN]-> (m:Movie {title: $movie_title})
      RETURN p.name AS actor
```

. **Start the Toolbox server.**
Run the Toolbox server, pointing to the `tools.yaml` file you created earlier.
\+

```bash
./toolbox --tools-file "tools.yaml"
```

## Step 3: Connect to MCP Inspector

. **Run the MCP Inspector:**
\+

```bash
npx @modelcontextprotocol/inspector
```

. Type `y` when it asks to install the inspector package.
. It should show the following when the MCP Inspector is up and running (please take note of `<YOUR_SESSION_TOKEN>`):
\+

```bash
Starting MCP inspector...
⚙️ Proxy server listening on localhost:6277
🔑 Session token: <YOUR_SESSION_TOKEN>
    Use this token to authenticate requests or set DANGEROUSLY_OMIT_AUTH=true to disable auth

🚀 MCP Inspector is up and running at:
    http://localhost:6274/?MCP_PROXY_AUTH_TOKEN=<YOUR_SESSION_TOKEN>
```

1. Open the above link in your browser.

1. For `Transport Type`, select `Streamable HTTP`.

1. For `URL`, type in `http://127.0.0.1:5000/mcp`.

1. For `Configuration` -\> `Proxy Session Token`, make sure `<YOUR_SESSION_TOKEN>` is present.

1. Click `Connect`.

1. Select `List Tools`, you will see a list of tools configured in `tools.yaml`.

1. Test out your tools here\!

