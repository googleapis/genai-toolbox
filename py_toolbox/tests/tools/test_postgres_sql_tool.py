import unittest
from unittest.mock import MagicMock, patch
import psycopg2 # For errors

# Attempt to import the items to be tested
try:
    from py_toolbox.internal.tools.postgres_sql import PostgresSQLConfig, PostgresSQLTool, TOOL_KIND
    from py_toolbox.internal.sources.postgres import PostgresSource # Actual PostgresSource for isinstance checks
    from py_toolbox.internal.sources.postgres import PostgresConfig as PostgresSourceConfig # Renamed for clarity
    from py_toolbox.internal.tools.base import Manifest, McpManifest, ParameterManifest
    from py_toolbox.internal.core.logging import get_logger # for logger testing
    IMPORT_ERROR = False
except ImportError as e:
    print(f"Import error in test_postgres_sql_tool.py: {e}")
    IMPORT_ERROR = True
    # Dummy classes if import fails
    class PostgresSQLConfig: pass
    class PostgresSQLTool: pass
    TOOL_KIND = "postgres-sql"
    class PostgresSource: pass # This will be a dummy, so isinstance checks might behave unexpectedly
    class PostgresSourceConfig: pass
    class Manifest: pass
    class McpManifest: pass
    class ParameterManifest: pass

@unittest.skipIf(IMPORT_ERROR, "Skipping tests due to import error")
class TestPostgresSQLConfig(unittest.TestCase):
    def test_config_creation_minimal(self):
        data = {
            "kind": TOOL_KIND,
            "description": "Test PG SQL Tool",
            "source": "my_pg_source", # Name of the source instance
        }
        config = PostgresSQLConfig.from_dict("pg_sql_test", data)
        self.assertEqual(config.name, "pg_sql_test")
        self.assertEqual(config.tool_config_kind(), TOOL_KIND)
        self.assertEqual(config.description, "Test PG SQL Tool")
        self.assertEqual(config.source_name, "my_pg_source")
        self.assertIsNone(config.default_statement)
        self.assertEqual(config.auth_required, [])
        self.assertEqual(config.parameters_config, [])

    def test_config_creation_full(self):
        data = {
            "kind": TOOL_KIND,
            "description": "Detailed PG SQL Tool",
            "source": "another_pg_source",
            "statement": "SELECT * FROM users WHERE id = %s;",
            "authRequired": ["user_auth"],
            "parameters": [
                {"name": "id", "type": "integer", "description": "User ID", "required": True}
            ],
            "custom_field": "custom_value" # Extra field
        }
        config = PostgresSQLConfig.from_dict("pg_sql_detailed", data)
        self.assertEqual(config.description, "Detailed PG SQL Tool")
        self.assertEqual(config.source_name, "another_pg_source")
        self.assertEqual(config.default_statement, "SELECT * FROM users WHERE id = %s;")
        self.assertEqual(config.auth_required, ["user_auth"])
        self.assertEqual(len(config.parameters_config), 1)
        self.assertEqual(config.parameters_config[0]['name'], "id")
        self.assertTrue(hasattr(config, "custom_field"))
        self.assertEqual(getattr(config, "custom_field"), "custom_value")


    def test_config_missing_required_fields(self):
        with self.assertRaisesRegex(ValueError, "missing required field: 'source'"):
            PostgresSQLConfig.from_dict("test_fail_source", {"description": "desc"})
        with self.assertRaisesRegex(ValueError, "missing required field: 'description'"):
            PostgresSQLConfig.from_dict("test_fail_desc", {"source": "src"})

    def test_config_kind_mismatch_in_constructor(self):
        with self.assertRaisesRegex(ValueError, "Kind mismatch for PostgresSQLConfig"):
            PostgresSQLConfig(name="test", kind="wrong_kind", description="d", source_name="s")


