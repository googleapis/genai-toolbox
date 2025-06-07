import click
from typing import Optional
import json
import os
import sys
import requests
import uuid
# server_load_config is called internally by server.py's startup_event
# from py_toolbox.server import load_toolbox_config as server_load_config
from py_toolbox.internal.core.logging import get_logger, logging
from py_toolbox.internal.core.registry import SourceRegistry, ToolRegistry
from py_toolbox.internal.sources.base import Source
from py_toolbox.internal.tools.base import Tool

# Import registration functions
register_postgres_source = None
register_postgres_sql_tool = None

try:
    from py_toolbox.internal.sources.postgres import register_source as pg_src_reg
    register_postgres_source = pg_src_reg
except ImportError as e:
    # This warning is now less critical as the CLI might work without specific types if no config is used.
    # logger might not be configured yet if this happens very early.
    click.echo(f"Initial Warning: Could not import PostgreSQL source module: {e}", err=True)

try:
    from py_toolbox.internal.tools.postgres_sql import register_tool as pg_tool_reg
    register_postgres_sql_tool = pg_tool_reg
except ImportError as e:
    click.echo(f"Initial Warning: Could not import PostgreSQL SQL tool module: {e}", err=True)

# MySQL registration imports
register_mysql_source_func = None # Use a distinct name pattern
register_mysql_sql_tool_func = None # Use a distinct name pattern

try:
    from py_toolbox.internal.sources.mysql import register_source as mysql_src_reg_actual # Use distinct alias
    register_mysql_source_func = mysql_src_reg_actual
    click.echo("Successfully imported 'register_source' from MySQL source.", err=True)
except ImportError as e:
    click.echo(f"Warning: Could not import 'register_source' from MySQL source module: {e}", err=True)
except Exception as e:
    click.echo(f"Error importing 'register_source' from MySQL source module: {e}", err=True)

try:
    from py_toolbox.internal.tools.mysql_sql import register_tool as mysql_tool_reg_actual # Use distinct alias
    register_mysql_sql_tool_func = mysql_tool_reg_actual
    click.echo("Successfully imported 'register_tool' from MySQL SQL tool.", err=True)
except ImportError as e:
    click.echo(f"Warning: Could not import 'register_tool' from MySQL SQL tool module: {e}", err=True)
except Exception as e:
    click.echo(f"Error importing 'register_tool' from MySQL SQL tool module: {e}", err=True)

# SQLite registration imports
register_sqlite_source_func = None # Use distinct _func suffix
register_sqlite_sql_tool_func = None # Use distinct _func suffix

try:
    from py_toolbox.internal.sources.sqlite import register_source as sqlite_src_reg_actual # Use distinct alias
    register_sqlite_source_func = sqlite_src_reg_actual
    click.echo("Successfully imported 'register_source' from SQLite source.", err=True)
except ImportError as e:
    click.echo(f"Warning: Could not import 'register_source' from SQLite source module: {e}", err=True)
except Exception as e:
    click.echo(f"Error importing 'register_source' from SQLite source module: {e}", err=True)

try:
    from py_toolbox.internal.tools.sqlite_sql import register_tool as sqlite_tool_reg_actual # Use distinct alias
    register_sqlite_sql_tool_func = sqlite_tool_reg_actual
    click.echo("Successfully imported 'register_tool' from SQLite SQL tool.", err=True)
except ImportError as e:
    click.echo(f"Warning: Could not import 'register_tool' from SQLite SQL tool module: {e}", err=True)
except Exception as e:
    click.echo(f"Error importing 'register_tool' from SQLite SQL tool module: {e}", err=True)

# Neo4j registration imports
register_neo4j_source_func = None # Using _func suffix for consistency
register_neo4j_cypher_tool_func = None # Using _func suffix

