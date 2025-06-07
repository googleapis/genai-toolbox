# MCP Hub Server

## Overview

The MCP Hub Server is a central registry for discovering and managing tools offered by `py_toolbox` microservice instances. Each `py_toolbox` instance, when configured to connect to a specific data source and run in `mcp-serve` mode, can register the tools it provides with this Hub. Consumers can then query the Hub to find available tools across all registered microservices.

This Hub acts as a service discovery mechanism, enabling a decoupled architecture where tool providers (data source microservices) and tool consumers can find each other.

## Features

-   **Tool Registration:** Allows `py_toolbox` microservices to register the tools they offer.
-   **Tool Discovery:** Provides HTTP REST API endpoints for consumers to list and get details of available tools.
-   **Persistence:** Uses an SQLite database (`mcp_hub.db`) to store registered tool information.
-   **FastAPI Backend:** Built with FastAPI for a modern, high-performance API.
-   **Heartbeat Mechanism:** Allows microservices to signal their continued availability.

## Project Structure

```
mcp_hub/
├── api/
│   └── routes.py       # API endpoint definitions
├── db/
│   └── database.py     # SQLAlchemy setup, init_db, SQLite DB file location
├── models/
│   └── tool_registry_models.py # SQLAlchemy and Pydantic models
├── tests/
│   ├── test_api_routes.py # Integration tests for the API
│   └── test_database.py   # Test database setup
├── main.py               # Main FastAPI application, startup logic
└── requirements.txt      # Python dependencies
README.md                 # This file
mcp_hub.db                # SQLite database file (created on first run)
```

## Setup

### Prerequisites

-   Python 3.8 or higher.

### Installation

1.  **Clone the repository (if this Hub is part of a larger project structure).**
2.  **Navigate to the `mcp_hub` directory.**
3.  **Create a virtual environment (recommended):**
    ```bash
    python -m venv venv
    source venv/bin/activate  # On Windows: venv\Scripts\activate
    ```
4.  **Install dependencies:**
    ```bash
    pip install -r requirements.txt
    ```

## Running the MCP Hub Server

To run the MCP Hub server:

```bash
python mcp_hub/main.py
```
Or, using Uvicorn directly (often preferred for development with auto-reload):
```bash
uvicorn mcp_hub.main:app --reload --host 0.0.0.0 --port 8080
```
The server will typically run on `http://localhost:8080`. The port can be changed in `mcp_hub/main.py` or via Uvicorn CLI options.

On first startup, an SQLite database file (`mcp_hub.db` inside the `mcp_hub/db/` directory) will be automatically created if it doesn't exist, along with the necessary tables.

## API Endpoints

All API endpoints are prefixed with `/api/v1`.

### Tool Registration

-   **`POST /tools`**
    -   **Description:** Registers a new tool or updates an existing one if the same `microservice_id` and `tool_name` are provided.
    -   **Request Body:** (See `ToolRegistrationRequest` Pydantic model in `mcp_hub/models/tool_registry_models.py`)
        ```json
        {
          "tool_name": "my_specific_query_tool",
          "microservice_id": "unique_id_for_my_pytoolbox_instance",
          "description": "Executes a specific query on a particular database.",
          "invocation_info": {
            "type": "mcp_jsonrpc_stdio",
            "command_template": "python /path/to/py_toolbox/main.py --config /path/to/config.yaml mcp-serve",
            "config_file_path_for_this_instance": "/path/to/config.yaml",
            "json_rpc_request_template": {
              "jsonrpc": "2.0",
              "method": "invoke_tool",
              "params": {
                "tool_name": "my_specific_query_tool",
                "invoke_params": { "...": "..." }
              },
              "id": "<client_generated_request_id>"
            },
            "notes": "Ensure 'invoke_params' matches the tool's input_schema."
          },
          "mcp_manifest": {
            "name": "my_specific_query_tool",
            "description": "Tool description from microservice.",
            "input_schema": {
              "type": "object",
              "properties": { "param1": { "type": "string" } },
              "required": ["param1"]
            }
          }
        }
        ```
    -   **Response (201 Created or 200 OK):** (See `ToolRegistrationResponse`)
        ```json
        {
          "id": 123, // Hub's internal DB ID for the tool registration
          "tool_name": "my_specific_query_tool",
          "microservice_id": "unique_id_for_my_pytoolbox_instance",
          "registered_at": "2023-10-27T10:00:00Z"
        }
        ```

### Tool Discovery

-   **`GET /tools`**
    -   **Description:** Lists all registered tools.
    -   **Query Parameters:**
        -   `microservice_id` (optional, string): Filter tools by a specific microservice ID.
        -   `skip` (optional, int, default: 0): Number of records to skip (for pagination).
        -   `limit` (optional, int, default: 100): Maximum number of records to return.
    -   **Response (200 OK):** A list of `ToolDisplay` objects.
        ```json
        [
          {
            "id": 123,
            "tool_name": "my_specific_query_tool",
            "microservice_id": "unique_id_for_my_pytoolbox_instance",
            "description": "Executes a specific query...",
            "registered_at": "2023-10-27T10:00:00Z",
            "last_heartbeat_at": "2023-10-27T10:05:00Z"
          }
        ]
        ```

-   **`GET /tools/{tool_id}`**
    -   **Description:** Retrieves detailed information for a specific tool by its **Hub-assigned ID**.
    -   **Response (200 OK):** `ToolDetail` object (includes `invocation_info` and `mcp_manifest`).

-   **`GET /tools/lookup?microservice_id=<MS_ID>&tool_name=<TOOL_NAME>`**
    -   **Description:** Retrieves detailed information for a specific tool by its `microservice_id` and `tool_name`.
    -   **Response (200 OK):** `ToolDetail` object.

### Tool Health & Management

-   **`POST /tools/heartbeat/{microservice_id}/{tool_name}`**
    -   **Description:** Allows a microservice to signal that a specific tool it offers is still alive. Updates the `last_heartbeat_at` timestamp for the tool.
    -   **Response (200 OK):** Confirmation message.

-   **`DELETE /tools/{tool_id}`**
    -   **Description:** Deletes a tool registration by its Hub-assigned ID.
    -   **Response (204 No Content).**

## Running Tests

Unit and integration tests are located in the `mcp_hub/tests/` directory.

1.  Ensure you are in the `mcp_hub` directory with the virtual environment activated.
2.  Install test dependencies if any (currently covered by `requirements.txt`).
3.  Run tests using `unittest` discovery:
    ```bash
    python -m unittest discover -s tests -p "test_*.py"
    ```

This will run all tests in files named `test_*.py` within the `tests` directory. The API tests use an in-memory SQLite database, so they won't affect `mcp_hub.db`.
