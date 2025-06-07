# Python Toolbox (py_toolbox)

## Overview

`py_toolbox` is a Python-based toolkit designed for flexible interaction with various data sources. It is inspired by an original Go-based project and aims to provide a general-purpose framework suitable for local development and potential enterprise use.

The core design emphasizes a pluggable architecture where different data sources (databases) and tools (operations on those sources) can be easily added and configured.

## Features

*   **Multiple Database Support:**
    *   PostgreSQL
    *   MySQL
    *   SQLite
    *   Neo4j (Graph Database)
*   **Configuration via YAML:** Define and configure data sources and tools using a simple `tools.yaml` file.
*   **Command-Line Interface (CLI):** Interact with the toolbox using a CLI built with `click`.
    *   List available tools.
    *   Invoke tools with parameters.
*   **Extensible Architecture:**
    *   **Source Registry:** Manages connections to different data sources.
    *   **Tool Registry:** Manages tools that operate on these sources.
    *   New sources and tools can be added by following the established patterns.
*   **Basic Logging:** Provides informative logs about operations.

## Project Structure

```
py_toolbox/
├── cmd/                  # CLI related logic (though main.py is currently at root of py_toolbox)
├── internal/             # Core internal logic
│   ├── core/             # Core components like registries, logging, config parsing
│   ├── sources/          # Data source implementations (postgres.py, mysql.py, etc.)
│   └── tools/            # Tool implementations (postgres_sql.py, etc.)
├── tests/                # Unit tests
│   ├── sources/
│   └── tools/
├── main.py               # Main CLI entry point
├── requirements.txt      # Python dependencies
└── README.md             # This file
tools.yaml                # Example configuration (should be at the root where app is run)
```

## Setup

### Prerequisites

*   Python 3.8 or higher is recommended.
*   Access to the databases you intend to connect to (e.g., running instances of PostgreSQL, MySQL, Neo4j).

### Installation