try:
    from py_toolbox.internal.sources.neo4j_source import register_source as neo4j_src_reg_actual # Using _actual alias
    register_neo4j_source_func = neo4j_src_reg_actual
    click.echo("Successfully imported 'register_source' from Neo4j source.", err=True)
except ImportError as e:
    click.echo(f"Warning: Could not import 'register_source' from Neo4j source module: {e}", err=True)
except Exception as e:
    click.echo(f"Error importing 'register_source' from Neo4j source module: {e}", err=True)

try:
    from py_toolbox.internal.tools.neo4j_cypher import register_tool as neo4j_tool_reg_actual # Using _actual alias
    register_neo4j_cypher_tool_func = neo4j_tool_reg_actual
    click.echo("Successfully imported 'register_tool' from Neo4j Cypher tool.", err=True)
except ImportError as e:
    click.echo(f"Warning: Could not import 'register_tool' from Neo4j Cypher tool module: {e}", err=True)
except Exception as e:
    click.echo(f"Error importing 'register_tool' from Neo4j Cypher tool module: {e}", err=True)
logger = get_logger(__name__) # Logger is now configured after this line via CLI options

source_registry = SourceRegistry()
tool_registry = ToolRegistry()

initialized_sources: dict[str, Source] = {}
initialized_tools: dict[str, Tool] = {}

def setup_registries():
    """Calls the registration functions for known sources and tools."""
    if register_postgres_source and callable(register_postgres_source):
        try:
            register_postgres_source(source_registry)
            logger.info("Called PostgreSQL source registration.")
        except Exception as e:
            logger.error(f"Error during PostgreSQL source registration: {e}", exc_info=True)
    else:
        logger.warning("PostgreSQL source registration function not found or not callable.")

    if register_postgres_sql_tool and callable(register_postgres_sql_tool):
        try:
            register_postgres_sql_tool(tool_registry)
            logger.info("Called PostgreSQL SQL tool registration.")
        except Exception as e:
            logger.error(f"Error during PostgreSQL SQL tool registration: {e}", exc_info=True)
    else:
        logger.warning("PostgreSQL SQL tool registration function not found or not callable.")

    if register_mysql_source_func and callable(register_mysql_source_func):
        try:
            register_mysql_source_func(source_registry)
            logger.info("Called MySQL source registration function.")
        except Exception as e:
            logger.error(f"Error during MySQL source registration: {e}", exc_info=True)
    else:
        logger.warning("MySQL source registration function ('register_source') not found or not callable.")

    if register_mysql_sql_tool_func and callable(register_mysql_sql_tool_func):
        try:
            register_mysql_sql_tool_func(tool_registry)
            logger.info("Called MySQL SQL tool registration function.")
        except Exception as e:
            logger.error(f"Error during MySQL SQL tool registration: {e}", exc_info=True)
    else:
        logger.warning("MySQL SQL tool registration function ('register_tool') not found or not callable.")

    if register_sqlite_source_func and callable(register_sqlite_source_func):
        try:
            register_sqlite_source_func(source_registry)
            logger.info("Called SQLite source registration function.")
        except Exception as e:
            logger.error(f"Error during SQLite source registration: {e}", exc_info=True)
    else:
        logger.warning("SQLite source registration function ('register_source') not found or not callable.")

    if register_sqlite_sql_tool_func and callable(register_sqlite_sql_tool_func):
        try:
            register_sqlite_sql_tool_func(tool_registry)
            logger.info("Called SQLite SQL tool registration function.")
        except Exception as e:
            logger.error(f"Error during SQLite SQL tool registration: {e}", exc_info=True)
    else:
        logger.warning("SQLite SQL tool registration function ('register_tool') not found or not callable.")

    if register_neo4j_source_func and callable(register_neo4j_source_func):
        try:
            register_neo4j_source_func(source_registry)
            logger.info("Called Neo4j source registration function.")
        except Exception as e:
            logger.error(f"Error during Neo4j source registration: {e}", exc_info=True)
    else:
        logger.warning("Neo4j source registration function ('register_source') not found or not callable.")

    if register_neo4j_cypher_tool_func and callable(register_neo4j_cypher_tool_func):
        try:
            register_neo4j_cypher_tool_func(tool_registry)
            logger.info("Called Neo4j Cypher tool registration function.")
        except Exception as e:
            logger.error(f"Error during Neo4j Cypher tool registration: {e}", exc_info=True)
    else:
        logger.warning("Neo4j Cypher tool registration function ('register_tool') not found or not callable.")

