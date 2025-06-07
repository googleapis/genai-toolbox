import click
import json
import os
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
@click.group()
@click.option('--config', default='tools.yaml', help='Path to the configuration file.', type=click.Path(exists=False, dir_okay=False))
@click.option('--log-level', default='INFO', type=click.Choice(['DEBUG', 'INFO', 'WARNING', 'ERROR', 'CRITICAL'], case_sensitive=False))
@click.pass_context
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

if __name__ == '__main__':
    cli()
