from typing import Any, Dict, List, Mapping, Optional

from py_toolbox.internal.tools.base import Tool, ToolConfig, Manifest, McpManifest, ParameterManifest
from py_toolbox.internal.sources.base import Source
from py_toolbox.internal.sources.neo4j_source import Neo4jSource, SOURCE_KIND as NEO4J_SOURCE_KIND # Corrected import
from py_toolbox.internal.core.logging import get_logger
from neo4j import exceptions as neo4j_exceptions

logger = get_logger(__name__)

TOOL_KIND = "neo4j-cypher"

class Neo4jCypherConfig(ToolConfig):
    def __init__(self, name: str, kind: str, description: str, source_name: str, default_cypher: Optional[str] = None, auth_required: Optional[List[str]] = None, parameters: Optional[List[Dict[str, Any]]] = None, **kwargs):
        super().__init__(name, kind, description, **kwargs) # Pass kwargs to parent
        if kind != TOOL_KIND:
            raise ValueError(f"Kind mismatch for Neo4jCypherConfig. Expected '{TOOL_KIND}', got '{kind}'")
        self.source_name = source_name
        self.default_cypher = default_cypher
        self.auth_required = auth_required if auth_required is not None else []
        self.parameters_config = parameters if parameters is not None else []
        # self.extra_kwargs are handled by super().__init__

    def tool_config_kind(self) -> str:
        return TOOL_KIND

    def initialize(self, sources: Mapping[str, Source]) -> Tool:
        logger.info(f"Initializing Neo4j Cypher tool: {self.name} using source '{self.source_name}'")
        source_instance = sources.get(self.source_name)
        if not source_instance:
            raise ValueError(f"Source '{self.source_name}' not found for tool '{self.name}'. Available: {list(sources.keys())}")
        if not isinstance(source_instance, Neo4jSource):
            raise ValueError(f"Source '{self.source_name}' for tool '{self.name}' is not a Neo4jSource. Actual type: {type(source_instance).__name__}")
        return Neo4jCypherTool(self, source_instance)

    @classmethod
    def from_dict(cls, name: str, data: Dict[str, Any]) -> 'Neo4jCypherConfig':
        required_fields = ['source', 'description']
        for field in required_fields:
            if field not in data:
                raise ValueError(f"Neo4j Cypher tool config for '{name}' missing field: '{field}'")

        current_fields = ['name', 'kind', 'description', 'source', 'cypher', 'statement', 'authRequired', 'parameters']
        extra_kwargs = {k: v for k, v in data.items() if k not in current_fields}

        return cls(
            name=name,
            kind=data.get("kind", TOOL_KIND),
            description=data['description'],
            source_name=data['source'],
            default_cypher=data.get('cypher', data.get('statement')), # Allow 'cypher' or 'statement' for default query
            auth_required=data.get('authRequired'),
            parameters=data.get('parameters'),
            **extra_kwargs
        )

class Neo4jCypherTool(Tool):
    def __init__(self, config: Neo4jCypherConfig, source: Neo4jSource):
        super().__init__(config.name, config.tool_config_kind())
        self.config = config
        self.source = source # Instance of Neo4jSource
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

    def _execute_read_transaction_work(self, tx, cypher, query_params):
        result = tx.run(cypher, query_params)
        return [record.data() for record in result] # Convert Records to list of dicts

    def _execute_write_transaction_work(self, tx, cypher, query_params):
        result_summary = tx.run(cypher, query_params).consume() # Consume the result to get summary
        # Convert ResultSummary properties to a more serializable dict
        return {
            "counters": dict(result_summary.counters),
            "query_type": result_summary.query_type,
            "database": result_summary.database if result_summary.database else "default", # Ensure database is a string
            "notifications": result_summary.notifications if result_summary.notifications else [],
            "plan": result_summary.plan if result_summary.plan else None, # Plan can be None
            "profile": result_summary.profile if result_summary.profile else None, # Profile can be None
            "result_available_after": result_summary.result_available_after,
            "result_consumed_after": result_summary.result_consumed_after,
        }

    def invoke(self, params: Dict[str, Any]) -> Any: # Return type can be List[Dict] or Dict
        cypher_query = params.get("cypher", self.config.default_cypher)
        if not cypher_query:
            raise ValueError("No Cypher query provided for Neo4j tool.")

        query_params = params.get("params", params.get("args", {}))
        if not isinstance(query_params, dict):
            raise ValueError("Parameters for Neo4j Cypher query must be a dictionary (key-value object).")

        transaction_type = params.get("transaction_type", "read").lower()
        # Optional session configuration parameters from tool params
        session_database = params.get("session_database") # Allow overriding the source's default DB for this call
        session_kwargs = params.get("session_kwargs", {})
        if session_database:
            session_kwargs['database'] = session_database

        db_for_log = session_database if session_database else (self.source.config.database if self.source.config.database else "default")

        logger.info(f"Invoking Neo4j Cypher tool '{self.name}' on source '{self.source.name}' (DB: {db_for_log}) "
                    f"with type '{transaction_type}', query: {cypher_query[:100]}... and params: {query_params}")

        results: Any = []
        try:
            # Use the source's get_session context manager
            with self.source.get_session(**session_kwargs) as session:
                if transaction_type == "write":
                    summary = session.write_transaction(self._execute_write_transaction_work, cypher_query, query_params)
                    results = summary
                elif transaction_type == "read":
                    records = session.read_transaction(self._execute_read_transaction_work, cypher_query, query_params)
                    results = records
                else:
                    raise ValueError(f"Invalid transaction_type: '{transaction_type}'. Must be 'read' or 'write'.")

        except neo4j_exceptions.Neo4jError as e:
            logger.error(f"Neo4j database error in Cypher tool '{self.name}': {e.message} (Code: {e.code})")
            # Consider how to propagate Neo4j-specific error details if needed
            raise ValueError(f"Neo4j Error ({e.code}): {e.message}") from e
        except Exception as e:
            logger.error(f"Unexpected error in Neo4j Cypher tool '{self.name}': {e}", exc_info=True)
            raise

        return results

# Registration function
def register_tool(registry: Any): # Using Any for registry type
    registry.register(TOOL_KIND, Neo4jCypherConfig.from_dict)
    logger.info(f"Neo4j Cypher tool kind '{TOOL_KIND}' registration function called.")
