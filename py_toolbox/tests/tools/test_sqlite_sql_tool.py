import unittest
from unittest.mock import MagicMock, patch
import sqlite3

try:
    from py_toolbox.internal.tools.sqlite_sql import SQLiteSQLConfig, SQLiteSQLTool, TOOL_KIND
    from py_toolbox.internal.sources.sqlite import SQLiteSource, SQLiteConfig # For spec and type hints
    from py_toolbox.internal.tools.base import Manifest # For type checks
    IMPORT_SUCCESSFUL = True
except ImportError as e:
    print(f"Import error in test_sqlite_sql_tool.py: {e}. Ensure py_toolbox is in PYTHONPATH.")
    IMPORT_SUCCESSFUL = False
    class SQLiteSQLConfig: pass
    class SQLiteSQLTool: pass
    TOOL_KIND = "sqlite-sql"
    class SQLiteSource: pass
    class SQLiteConfig: pass
    class Manifest: pass


TEST_DB_FILE_TOOL = ":memory:" # Use in-memory for tool tests

@unittest.skipIf(not IMPORT_SUCCESSFUL, "Skipping SQLite tool tests due to import error")
class TestSQLiteSQLConfig(unittest.TestCase):
    def test_config_creation(self):
        data = {
            "kind": TOOL_KIND,
            "description": "Test SQLite Tool for querying",
            "source": "my_sqlite_source_ref"
        }
        config = SQLiteSQLConfig.from_dict("sqlite_tool_test_cfg", data)
        self.assertEqual(config.name, "sqlite_tool_test_cfg")
        self.assertEqual(config.kind, TOOL_KIND)
        self.assertEqual(config.description, "Test SQLite Tool for querying")
        self.assertEqual(config.source_name, "my_sqlite_source_ref")

    def test_config_full_options(self):
        data = {
            "kind": TOOL_KIND,
            "description": "Full SQLite Tool",
            "source": "db1",
            "statement": "SELECT * FROM logs;",
            "authRequired": ["local_user"],
            "parameters": [{"name": "level", "type": "string", "description": "Log level"}],
            "custom_timeout": 3000 # Extra kwarg
        }
        config = SQLiteSQLConfig.from_dict("sqlite_tool_full_cfg", data)
        self.assertEqual(config.default_statement, "SELECT * FROM logs;")
        self.assertEqual(config.auth_required, ["local_user"])
        self.assertEqual(len(config.parameters_config), 1)
        self.assertTrue(hasattr(config, "custom_timeout"))
        self.assertEqual(getattr(config, "custom_timeout"), 3000)


