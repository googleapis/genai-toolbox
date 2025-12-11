---
title: "Prompts using Gemini CLI"
type: docs
weight: 5
description: >
  How to get started using Toolbox prompts locally with PostgreSQL and [Gemini CLI](https://pypi.org/project/gemini-cli/).
---

## Before you begin

This guide assumes you have already done the following:

1. Installed [PostgreSQL 16+ and the `psql` client][install-postgres].
 
[install-postgres]: https://www.postgresql.org/download/

## Step 1: Set up your database

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

1. Create the required tables using the following commands:

    ```sql
    CREATE TABLE users (
      id SERIAL PRIMARY KEY,
      username VARCHAR(50) NOT NULL,
      email VARCHAR(100) UNIQUE NOT NULL,
      created_at TIMESTAMPTZ DEFAULT NOW()
    );

    CREATE TABLE restaurants (
      id SERIAL PRIMARY KEY,
      name VARCHAR(100) NOT NULL,
      location VARCHAR(100)
    );

    CREATE TABLE reviews (
      id SERIAL PRIMARY KEY,
      user_id INT REFERENCES users(id),
      restaurant_id INT REFERENCES restaurants(id),
      rating INT CHECK (rating >= 1 AND rating <= 5),
      review_text TEXT,
      is_published BOOLEAN DEFAULT false,
      moderation_status VARCHAR(50) DEFAULT 'pending_manual_review',      
      created_at TIMESTAMPTZ DEFAULT NOW()
    );
    ```

1. Insert dummy data into the tables.

    ```sql
    INSERT INTO users (id, username, email) VALUES
    (123, 'jane_d', 'jane.d@example.com'),
    (124, 'john_s', 'john.s@example.com'),
    (125, 'sam_b', 'sam.b@example.com');

    INSERT INTO restaurants (id, name, location) VALUES
    (455, 'Pizza Palace', '123 Main St'),
    (456, 'The Corner Bistro', '456 Oak Ave'),
    (457, 'Sushi Spot', '789 Pine Ln');

    INSERT INTO reviews (user_id, restaurant_id, rating, review_text, is_published, moderation_status) VALUES
    (124, 455, 5, 'Best pizza in town! The crust was perfect.', true, 'approved'),
    (125, 457, 4, 'Great sushi, very fresh. A bit pricey but worth it.', true, 'approved'),
    (123, 457, 5, 'Absolutely loved the dragon roll. Will be back!', true, 'approved'),
    (123, 456, 4, 'The atmosphere was lovely and the food was great. My photo upload might have been weird though.', false, 'pending_manual_review'),
    (125, 456, 1, 'This review contains inappropriate language.', false, 'rejected');
    ```

1. End the database session:

    ```bash
    \q
    ```

## Step 2: Configure Toolbox

Create a file named `tools.yaml`. This file defines the database connection, the SQL tools available, and the prompts the agents will use.

```yaml
sources:
  my-foodiefind-db:
    kind: postgres
    host: 127.0.0.1
    port: 5432
    database: toolbox_db
    user: toolbox_user
    password: my-password
tools:
  find_user_by_email:
    kind: postgres-sql
    source: my-foodiefind-db
    description: Find a user's ID by their email address.
    parameters:
      - name: email
        type: string
        description: The email address of the user to find.
    statement: SELECT id FROM users WHERE email = $1;
  find_restaurant_by_name:
    kind: postgres-sql
    source: my-foodiefind-db
    description: Find a restaurant's ID by its exact name.
    parameters:
      - name: name
        type: string
        description: The name of the restaurant to find.
    statement: SELECT id FROM restaurants WHERE name = $1;
  find_review_by_user_and_restaurant:
    kind: postgres-sql
    source: my-foodiefind-db
    description: Find the full record for a specific review using the user's ID and the restaurant's ID.
    parameters:
      - name: user_id
        type: integer
        description: The numerical ID of the user.
      - name: restaurant_id
        type: integer
        description: The numerical ID of the restaurant.
    statement: SELECT * FROM reviews WHERE user_id = $1 AND restaurant_id = $2;
prompts:
  investigate_missing_review:
    description: "Investigates a user's missing review by finding the user, restaurant, and the review itself, then analyzing its status."
    arguments:
      - name: "user_email"
        description: "The email of the user who wrote the review."
      - name: "restaurant_name"
        description: "The name of the restaurant being reviewed."
    messages:
      - content: >-
          **Goal:** Find the review written by the user with email '{{.user_email}}' for the restaurant named '{{.restaurant_name}}' and understand its status.
          **Workflow:**
          1. Use the `find_user_by_email` tool with the email '{{.user_email}}' to get the `user_id`.
          2. Use the `find_restaurant_by_name` tool with the name '{{.restaurant_name}}' to get the `restaurant_id`.
          3. Use the `find_review_by_user_and_restaurant` tool with the `user_id` and `restaurant_id` you just found.
          4. Analyze the results from the final tool call. Examine the `is_published` and `moderation_status` fields and explain the review's status to the user in a clear, human-readable sentence.
```

## Step 3: Connect to Gemini CLI

Configure the Gemini CLI to talk to your local Toolbox MCP server.

1. Open or create your Gemini settings file: `~/.gemini/settings.json`.
2. Add the following configuration to the file:

    ```json
    {
      "mcpServers": {
        "MCPToolbox": {
          "httpUrl": "http://localhost:5000/mcp"
        }
      },
      "mcp": {
        "allowed": ["MCPToolbox"]
      }
    }
    ```
3. Start Gemini CLI using 
    ```sh
    gemini
    ```
    In case Gemini CLI is already running, use `/mcp refresh` to refresh the MCP server.

4. Use gemini slash commands to run your prompt: 
    ```sh
    /investigate_missing_review --user_email="jane.d@example.com" --restaurant_name="The Corner Bistro"
    ```
