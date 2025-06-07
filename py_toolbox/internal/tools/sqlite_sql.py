from typing import Any, Dict, List, Mapping, Optional
import sqlite3 # For error types

from py_toolbox.internal.tools.base import Tool, ToolConfig, Manifest, McpManifest, ParameterManifest
from py_toolbox.internal.sources.base import Source
from py_toolbox.internal.sources.sqlite import SQLiteSource, SOURCE_KIND as SQLITE_SOURCE_KIND # Correct import
from py_toolbox.internal.core.logging import get_logger
# from py_toolbox.internal.core.registry import ToolRegistry # Avoid direct import for registration

logger = get_logger(__name__)

TOOL_KIND = "sqlite-sql"

class SQLiteSQLConfig(ToolConfig):
    def __init__(self, name: str, kind: str, description: str, source_name: str, default_statement: Optional[str] = None, auth_required: Optional[List[str]] = None, parameters: Optional[List[Dict[str, Any]]] = None, **kwargs):
        super().__init__(name, kind, description, **kwargs) # Pass kwargs to parent
        if kind != TOOL_KIND:
            raise ValueError(f"Kind mismatch for SQLiteSQLConfig. Expected '{TOOL_KIND}', got '{kind}'")
        self.source_name = source_name
        self.default_statement = default_statement
        self.auth_required = auth_required if auth_required is not None else []
        self.parameters_config = parameters if parameters is not None else []
        # self.extra_kwargs are handled by super().__init__

    def tool_config_kind(self) -> str:
        return TOOL_KIND

    def initialize(self, sources: Mapping[str, Source]) -> Tool:
        logger.info(f"Initializing SQLite SQL tool: {self.name} using source '{self.source_name}'")
        source_instance = sources.get(self.source_name)
        if not source_instance:
            raise ValueError(f"Source '{self.source_name}' not found for tool '{self.name}'. Available: {list(sources.keys())}")
        if not isinstance(source_instance, SQLiteSource):
            raise ValueError(f"Source '{self.source_name}' for tool '{self.name}' is not a SQLiteSource. Actual type: {type(source_instance).__name__}")
        return SQLiteSQLTool(self, source_instance)

    @classmethod
    def from_dict(cls, name: str, data: Dict[str, Any]) -> 'SQLiteSQLConfig':
        required_fields = ['source', 'description']
        for field in required_fields:
            if field not in data:
                raise ValueError(f"SQLite SQL tool config for '{name}' missing field: '{field}'")

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

class SQLiteSQLTool(Tool):
    def __init__(self, config: SQLiteSQLConfig, source: SQLiteSource):
        super().__init__(config.name, config.tool_config_kind())
        self.config = config
        self.source = source # Instance of SQLiteSource
        self._manifest = self._build_manifest()
        self._mcp_manifest = self._build_mcp_manifest()

    def tool_kind(self) -> str:
        return TOOL_KIND

    def _build_manifest(self) -> Manifest:
        params = [ParameterManifest(**p) for p in self.config.parameters_config]
        return Manifest(description=self.config.description, parameters=params, auth_required=self.config.auth_required)

    def _build_mcp_manifest(self) -> McpManifest:
        properties = {}
        required_props = []
        for p_cfg in self.config.parameters_config:
            p_name = p_cfg.get("name")
            if not p_name: continue
            properties[p_name] = {"type": p_cfg.get("type", "string"), "description": p_cfg.get("description", "")}
            if p_cfg.get("required", False): required_props.append(p_name)
        return McpManifest(name=self.config.name, description=self.config.description, input_schema={"type": "object", "properties": properties, "required": required_props})

    def get_manifest(self) -> Manifest: return self._manifest
    def get_mcp_manifest(self) -> McpManifest: return self._mcp_manifest

    def is_authorized(self, verified_auth_services: List[str]) -> bool:
        if not self.config.auth_required: return True
        for req_auth in self.config.auth_required:
            if req_auth in verified_auth_services: return True
        logger.warning(f"Tool '{self.name}' not authorized. Required: {self.config.auth_required}, Provided: {verified_auth_services}")
        return False

    def invoke(self, params: Dict[str, Any]) -> List[Dict[str, Any]]:
        sql_statement = params.get("statement", self.config.default_statement)
        if not sql_statement:
            raise ValueError("No SQL statement provided for SQLite tool.")

        query_args = params.get("args", [])
        # SQLite uses '?' for placeholders, ensure args is a tuple or list
        if not isinstance(query_args, (list, tuple)):
            if query_args is None: # No args is fine
                query_args = []
            else: # Single arg should be wrapped in a list/tuple
                query_args = [query_args]

        logger.info(f"Invoking SQLite SQL tool '{self.name}' on file '{self.source.config.database_file}' with statement: {sql_statement[:100]}... and args: {query_args}")

        results = []
        try:
            # get_connection is a context manager from SQLiteSource
            with self.source.get_connection() as conn:
                cursor = conn.cursor() # sqlite3.Row factory is set on connection by the source

                # SQLite uses '?' for placeholders
                cursor.execute(sql_statement, tuple(query_args)) # Ensure args is a tuple

                if cursor.description:
                    raw_results = cursor.fetchall()
                    results = [dict(row) for row in raw_results]
                    logger.debug(f"Query executed successfully. Rows returned: {len(results)}")
                else:
                    logger.debug(f"DML/DDL query executed. Row count (affected rows): {cursor.rowcount}")
                    results = [{"status": "success", "rowcount": cursor.rowcount, "lastrowid": cursor.lastrowid}]

                conn.commit() # Commit changes for DML/DDL statements
        except sqlite3.Error as e:
            logger.error(f"Database error in SQLite SQL tool '{self.name}': {e}")
            # SQLite typically rolls back transactions automatically on error if they were started implicitly.
            # If using conn.execute("BEGIN TRANSACTION"), explicit rollback might be needed in error handling.
            raise
        except Exception as e: # Catch any other unexpected errors
            logger.error(f"Unexpected error in SQLite SQL tool '{self.name}': {e}", exc_info=True)
            raise
        return results

# Registration function for this tool
def register_tool(registry: Any): # Using Any for registry type
    registry.register(TOOL_KIND, SQLiteSQLConfig.from_dict)
    logger.info(f"SQLite SQL tool kind '{TOOL_KIND}' registration function called.")
