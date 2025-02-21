---
title: "Deploy using Docker Compose"
type: docs
weight: 2
description: >
  How to deploy Toolbox using Docker Compose. 
---


## Before you begin

1. [Install Docker Compose.](https://docs.docker.com/compose/install/)

## Modify the configurations

The configurations can be left as is for a default deployment. If you intend to customize the deployment,

1. Modify the `config/init.sql` and `config/tools.yaml` as per the customization need.
2. Modify the `docker-compose.yml` as per the customization need.

## Deploy to Docker

1. Run the following command to bring up the Toolbox and Preloaded Postgres instance

    ```bash
    docker-compose up -d
    ```

This will bring up a functional Gen AI Toolbox and a Postgres instance with a database and table as described in the [Getting started -> Quickstart -> Step 1: Set up your database](../getting-started/local_quickstart.md#step-1-set-up-your-database), so that you can proceed with [Step 2](../getting-started/local_quickstart.md#step-2-install-and-configure-toolbox).