@unittest.skipIf(IMPORT_ERROR, "Skipping tests due to import error")
class TestPostgresSQLTool(unittest.TestCase):
    def setUp(self):
        # Mock PostgresSourceConfig
        self.mock_pg_source_config = MagicMock(spec=PostgresSourceConfig)
        self.mock_pg_source_config.name = "test_pg_db_config"

        # Mock PostgresSource - crucial to use spec=PostgresSource for isinstance checks
        self.mock_pg_source = MagicMock(spec=PostgresSource)
        self.mock_pg_source.name = "test_pg_db"
        self.mock_pg_source.config = self.mock_pg_source_config
        self.mock_pg_source._pool = MagicMock() # Assume pool exists and is valid initially

        # Mock connection and cursor from the source
        self.mock_conn = MagicMock()
        self.mock_cursor = MagicMock()
        # Configure the context manager behavior for get_connection
        self.mock_pg_source.get_connection.return_value.__enter__.return_value = self.mock_conn
        self.mock_conn.cursor.return_value.__enter__.return_value = self.mock_cursor

        # Basic Tool Config
        self.tool_config_dict = {
            "description": "A test SQL tool",
            "source_name": "test_pg_db", # Matches mock_pg_source.name
            "default_statement": "SELECT * FROM test_table WHERE id = %s;",
            "parameters": [
                {"name": "id", "type": "integer", "description": "ID of the record", "required": True}
            ],
            "auth_required": ["service_A"]
        }
        # Use the actual PostgresSQLConfig for the tool's config
        self.tool_config = PostgresSQLConfig(name="TestSQLTool", kind=TOOL_KIND, **self.tool_config_dict)

        # Sources map for initialization
        self.sources_map = { "test_pg_db": self.mock_pg_source }
        # Initialize the tool
        self.tool = self.tool_config.initialize(self.sources_map)

    def test_tool_initialization(self):
        self.assertIsInstance(self.tool, PostgresSQLTool)
        self.assertEqual(self.tool.source, self.mock_pg_source)
        self.assertEqual(self.tool.tool_kind(), TOOL_KIND)
        self.assertEqual(self.tool.config.name, "TestSQLTool")

    def test_tool_initialization_source_not_found(self):
        broken_sources_map = {"wrong_source_name": self.mock_pg_source}
        with self.assertRaisesRegex(ValueError, "Source 'test_pg_db' not found"):
            self.tool_config.initialize(broken_sources_map)

    def test_tool_initialization_source_wrong_type(self):
        wrong_type_source = MagicMock(spec=object) # Not a PostgresSource
        wrong_type_sources_map = {"test_pg_db": wrong_type_source}
        with self.assertRaisesRegex(ValueError, "is not a PostgresSource"):
            self.tool_config.initialize(wrong_type_sources_map)

    def test_invoke_success_select_statement(self):
        self.mock_cursor.fetchall.return_value = [{"id": 1, "name": "Test Name"}]
        self.mock_cursor.description = True # Indicates a SELECT query with columns

        params = {"statement": "SELECT * FROM example WHERE id = %s;", "args": [1]}
        result = self.tool.invoke(params)

        self.mock_cursor.execute.assert_called_once_with("SELECT * FROM example WHERE id = %s;", (1,))
        self.mock_conn.commit.assert_called_once() # Commit should be called
        self.assertEqual(result, [{"id": 1, "name": "Test Name"}])

    def test_invoke_success_select_default_statement(self):
        self.mock_cursor.fetchall.return_value = [{"id": 1, "name": "Test Name"}]
        self.mock_cursor.description = True

        params = {"args": [1]} # Uses default_statement from config
        result = self.tool.invoke(params)

        self.mock_cursor.execute.assert_called_once_with(self.tool_config.default_statement, (1,))
        self.assertEqual(result, [{"id": 1, "name": "Test Name"}])


    def test_invoke_success_dml_statement(self):
        self.mock_cursor.description = None # Indicates DML/DDL, no rows to fetch
        self.mock_cursor.rowcount = 1 # e.g., 1 row updated

        params = {"statement": "UPDATE test_table SET name = %s WHERE id = %s;", "args": ["NewName", 1]}
        result = self.tool.invoke(params)

        self.mock_cursor.execute.assert_called_once_with("UPDATE test_table SET name = %s WHERE id = %s;", ("NewName", 1))
        self.mock_conn.commit.assert_called_once()
        self.assertEqual(result, [{"status": "success", "rowcount": 1}])

    @patch.object(PostgresSource, 'connect', wraps=MagicMock(side_effect=Exception("Deliberate connection failure during invoke")))
    def test_invoke_connects_if_not_connected_failure(self, mock_source_connect_method):
        self.tool.source._pool = None # Simulate source not connected (pool is None)

        # Ensure the mocked source's connect method is what's called
        self.tool.source.connect = mock_source_connect_method

        params = {"statement": "SELECT 1;", "args": []}
        with self.assertRaisesRegex(ConnectionError, "Source 'test_pg_db' not connected: Deliberate connection failure during invoke"):
            self.tool.invoke(params)

        self.tool.source.connect.assert_called_once()


    @patch.object(PostgresSource, 'connect') # More specific patch
    def test_invoke_connects_if_not_connected_success(self, mock_source_connect_method):
        self.tool.source._pool = None # Simulate not connected
        # Re-assign the mocked method to the instance for this test
        self.tool.source.connect = mock_source_connect_method

        self.mock_cursor.fetchall.return_value = [{"id": 1}]
        self.mock_cursor.description = True

        params = {"statement": "SELECT 1;", "args": []}
        self.tool.invoke(params)

        mock_source_connect_method.assert_called_once()


    def test_invoke_db_error_triggers_rollback(self):
        self.mock_cursor.execute.side_effect = psycopg2.Error("DB Error on execute")

        params = {"statement": "SELECT NONSENSE;", "args": []}
        with self.assertRaisesRegex(psycopg2.Error, "DB Error on execute"):
            self.tool.invoke(params)

        self.mock_conn.rollback.assert_called_once()
        self.mock_conn.commit.assert_not_called() # Should not commit on error


    def test_invoke_no_statement_error(self):
        # Clear default statement for this test case
        self.tool.config.default_statement = None
        params = {} # No statement, no args
        with self.assertRaisesRegex(ValueError, "No SQL statement provided"):
            self.tool.invoke(params)

    def test_get_manifest(self):
        manifest = self.tool.get_manifest()
        self.assertIsInstance(manifest, Manifest)
        self.assertEqual(manifest.description, self.tool_config.description)
        self.assertEqual(len(manifest.parameters), 1)
        param = manifest.parameters[0]
        self.assertIsInstance(param, ParameterManifest)
        self.assertEqual(param.name, "id")
        self.assertEqual(param.type, "integer")
        self.assertTrue(param.required)
        self.assertEqual(manifest.auth_required, ["service_A"])

    def test_get_mcp_manifest(self):
        mcp_manifest = self.tool.get_mcp_manifest()
        self.assertIsInstance(mcp_manifest, McpManifest)
        self.assertEqual(mcp_manifest.name, self.tool_config.name)
        self.assertEqual(mcp_manifest.description, self.tool_config.description)
        self.assertIn("id", mcp_manifest.input_schema['properties'])
        self.assertIn("id", mcp_manifest.input_schema['required'])

    def test_is_authorized(self):
        self.assertTrue(self.tool.is_authorized(["service_A"]))
        self.assertTrue(self.tool.is_authorized(["service_A", "service_B"]))
        self.assertFalse(self.tool.is_authorized(["service_C"]))

        # Test with no auth required by tool
        self.tool.config.auth_required = []
        self.assertTrue(self.tool.is_authorized(["service_C"])) # Should be true as no auth is specified
        self.assertTrue(self.tool.is_authorized([])) # Also true

if __name__ == '__main__':
    import sys
    import os
    if not IMPORT_ERROR:
        # sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '../../../')))
        unittest.main()
    else:
        print("Skipping test run due to import errors.")
