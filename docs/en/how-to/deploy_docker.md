---
title: "Deploy using Docker Compose"
type: docs
weight: 4
description: >
  How to deploy Toolbox using Docker Compose.
---

<!-- Contributor: Sujith R Pillai <sujithrpillai@gmail.com> -->

## Before you begin

1. [Install Docker Compose.](https://docs.docker.com/compose/install/)

## Configure `tools.yaml` file

Create a `tools.yaml` file that contains your configuration for Toolbox. For
details, see the
[configuration](https://github.com/googleapis/genai-toolbox/blob/main/README.md#configuration)
section.

## Deploy using Docker Compose

1. Create a `docker-compose.yml` file, customizing as needed:

```yaml
services:
  toolbox:
    # TODO: It is recommended to pin to a specific image version instead of latest.
    image:  us-central1-docker.pkg.dev/database-toolbox/toolbox/toolbox:latest
    hostname: toolbox
    platform: linux/amd64
    ports:
      - "5000:5000"
    volumes:
      - ./config:/config
    command: [ "toolbox", "--tools-file", "/config/tools.yaml", "--address", "0.0.0.0"]
    depends_on:
      db:
        condition: service_healthy
    networks:
      - tool-network
  db:
    # TODO: It is recommended to pin to a specific image version instead of latest.
    image: postgres
    hostname: db
    environment:
      POSTGRES_USER: toolbox_user
      POSTGRES_PASSWORD: my-password
      POSTGRES_DB: toolbox_db
    ports:
      - "5432:5432"
    volumes:
      - ./db:/var/lib/postgresql/data
      # This file can be used to bootstrap your schema if needed.
      # See "initialization scripts" on https://hub.docker.com/_/postgres/ for more info
      - ./config/init.sql:/docker-entrypoint-initdb.d/init.sql
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U toolbox_user -d toolbox_db"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - tool-network
networks:
  tool-network:

```

    {{< notice tip >}}  
To prevent DNS rebinding attack, use the `--allowed-origins` flag to specify a
list of origins permitted to access the server. E.g. `command: [ "toolbox",
"--tools-file", "/config/tools.yaml", "--address", "0.0.0.0",
"--allowed-origins", "https://foo.bar"]`
{{< /notice >}}

1. Run the following command to bring up the Toolbox and Postgres instance

    ```bash
    docker-compose up -d
    ```

{{< notice tip >}}

You can use this setup to quickly set up Toolbox + Postgres to follow along in our
[Quickstart](../getting-started/local_quickstart.md)

{{< /notice >}}

## Connecting with Toolbox Client SDK

Next, we will use Toolbox with the Client SDKs:

1. The url for the Toolbox server running using docker-compose will be:

    ```
    http://localhost:5000
    ```

1. Import and initialize the client with the URL:

   {{< tabpane persist=header >}}
{{< tab header="LangChain" lang="Python" >}}
from toolbox_langchain import ToolboxClient

# Replace with the cloud run service URL generated above

async with ToolboxClient("http://$YOUR_URL") as toolbox:
{{< /tab >}}
{{< tab header="Llamaindex" lang="Python" >}}
from toolbox_llamaindex import ToolboxClient

# Replace with the cloud run service URL generated above

async with ToolboxClient("http://$YOUR_URL") as toolbox:
{{< /tab >}}
{{< /tabpane >}}
