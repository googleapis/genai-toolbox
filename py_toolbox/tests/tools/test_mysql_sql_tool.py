import unittest
from unittest.mock import MagicMock, patch
import mysql.connector # For exception types

try:
    from py_toolbox.internal.tools.mysql_sql import MySQLSQLConfig, MySQLSQLTool, TOOL_KIND
    from py_toolbox.internal.sources.mysql import MySQLSource # Actual source for isinstance checks
    from py_toolbox.internal.tools.base import Manifest, McpManifest, ParameterManifest
    IMPORT_SUCCESSFUL = True
except ImportError as e:
    print(f"Import error in test_mysql_sql_tool.py: {e}. Ensure py_toolbox is in PYTHONPATH.")
    IMPORT_SUCCESSFUL = False
    # Dummy classes for test discovery
    class MySQLSQLConfig: pass
    class MySQLSQLTool: pass
    TOOL_KIND = "mysql-sql"
    class MySQLSource: pass
    class Manifest: pass
    class McpManifest: pass
    class ParameterManifest: pass


@unittest.skipIf(not IMPORT_SUCCESSFUL, "Skipping MySQL tool tests due to import error")
class TestMySQLSQLConfig(unittest.TestCase):
    def test_config_creation_minimal(self):
        data = {
            "kind": TOOL_KIND,
            "description": "Test MySQL Tool",
            "source": "my_mysql_source_ref"
        }
        config = MySQLSQLConfig.from_dict("mysql_tool_min", data)
        self.assertEqual(config.name, "mysql_tool_min")
        self.assertEqual(config.kind, TOOL_KIND)
        self.assertEqual(config.description, "Test MySQL Tool")
        self.assertEqual(config.source_name, "my_mysql_source_ref")
        self.assertIsNone(config.default_statement)
        self.assertEqual(config.auth_required, [])
        self.assertEqual(config.parameters_config, [])

    def test_config_creation_full(self):
        data = {
            "kind": TOOL_KIND,
            "description": "Full MySQL Tool with all options",
            "source": "prod_mysql_source",
            "statement": "SELECT * FROM customers WHERE email = %s;",
            "authRequired": ["customer_service_auth"],
            "parameters": [
                {"name": "email", "type": "string", "description": "Customer email", "required": True}
            ],
            "custom_option": "enabled" # Example of an extra kwarg
        }
        config = MySQLSQLConfig.from_dict("mysql_tool_options", data)
        self.assertEqual(config.description, "Full MySQL Tool with all options")
        self.assertEqual(config.source_name, "prod_mysql_source")
        self.assertEqual(config.default_statement, "SELECT * FROM customers WHERE email = %s;")
        self.assertEqual(config.auth_required, ["customer_service_auth"])
        self.assertEqual(len(config.parameters_config), 1)
        self.assertEqual(config.parameters_config[0]['name'], "email")
        self.assertTrue(hasattr(config, "custom_option"))
        self.assertEqual(getattr(config, "custom_option"), "enabled")

    def test_config_missing_required_fields(self):
        with self.assertRaisesRegex(ValueError, "missing required field: 'source'"):
            MySQLSQLConfig.from_dict("test_fail_src", {"description": "desc only"})
        with self.assertRaisesRegex(ValueError, "missing required field: 'description'"):
            MySQLSQLConfig.from_dict("test_fail_desc", {"source": "src_only"})


