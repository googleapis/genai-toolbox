---
title: "Deploy ADK Agent and MCP Toolbox"
type: docs
weight: 4
description: >
  How to deploy your ADK Agent to Vertex AI Agent Engine and connect it to an MCP Toolbox deployed on Cloud Run.
---

## Before you begin

This guide assumes you have already done the following:

1.  Completed the [Python Quickstart
    (Local)](../getting-started/local_quickstart.md) and have a working ADK
    agent running locally.
2.  Installed the [Google Cloud CLI](https://cloud.google.com/sdk/docs/install).
3.  A Google Cloud project with billing enabled.

## Step 1: Deploy MCP Toolbox to Cloud Run

Before deploying your agent, your MCP Toolbox server needs to be accessible from
the cloud. We will deploy MCP Toolbox to Cloud Run.

Follow the [Deploy to Cloud Run](deploy_toolbox.md) guide to deploy your MCP
Toolbox instance.

{{% alert title="Important" %}}
After deployment, note down the Service URL of your MCP Toolbox Cloud Run
service. You will need this to configure your agent.
{{% /alert %}}
## Step 2: Prepare your Agent for Deployment

We will use the `agent-starter-pack` tool to enhance your local agent project
with the necessary configuration for deployment to Vertex AI Agent Engine.

1.  Open a terminal and navigate to the **parent directory** of your agent
    project (the directory containing the `my_agent` folder).

2.  Run the following command to enhance your project:

    ```bash
    uvx agent-starter-pack enhance --adk -d agent_engine
    ```

3.  Follow the interactive prompts to configure your deployment settings. This
    process will generate deployment configuration files (like a `Makefile` and
    `Dockerfile`) in your project directory.

4.  Add `google-adk[toolbox]` as a dependency to the new project:

    ```bash
    uv add google-adk[toolbox]
    ```

## Step 3: Configure Google Cloud Authentication

Ensure your local environment is authenticated with Google Cloud to perform the
deployment.

1.  Login with Application Default Credentials (ADC):

    ```bash
    gcloud auth application-default login
    ```

2.  Set your active project:

    ```bash
    gcloud config set project <YOUR_PROJECT_ID>
    ```

## Step 4: Connect Agent to Deployed MCP Toolbox

You need to update your agent's code to connect to the Cloud Run URL of your MCP
Toolbox instead of the local address.

1.  Recall that you can find the Cloud Run deployment URL of the MCP Toolbox
    server using the following command:

    ```bash
    gcloud run services describe toolbox --format 'value(status.url)'
    ```

2.  Open your agent file (`my_agent/agent.py`).

3.  Update the `ToolboxToolset` initialization to point to your Cloud Run service URL. Replace the existing initialization code with the following:

    {{% alert color="info" title="Note" %}}
Since Cloud Run services are secured by default, you also need to provide a workload identity.
    {{% /alert %}}

    ```python
    from google.adk import Agent
    from google.adk.apps import App
    from google.adk.tools.toolbox_toolset import ToolboxToolset
    from toolbox_adk import CredentialStrategy

    # TODO(developer): Replace with your Toolbox Cloud Run Service URL
    TOOLBOX_URL = "https://your-toolbox-service-xyz.a.run.app"

    # Initialize the toolset with Workload Identity (generates ID token for the URL)
    toolset = ToolboxToolset(
        server_url=TOOLBOX_URL,
        credentials=CredentialStrategy.workload_identity(target_audience=TOOLBOX_URL)
    )

    root_agent = Agent(
        name='root_agent',
        model='gemini-2.5-flash',
        instruction="You are a helpful AI assistant designed to provide accurate and useful information.",
        tools=[toolset],
    )

    app = App(root_agent=root_agent, name="my_agent")
    ```

    {{% alert title="Important" %}}
Ensure that the `name` parameter in the `App` initialization matches the name of
your agent's parent directory (e.g., `my_agent`).
```python
...

app = App(root_agent=root_agent, name="my_agent")
```
    {{% /alert %}}

## Step 5: Deploy to Agent Engine

Run the deployment command:

```bash
make deploy
```

This command will build your agent's container image and deploy it to Vertex AI.

## Step 6: Test your Deployment

Once the deployment command (`make deploy`) completes, it will output the URL
for the Agent Engine Playground. You can click on this URL to open the
Playground in your browser and start chatting with your agent to test the tools.

For additional test scenarios, refer to the [Test deployed
agent](https://google.github.io/adk-docs/deploy/agent-engine/#test-deployment)
section in the ADK documentation.