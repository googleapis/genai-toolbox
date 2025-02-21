---
title: "Deploy using Docker Compose"
type: docs
weight: 2
description: >
  How to deploy Toolbox using Docker Compose. 
---


## Before you begin

1. [Install Docker Compose.](https://docs.docker.com/compose/install/)

## Configure `tools.yaml` file

Create a `tools.yaml` file that contains your configuration for Toolbox. For
details, see the
[configuration](https://github.com/googleapis/genai-toolbox/blob/main/README.md#configuration)
section.

For reference, a sample for `tools.yaml` is provided in the `config/tools.yaml` path of this repository.
Similarly, a sample SQL for initializing the database is provided in the `config/init.sql` path of this repository.

## Modify the configurations

The configurations can be left as is for a default deployment. If you intend to customize the deployment,

1. Modify the `docker-compose.yml` as per the customization need.

## Deploy to Docker

1. Run the following command to bring up the Toolbox and Preloaded Postgres instance

    ```bash
    docker-compose up -d
    ```

This will bring up a functional Gen AI Toolbox and a Postgres instance with a database and table as described in the [Getting started -> Quickstart -> Step 1: Set up your database](../getting-started/local_quickstart.md#step-1-set-up-your-database), so that you can proceed with [Step 2](../getting-started/local_quickstart.md#step-2-install-and-configure-toolbox).

## Connecting with Toolbox Client Python modules

Next, we will use Toolbox with Python modules.

1. Below is a list of Client SDKs that are supported:
    - LangChain / LangGraph
    - LlamaIndex
2. The Toolbox brought up using docker-compose will be serving in [http://localhost:5000](http://localhost:5000)
3. You can follow the steps described in [Quickstart guide](https://googleapis.github.io/genai-toolbox/getting-started/local_quickstart/#step-3-connect-your-agent-to-toolbox) to use the Toolbox.