# JSON-RPC 2.0 Error Codes (subset)
JSONRPC_PARSE_ERROR = -32700
JSONRPC_INVALID_REQUEST = -32600
JSONRPC_METHOD_NOT_FOUND = -32601
JSONRPC_INVALID_PARAMS = -32602
JSONRPC_INTERNAL_ERROR = -32603
# Application-specific errors: -32000 to -32099

def create_jsonrpc_success_response(request_id, result):
    return {
        "jsonrpc": "2.0",
        "result": result,
        "id": request_id
    }

def create_jsonrpc_error_response(request_id, code, message, data=None):
    error_obj = {"code": code, "message": message}
    if data is not None:
        error_obj["data"] = data
    return {
        "jsonrpc": "2.0",
        "error": error_obj,
        "id": request_id # Can be null if request_id is unknown (e.g. parse error)
    }

def mcp_handle_request(request_obj: dict) -> dict:
    global initialized_tools # Ensure we are using the globally loaded tools
    """
    Handles a parsed JSON-RPC request object and returns a JSON-RPC response object.
    """
    request_id = request_obj.get("id")

    # Validation of method presence is already done before calling this in the mcp_serve_cmd loop.
    method = request_obj["method"]
    # Params can be an array or object for JSON-RPC, but we'll expect an object (dict) for our methods.
    params = request_obj.get("params", {})
    if not isinstance(params, dict):
        logger.warning(f"MCP Request (id: {request_id}, method: {method}): 'params' should be an object/dictionary, got {type(params)}. Proceeding with empty params if applicable.")
        # For methods that don't strictly need params (like list_tools), this might be okay.
        # For others, it will likely lead to an Invalid Params error below.
        # Or, we can enforce it strictly here:
        # return create_jsonrpc_error_response(request_id, JSONRPC_INVALID_PARAMS, "'params' field must be an object if present.")

    logger.debug(f"MCP Request: id={request_id}, method='{method}', params='{str(params)[:200]}...'")

    if method == "list_tools":
        try:
            tools_list = []
            for name, tool_instance in initialized_tools.items():
                try:
                    # Using McpManifest for consistency as it's richer; client can pick description.
                    manifest = tool_instance.get_mcp_manifest() # Changed from get_manifest to get_mcp_manifest
                    tools_list.append({
                        "name": name,
                        "description": manifest.description,
                        # Optionally include full McpManifest if client prefers:
                        # "manifest": manifest.model_dump(exclude_none=True)
                    })
                except Exception as e:
                    logger.error(f"Error getting manifest for tool '{name}' during list_tools: {e}")
                    tools_list.append({"name": name, "description": "Error retrieving description."})
            return create_jsonrpc_success_response(request_id, tools_list)
        except Exception as e:
            logger.error(f"Internal error during 'list_tools': {e}", exc_info=True)
            return create_jsonrpc_error_response(request_id, JSONRPC_INTERNAL_ERROR, "Internal server error processing list_tools.")

    elif method == "get_tool_description":
        if not isinstance(params, dict) or "tool_name" not in params or not isinstance(params["tool_name"], str):
            return create_jsonrpc_error_response(request_id, JSONRPC_INVALID_PARAMS, "Invalid params: 'tool_name' (string) is required.")

        tool_name = params["tool_name"]
        if tool_name not in initialized_tools:
            return create_jsonrpc_error_response(request_id, JSONRPC_METHOD_NOT_FOUND, f"Tool '{tool_name}' not found.") # Re-using METHOD_NOT_FOUND as it fits tool context

        try:
            tool_instance = initialized_tools[tool_name]
            mcp_manifest = tool_instance.get_mcp_manifest()
            # Ensure name is part of the returned manifest if tool doesn't set it
            if not mcp_manifest.name and tool_name: # Check if mcp_manifest.name is empty or None
                 mcp_manifest.name = tool_name
            return create_jsonrpc_success_response(request_id, mcp_manifest.model_dump(exclude_none=True))
        except Exception as e:
            logger.error(f"Internal error during 'get_tool_description' for '{tool_name}': {e}", exc_info=True)
            return create_jsonrpc_error_response(request_id, JSONRPC_INTERNAL_ERROR, f"Internal error getting description for tool '{tool_name}'.")

    elif method == "invoke_tool":
        if not isinstance(params, dict):
             return create_jsonrpc_error_response(request_id, JSONRPC_INVALID_PARAMS, "Invalid params: Expected an object with 'tool_name' and 'invoke_params'.")

        tool_name = params.get("tool_name")
        actual_tool_params = params.get("invoke_params", params.get("tool_params", {}))

        if not isinstance(tool_name, str):
            return create_jsonrpc_error_response(request_id, JSONRPC_INVALID_PARAMS, "Invalid params: 'tool_name' (string) is required.")
        if not isinstance(actual_tool_params, dict):
            # If actual_tool_params ended up being non-dict due to params.get("tool_params", {}) where params itself wasn't a dict.
            # This case might be redundant if the top-level params check is strict.
            return create_jsonrpc_error_response(request_id, JSONRPC_INVALID_PARAMS, "Invalid params: 'invoke_params' or 'tool_params' must be an object/dictionary.")

        if tool_name not in initialized_tools:
            return create_jsonrpc_error_response(request_id, JSONRPC_METHOD_NOT_FOUND, f"Tool '{tool_name}' not found.")

        try:
            tool_instance = initialized_tools[tool_name]

            auth_required_by_tool = tool_instance.get_manifest().auth_required
            if auth_required_by_tool:
                logger.warning(f"Tool '{tool_name}' requires authorization: {auth_required_by_tool}. "
                               "MCP server currently does not process/pass specific auth context from client requests. "
                               "Tool's is_authorized() will be called with an empty list of verified services.")
                if not tool_instance.is_authorized([]):
                    return create_jsonrpc_error_response(request_id, JSONRPC_INTERNAL_ERROR,
                                                       f"Tool '{tool_name}' not authorized.",
                                                       {"reason": "authorization_failed", "required": auth_required_by_tool})
            else:
                 if not tool_instance.is_authorized([]):
                    logger.error(f"Tool '{tool_name}' (requires no auth) failed its authorization check unexpectedly for MCP call.")
                    return create_jsonrpc_error_response(request_id, JSONRPC_INTERNAL_ERROR, "Tool authorization logic error.")

            result = tool_instance.invoke(actual_tool_params)
            return create_jsonrpc_success_response(request_id, result)
        except ConnectionError as e:
            logger.error(f"MCP: Connection error invoking tool '{tool_name}': {e}", exc_info=True)
            return create_jsonrpc_error_response(request_id, JSONRPC_INTERNAL_ERROR, f"Connection error for tool '{tool_name}': {str(e)}", {"tool_name": tool_name, "type": "ConnectionError"})
        except ValueError as e:
            logger.warning(f"MCP: Value error invoking tool '{tool_name}': {e}", exc_info=False) # Less verbose for value errors
            return create_jsonrpc_error_response(request_id, JSONRPC_INVALID_PARAMS, f"Invalid parameters or value for tool '{tool_name}': {str(e)}", {"tool_name": tool_name, "type": "ValueError"})
        except Exception as e:
            logger.error(f"MCP: Error invoking tool '{tool_name}': {e}", exc_info=True)
            return create_jsonrpc_error_response(request_id, JSONRPC_INTERNAL_ERROR, f"Error during tool '{tool_name}' execution: {str(e)}", {"tool_name": tool_name, "type": type(e).__name__})

    else:
        logger.warning(f"MCP: Method not found: {method} (id: {request_id})")
        return create_jsonrpc_error_response(request_id, JSONRPC_METHOD_NOT_FOUND, f"Method '{method}' not found.")

