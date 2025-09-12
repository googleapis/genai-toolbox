---
title: "AlloyDB"
type: docs
weight: 2
description: >
  How to get started running Toolbox with MCP Inspector and AlloyDB as the source.
---

## Overview

[Model Context Protocol](https://modelcontextprotocol.io) is an open protocol
that standardizes how applications provide context to LLMs. Check out this page
on how to [connect to Toolbox via MCP](../../how-to/connect_via_mcp.md).

## Before you begin

This guide assumes you have already done the following:

1.  [Create a AlloyDB cluster and instance](https://cloud.google.com/alloydb/docs/cluster-create) with a database and user.
1. Connect to the instance using [AlloyDB Studio](https://cloud.google.com/alloydb/docs/manage-data-using-studio), [`psql` command-line tool](https://www.postgresql.org/download/), or any other PostgreSQL client.

2.  Enable the `pgvector` and `google_ml_integration` [extensions](https://cloud.google.com/alloydb/docs/ai). These are required for Semantic Search and Natural Language to SQL tools. Run the following SQL commands:

    ```sql
    CREATE EXTENSION IF NOT EXISTS "vector";
    CREATE EXTENSION IF NOT EXISTS "google_ml_integration";
    CREATE EXTENSION IF NOT EXISTS alloydb_ai_nl cascade;
    CREATE EXTENSION IF NOT EXISTS parameterized_views;
    ```

## Step 1: Set up your AlloyDB database

In this section, we will create the necessary tables and functions in your AlloyDB instance.

1.  Create tables using the following commands:

    ```sql
    CREATE TABLE products (
      product_id SERIAL PRIMARY KEY,
      name VARCHAR(255) NOT NULL,
      description TEXT,
      price DECIMAL(10, 2) NOT NULL,
      category_id INT,
      embedding vector(3072) -- Vector size for model(gemini-embedding-001)
    );

    CREATE TABLE customers (
      customer_id SERIAL PRIMARY KEY,
      name VARCHAR(255) NOT NULL,
      email VARCHAR(255) UNIQUE NOT NULL
    );

    CREATE TABLE cart (
      cart_id SERIAL PRIMARY KEY,
      customer_id INT UNIQUE NOT NULL,
      created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
      FOREIGN KEY (customer_id) REFERENCES customers(customer_id)
    );

    CREATE TABLE cart_items (
      cart_item_id SERIAL PRIMARY KEY,
      cart_id INT NOT NULL,
      product_id INT NOT NULL,
      quantity INT NOT NULL,
      price DECIMAL(10, 2) NOT NULL,
      FOREIGN KEY (cart_id) REFERENCES cart(cart_id),
      FOREIGN KEY (product_id) REFERENCES products(product_id)
    );

    CREATE TABLE categories (
      category_id SERIAL PRIMARY KEY,
      name VARCHAR(255) NOT NULL
    );
    ```

2.  Insert sample data into the tables:

    ```sql
    INSERT INTO categories (category_id, name) VALUES
    (1, 'Flowers'),
    (2, 'Vases');

    INSERT INTO products (product_id, name, description, price, category_id, embedding) VALUES
    (1, 'Rose', 'A beautiful red rose', 2.50, 1, embedding('gemini-embedding-001', 'A beautiful red rose')),
    (2, 'Tulip', 'A colorful tulip', 1.50, 1, embedding('gemini-embedding-001', 'A colorful tulip')),
    (3, 'Glass Vase', 'A transparent glass vase', 10.00, 2, embedding('gemini-embedding-001', 'A transparent glass vase')),
    (4, 'Ceramic Vase', 'A handmade ceramic vase', 15.00, 2, embedding('gemini-embedding-001', 'A handmade ceramic vase'));

    INSERT INTO customers (customer_id, name, email) VALUES
    (1, 'John Doe', 'john.doe@example.com'),
    (2, 'Jane Smith', 'jane.smith@example.com');

    INSERT INTO cart (cart_id, customer_id) VALUES
    (1, 1),
    (2, 2);

    INSERT INTO cart_items (cart_id, product_id, quantity, price) VALUES
    (1, 1, 2, 2.50),
    (1, 3, 1, 10.00),
    (2, 2, 5, 1.50);
    ```

## Step 2: Install Toolbox

In this section, we will download and install the Toolbox binary.

1. Download the latest version of Toolbox as a binary:

    {{< notice tip >}}
   Select the
   [correct binary](https://github.com/googleapis/genai-toolbox/releases)
   corresponding to your OS and CPU architecture.
    {{< /notice >}}
    <!-- {x-release-please-start-version} -->
    ```bash
    export OS="linux/amd64" # one of linux/amd64, darwin/arm64, darwin/amd64, or windows/amd64
    export VERSION="0.11.0"
    curl -O https://storage.googleapis.com/genai-toolbox/v$VERSION/$OS/toolbox
    ```
    <!-- {x-release-please-end} -->

1. Make the binary executable:

    ```bash
    chmod +x toolbox
    ```

## Step 3: Configure the tools

Create a `tools.yaml` file and add the following content. You must replace the placeholders with your actual AlloyDB configuration.

First, define the data source for your tools. This tells Toolbox how to connect to your AlloyDB instance.

```yaml
sources:
  alloydb-pg-source:
    kind: alloydb-postgres
    project: YOUR_PROJECT_ID
    region: YOUR_REGION
    cluster: YOUR_CLUSTER
    instance: YOUR_INSTANCE
    database: YOUR_DATABASE
    user: YOUR_USER
    password: YOUR_PASSWORD
```

Next, define the tools the agent can use. We will categorize them into three types:

### 1. Structured Queries Tools

These tools execute predefined SQL statements. They are ideal for common, structured queries like managing a shopping cart. Add the following to your `tools.yaml` file:

```yaml
tools:

  access-cart-information:
    kind: postgres-sql
    source: alloydb-pg-source
    description: >-
      List items in customer cart.
      Use this tool to list items in a customer cart. This tool requires the cart ID.
    parameters:
      - name: cart_id
        type: integer
        description: The id of the cart.
    statement: |
      SELECT
        p.name AS product_name,
        ci.quantity,
        ci.price AS item_price,
        (ci.quantity * ci.price) AS total_item_price,
        c.created_at AS cart_created_at,
        ci.product_id AS product_id
      FROM
        cart_items ci JOIN cart c ON ci.cart_id = c.cart_id
        JOIN products p ON ci.product_id = p.product_id
      WHERE
        c.cart_id = $1;

  add-to-cart:
    kind: postgres-sql
    source: alloydb-pg-source
    description: >-
      Add items to customer cart using the product ID and product prices from the product list.
      Use this tool to add items to a customer cart.
      This tool requires the cart ID, product ID, quantity, and price.
    parameters:
      - name: cart_id
        type: integer
        description: The id of the cart.
      - name: product_id
        type: integer
        description: The id of the product.
      - name: quantity
        type: integer
        description: The quantity of items to add.
      - name: price
        type: float
        description: The price of items to add.
    statement: |
      INSERT INTO
        cart_items (cart_id, product_id, quantity, price)
      VALUES($1,$2,$3,$4);

  delete-from-cart:
    kind: postgres-sql
    source: alloydb-pg-source
    description: >-
      Remove products from customer cart.
      Use this tool to remove products from a customer cart.
      This tool requires the cart ID and product ID.
    parameters:
      - name: cart_id
        type: integer
        description: The id of the cart.
      - name: product_id
        type: integer
        description: The id of the product.
    statement: |
      DELETE FROM
        cart_items
      WHERE
        cart_id = $1 AND product_id = $2;
```

### 2. Semantic Search Tools

These tools use vector embeddings to find the most relevant results based on the meaning of a user's query, rather than just keywords. Append the following tools to the `tools` section in your `tools.yaml`:

```yaml
  search-product-recommendations:
    kind: postgres-sql
    source: alloydb-pg-source
    description: >-
      Search for products based on user needs.
      Use this tool to search for products. This tool requires the user's needs.
    parameters:
      - name: query
        type: string
        description: The product characteristics
    statement: |
      SELECT
        product_id,
        name,
        description,
        ROUND(CAST(price AS numeric), 2) as price
      FROM
        products
      ORDER BY
        embedding('gemini-embedding-001', $1)::vector <=> embedding
      LIMIT 5;
```

### 3. Natural Language to SQL (NL2SQL) Tools

1. Create a [natural language configuration](https://cloud.google.com/alloydb/docs/ai/use-natural-language-generate-sql-queries#create-config) for your AlloyDB cluster.

    {{< notice tip >}}Before using NL2SQL tools,
    you must first install the `alloydb_ai_nl` extension and
    create the [semantic layer](https://cloud.google.com/alloydb/docs/ai/natural-language-overview) under a configuration named `flower_shop`.
    {{< /notice >}}

2. Configure your NL2SQL tool to use your configuration. These tools translate natural language questions into SQL queries, allowing users to interact with the database conversationally. Append the following tool to the `tools` section:

```yaml
  ask-questions-about-products:
    kind: alloydb-ai-nl
    source: alloydb-pg-source
    nlConfig: flower_shop
    description: >-
      Ask questions related to products or brands.
      Use this tool to ask questions about products or brands.
      Always SELECT the IDs of objects when generating queries.
```

Finally, group the tools into a `toolset` to make them available to the model. Add the following to the end of your `tools.yaml` file:

```yaml
toolsets:
  flower_shop:
    - access-cart-information
    - search-product-recommendations
    - ask-questions-about-products
    - add-to-cart
    - delete-from-cart
```

For more info on tools, check out the
[Tools](../../resources/tools/) section.

## Step 4: Run the Toolbox server

Run the Toolbox server, pointing to the `tools.yaml` file created earlier:

```bash
./toolbox --tools-file "tools.yaml"
```

## Step 5: Connect to MCP Inspector

1. Run the MCP Inspector:

    ```bash
    npx @modelcontextprotocol/inspector
    ```

1. Type `y` when it asks to install the inspector package.

1. It should show the following when the MCP Inspector is up and running (please take note of `<YOUR_SESSION_TOKEN>`):

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

1. For `Configuration` -> `Proxy Session Token`, make sure `<YOUR_SESSION_TOKEN>` is present.

1. Click Connect.

1. Select `List Tools`, you will see a list of tools configured in `tools.yaml`.

1. Test out your tools here!

## What's next

- Learn more about [MCP Inspector](../../how-to/connect_via_mcp.md).
- Learn more about [Toolbox Resources](../../resources/).
- Learn more about [Toolbox How-to guides](../../how-to/).

