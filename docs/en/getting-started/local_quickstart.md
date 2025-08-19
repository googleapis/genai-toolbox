---
title: "Python Quickstart (Local)"
type: docs
weight: 2
description: >
  How to get started running Toolbox locally with [Python](https://github.com/googleapis/mcp-toolbox-sdk-python), PostgreSQL, and  [Agent Development Kit](https://google.github.io/adk-docs/),
  [LangGraph](https://www.langchain.com/langgraph), [LlamaIndex](https://www.llamaindex.ai/) or [GoogleGenAI](https://pypi.org/project/google-genai/).
---

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/googleapis/genai-toolbox/blob/main/docs/en/getting-started/colab_quickstart.ipynb)

## Before you begin

This guide assumes you have already done the following:

1. Installed [Python 3.9+][install-python] (including [pip][install-pip] and
   your preferred virtual environment tool for managing dependencies e.g. [venv][install-venv]).
1. Installed [PostgreSQL 16+ and the `psql` client][install-postgres].

### Cloud Setup (Optional)
<!-- [START cloud_setup] -->

If you plan to use **Google Cloud’s Vertex AI** with your agent (e.g., using
`vertexai=True` or a Google GenAI model), follow these one-time setup steps for
local development:

1. [Install the Google Cloud CLI](https://cloud.google.com/sdk/docs/install)
1. [Set up Application Default Credentials (ADC)](https://cloud.google.com/docs/authentication/set-up-adc-local-dev-environment)
1. Set your project and enable Vertex AI

    ```bash
    gcloud config set project YOUR_PROJECT_ID
    gcloud services enable aiplatform.googleapis.com
    ```

[install-python]: https://wiki.python.org/moin/BeginnersGuide/Download
[install-pip]: https://pip.pypa.io/en/stable/installation/
[install-venv]: https://packaging.python.org/en/latest/tutorials/installing-packages/#creating-virtual-environments
[install-postgres]: https://www.postgresql.org/download/
<!-- [END cloud_setup] -->

## Step 1: Set up your database
<!-- [START database_setup] -->

In this section, we will create a database, insert some data that needs to be
accessed by our agent, and create a database user for Toolbox to connect with.

1. Connect to postgres using the `psql` command:

    ```bash
    psql -h 127.0.0.1 -U postgres
    ```

    Here, `postgres` denotes the default postgres superuser.

    {{< notice info >}}

#### **Having trouble connecting?**

* **Password Prompt:** If you are prompted for a password for the `postgres`
  user and do not know it (or a blank password doesn't work), your PostgreSQL
  installation might require a password or a different authentication method.
* **`FATAL: role "postgres" does not exist`:** This error means the default
  `postgres` superuser role isn't available under that name on your system.
* **`Connection refused`:** Ensure your PostgreSQL server is actually running.
  You can typically check with `sudo systemctl status postgresql` and start it
  with `sudo systemctl start postgresql` on Linux systems.

<br/>

#### **Common Solution**

For password issues or if the `postgres` role seems inaccessible directly, try
switching to the `postgres` operating system user first. This user often has
permission to connect without a password for local connections (this is called
peer authentication).

```bash
sudo -i -u postgres
psql -h 127.0.0.1
```

Once you are in the `psql` shell using this method, you can proceed with the
database creation steps below. Afterwards, type `\q` to exit `psql`, and then
`exit` to return to your normal user shell.

If desired, once connected to `psql` as the `postgres` OS user, you can set a
password for the `postgres` *database* user using: `ALTER USER postgres WITH
PASSWORD 'your_chosen_password';`. This would allow direct connection with `-U
postgres` and a password next time.
    {{< /notice >}}

1. Create a new database and a new user:

    {{< notice tip >}}
  For a real application, it's best to follow the principle of least permission
  and only grant the privileges your application needs.
    {{< /notice >}}

    ```sql
      CREATE USER toolbox_user WITH PASSWORD 'my-password';

      CREATE DATABASE toolbox_db;
      GRANT ALL PRIVILEGES ON DATABASE toolbox_db TO toolbox_user;

      ALTER DATABASE toolbox_db OWNER TO toolbox_user;
    ```

1. End the database session:

    ```bash
    \q
    ```

    (If you used `sudo -i -u postgres` and then `psql`, remember you might also
    need to type `exit` after `\q` to leave the `postgres` user's shell
    session.)

1. Connect to your database with your new user:

    ```bash
    psql -h 127.0.0.1 -U toolbox_user -d toolbox_db
    ```

1. Create a table using the following command:

    ```sql
    CREATE TABLE hotels(
      id            INTEGER NOT NULL PRIMARY KEY,
      name          VARCHAR NOT NULL,
      location      VARCHAR NOT NULL,
      price_tier    VARCHAR NOT NULL,
      checkin_date  DATE    NOT NULL,
      checkout_date DATE    NOT NULL,
      booked        BIT     NOT NULL
    );
    ```

1. Insert data into the table.

    ```sql
    INSERT INTO hotels(id, name, location, price_tier, checkin_date, checkout_date, booked)
    VALUES
      (1, 'Hilton Basel', 'Basel', 'Luxury', '2024-04-22', '2024-04-20', B'0'),
      (2, 'Marriott Zurich', 'Zurich', 'Upscale', '2024-04-14', '2024-04-21', B'0'),
      (3, 'Hyatt Regency Basel', 'Basel', 'Upper Upscale', '2024-04-02', '2024-04-20', B'0'),
      (4, 'Radisson Blu Lucerne', 'Lucerne', 'Midscale', '2024-04-24', '2024-04-05', B'0'),
      (5, 'Best Western Bern', 'Bern', 'Upper Midscale', '2024-04-23', '2024-04-01', B'0'),
      (6, 'InterContinental Geneva', 'Geneva', 'Luxury', '2024-04-23', '2024-04-28', B'0'),
      (7, 'Sheraton Zurich', 'Zurich', 'Upper Upscale', '2024-04-27', '2024-04-02', B'0'),
      (8, 'Holiday Inn Basel', 'Basel', 'Upper Midscale', '2024-04-24', '2024-04-09', B'0'),
      (9, 'Courtyard Zurich', 'Zurich', 'Upscale', '2024-04-03', '2024-04-13', B'0'),
      (10, 'Comfort Inn Bern', 'Bern', 'Midscale', '2024-04-04', '2024-04-16', B'0');
    ```

1. End the database session:

    ```bash
    \q
    ```
<!-- [END database_setup] -->


## Step 2: Install and configure Toolbox
<!-- [START configure_toolbox] -->

In this section, we will download Toolbox, configure our tools in a
`tools.yaml`, and then run the Toolbox server.

1. Download the latest version of Toolbox as a binary:

    {{< notice tip >}}
  Select the
  [correct binary](https://github.com/googleapis/genai-toolbox/releases)
  corresponding to your OS and CPU architecture.
    {{< /notice >}}
    <!-- {x-release-please-start-version} -->
    ```bash
    export OS="linux/amd64" # one of linux/amd64, darwin/arm64, darwin/amd64, or windows/amd64
    curl -O https://storage.googleapis.com/genai-toolbox/v0.12.0/$OS/toolbox
    ```
    <!-- {x-release-please-end} -->

1. Make the binary executable:

    ```bash
    chmod +x toolbox
    ```

1. Write the following into a `tools.yaml` file. Be sure to update any fields
   such as `user`, `password`, or `database` that you may have customized in the
   previous step.

    {{< notice tip >}}
  In practice, use environment variable replacement with the format ${ENV_NAME}
  instead of hardcoding your secrets into the configuration file.
    {{< /notice >}}

    ```yaml
    sources:
      my-pg-source:
        kind: postgres
        host: 127.0.0.1
        port: 5432
        database: toolbox_db
        user: ${USER_NAME}
        password: ${PASSWORD}
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
      search-hotels-by-location:
        kind: postgres-sql
        source: my-pg-source
        description: Search for hotels based on location.
        parameters:
          - name: location
            type: string
            description: The location of the hotel.
        statement: SELECT * FROM hotels WHERE location ILIKE '%' || $1 || '%';
      book-hotel:
        kind: postgres-sql
        source: my-pg-source
        description: >-
           Book a hotel by its ID. If the hotel is successfully booked, returns a NULL, raises an error if not.
        parameters:
          - name: hotel_id
            type: string
            description: The ID of the hotel to book.
        statement: UPDATE hotels SET booked = B'1' WHERE id = $1;
      update-hotel:
        kind: postgres-sql
        source: my-pg-source
        description: >-
          Update a hotel's check-in and check-out dates by its ID. Returns a message
          indicating  whether the hotel was successfully updated or not.
        parameters:
          - name: hotel_id
            type: string
            description: The ID of the hotel to update.
          - name: checkin_date
            type: string
            description: The new check-in date of the hotel.
          - name: checkout_date
            type: string
            description: The new check-out date of the hotel.
        statement: >-
          UPDATE hotels SET checkin_date = CAST($2 as date), checkout_date = CAST($3
          as date) WHERE id = $1;
      cancel-hotel:
        kind: postgres-sql
        source: my-pg-source
        description: Cancel a hotel by its ID.
        parameters:
          - name: hotel_id
            type: string
            description: The ID of the hotel to cancel.
        statement: UPDATE hotels SET booked = B'0' WHERE id = $1;
   toolsets:
      my-toolset:
        - search-hotels-by-name
        - search-hotels-by-location
        - book-hotel
        - update-hotel
        - cancel-hotel
    ```

    For more info on tools, check out the `Resources` section of the docs.

1. Run the Toolbox server, pointing to the `tools.yaml` file created earlier:

    ```bash
    ./toolbox --tools-file "tools.yaml"
    ```

    {{< notice note >}}
    Toolbox enables dynamic reloading by default. To disable, use the
    `--disable-reload` flag.
    {{< /notice >}}

<!-- [END configure_toolbox] -->

## Step 3: Connect your agent to Toolbox

In this section, we will write and run an agent that will load the Tools
from Toolbox.

{{< notice tip>}} If you prefer to experiment within a Google Colab environment,
you can connect to a
[local runtime](https://research.google.com/colaboratory/local-runtimes.html).
{{< /notice >}}

1. In a new terminal, install the SDK package.

    {{< tabpane persist=header >}}
{{< tab header="ADK" lang="bash" >}}

pip install toolbox-core
{{< /tab >}}
{{< tab header="Langchain" lang="bash" >}}

pip install toolbox-langchain
{{< /tab >}}
{{< tab header="LlamaIndex" lang="bash" >}}

pip install toolbox-llamaindex
{{< /tab >}}
{{< tab header="Core" lang="bash" >}}

pip install toolbox-core
{{< /tab >}}
{{< /tabpane >}}

1. Install other required dependencies:

    {{< tabpane persist=header >}}
{{< tab header="ADK" lang="bash" >}}

pip install google-adk
{{< /tab >}}
{{< tab header="Langchain" lang="bash" >}}

# TODO(developer): replace with correct package if needed

pip install langgraph langchain-google-vertexai

# pip install langchain-google-genai

# pip install langchain-anthropic

{{< /tab >}}
{{< tab header="LlamaIndex" lang="bash" >}}

# TODO(developer): replace with correct package if needed

pip install llama-index-llms-google-genai

# pip install llama-index-llms-anthropic

{{< /tab >}}
{{< tab header="Core" lang="bash" >}}

pip install google-genai
{{< /tab >}}
{{< /tabpane >}}

1. Create a new file named `hotel_agent.py` and copy the following
   code to create an agent:
    {{< tabpane persist=header >}}
{{< tab header="ADK" lang="python" >}}
{{< include "quickstart/python/adk/quickstart.py" >}}
{{< /tab >}}
{{< tab header="LangChain" lang="python" >}}
{{< include "quickstart/python/langchain/quickstart.py" >}}
{{< /tab >}}
{{< tab header="LlamaIndex" lang="python" >}}
{{< include "quickstart/python/llamaindex/quickstart.py" >}}
{{< /tab >}}
{{< tab header="Core" lang="python" >}}
{{< include "quickstart/python/core/quickstart.py" >}}
{{< /tab >}}
{{< /tabpane >}}

    {{< tabpane text=true persist=header >}}
{{% tab header="ADK" lang="en" %}}
To learn more about Agent Development Kit, check out the [ADK
documentation.](https://google.github.io/adk-docs/)
{{% /tab %}}
{{% tab header="Langchain" lang="en" %}}
To learn more about Agents in LangChain, check out the [LangGraph Agent
documentation.](https://langchain-ai.github.io/langgraph/reference/prebuilt/#langgraph.prebuilt.chat_agent_executor.create_react_agent)
{{% /tab %}}
{{% tab header="LlamaIndex" lang="en" %}}
To learn more about Agents in LlamaIndex, check out the [LlamaIndex
AgentWorkflow
documentation.](https://docs.llamaindex.ai/en/stable/examples/agent/agent_workflow_basic/)
{{% /tab %}}
{{% tab header="Core" lang="en" %}}
To learn more about tool calling with Google GenAI, check out the
[Google GenAI
Documentation](https://github.com/googleapis/python-genai?tab=readme-ov-file#manually-declare-and-invoke-a-function-for-function-calling).
{{% /tab %}}
{{< /tabpane >}}

1. Run your agent, and observe the results:

    ```sh
    python hotel_agent.py
    ```

{{< notice info >}}
For more information, visit the [Python SDK repo](https://github.com/googleapis/mcp-toolbox-sdk-python).
{{</ notice >}}