def cli(ctx, config, log_level):
    """A Python toolkit for database interactions and other tools."""
    # Set logger level for the root logger and any handlers already configured by get_logger
    # This ensures all loggers created by get_logger() will adhere to this level.
    # logging.getLogger().setLevel(getattr(logging, log_level.upper()))
    # If get_logger configures handlers on a specific logger, ensure its level is also set.
    # logger.setLevel(getattr(logging, log_level.upper())) # For the main module's logger
    # Also, ensure any handlers on the root logger are set to this level or lower.
    # for handler in logging.getLogger().handlers:
    #    handler.setLevel(getattr(logging, log_level.upper()))

    # Simplified logging setup: basicConfig affects the root logger.
    # get_logger in core.logging gets a logger by name; its level will be affected if it's a child of root
    # and doesn't have its own level explicitly set higher.
    # The handler formatting is set in core.logging.
    effective_log_level = getattr(logging, log_level.upper())
    logging.basicConfig(level=effective_log_level) #This will set the root logger level
    logger.setLevel(effective_log_level) # Ensure our module's logger is also set

    # If core.logging adds a handler to its own logger, ensure its level is also updated.
    # Or, ensure core.logging's handler respects the logger's effective level.
    # The current core.logging.get_logger sets the logger's level, so this should be fine.

    logger.info(f"Log level set to {log_level.upper()}")

    ctx.obj = {'CONFIG_PATH': config}

    if not os.path.exists(config):
        logger.warning(f"Configuration file '{config}' not found. Tool/source loading will be skipped.")

    setup_registries()

    global initialized_sources, initialized_tools
    try:
        if os.path.exists(config):
            initialized_sources = source_registry.load_sources_from_config(config)
            initialized_tools = tool_registry.load_tools_from_config(config, initialized_sources)
            logger.info(f"Loaded {len(initialized_sources)} sources and {len(initialized_tools)} tools from '{config}'.")
        else:
            initialized_sources = {}
            initialized_tools = {}
            logger.info("No config file found; registries initialized but no instances loaded.")

    except Exception as e:
        logger.error(f"Failed to load configuration or initialize components: {e}", exc_info=True)
        click.echo(f"Error during setup: {e}", err=True)