@unittest.skipIf(not IMPORT_SUCCESSFUL, "Skipping SQLite tool tests due to import error")
class TestSQLiteSQLTool(unittest.TestCase):
    def setUp(self):
        # Mock SQLiteSource that the tool will use
        self.mock_sqlite_source = MagicMock(spec=SQLiteSource)
        self.mock_sqlite_source.name = "mocked_sqlite_for_tool"

        # Mock the source's config attribute, as the tool uses it for logging the db file path
        mock_src_config = MagicMock(spec=SQLiteConfig)
        mock_src_config.database_file = TEST_DB_FILE_TOOL
        self.mock_sqlite_source.config = mock_src_config

        # Setup a real in-memory SQLite connection that the mocked source's get_connection will yield
        self.real_conn = sqlite3.connect(TEST_DB_FILE_TOOL)
        self.real_conn.row_factory = sqlite3.Row # Tool expects dict-like rows

        # Configure the mocked source's get_connection to work as a context manager yielding the real connection
        self.mock_sqlite_source.get_connection.return_value.__enter__.return_value = self.real_conn

        # Pre-populate the in-memory database with some test data
        with self.real_conn as conn:
            conn.execute("CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT, email TEXT);")
            conn.execute("INSERT INTO users (name, email) VALUES ('Alice', 'alice@example.com'), ('Bob', 'bob@example.com');")

        # Tool Configuration
        tool_config_data = {
            "description": "User query tool for SQLite",
            "source_name": "mocked_sqlite_for_tool",
            "parameters_config": [
                {"name": "user_id", "type": "integer", "description": "ID of the user", "required": False}
            ]
        }
        # Use direct instantiation for the config object passed to the tool
        self.tool_config = SQLiteSQLConfig(name="QueryUsersTool", kind=TOOL_KIND, **tool_config_data)

        self.sources_map = { "mocked_sqlite_for_tool": self.mock_sqlite_source }
        self.tool = self.tool_config.initialize(self.sources_map)

    def tearDown(self):
        # Close the real in-memory SQLite connection
        self.real_conn.close()

    def test_invoke_select_all_users(self):
        params = {"statement": "SELECT id, name, email FROM users;"}
        results = self.tool.invoke(params)

        self.assertEqual(len(results), 2)
        self.assertIsInstance(results[0], dict) # Check conversion from sqlite3.Row
        self.assertEqual(results[0]['name'], 'Alice')
        self.assertEqual(results[1]['name'], 'Bob')
        self.mock_sqlite_source.get_connection.assert_called_once()

    def test_invoke_select_with_args(self):
        params = {"statement": "SELECT name, email FROM users WHERE name = ?;", "args": ["Bob"]}
        results = self.tool.invoke(params)

        self.assertEqual(len(results), 1)
        self.assertEqual(results[0]['name'], 'Bob')
        self.assertEqual(results[0]['email'], 'bob@example.com')

    def test_invoke_insert_user(self):
        params = {"statement": "INSERT INTO users (name, email) VALUES (?, ?);", "args": ["Charlie", "charlie@example.com"]}
        results = self.tool.invoke(params)

        self.assertEqual(results[0]['status'], 'success')
        self.assertIsNotNone(results[0]['lastrowid'])
        new_user_id = results[0]['lastrowid']

        # Verify insertion using the real connection
        with self.real_conn as conn:
            user = conn.execute("SELECT name FROM users WHERE id = ?", (new_user_id,)).fetchone()
            self.assertEqual(user['name'], 'Charlie')

    def test_invoke_update_user(self):
        params_insert = {"statement": "INSERT INTO users (name, email) VALUES (?, ?);", "args": ["David", "david_old@example.com"]}
        insert_result = self.tool.invoke(params_insert)
        david_id = insert_result[0]['lastrowid']

        params_update = {"statement": "UPDATE users SET email = ? WHERE id = ?;", "args": ["david_new@example.com", david_id]}
        update_result = self.tool.invoke(params_update)

        self.assertEqual(update_result[0]['status'], 'success')
        self.assertEqual(update_result[0]['rowcount'], 1)

        with self.real_conn as conn:
            user = conn.execute("SELECT email FROM users WHERE id = ?", (david_id,)).fetchone()
            self.assertEqual(user['email'], 'david_new@example.com')


    def test_invoke_delete_user(self):
        params_insert = {"statement": "INSERT INTO users (name, email) VALUES (?, ?);", "args": ["Eve", "eve@example.com"]}
        insert_result = self.tool.invoke(params_insert)
        eve_id = insert_result[0]['lastrowid']

        params_delete = {"statement": "DELETE FROM users WHERE id = ?;", "args": [eve_id]}
        delete_result = self.tool.invoke(params_delete)
        self.assertEqual(delete_result[0]['status'], 'success')
        self.assertEqual(delete_result[0]['rowcount'], 1)

        with self.real_conn as conn:
            user = conn.execute("SELECT * FROM users WHERE id = ?", (eve_id,)).fetchone()
            self.assertIsNone(user)


    def test_invoke_statement_syntax_error(self):
        params = {"statement": "SELECT FROM WHERE bad_syntax;"} # Invalid SQL
        with self.assertRaises(sqlite3.Error): # Expecting an sqlite3.Error (OperationalError)
            self.tool.invoke(params)

    def test_invoke_no_statement_error(self):
        self.tool.config.default_statement = None # Ensure no default
        params = {} # No statement provided
        with self.assertRaisesRegex(ValueError, "No SQL statement provided for SQLite tool."):
            self.tool.invoke(params)

    def test_get_manifest(self):
        manifest = self.tool.get_manifest()
        self.assertIsInstance(manifest, Manifest)
        self.assertEqual(manifest.description, "User query tool for SQLite")
        self.assertEqual(len(manifest.parameters), 1)
        self.assertEqual(manifest.parameters[0].name, "user_id")


if __name__ == '__main__':
    unittest.main()
