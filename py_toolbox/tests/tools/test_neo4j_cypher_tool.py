import unittest
from unittest.mock import MagicMock, patch
from neo4j import exceptions as neo4j_exceptions # For Neo4j specific exceptions

try:
    from py_toolbox.internal.tools.neo4j_cypher import Neo4jCypherConfig, Neo4jCypherTool, TOOL_KIND
    from py_toolbox.internal.sources.neo4j_source import Neo4jSource, Neo4jConfig # For spec and type hints
    from py_toolbox.internal.tools.base import Manifest # For type checks
    IMPORT_SUCCESSFUL = True
except ImportError as e:
    print(f"Import error in test_neo4j_cypher_tool.py: {e}. Ensure py_toolbox is in PYTHONPATH.")
    IMPORT_SUCCESSFUL = False
    # Dummy classes for test discovery if imports fail
    class Neo4jCypherConfig: pass
    class Neo4jCypherTool: pass
    TOOL_KIND = "neo4j-cypher"
    class Neo4jSource: pass
    class Neo4jConfig: pass # If Neo4jSource needs it for spec
    class Manifest: pass


@unittest.skipIf(not IMPORT_SUCCESSFUL, "Skipping Neo4j Cypher tool tests due to import error")
class TestNeo4jCypherConfig(unittest.TestCase):
    def test_config_creation_minimal(self):
        data = {
            "kind": TOOL_KIND,
            "description": "Minimal Neo4j Cypher tool",
            "source": "my_neo4j_db_ref"
        }
        config = Neo4jCypherConfig.from_dict("neo_cypher_min", data)
        self.assertEqual(config.name, "neo_cypher_min")
        self.assertEqual(config.kind, TOOL_KIND)
        self.assertEqual(config.description, "Minimal Neo4j Cypher tool")
        self.assertEqual(config.source_name, "my_neo4j_db_ref")
        self.assertIsNone(config.default_cypher)

    def test_config_creation_with_cypher_and_statement_alias(self):
        data_cypher = {"kind": TOOL_KIND, "description": "Tool with cypher", "source": "s1", "cypher": "MATCH (n) RETURN n"}
        config_cypher = Neo4jCypherConfig.from_dict("tool_c", data_cypher)
        self.assertEqual(config_cypher.default_cypher, "MATCH (n) RETURN n")

        data_statement = {"kind": TOOL_KIND, "description": "Tool with statement", "source": "s2", "statement": "CREATE (a)"}
        config_statement = Neo4jCypherConfig.from_dict("tool_s", data_statement)
        self.assertEqual(config_statement.default_cypher, "CREATE (a)")

    def test_config_creation_full(self):
        data = {
            "kind": TOOL_KIND,
            "description": "Full Neo4j tool",
            "source": "neo_prod",
            "cypher": "MATCH (u:User {id: $userId}) RETURN u.name",
            "authRequired": ["admin_role"],
            "parameters": [{"name": "userId", "type": "string", "description": "User ID"}],
            "custom_timeout_ms": 5000 # Example of an extra kwarg
        }
        config = Neo4jCypherConfig.from_dict("neo_cypher_full", data)
        self.assertEqual(config.default_cypher, "MATCH (u:User {id: $userId}) RETURN u.name")
        self.assertEqual(config.auth_required, ["admin_role"])
        self.assertEqual(len(config.parameters_config), 1)
        self.assertTrue(hasattr(config, "custom_timeout_ms"))
        self.assertEqual(getattr(config, "custom_timeout_ms"), 5000)


