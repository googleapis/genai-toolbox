import unittest
from unittest.mock import patch, MagicMock
import psycopg2 # Import for psycopg2.Error

# Attempt to import the items to be tested
try:
    from py_toolbox.internal.sources.postgres import PostgresConfig, PostgresSource, SOURCE_KIND
    from py_toolbox.internal.core.registry import SourceRegistry
    IMPORT_ERROR = False
except ImportError as e:
    # This helps in diagnosing path issues when running tests
    print(f"Import error in test_postgres_source.py: {e}")
    IMPORT_ERROR = True
    # Define dummy classes if import fails, so tests can be discovered / partially run
    class PostgresConfig: pass
    class PostgresSource: pass
    SOURCE_KIND = "postgres"


@unittest.skipIf(IMPORT_ERROR, "Skipping tests due to import error")
class TestPostgresConfig(unittest.TestCase):
    def test_config_creation_minimal(self):
        data = {
            "kind": SOURCE_KIND,
            "host": "localhost",
            "port": 5432,
            "user": "testuser",
            "password": "password",
            "database": "testdb"
        }
        config = PostgresConfig.from_dict("pg_test", data)
        self.assertEqual(config.name, "pg_test")
        self.assertEqual(config.host, "localhost")
        self.assertEqual(config.port, 5432)
        self.assertEqual(config.user, "testuser")
        self.assertEqual(config.password, "password")
        self.assertEqual(config.database, "testdb")
        self.assertEqual(config.source_config_kind(), SOURCE_KIND)
        self.assertEqual(config.pool_min_conn, 1) # Default
        self.assertEqual(config.pool_max_conn, 10) # Default

    def test_config_creation_with_pool_params(self):
        data = {
            "kind": SOURCE_KIND,
            "host": "localhost",
            "port": "5433", # Test string port
            "user": "testuser",
            "password": "password",
            "database": "testdb",
            "pool_min_conn": "2", # Test string pool_min_conn
            "pool_max_conn": 20
        }
        config = PostgresConfig.from_dict("pg_test_pool", data)
        self.assertEqual(config.port, 5433)
        self.assertEqual(config.pool_min_conn, 2)
        self.assertEqual(config.pool_max_conn, 20)

    def test_config_creation_with_extra_params(self):
        data = {
            "kind": SOURCE_KIND,
            "host": "localhost",
            "port": 5432,
            "user": "testuser",
            "password": "password",
            "database": "testdb",
            "application_name": "my_app", # Extra psycopg2 param
            "connect_timeout": 5
        }
        config = PostgresConfig.from_dict("pg_test_extra", data)
        self.assertEqual(config.name, "pg_test_extra")
        self.assertTrue(hasattr(config, "application_name"))
        self.assertEqual(getattr(config, "application_name"), "my_app")
        self.assertTrue(hasattr(config, "connect_timeout"))
        self.assertEqual(getattr(config, "connect_timeout"), 5)


    def test_config_missing_field(self):
        data = {"kind": SOURCE_KIND, "host": "localhost", "port": 5432, "user": "testuser"} # Missing password and database
        with self.assertRaisesRegex(ValueError, "missing required field: 'password'"):
            PostgresConfig.from_dict("pg_test_fail", data)

    def test_config_kind_mismatch_in_constructor(self):
        with self.assertRaisesRegex(ValueError, "Kind mismatch for PostgresConfig"):
            PostgresConfig(name="test", kind="wrong_kind", host="h", port=1, user="u", password="p", database="d")