@cli.command("list-tools")
def list_tools_cmd():
    """Lists all available/configured tools."""
    if not initialized_tools:
        click.echo("No tools configured or loaded. Check your configuration file and ensure it's correctly specified.")
        return

    click.echo("Available tools:")
    for name, tool_instance in initialized_tools.items():
        try:
            manifest = tool_instance.get_manifest()
            click.echo(f"- {name}: {manifest.description}")
            if manifest.parameters:
                click.echo("  Parameters:")
                for param in manifest.parameters:
                    click.echo(f"    - {param.name} ({param.type}, required: {param.required}): {param.description}")
            auth_req = manifest.auth_required
            if auth_req:
                click.echo(f"  Requires authorization: {auth_req}")
            click.echo("")
        except Exception as e:
            click.echo(f"Error retrieving manifest for tool {name}: {e}", err=True)

@cli.command("invoke-tool")
@click.argument('tool_name')
@click.argument('tool_params_json', required=False, default='{}')
@click.pass_context
def invoke_tool_cmd(ctx, tool_name: str, tool_params_json: str):
    """Invokes a specified tool with given parameters (as JSON string)."""
    if not initialized_tools:
        click.echo(f"Error: No tools loaded. Cannot find tool '{tool_name}'. Ensure config is loaded.", err=True)
        ctx.exit(1)

    if tool_name not in initialized_tools:
        click.echo(f"Error: Tool '{tool_name}' not found or not configured.", err=True)
        available = ", ".join(initialized_tools.keys())
        if not available: available = "None"
        click.echo(f"Available tools are: {available}")
        ctx.exit(1)

    tool_instance = initialized_tools[tool_name]

    try:
        params = json.loads(tool_params_json)
        if not isinstance(params, dict):
            raise ValueError("Tool parameters must be a JSON object (dictionary).")
    except json.JSONDecodeError as e:
        click.echo(f"Error: Invalid JSON provided for tool parameters: {e}", err=True)
        ctx.exit(1)
    except ValueError as e:
        click.echo(f"Error: {e}", err=True)
        ctx.exit(1)

    try:
        # Authentication handling for local CLI:
        auth_required_by_tool = tool_instance.get_manifest().auth_required
        if auth_required_by_tool:
            logger.warning(f"Tool '{tool_name}' requires authorization: {auth_required_by_tool}. "
                           "For local CLI execution, this check is currently bypassed by providing the tool's own requirements as 'verified'.")
            # Pass the tool's own requirements as if they were verified services.
            # This allows the tool's internal is_authorized logic to proceed as if these services were confirmed.
            if not tool_instance.is_authorized(auth_required_by_tool):
                 logger.error(f"Tool '{tool_name}' authorization check failed unexpectedly even when its own required services {auth_required_by_tool} were notionally provided. This may indicate an issue in the tool's is_authorized() logic when presented with its requirements.")
                 click.echo(f"Error: Tool '{tool_name}' authorization check failed. See logs for details.", err=True)
                 ctx.exit(1)
            else:
                logger.info(f"Tool '{tool_name}' considered authorized for local CLI execution by providing its own auth requirements ({auth_required_by_tool}) as verified services.")
        else: # No auth explicitly required by tool's manifest
            # Call is_authorized with an empty list, meaning no auth services were verified externally.
            # The tool should return True if it genuinely requires no auth.
            if not tool_instance.is_authorized([]):
                 logger.error(f"Tool '{tool_name}' (which lists no specific auth_required in manifest) failed its authorization check when no verified services were provided. This tool might have internal auth logic not declared in manifest or an issue in is_authorized().")
                 click.echo(f"Error: Tool '{tool_name}' authorization check failed. It requires no auth in manifest but is_authorized([]) returned false. See logs for details.", err=True)
                 ctx.exit(1)
            else:
                logger.debug(f"Tool '{tool_name}' requires no auth per manifest and passed is_authorized([]).")

        logger.info(f"Invoking tool '{tool_name}' with parameters: {params}")
        result = tool_instance.invoke(params)
        click.echo("Tool invocation successful. Result:")
        try:
            click.echo(json.dumps(result, indent=2, default=str)) # Use default=str for non-serializable
        except TypeError: # Should be caught by default=str, but as a fallback
            logger.warning("Result for tool '{tool_name}' was not directly JSON serializable, used str().")
            click.echo(str(result))

    except ConnectionError as e:
        logger.error(f"Connection error while invoking tool '{tool_name}': {e}", exc_info=True)
        click.echo(f"Connection Error: {e}", err=True)
        ctx.exit(1)
    except ValueError as e:
        logger.error(f"Value error while invoking tool '{tool_name}': {e}", exc_info=True)
        click.echo(f"Value Error: {e}", err=True)
        ctx.exit(1)
    except Exception as e:
        logger.error(f"Error invoking tool '{tool_name}': {e}", exc_info=True)
        click.echo(f"An unexpected error occurred: {e}", err=True)
        ctx.exit(1)

