---
title: "Quickstart (Local) using Js SDK"
type: docs
weight: 2
description: >
  How to get started running Toolbox locally with Javascript, PostgreSQL, and  [Agent Development Kit](https://google.github.io/adk-docs/),
  [LangGraph](https://www.langchain.com/langgraph), [LlamaIndex](https://www.llamaindex.ai/) or [GoogleGenAI](https://pypi.org/project/google-genai/).
---

## Before you begin

This guide assumes you have already done the following:

1. Installed [Node.js (v18 or higher)]
2. Installed [PostgreSQL 16+ and the `psql` client][install-postgres]

### Cloud Setup (Optional)

If you plan to use **Google Cloud’s Vertex AI** with your agent (e.g., using Gemini or PaLM models), follow these one-time setup steps:

> 📚 Before you begin:
> - [Install the Google Cloud CLI]
> - [Set up Application Default Credentials (ADC)]

#### Set your project and enable Vertex AI

```bash
gcloud config set project YOUR_PROJECT_ID
gcloud services enable aiplatform.googleapis.com
```

[Node.js (v18 or higher)]: https://nodejs.org/
[install-postgres]: https://www.postgresql.org/download/
[Install the Google Cloud CLI]: https://cloud.google.com/sdk/docs/install
[Set up Application Default Credentials (ADC)]: https://cloud.google.com/docs/authentication/set-up-adc-local-dev-environment

---

## Step 1: Set up your database

Follow the same steps as in the [PostgreSQL setup](../local_quickstart#step-1-set-up-your-database) of the Python quickstart. Create:

- A database (e.g., `toolbox_db`)
- A user (e.g., `toolbox_user`)
- A `hotels` table and insert sample data.

Once your database is set up and populated, continue to the next step.

---

## Step 2: Install and run the Toolbox server

Download the Toolbox binary and create your `tools.yaml` file as shown in the [Python quickstart](../local_quickstart#step-2-install-and-configure-toolbox).

Then run:

```bash
./toolbox --tools-file tools.yaml
```

Your tools will now be accessible at `http://localhost:5000`.

---