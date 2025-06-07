from typing import Any, Dict, List, Mapping, Optional
import mysql.connector # For error types

from py_toolbox.internal.tools.base import Tool, ToolConfig, Manifest, McpManifest, ParameterManifest
from py_toolbox.internal.sources.base import Source
from py_toolbox.internal.sources.mysql import MySQLSource, SOURCE_KIND as MYSQL_SOURCE_KIND # Ensure correct import
from py_toolbox.internal.core.logging import get_logger
# from py_toolbox.internal.core.registry import ToolRegistry # Avoid direct import for registration

logger = get_logger(__name__)

TOOL_KIND = "mysql-sql"

class MySQLSQLConfig(ToolConfig):
    def __init__(self, name: str, kind: str, description: str, source_name: str, default_statement: Optional[str] = None, auth_required: Optional[List[str]] = None, parameters: Optional[List[Dict[str, Any]]] = None, **kwargs):
        super().__init__(name, kind, description, **kwargs) # Pass kwargs to parent
        if kind != TOOL_KIND:
            raise ValueError(f"Kind mismatch for MySQLSQLConfig. Expected '{TOOL_KIND}', got '{kind}'")
        self.source_name = source_name
        self.default_statement = default_statement
        self.auth_required = auth_required if auth_required is not None else []
        self.parameters_config = parameters if parameters is not None else []
        # self.extra_kwargs are handled by super().__init__

    def tool_config_kind(self) -> str:
        return TOOL_KIND

    def initialize(self, sources: Mapping[str, Source]) -> Tool:
        logger.info(f"Initializing MySQL SQL tool: {self.name} using source '{self.source_name}'")

        source_instance = sources.get(self.source_name)
        if not source_instance:
            raise ValueError(f"Source '{self.source_name}' not found for tool '{self.name}'. Available sources: {list(sources.keys())}")

        if not isinstance(source_instance, MySQLSource):
            raise ValueError(f"Source '{self.source_name}' for tool '{self.name}' is not a MySQLSource. Actual type: {type(source_instance).__name__}")
        return MySQLSQLTool(self, source_instance)

    @classmethod
    def from_dict(cls, name: str, data: Dict[str, Any]) -> 'MySQLSQLConfig':
        required_fields = ['source', 'description']
        for field in required_fields:
            if field not in data:
                raise ValueError(f"MySQL SQL tool config for '{name}' is missing required field: '{field}'")

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

class MySQLSQLTool(Tool):
    def __init__(self, config: MySQLSQLConfig, source: MySQLSource):
        super().__init__(config.name, config.tool_config_kind())
        self.config = config
        self.source = source # Instance of MySQLSource
        self._manifest = self._build_manifest()
        self._mcp_manifest = self._build_mcp_manifest()

    def tool_kind(self) -> str:
        return TOOL_KIND

    def _build_manifest(self) -> Manifest:
        params = [ParameterManifest(**p) for p in self.config.parameters_config]
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
            return True # No auth configured for this tool
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
        logger.info(f"Invoking MySQL SQL tool '{self.name}' on source '{self.source.name}' with statement: {sql_statement[:100]}... and args: {query_args}")

        if not self.source._pool: # Ensure source is connected (pool is initialized)
            try:
                logger.info(f"Source '{self.source.name}' pool is not initialized for tool '{self.name}'. Attempting connect.")
                self.source.connect()
            except Exception as e: # Catch connection errors from source.connect()
                logger.error(f"Failed to connect source '{self.source.name}' for tool '{self.name}': {e}")
                raise ConnectionError(f"Source '{self.source.name}' not connected: {e}") from e

        results = []
        conn_from_pool = None
        try:
            # get_connection is a context manager
            with self.source.get_connection() as conn_from_pool:
                # conn_from_pool is the actual connection object
                with conn_from_pool.cursor(dictionary=True) as cursor: # dictionary=True for dict-like rows
                    # MySQL uses %s for placeholders, ensure args is a tuple
                    cursor.execute(sql_statement, tuple(query_args) if query_args else None)

                    if cursor.description: # Check if the query returns rows (e.g., SELECT)
                        results = cursor.fetchall() # list of dicts
                        logger.debug(f"Query executed successfully. Rows returned: {len(results)}")
                    else: # For INSERT, UPDATE, DELETE etc.
                        logger.debug(f"Query executed successfully. No rows returned (e.g., DML/DDL). Row count: {cursor.rowcount}")
                        results = [{"status": "success", "rowcount": cursor.rowcount, "lastrowid": cursor.lastrowid}]

                    # Important: Commit changes for DML statements
                    if not sql_statement.strip().upper().startswith("SELECT"):
                        conn_from_pool.commit()
                        logger.debug("Transaction committed for DML/DDL statement.")

        except mysql.connector.Error as e:
            logger.error(f"Database error in MySQL SQL tool '{self.name}' on source '{self.source.name}': {e}")
            if conn_from_pool and conn_from_pool.is_connected():
                try:
                    conn_from_pool.rollback() # Attempt rollback on database error
                    logger.info("Transaction rolled back due to database error.")
                except mysql.connector.Error as rb_err:
                    logger.error(f"Failed to rollback transaction for tool '{self.name}': {rb_err}")
            raise # Re-raise the original database error
        except ConnectionError as e: # Raised by self.source.get_connection() if pool is gone or connect fails
             logger.error(f"Connection acquisition error for tool '{self.name}': {e}")
             raise
        except Exception as e: # Catch any other unexpected errors
            logger.error(f"Unexpected error in MySQL SQL tool '{self.name}': {e}", exc_info=True)
            if conn_from_pool and conn_from_pool.is_connected(): # If there's a live connection, try rollback
                 try:
                    conn_from_pool.rollback()
                 except mysql.connector.Error as rb_err:
                    logger.error(f"Failed to rollback transaction for tool '{self.name}' during unexpected error: {rb_err}")
            raise

        return results

# Registration function for this tool (to be called by main.py or similar)
def register_tool(registry: Any): # Using Any for registry type
    registry.register(TOOL_KIND, MySQLSQLConfig.from_dict)
    logger.info(f"MySQL SQL tool kind '{TOOL_KIND}' registration function called.")