@unittest.skipIf(not IMPORT_SUCCESSFUL, "Skipping MySQL tool tests due to import error")
class TestMySQLSQLTool(unittest.TestCase):
    def setUp(self):
        # Mock MySQLSource instance
        self.mock_mysql_source = MagicMock(spec=MySQLSource)
        self.mock_mysql_source.name = "mock_mysql_db_instance"
        # Simulate that the source's pool is initialized (or connect will be called)
        self.mock_mysql_source._pool = MagicMock(spec=mysql.connector.pooling.MySQLConnectionPool)

        # Mock connection and cursor that would be returned by the source
        self.mock_conn = MagicMock(spec=mysql.connector.connection.MySQLConnection)
        self.mock_conn.is_connected.return_value = True # Important for finally block in source
        self.mock_cursor = MagicMock(spec=mysql.connector.cursor.MySQLCursorDict)

        # Configure the source's get_connection context manager
        self.mock_mysql_source.get_connection.return_value.__enter__.return_value = self.mock_conn
        self.mock_conn.cursor.return_value.__enter__.return_value = self.mock_cursor

        # Tool Configuration (using direct instantiation for clarity in what config object holds)
        self.tool_config_obj = MySQLSQLConfig(
            name="TestMySQLToolInstance",
            kind=TOOL_KIND,
            description="A MySQL tool for testing",
            source_name="mock_mysql_db_instance", # Matches the name of the mocked source
            default_statement="SELECT data FROM my_table WHERE key = %s;",
            parameters=[ # Note: in class this is parameters_config, from_dict handles 'parameters'
                {"name": "key_param", "type": "string", "description": "The key for lookup", "required": True}
            ],
            auth_required=["user_access_token"]
        )

        self.sources_map = { "mock_mysql_db_instance": self.mock_mysql_source }
        # Initialize the tool with the config object and mocked sources map
        self.tool = self.tool_config_obj.initialize(self.sources_map)

    def test_tool_initialization(self):
        self.assertIsInstance(self.tool, MySQLSQLTool)
        self.assertEqual(self.tool.source, self.mock_mysql_source)
        self.assertEqual(self.tool.config.name, "TestMySQLToolInstance")
        self.assertEqual(self.tool.tool_kind(), TOOL_KIND)

    def test_tool_initialization_source_not_found(self):
        broken_sources_map = {"wrong_source": self.mock_mysql_source}
        with self.assertRaisesRegex(ValueError, "Source 'mock_mysql_db_instance' not found"):
            self.tool_config_obj.initialize(broken_sources_map)

    def test_tool_initialization_source_wrong_type(self):
        wrong_source = MagicMock(spec=object) # Not a MySQLSource
        wrong_type_map = {"mock_mysql_db_instance": wrong_source}
        with self.assertRaisesRegex(ValueError, "is not a MySQLSource"):
            self.tool_config_obj.initialize(wrong_type_map)


    def test_invoke_success_select_with_params(self):
        self.mock_cursor.fetchall.return_value = [{"data": "value1"}]
        self.mock_cursor.description = True # Indicates SELECT

        params = {"statement": "SELECT data FROM my_table WHERE key = %s;", "args": ["test_key"]}
        result = self.tool.invoke(params)

        self.mock_cursor.execute.assert_called_once_with("SELECT data FROM my_table WHERE key = %s;", ("test_key",))
        self.mock_conn.commit.assert_called_once() # Commit is called even for SELECT in current impl
        self.assertEqual(result, [{"data": "value1"}])

    def test_invoke_success_select_default_statement(self):
        self.mock_cursor.fetchall.return_value = [{"data": "default_value"}]
        self.mock_cursor.description = True

        params = {"args": ["default_key"]} # Uses default_statement
        result = self.tool.invoke(params)

        self.mock_cursor.execute.assert_called_once_with(self.tool_config_obj.default_statement, ("default_key",))
        self.assertEqual(result, [{"data": "default_value"}])


    def test_invoke_success_dml_no_description(self):
        self.mock_cursor.description = None # Indicates DML/DDL
        self.mock_cursor.rowcount = 1
        self.mock_cursor.lastrowid = 123

        params = {"statement": "INSERT INTO logs (message) VALUES (%s);", "args": ["log message"]}
        result = self.tool.invoke(params)

        self.mock_cursor.execute.assert_called_once_with("INSERT INTO logs (message) VALUES (%s);", ("log message",))
        self.mock_conn.commit.assert_called_once()
        self.assertEqual(result, [{"status": "success", "rowcount": 1, "lastrowid": 123}])

    @patch.object(MySQLSource, 'connect') # Patch connect on the class
    def test_invoke_connects_if_source_pool_is_none(self, mock_source_connect_method):
        self.tool.source._pool = None # Simulate source not connected (pool is None)
        # Re-assign the mocked method to the instance for this test to ensure it's the one called
        self.tool.source.connect = mock_source_connect_method

        self.mock_cursor.fetchall.return_value = [{"id": 1}] # Dummy result
        self.mock_cursor.description = True

        params = {"statement": "SELECT 1;", "args": []}
        self.tool.invoke(params)

        mock_source_connect_method.assert_called_once() # Source.connect() should have been called by tool

    def test_invoke_db_execution_error_triggers_rollback(self):
        self.mock_cursor.execute.side_effect = mysql.connector.Error("Error during query execution")

        params = {"statement": "SELECT * FROM non_existent_table;", "args": []}
        with self.assertRaisesRegex(mysql.connector.Error, "Error during query execution"):
            self.tool.invoke(params)

        self.mock_conn.rollback.assert_called_once()
        self.mock_conn.commit.assert_not_called()


    def test_invoke_no_statement_provided_error(self):
        self.tool.config.default_statement = None # Ensure no default statement
        params = {"args": ["some_arg"]} # No 'statement' key
        with self.assertRaisesRegex(ValueError, "No SQL statement provided"):
            self.tool.invoke(params)

    def test_get_manifest_generation(self):
        manifest = self.tool.get_manifest()
        self.assertIsInstance(manifest, Manifest)
        self.assertEqual(manifest.description, self.tool_config_obj.description)
        self.assertEqual(len(manifest.parameters), 1)
        param_manifest = manifest.parameters[0]
        self.assertIsInstance(param_manifest, ParameterManifest)
        self.assertEqual(param_manifest.name, "key_param")
        self.assertEqual(param_manifest.type, "string")
        self.assertTrue(param_manifest.required)
        self.assertEqual(manifest.auth_required, ["user_access_token"])

    def test_get_mcp_manifest_generation(self):
        mcp_manifest = self.tool.get_mcp_manifest()
        self.assertIsInstance(mcp_manifest, McpManifest)
        self.assertEqual(mcp_manifest.name, self.tool_config_obj.name)
        self.assertEqual(mcp_manifest.description, self.tool_config_obj.description)
        self.assertIn("key_param", mcp_manifest.input_schema['properties'])
        self.assertIn("key_param", mcp_manifest.input_schema['required'])

    def test_is_authorized_logic(self):
        self.assertTrue(self.tool.is_authorized(["user_access_token"]))
        self.assertTrue(self.tool.is_authorized(["user_access_token", "another_service"]))
        self.assertFalse(self.tool.is_authorized(["unrelated_service"]))
        self.assertFalse(self.tool.is_authorized([])) # Requires "user_access_token"

        # Test with no auth required in config
        self.tool.config.auth_required = [] # Modify config for this test case
        self.assertTrue(self.tool.is_authorized([]))
        self.assertTrue(self.tool.is_authorized(["any_service"]))


if __name__ == '__main__':
    unittest.main()