def cleanup_sources_on_exit():
    logger.debug("CLI exiting. Cleaning up sources...")
    if not initialized_sources:
        logger.debug("No sources were initialized, skipping cleanup.")
        return
    for src_name, src_instance in initialized_sources.items():
        try:
            logger.info(f"Attempting to close source: {src_name}")
            src_instance.close() # Ensure close method is robust
            logger.info(f"Closed source: {src_name}")
        except Exception as e:
            logger.error(f"Error closing source {src_name}: {e}", exc_info=True)

# Using click's recommended way for teardown for groups
@cli.result_callback()
def process_result(result, **kwargs):
    # This function is called after any command within the group has finished.
    cleanup_sources_on_exit()

def register_tools_with_hub(hub_api_url: str, microservice_id: str, tools_to_register: dict, current_config_path: str):
    if not hub_api_url:
        logger.info("MCP_HUB_API_URL not configured. Skipping registration with Hub.")
        return

    if not tools_to_register:
        logger.info("No tools initialized in this py_toolbox instance. Nothing to register with Hub.")
        return

    logger.info(f"Attempting to register {len(tools_to_register)} tools with MCP Hub at {hub_api_url} for microservice_id '{microservice_id}'...")

    for tool_name, tool_instance in tools_to_register.items():
        try:
            mcp_manifest = tool_instance.get_mcp_manifest().model_dump(exclude_none=True)

            # Define invocation_info (Refined in Step 6)


                        # Get absolute path for the config_file for clarity in Hub registration


                        abs_config_path = os.path.abspath(current_config_path)





                        py_toolbox_main_executable = f"python {os.path.abspath(sys.argv[0])}"


                        # A more robust way for general case might be needed if not run as "python main.py"


                        # For instance, if it's an installed script, sys.argv[0] is correct.


                        # If run with "python -m py_toolbox.main", sys.argv[0] might be different.


                        # This simple version assumes direct script execution for now.





                        invocation_info = {


                            "type": "mcp_jsonrpc_stdio",


                            "command_template": f"{py_toolbox_main_executable} --config {{config_file_path}} mcp-serve",


                            "config_file_path_for_this_instance": abs_config_path,


                            "json_rpc_request_template": {


                                "jsonrpc": "2.0",


                                "method": "invoke_tool",


                                "params": {


                                    "tool_name": tool_name,


                                    "invoke_params": {


                                        # Placeholder, consumer fills this based on tool's McpManifest


                                        "param1": "value1_example",


                                        "...": "..."


                                    }


                                },


                                "id": "<client_generated_request_id>"


                            },


                            "notes": "Replace placeholders in command_template and json_rpc_request_template. " +


                                     "'invoke_params' must match the tool's input_schema from its McpManifest."


                        }

            registration_payload = {
                "tool_name": tool_name,
                "microservice_id": microservice_id,
                "description": mcp_manifest.get("description", "No description provided."),
                "invocation_info": invocation_info,
                "mcp_manifest": mcp_manifest
            }

            headers = {"Content-Type": "application/json"}
            target_url = f"{hub_api_url.rstrip('/')}/tools"

            logger.debug(f"Registering tool '{tool_name}' with payload: {json.dumps(registration_payload)}")
            response = requests.post(target_url, json=registration_payload, headers=headers, timeout=10)

            if response.status_code == 201 or response.status_code == 200:
                logger.info(f"Successfully registered/updated tool '{tool_name}' with MCP Hub. Status: {response.status_code}. Response: {response.json()}")
            else:
                logger.error(f"Failed to register tool '{tool_name}' with MCP Hub. Status: {response.status_code}. Response: {response.text}")

        except requests.exceptions.RequestException as e:
            logger.error(f"HTTP request error while registering tool '{tool_name}' with MCP Hub: {e}", exc_info=True)
        except Exception as e:
            logger.error(f"Error preparing registration for tool '{tool_name}': {e}", exc_info=True)

