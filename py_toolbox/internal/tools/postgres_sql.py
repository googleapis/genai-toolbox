from typing import Any, Dict, List, Mapping, Optional
import json
import psycopg2 # For error handling and RealDictCursor
from psycopg2 import extras # For RealDictCursor

from py_toolbox.internal.tools.base import Tool, ToolConfig, Manifest, McpManifest, ParameterManifest
from py_toolbox.internal.sources.base import Source
from py_toolbox.internal.sources.postgres import PostgresSource, SOURCE_KIND as POSTGRES_SOURCE_KIND
# Ensure ToolRegistry is imported correctly for registration
from py_toolbox.internal.core import ToolRegistry
from py_toolbox.internal.core.logging import get_logger

logger = get_logger(__name__)

TOOL_KIND = "postgres-sql"

class PostgresSQLConfig(ToolConfig):
    def __init__(self, name: str, kind: str, description: str, source_name: str, default_statement: Optional[str] = None, auth_required: Optional[List[str]] = None, parameters: Optional[List[Dict[str, Any]]] = None, **kwargs):
        super().__init__(name, kind, description, **kwargs) # Pass kwargs to parent
        if kind != TOOL_KIND:
            raise ValueError(f"Kind mismatch for PostgresSQLConfig. Expected '{TOOL_KIND}', got '{kind}'")
        self.source_name = source_name
        self.default_statement = default_statement
        self.auth_required = auth_required if auth_required is not None else []
        self.parameters_config = parameters if parameters is not None else []
        # self.extra_kwargs are handled by super().__init__

    def tool_config_kind(self) -> str:
        return TOOL_KIND

    def initialize(self, sources: Mapping[str, Source]) -> Tool:
        logger.info(f"Initializing PostgreSQL SQL tool: {self.name} using source '{self.source_name}'")

        source_instance = sources.get(self.source_name)
        if not source_instance:
            raise ValueError(f"Source '{self.source_name}' not found for tool '{self.name}'. Available sources: {list(sources.keys())}")

        if not isinstance(source_instance, PostgresSource):
            raise ValueError(f"Source '{self.source_name}' for tool '{self.name}' is not a PostgresSource. Got type: {type(source_instance).__name__}")

        return PostgresSQLTool(self, source_instance)

    @classmethod
    def from_dict(cls, name: str, data: Dict[str, Any]) -> 'PostgresSQLConfig':
        required_fields = ['source', 'description']
        for field in required_fields:
            if field not in data:
                raise ValueError(f"PostgreSQL SQL tool config for '{name}' is missing required field: '{field}'")

        current_fields = ['name', 'kind', 'description', 'source', 'statement', 'authRequired', 'parameters']
        extra_kwargs = {k: v for k, v in data.items() if k not in current_fields}

        return cls(
            name=name,
            kind=data.get("kind", TOOL_KIND),
            description=data['description'],
            source_name=data['source'],
            default_statement=data.get('statement'),
            auth_required=data.get('authRequired'),
            parameters=data.get('parameters'),
            **extra_kwargs
        )