1.  **Clone the repository (if applicable).**
2.  **Install dependencies:**
    Open a terminal in the `py_toolbox` directory (or the directory containing `requirements.txt`) and run:
    ```bash
    pip install -r requirements.txt
    ```
    (It's recommended to do this within a Python virtual environment.)

## Configuration (`tools.yaml`)

The behavior of `py_toolbox` is controlled by a `tools.yaml` file. This file should typically reside in the directory from which you run the `main.py` script, or its path can be specified using the `--config` option.

The `tools.yaml` file has two main sections: `sources` and `tools`.

### `sources`

This section defines the data sources (databases) you want to connect to. Each source has a unique name and a `kind` that specifies its type.

**Example Structure:**

```yaml
sources:
  my_postgres_instance:
    kind: "postgres"  # Supported kinds: postgres, mysql, sqlite, neo4j
    host: "localhost"
    port: 5432
    user: "your_pg_user"
    password: "your_pg_password"
    database: "your_pg_db"
    # pool_min_conn: 1 # Optional
    # pool_max_conn: 5 # Optional

  my_neo4j_instance:
    kind: "neo4j"
    uri: "neo4j://localhost:7687"
    user: "neo4j"
    password: "your_neo4j_password"
    database: "neo4j" # Optional, defaults to system/neo4j based on version
    # max_connection_lifetime: 3600 # Optional driver config
```

**Important:** Replace placeholder values (like `your_pg_user`, `your_pg_password`, hostnames, etc.) with your actual database credentials and connection details.

### `tools`

This section defines the tools that operate on the configured sources. Each tool has a unique name, a `kind` (e.g., `postgres-sql`, `mysql-sql`, `sqlite-sql`, `neo4j-cypher`), a `description`, and specifies the `source` it uses (which must match a name defined in the `sources` section).

**Example Structure:**

```yaml
tools:
  fetch_pg_users:
    kind: "postgres-sql"
    description: "Fetches user data from PostgreSQL."
    source: "my_postgres_instance" # Links to a source defined above
    # Default statement (can be overridden at invocation)
    # statement: "SELECT id, name, email FROM users WHERE status = %s;"
    parameters:
      - name: "statement"
        type: "string"
        description: "The SQL query to execute."
        required: true
      - name: "args"
        type: "array"
        description: "List of arguments for the SQL query's placeholders."
        required: false

  create_neo4j_node:
    kind: "neo4j-cypher"
    description: "Creates a new node in Neo4j."
    source: "my_neo4j_instance"
    parameters:
      - name: "cypher"
        type: "string"
        description: "The Cypher query to execute (e.g., CREATE (n:Person {name: $name}))."
        required: true
      - name: "params"
        type: "object"
        description: "Dictionary of parameters for the Cypher query."
        required: false
      - name: "transaction_type"
        type: "string"
        description: "'read' or 'write'. Defaults to 'read'."
        required: false # Defaults to 'read' in the tool
```
*(Refer to the provided `tools.yaml` for a complete example covering all supported database types.)*

## Usage (CLI)

The primary way to interact with `py_toolbox` is through its command-line interface, accessed via `py_toolbox/main.py`.

```bash
python py_toolbox/main.py [OPTIONS] COMMAND [ARGS]...
```

**Global Options:**

*   `--config FILE_PATH`: Path to the `tools.yaml` configuration file (defaults to `tools.yaml` in the current directory).
*   `--log-level [DEBUG|INFO|WARNING|ERROR|CRITICAL]`: Set the logging level.

### Commands

#### `list-tools`

Lists all tools configured in `tools.yaml`.

```bash
python py_toolbox/main.py list-tools
```

**Example Output:**

```
Available tools:
- fetch_pg_users: Fetches user data from PostgreSQL.
  Parameters:
    - statement (string, required: true): The SQL query to execute.
    - args (array, required: false): List of arguments for the SQL query's placeholders.

- create_neo4j_node: Creates a new node in Neo4j.
  Parameters:
    - cypher (string, required: true): The Cypher query to execute (e.g., CREATE (n:Person {name: $name})).
    - params (object, required: false): Dictionary of parameters for the Cypher query.
    - transaction_type (string, required: false): 'read' or 'write'. Defaults to 'read'.
```

#### `invoke-tool <TOOL_NAME> [TOOL_PARAMS_JSON]`

Invokes a specified tool.

*   `<TOOL_NAME>`: The name of the tool as defined in `tools.yaml`.
*   `[TOOL_PARAMS_JSON]` (Optional): A JSON string representing the parameters for the tool. This should be a JSON object (dictionary).

**Example: Invoking a PostgreSQL tool**

Assuming `fetch_pg_users` is configured and expects a `statement` and `args`:

```bash
python py_toolbox/main.py invoke-tool fetch_pg_users '{"statement": "SELECT name FROM users WHERE id = %s;", "args": [123]}'
```

**Example: Invoking a Neo4j tool**

Assuming `create_neo4j_node` is configured:

```bash
python py_toolbox/main.py invoke-tool create_neo4j_node '{"cypher": "CREATE (p:Person {name: $personName, age: $personAge})", "params": {"personName": "Alice", "personAge": 30}, "transaction_type": "write"}'
```

The tool's output (e.g., query results or status messages) will be printed to the console as a JSON string.

## MCP Server Mode (STDIN/STDOUT)

 can run as a dedicated MCP (Management Control Plane) server, listening for requests on standard input (STDIN) and sending responses to standard output (STDOUT). This mode is designed for programmatic interaction, typically where  is managed as a subprocess by a parent application (e.g., a chatbot backend, an orchestration script).

### Running as an MCP Server

To start  in MCP server mode, use the  command:

For example:

Once started, the server will:
- Load tools based on the specified configuration file.
- Listen for incoming messages on STDIN.
- Process one message per line.
- Send one response per line to STDOUT for each valid request that isn't a notification.

### Communication Protocol

- **Framing:** Messages are line-delimited. Each line on STDIN is expected to be a complete JSON message. Each response on STDOUT will also be a single line containing a complete JSON message.
- **Message Format:** JSON-RPC 2.0 is used for the content of each JSON message.

#### JSON-RPC 2.0 Request Structure
A typical request from the client (parent process) to  will look like this:

- : The name of the operation to perform (see supported methods below).
- : An object/dictionary containing parameters for the method.
- uid=1001(jules) gid=0(root) groups=0(root),27(sudo): A unique identifier for the request, which will be echoed in the response. If uid=1001(jules) gid=0(root) groups=0(root),27(sudo) is  or omitted, it's treated as a notification.

#### JSON-RPC 2.0 Response Structure

**Success:**

**Error:**

### Supported MCP Methods

1.  ****
    -   Description: Retrieves a list of all available tools configured in .
    -   Request : (empty object or null)
    -   Response : An array of objects, where each object contains  (string) and  (string) of a tool.
    -   Example Request:


2.  ****
    -   Description: Retrieves the detailed manifest (including input schema) for a specific tool.
    -   Request :
    -   Response : The  object for the tool (see Pydantic models in  for structure, includes , , ).
    -   Example Request:


3.  ****
    -   Description: Executes a specified tool with the given parameters.
    -   Request :

        (Note:  holds the actual arguments for the tool.  is also accepted as an alias.)
    -   Response : The output from the tool's execution. The structure depends on the tool.
    -   Example Request (for a SQL tool):


### Interacting as a Subprocess
A parent application would typically:
1.  Spawn  as a subprocess.
2.  Capture its STDIN and STDOUT pipes.
3.  Send JSON-RPC request strings (each terminated by a newline) to the subprocess's STDIN.
4.  Read response lines from the subprocess's STDOUT and parse them as JSON.
5.  Handle logs or other output from STDERR separately if needed.

## Integration with MCP Hub (as a Microservice)

A  instance running in  mode can register the tools it offers with a central MCP Hub server. This allows consumers to discover tools from various  microservices through the Hub.

### Configuration for Hub Registration

When starting  in  mode, the following environment variables control its interaction with the MCP Hub:

-   ****: The base URL of the MCP Hub's REST API (e.g., ). If this variable is not set,  will skip the registration process.
-   **** (Optional): A unique identifier for this  instance. If not set, a default ID will be generated based on the configuration file path and hostname (e.g., ). It's recommended to set a stable, unique ID for production deployments.

### Registration Process

On startup, if  is configured,  will:
1.  Iterate through all tools loaded from its local .
2.  For each tool, it sends a registration request (HTTP POST) to the Hub's  endpoint.
3.  The payload includes:
    -   : The name of the tool within this  instance.
    -   : The ID of this  instance.
    -   : The tool's description.
    -   : The tool's detailed manifest (including input schema).
    -   : A JSON object explaining how a consumer can invoke this tool via this  instance's MCP (STDIN/STDOUT JSON-RPC) interface. This includes a command template and a JSON-RPC request template.

If a tool is already registered with the Hub by the same microservice and tool name, its registration details (including ) are updated.

### Example  sent to Hub

This information helps consumers understand how to use the tool after discovering it via the Hub:

## Extending the Toolbox

To add support for a new database or a new type of tool:

1.  **Create a new Source:**
    *   Implement a `SourceConfig` and `Source` class in `py_toolbox/internal/sources/your_new_source.py`.
    *   Include a `register_source(registry)` function.
2.  **Create a new Tool:**
    *   Implement a `ToolConfig` and `Tool` class in `py_toolbox/internal/tools/your_new_tool.py`.
    *   Include a `register_tool(registry)` function.
3.  **Register:**
    *   In `py_toolbox/main.py`, import and call your new registration functions within `setup_registries()`.
4.  **Configure:**
    *   Add configurations for your new source and tool in `tools.yaml`.

## Running Tests

Unit tests are located in the `py_toolbox/tests/` directory. To run them:

1.  Ensure you have installed any testing-related dependencies if specified (currently, only `unittest` from the standard library is used directly by the tests, but project dependencies in `requirements.txt` are needed for the tested code).
2.  Navigate to the directory containing the `py_toolbox` folder (i.e., the parent of `py_toolbox`).
3.  Run the `unittest` discovery:

    ```bash
    python -m unittest discover -s py_toolbox/tests -p "test_*.py"
    ```
    Or, if you are inside the `py_toolbox` directory:
    ```bash
    python -m unittest discover -s tests -p "test_*.py"
    ```

This will discover and run all test cases in files named `test_*.py` within the specified directory.

Additionally, integration tests for the MCP server mode can be found in . These tests demonstrate how to interact with  as a subprocess using the JSON-RPC 2.0 protocol over STDIN/STDOUT.