@cli.command("mcp-serve")
@click.pass_context # To access global options like --config and --log-level
def mcp_serve_cmd(ctx):
    """Starts the server in MCP (STDIN/STDOUT JSON-RPC) mode."""
    global initialized_sources, initialized_tools # Declare them global to modify
    config_path = ctx.obj.get('CONFIG_PATH', 'tools.yaml') # From global option
    logger.info(f"Starting MCP server. Reading from STDIN, writing to STDOUT.")
    logger.info(f"Using toolbox config file: {config_path}")

    if not initialized_tools and os.path.exists(config_path):
        logger.warning("MCP Server: Tools not initialized by main CLI context. Attempting to load now.")
        try:
            if not source_registry._factories:
                 logger.error("MCP Server: Registries are empty. Cannot load tools. Ensure setup_registries() is called.")
                 sys.exit(1)

            initialized_sources = source_registry.load_sources_from_config(config_path)
            initialized_tools = tool_registry.load_tools_from_config(config_path, initialized_sources)
            logger.info(f"MCP Server: Loaded {len(initialized_sources)} sources and {len(initialized_tools)} tools from '{config_path}'.")
# Register tools with MCP Hub
    hub_api_url = os.getenv("MCP_HUB_API_URL")
    default_ms_id_suffix = str(uuid.uuid5(uuid.NAMESPACE_DNS, config_path + os.uname().nodename))[:8]
    microservice_id = os.getenv("PYTOOLBOX_MICROSERVICE_ID", f"pytoolbox_ms_{default_ms_id_suffix}")

    register_tools_with_hub(hub_api_url, microservice_id, initialized_tools, config_path)
        except Exception as e:
            logger.error(f"MCP Server: Failed to load config/tools in mcp-serve: {e}", exc_info=True)
            error_resp = create_jsonrpc_error_response(None, JSONRPC_INTERNAL_ERROR, f"Critical server error during tool loading: {e}")
            sys.stdout.write(json.dumps(error_resp) + "\\n")
            sys.stdout.flush()
            sys.exit(1)

    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue

        request_id = None
        try:
            logger.debug(f"MCP In: {line}")
            request_obj = json.loads(line)

            if not isinstance(request_obj, dict) or request_obj.get("jsonrpc") != "2.0" or "method" not in request_obj:
                logger.error(f"Invalid JSON-RPC request: {line}")
                response_obj = create_jsonrpc_error_response(request_obj.get("id"), JSONRPC_INVALID_REQUEST, "Invalid JSON-RPC 2.0 request structure.")
            else:
                request_id = request_obj.get("id")
                response_obj = mcp_handle_request(request_obj)

        except json.JSONDecodeError:
            logger.error(f"Failed to parse JSON from STDIN: {line}", exc_info=True)
            response_obj = create_jsonrpc_error_response(None, JSONRPC_PARSE_ERROR, "Failed to parse JSON request.")
        except Exception as e:
            logger.error(f"Unexpected error processing MCP request: {line}", exc_info=True)
            response_obj = create_jsonrpc_error_response(request_id, JSONRPC_INTERNAL_ERROR, f"Internal server error: {e}")

        if response_obj:
            try:
                response_str = json.dumps(response_obj)
                logger.debug(f"MCP Out: {response_str}")
                sys.stdout.write(response_str + "\\n")
                sys.stdout.flush()
            except Exception as e:
                logger.critical(f"FATAL: Failed to serialize JSON-RPC response: {response_obj}", exc_info=True)
                fallback_error = {"jsonrpc": "2.0", "error": {"code": JSONRPC_INTERNAL_ERROR, "message": "Fatal: Response serialization failed."}, "id": request_id}
                try:
                    sys.stdout.write(json.dumps(fallback_error) + "\\n")
                    sys.stdout.flush()
                except:
                    pass

    logger.info("MCP server STDIN stream ended. Shutting down.")

if __name__ == '__main__':
    cli()