@unittest.skipIf(IMPORT_ERROR, "Skipping tests due to import error")
class TestPostgresSource(unittest.TestCase):
    def _get_valid_config(self, name="pg_dummy", **kwargs):
        config_data = {
            "host": "dummyhost", "port": 1234, "user": "u", "password": "p", "database": "db",
            **kwargs
        }
        return PostgresConfig(name, SOURCE_KIND, **config_data)

    @patch('psycopg2.pool.ThreadedConnectionPool')
    def test_source_connect_success(self, mock_pool_constructor):
        mock_pool_instance = MagicMock()
        mock_conn = MagicMock()
        mock_pool_instance.getconn.return_value = mock_conn # Simulate successful connection retrieval
        mock_pool_constructor.return_value = mock_pool_instance

        config = self._get_valid_config(application_name="test_app") # With extra param
        source = PostgresSource(config)

        source.connect()

        mock_pool_constructor.assert_called_once_with(
            minconn=config.pool_min_conn,
            maxconn=config.pool_max_conn,
            user=config.user,
            password=config.password,
            host=config.host,
            port=config.port,
            database=config.database,
            application_name="test_app" # Ensure extra param is passed
        )
        mock_pool_instance.getconn.assert_called_once() # To test connection
        mock_pool_instance.putconn.assert_called_once_with(mock_conn)
        self.assertIsNotNone(source._pool)
        self.assertEqual(source.source_kind(), SOURCE_KIND)

    @patch('psycopg2.pool.ThreadedConnectionPool', side_effect=psycopg2.Error("Connection failed"))
    def test_source_connect_failure(self, mock_pool_constructor):
        config = self._get_valid_config()
        source = PostgresSource(config)

        with self.assertRaisesRegex(psycopg2.Error, "Connection failed"):
            source.connect()
        self.assertIsNone(source._pool)

    def test_source_connect_already_connected(self):
        config = self._get_valid_config()
        source = PostgresSource(config)
        source._pool = MagicMock() # Simulate already connected

        with patch.object(logger, 'info') as mock_logger_info:
             source.connect()
             mock_logger_info.assert_any_call(f"PostgreSQL source '{source.name}' connection pool already initialized.")
        self.assertIsNotNone(source._pool) # Should still be the mock

    def test_close_pool(self):
        config = self._get_valid_config()
        source = PostgresSource(config)
        source._pool = MagicMock(spec=psycopg2.pool.ThreadedConnectionPool)

        source.close()
        source._pool.closeall.assert_called_once()
        self.assertIsNone(source._pool)

    def test_close_no_pool(self):
        config = self._get_valid_config()
        source = PostgresSource(config)
        source._pool = None # Ensure no pool

        with patch.object(logger, 'info') as mock_logger_info:
            source.close()
            mock_logger_info.assert_any_call(f"PostgreSQL source '{source.name}' connection pool already closed or not initialized.")
        self.assertIsNone(source._pool)


    @patch('psycopg2.pool.ThreadedConnectionPool')
    def test_get_connection_success(self, mock_pool_constructor):
        mock_pool_instance = MagicMock()
        mock_conn = MagicMock()
        mock_pool_instance.getconn.return_value = mock_conn
        mock_pool_constructor.return_value = mock_pool_instance

        config = self._get_valid_config()
        source = PostgresSource(config)
        source.connect() # Initialize pool

        with source.get_connection() as conn:
            self.assertEqual(conn, mock_conn)

        mock_pool_instance.getconn.assert_called_with() # Called once by connect, once by get_connection
        self.assertEqual(mock_pool_instance.getconn.call_count, 2) # 1 from connect, 1 from get_connection
        mock_pool_instance.putconn.assert_called_with(mock_conn) # 1 from connect, 1 from get_connection
        self.assertEqual(mock_pool_instance.putconn.call_count, 2)


    @patch('psycopg2.pool.ThreadedConnectionPool')
    def test_get_connection_connects_if_not_connected(self, mock_pool_constructor):
        mock_pool_instance = MagicMock()
        mock_conn = MagicMock()
        mock_pool_instance.getconn.return_value = mock_conn
        mock_pool_constructor.return_value = mock_pool_instance

        config = self._get_valid_config()
        source = PostgresSource(config)
        # source._pool is None initially

        with source.get_connection() as conn:
            self.assertEqual(conn, mock_conn)

        mock_pool_constructor.assert_called_once() # connect() should be called by get_connection()
        self.assertIsNotNone(source._pool)


    def test_get_connection_no_pool_after_connect_attempt(self):
        config = self._get_valid_config()
        source = PostgresSource(config)

        # Mock connect to fail to set a pool
        with patch.object(PostgresSource, 'connect', side_effect=psycopg2.Error("Simulated connect error")):
            with self.assertRaisesRegex(ConnectionError, f"PostgreSQL source '{source.name}' is not connected."):
                with source.get_connection():
                    pass # Should not reach here

    @patch('psycopg2.pool.ThreadedConnectionPool')
    def test_get_connection_pool_error(self, mock_pool_constructor):
        mock_pool_instance = MagicMock()
        mock_pool_instance.getconn.side_effect = psycopg2.Error("Pool error")
        mock_pool_constructor.return_value = mock_pool_instance

        config = self._get_valid_config()
        source = PostgresSource(config)
        source.connect() # Initialize pool, first getconn works

        # Reset mock for the specific getconn call in the context manager
        source._pool.getconn.side_effect = psycopg2.Error("Pool error during context getconn")
        source._pool.putconn.reset_mock() # Reset putconn that was called during connect

        with self.assertRaisesRegex(psycopg2.Error, "Pool error during context getconn"):
            with source.get_connection():
                pass # Should not reach here

        # Check that putconn was not called for the failed getconn
        source._pool.putconn.assert_not_called()


if __name__ == '__main__':
    # This allows running the tests directly from this file
    # Add project root to sys.path for imports to work if run directly
    import sys
    import os
    if not IMPORT_ERROR: # Only run if imports were successful
        # To ensure py_toolbox can be found if test is run directly
        # sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '../../../')))
        unittest.main()
    else:
        print("Skipping test run due to import errors.")
