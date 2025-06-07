import unittest
from unittest.mock import patch, MagicMock
import sqlite3
import os

try:
    from py_toolbox.internal.sources.sqlite import SQLiteConfig, SQLiteSource, SOURCE_KIND
    IMPORT_SUCCESSFUL = True
except ImportError as e:
    print(f"Import error in test_sqlite_source.py: {e}. Ensure py_toolbox is in PYTHONPATH.")
    IMPORT_SUCCESSFUL = False
    class SQLiteConfig: pass
    class SQLiteSource: pass
    SOURCE_KIND = "sqlite"

# Use an in-memory database for most tests, or a temporary file
TEST_DB_FILE = ":memory:"
# TEST_DB_FILE = "test_sqlite_temp.db" # For file-based tests to check connect() pre-check

@unittest.skipIf(not IMPORT_SUCCESSFUL, "Skipping SQLite source tests due to import error")
class TestSQLiteConfig(unittest.TestCase):
    def test_config_creation_memory(self):
        data = {"kind": SOURCE_KIND, "database_file": ":memory:"}
        config = SQLiteConfig.from_dict("sqlite_mem", data)
        self.assertEqual(config.name, "sqlite_mem")
        self.assertEqual(config.database_file, ":memory:")
        self.assertEqual(config.source_config_kind(), SOURCE_KIND)

    def test_config_creation_file(self):
        data = {"kind": SOURCE_KIND, "database_file": "test.db", "timeout": 10}
        config = SQLiteConfig.from_dict("sqlite_file", data)
        self.assertEqual(config.database_file, "test.db")
        self.assertTrue(hasattr(config, "timeout"))
        self.assertEqual(getattr(config, "timeout"), 10)


    def test_config_alias_database(self):
        data = {"kind": SOURCE_KIND, "database": "aliased.db"} # Using 'database' alias
        config = SQLiteConfig.from_dict("sqlite_alias_test", data)
        self.assertEqual(config.database_file, "aliased.db")

    def test_config_missing_db_file(self):
        data = {"kind": SOURCE_KIND} # Missing database_file or database
        with self.assertRaisesRegex(ValueError, "missing required field: 'database_file' or 'database'"):
            SQLiteConfig.from_dict("sqlite_fail_test", data)

    def test_config_empty_db_file(self):
        data = {"kind": SOURCE_KIND, "database_file": ""}
        # The constructor of SQLiteConfig should raise this, not from_dict directly if check is there
        with self.assertRaisesRegex(ValueError, "requires 'database_file' path."):
             SQLiteConfig(name="test", kind=SOURCE_KIND, database_file="", timeout=5)


@unittest.skipIf(not IMPORT_SUCCESSFUL, "Skipping SQLite source tests due to import error")
class TestSQLiteSource(unittest.TestCase):
    def setUp(self):
        # Config for an in-memory database for most tests
        self.mem_config = SQLiteConfig.from_dict("sqlite_mem_src_test", {"database_file": ":memory:"})
        # Config for a file-based database for specific tests like connect() pre-check
        self.file_db_path = "temp_test_sqlite.db"
        self.file_config = SQLiteConfig.from_dict("sqlite_file_src_test", {"database_file": self.file_db_path, "timeout": 10})

        # Clean up any previous file DB
        if os.path.exists(self.file_db_path):
            os.remove(self.file_db_path)

    def tearDown(self):
        # Clean up file DB after tests
        if os.path.exists(self.file_db_path):
            os.remove(self.file_db_path)

    def test_source_connect_call_accessibility_check_file_db(self):
        source = SQLiteSource(self.file_config)
        # sqlite3.connect will create the file if it doesn't exist, so this test is more about path writability
        # For a more robust check, one might try creating a read-only file or an invalid path.
        with patch('sqlite3.connect') as mock_sqlite_connect:
            mock_conn = MagicMock(spec=sqlite3.Connection)
            mock_sqlite_connect.return_value = mock_conn

            source.connect() # This should attempt the test connection for file DBs

            mock_sqlite_connect.assert_called_once_with(self.file_db_path) # Minimal connect for pre-check
            mock_conn.close.assert_called_once()


    def test_source_connect_call_accessibility_check_memory_db(self):
        source = SQLiteSource(self.mem_config) # In-memory DB
        with patch('sqlite3.connect') as mock_sqlite_connect:
            source.connect()
            # For in-memory, the pre-check sqlite3.connect should not be called
            mock_sqlite_connect.assert_not_called()


    @patch('sqlite3.connect', side_effect=sqlite3.Error("Simulated DB access error"))
    def test_source_connect_accessibility_check_failure(self, mock_sqlite_connect_fail):
        source = SQLiteSource(self.file_config) # Use file config
        with self.assertRaisesRegex(sqlite3.Error, "Simulated DB access error"):
            source.connect()
        mock_sqlite_connect_fail.assert_called_once_with(self.file_db_path)


    def test_get_connection_success_memory(self):
        source = SQLiteSource(self.mem_config)
        with source.get_connection() as conn:
            self.assertIsInstance(conn, sqlite3.Connection)
            self.assertEqual(conn.row_factory, sqlite3.Row)
            # Perform a simple operation
            cursor = conn.cursor()
            cursor.execute("CREATE TABLE test_table (id INTEGER);")
            conn.commit()
            cursor.execute("INSERT INTO test_table (id) VALUES (1);")
            conn.commit()
            row = cursor.execute("SELECT id FROM test_table;").fetchone()
            self.assertEqual(row['id'], 1)
        # Connection should be closed by context manager implicitly

    def test_get_connection_success_file_with_extra_params(self):
        source = SQLiteSource(self.file_config) # self.file_config has timeout=10

        # We want to verify that sqlite3.connect is called with the extra_params
        with patch('sqlite3.connect') as mock_actual_connect:
            mock_conn_instance = MagicMock(spec=sqlite3.Connection)
            mock_conn_instance.row_factory = None # Simulate it's not set yet
            mock_actual_connect.return_value = mock_conn_instance

            with source.get_connection() as conn:
                self.assertEqual(conn, mock_conn_instance)
                # Check that sqlite3.connect was called with the database file and extra_kwargs
                # The extra_kwargs are extracted inside get_connection from self.config
                expected_extra_kwargs = {'timeout': 10}
                # Add other expected default extra_kwargs if SQLiteConfig sets them by default
                mock_actual_connect.assert_called_with(self.file_db_path, **expected_extra_kwargs)
                # Verify row_factory was set
                self.assertEqual(conn.row_factory, sqlite3.Row)


    @patch('sqlite3.connect', side_effect=sqlite3.Error("Failed to open DB"))
    def test_get_connection_failure(self, mock_sqlite_connect):
        source = SQLiteSource(self.mem_config) # Use in-memory for this, path doesn't matter with mock
        with self.assertRaisesRegex(sqlite3.Error, "Failed to open DB"):
            with source.get_connection():
                pass # Should not reach here

        # Check that sqlite3.connect was called with the database file and any extra_kwargs from config
        expected_extra_kwargs = {k: getattr(self.mem_config, k) for k in self.mem_config.__dict__.keys()
                                 if k not in ['name', 'kind', 'database_file']}
        mock_sqlite_connect.assert_called_once_with(self.mem_config.database_file, **expected_extra_kwargs)

    def test_close_no_persistent_connection(self):
        source = SQLiteSource(self.mem_config)
        source.close()
        self.assertIsNone(source._connection) # No persistent _connection is used by default

if __name__ == '__main__':
    unittest.main()