@unittest.skipIf(not IMPORT_SUCCESSFUL, "Skipping Neo4j Cypher tool tests due to import error")
class TestNeo4jCypherTool(unittest.TestCase):
    def setUp(self):
        # Mock Neo4jSource instance that the tool will use
        self.mock_neo4j_source = MagicMock(spec=Neo4jSource)
        self.mock_neo4j_source.name = "mocked_neo4j_for_cypher_tool"

        # Mock the source's config, as the tool might access it (e.g., for default database name)
        mock_src_config = MagicMock(spec=Neo4jConfig)
        mock_src_config.database = "system" # Example default DB on the source
        self.mock_neo4j_source.config = mock_src_config

        # Mock the session object that the source's get_session() context manager would yield
        self.mock_session = MagicMock() # spec=neo4j.Session if more detailed mocking needed

        # Configure the mocked source's get_session to work as a context manager yielding the mock_session
        self.mock_neo4j_source.get_session.return_value.__enter__.return_value = self.mock_session

        # Tool Configuration (using direct instantiation)
        tool_config_data = {
            "description": "Cypher query tool for Neo4j",
            "source_name": "mocked_neo4j_for_cypher_tool", # Must match the name of the mocked source
            "default_cypher": "MATCH (n:Node) RETURN count(n) AS node_count",
            "parameters_config": [
                {"name": "node_label", "type": "string", "description": "Label of the node to search for"}
            ]
        }
        self.tool_config = Neo4jCypherConfig(name="TestCypherQueryTool", kind=TOOL_KIND, **tool_config_data)

        self.sources_map = { "mocked_neo4j_for_cypher_tool": self.mock_neo4j_source }
        self.tool = self.tool_config.initialize(self.sources_map)

    def test_invoke_read_transaction_success(self):
        # Mock the return value of _execute_read_transaction_work (which is called by session.read_transaction)
        expected_records = [{"node_count": 100}]
        self.mock_session.read_transaction.return_value = expected_records

        params = {
            "cypher": "MATCH (n:$node_label) RETURN count(n) AS node_count",
            "params": {"node_label": "Person"},
            "transaction_type": "read"
        }
        results = self.tool.invoke(params)

        self.mock_neo4j_source.get_session.assert_called_once_with() # Default session args
        # Check that session.read_transaction was called with the tool's internal work function
        # and the correct cypher and params
        self.mock_session.read_transaction.assert_called_once_with(
            self.tool._execute_read_transaction_work,
            params["cypher"],
            params["params"]
        )
        self.assertEqual(results, expected_records)

    def test_invoke_write_transaction_success(self):
        expected_summary = {"counters": {"nodes_created": 1}, "query_type": "write", "database": "system"}
        self.mock_session.write_transaction.return_value = expected_summary

        params = {
            "cypher": "CREATE (p:Person {name: $name})",
            "params": {"name": "Alice"},
            "transaction_type": "write"
        }
        results = self.tool.invoke(params)

        self.mock_neo4j_source.get_session.assert_called_once_with()
        self.mock_session.write_transaction.assert_called_once_with(
            self.tool._execute_write_transaction_work,
            params["cypher"],
            params["params"]
        )
        self.assertEqual(results, expected_summary)

    def test_invoke_uses_default_cypher(self):
        self.mock_session.read_transaction.return_value = [{"node_count": 50}]
        params = {"params": {}, "transaction_type": "read"} # No 'cypher' key, should use default

        self.tool.invoke(params)

        self.mock_session.read_transaction.assert_called_with(
            self.tool._execute_read_transaction_work,
            self.tool_config.default_cypher, # Check default is used
            {}
        )

    def test_invoke_session_database_override(self):
        self.mock_session.read_transaction.return_value = []
        params = {
            "cypher": "MATCH (n) RETURN n",
            "session_database": "customdb", # Override source's default DB
            "transaction_type": "read"
        }
        self.tool.invoke(params)
        self.mock_neo4j_source.get_session.assert_called_once_with(database="customdb")


    def test_invoke_no_cypher_query_error(self):
        self.tool.config.default_cypher = None # Ensure no default
        params = {"params": {}, "transaction_type": "read"}
        with self.assertRaisesRegex(ValueError, "No Cypher query provided"):
            self.tool.invoke(params)

    def test_invoke_invalid_transaction_type_error(self):
        params = {"cypher": "MATCH (n) RETURN n", "transaction_type": "invalid_type"}
        with self.assertRaisesRegex(ValueError, "Invalid transaction_type: 'invalid_type'"):
            self.tool.invoke(params)

    def test_invoke_invalid_params_type_error(self):
        params = {"cypher": "MATCH (n) RETURN n", "params": ["not", "a", "dict"]}
        with self.assertRaisesRegex(ValueError, "Parameters for Neo4j Cypher query must be a dictionary"):
            self.tool.invoke(params)


    def test_invoke_neo4j_db_error_handling(self):
        # Simulate a Neo4jError (e.g., CypherSyntaxError) during transaction
        self.mock_session.read_transaction.side_effect = neo4j_exceptions.CypherSyntaxError("Invalid query", "Neo.ClientError.Statement.SyntaxError")

        params = {"cypher": "MATCH n RETURN nnnn", "transaction_type": "read"}
        with self.assertRaisesRegex(ValueError, "Neo4j Error \(Neo.ClientError.Statement.SyntaxError\): Invalid query"):
            self.tool.invoke(params)

    def test_get_manifest(self):
        manifest = self.tool.get_manifest()
        self.assertIsInstance(manifest, Manifest)
        self.assertEqual(manifest.description, "Cypher query tool for Neo4j")
        self.assertEqual(len(manifest.parameters), 1)
        self.assertEqual(manifest.parameters[0].name, "node_label")


if __name__ == '__main__':
    unittest.main()