class PostgresSQLTool(Tool):
    def __init__(self, config: PostgresSQLConfig, source: PostgresSource):
        super().__init__(config.name, config.tool_config_kind())
        self.config = config
        self.source = source
        self._manifest = self._build_manifest()
        self._mcp_manifest = self._build_mcp_manifest()

    def tool_kind(self) -> str:
        return TOOL_KIND

    def _build_manifest(self) -> Manifest:
        params = []
        for p_cfg in self.config.parameters_config:
            # Ensure all fields expected by ParameterManifest are present or have defaults
            params.append(ParameterManifest(
                name=p_cfg.get("name", ""), # Ensure name is not None
                type=p_cfg.get("type", "string"), # Default type if not specified
                description=p_cfg.get("description", ""), # Default description
                required=p_cfg.get("required", False) # Default required
            ))
        return Manifest(
            description=self.config.description,
            parameters=params,
            auth_required=self.config.auth_required
        )

    def _build_mcp_manifest(self) -> McpManifest:
        properties = {}
        required_props = []
        for p_cfg in self.config.parameters_config:
            p_name = p_cfg.get("name")
            if not p_name: continue
            properties[p_name] = {
                "type": p_cfg.get("type", "string"),
                "description": p_cfg.get("description", "")
            }
            if p_cfg.get("required", False):
                required_props.append(p_name)

        return McpManifest(
            name=self.config.name,
            description=self.config.description,
            input_schema={
                "type": "object",
                "properties": properties,
                "required": required_props
            }
        )

    def get_manifest(self) -> Manifest:
        return self._manifest

    def get_mcp_manifest(self) -> McpManifest:
        return self._mcp_manifest

    def is_authorized(self, verified_auth_services: List[str]) -> bool:
        if not self.config.auth_required:
            return True
        for required_auth in self.config.auth_required:
            if required_auth in verified_auth_services:
                return True
        logger.warning(f"Tool '{self.name}' not authorized. Required: {self.config.auth_required}, Provided: {verified_auth_services}")
        return False

    def invoke(self, params: Dict[str, Any]) -> List[Dict[str, Any]]:
        sql_statement = params.get("statement", self.config.default_statement)
        if not sql_statement:
            raise ValueError("No SQL statement provided either in parameters or default config.")

        query_args = params.get("args", [])

        logger.info(f"Invoking PostgreSQL SQL tool '{self.name}' with statement: {sql_statement[:100]}... and args: {query_args}")

        if not self.source._pool:
            try:
                logger.info(f"Source '{self.source.name}' pool is not initialized for tool '{self.name}'. Attempting connect.")
                self.source.connect()
            except Exception as e:
                logger.error(f"Failed to connect source '{self.source.name}' for tool '{self.name}': {e}")
                raise ConnectionError(f"Source '{self.source.name}' not connected: {e}") from e

        results = []
        conn_from_pool = None # Define conn_from_pool here to be accessible in finally
        try:
            with self.source.get_connection() as conn_from_pool: # conn_from_pool is the actual connection
                with conn_from_pool.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cursor:
                    cursor.execute(sql_statement, tuple(query_args) if query_args else None)
                    if cursor.description:
                        raw_results = cursor.fetchall()
                        results = [dict(row) for row in raw_results]
                        logger.debug(f"Query executed successfully. Rows returned: {len(results)}")
                    else:
                        logger.debug(f"Query executed successfully. No rows returned. Row count: {cursor.rowcount}")
                        results = [{"status": "success", "rowcount": cursor.rowcount}]
                conn_from_pool.commit()
        except psycopg2.Error as e:
            logger.error(f"Database error in PostgreSQL SQL tool '{self.name}' on source '{self.source.name}': {e}")
            if conn_from_pool: # Check if connection was established before error
                try:
                    conn_from_pool.rollback()
                except psycopg2.Error as rb_err: # If rollback fails
                    logger.error(f"Failed to rollback transaction for tool '{self.name}': {rb_err}")
            raise
        except ConnectionError as e: # Raised by self.source.get_connection() if pool is gone
             logger.error(f"Connection acquisition error for tool '{self.name}': {e}")
             raise
        except Exception as e:
            logger.error(f"Unexpected error in PostgreSQL SQL tool '{self.name}': {e}")
            if conn_from_pool: # If there's a connection, try to rollback
                 try:
                    conn_from_pool.rollback()
                 except psycopg2.Error as rb_err:
                    logger.error(f"Failed to rollback transaction for tool '{self.name}' during unexpected error: {rb_err}")
            raise

        return results

# Registration function for this tool

# Example of how to call it (application entry point would manage registries)
# if __name__ == '__main__':
#     # This is illustrative
#     # from py_toolbox.internal.core.registry import ToolRegistry
#     # global_tool_registry = ToolRegistry()
#     # register_postgres_sql_tool(global_tool_registry)
#     pass

# Registration function for this tool
def register_tool(registry: ToolRegistry):
    # This function is called by the main application to register the tool
    registry.register(TOOL_KIND, PostgresSQLConfig.from_dict)
    logger.info(f"PostgreSQL SQL tool kind '{TOOL_KIND}' registration function called by main app.")
