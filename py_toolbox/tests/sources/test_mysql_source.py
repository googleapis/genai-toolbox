import unittest
from unittest.mock import patch, MagicMock
import mysql.connector # Import for exception types

# Attempt to import the items to be tested
try:
    from py_toolbox.internal.sources.mysql import MySQLConfig, MySQLSource, SOURCE_KIND
    IMPORT_SUCCESSFUL = True
except ImportError as e:
    print(f"Import error in test_mysql_source.py: {e}. Ensure py_toolbox is in PYTHONPATH.")
    IMPORT_SUCCESSFUL = False
    # Define dummy classes if import fails, so tests can be discovered / partially run
    class MySQLConfig: pass
    class MySQLSource: pass
    SOURCE_KIND = "mysql"


@unittest.skipIf(not IMPORT_SUCCESSFUL, "Skipping MySQL source tests due to import error")
class TestMySQLConfig(unittest.TestCase):
    def test_config_creation_minimal(self):
        data = {
            "kind": SOURCE_KIND,
            "host": "localhost",
            "port": 3306,
            "user": "testuser",
            "password": "password",
            "database": "testdb"
        }
        config = MySQLConfig.from_dict("mysql_test_min", data)
        self.assertEqual(config.name, "mysql_test_min")
        self.assertEqual(config.host, "localhost")
        self.assertEqual(config.port, 3306)
        self.assertEqual(config.user, "testuser")
        self.assertEqual(config.password, "password")
        self.assertEqual(config.database, "testdb")
        self.assertEqual(config.source_config_kind(), SOURCE_KIND)
        self.assertEqual(config.pool_name, "mysql_test_min_mysql_pool") # Default pool name
        self.assertEqual(config.pool_size, 5) # Default pool size

    def test_config_creation_with_pool_options(self):
        data = {
            "kind": SOURCE_KIND,
            "host": "remotehost",
            "port": "3307", # Test string port
            "user": "anotheruser",
            "password": "securepassword",
            "database": "productiondb",
            "pool_name": "custom_prod_pool",
            "pool_size": "10" # Test string pool_size
        }
        config = MySQLConfig.from_dict("mysql_test_custom", data)
        self.assertEqual(config.host, "remotehost")
        self.assertEqual(config.port, 3307)
        self.assertEqual(config.pool_name, "custom_prod_pool")
        self.assertEqual(config.pool_size, 10)

    def test_config_creation_with_extra_params(self):
        data = {
            "kind": SOURCE_KIND,
            "host": "localhost",
            "port": 3306,
            "user": "testuser",
            "password": "password",
            "database": "testdb",
            "connection_timeout": 30, # Example extra param for mysql.connector
            "ssl_disabled": True
        }
        config = MySQLConfig.from_dict("mysql_test_extra", data)
        self.assertTrue(hasattr(config, "connection_timeout"))
        self.assertEqual(getattr(config, "connection_timeout"), 30)
        self.assertTrue(hasattr(config, "ssl_disabled"))
        self.assertEqual(getattr(config, "ssl_disabled"), True)


    def test_config_missing_required_field(self):
        data = {"kind": SOURCE_KIND, "host": "localhost", "port": 3306, "user": "test_user"} # Missing password, database
        with self.assertRaisesRegex(ValueError, "missing required field: 'password'"):
            MySQLConfig.from_dict("mysql_test_fail", data)

@unittest.skipIf(not IMPORT_SUCCESSFUL, "Skipping MySQL source tests due to import error")
class TestMySQLSource(unittest.TestCase):
    def setUp(self):
        self.base_config_data = {
            "host": "dummyhost", "port": 1234, "user": "u", "password": "p", "database": "db"
        }
        # Create a fully initialized config instance to pass to MySQLSource
        self.config = MySQLConfig.from_dict("mysql_dummy", {**self.base_config_data, "kind": SOURCE_KIND, "auth_plugin": "mysql_native_password"})


    @patch('mysql.connector.pooling.MySQLConnectionPool')
    def test_source_connect_success(self, mock_pool_constructor):
        mock_pool_instance = MagicMock(spec=mysql.connector.pooling.MySQLConnectionPool)
        mock_conn = MagicMock(spec=mysql.connector.connection.MySQLConnection)
        mock_pool_instance.get_connection.return_value = mock_conn
        mock_pool_constructor.return_value = mock_pool_instance

        source = MySQLSource(self.config)
        source.connect()

        expected_pool_args = {
            "pool_name": self.config.pool_name,
            "pool_size": self.config.pool_size,
            "host": self.config.host,
            "port": self.config.port,
            "user": self.config.user,
            "password": self.config.password,
            "database": self.config.database,
            "auth_plugin": "mysql_native_password" # from self.config init
        }
        mock_pool_constructor.assert_called_once_with(**expected_pool_args)
        mock_pool_instance.get_connection.assert_called_once()
        mock_conn.close.assert_called_once()
        self.assertIsNotNone(source._pool)
        self.assertEqual(source._pool, mock_pool_instance)
        self.assertEqual(source.source_kind(), SOURCE_KIND)

    @patch('mysql.connector.pooling.MySQLConnectionPool', side_effect=mysql.connector.Error("Simulated Connection failed"))
    def test_source_connect_failure(self, mock_pool_constructor):
        source = MySQLSource(self.config)
        with self.assertRaisesRegex(mysql.connector.Error, "Simulated Connection failed"):
            source.connect()
        self.assertIsNone(source._pool)

    @patch('mysql.connector.pooling.MySQLConnectionPool')
    def test_get_connection_success(self, mock_pool_constructor):
        mock_pool_instance = MagicMock(spec=mysql.connector.pooling.MySQLConnectionPool)
        mock_conn = MagicMock(spec=mysql.connector.connection.MySQLConnection)
        mock_conn.is_connected.return_value = True # Make sure connection is seen as active
        mock_pool_instance.get_connection.return_value = mock_conn
        mock_pool_constructor.return_value = mock_pool_instance

        source = MySQLSource(self.config)
        source.connect() # Initialize the pool

        with source.get_connection() as conn:
            self.assertEqual(conn, mock_conn)

        self.assertEqual(mock_pool_instance.get_connection.call_count, 2)
        mock_conn.close.assert_called_once() # Connection from context manager is closed

    @patch('mysql.connector.pooling.MySQLConnectionPool') # Mock the pool so connect() succeeds
    def test_get_connection_not_connected_attempts_reconnect(self, mock_pool_constructor_outer):
        mock_pool_instance_outer = MagicMock()
        mock_conn_outer = MagicMock()
        mock_conn_outer.is_connected.return_value = True
        mock_pool_instance_outer.get_connection.return_value = mock_conn_outer
        mock_pool_constructor_outer.return_value = mock_pool_instance_outer

        source = MySQLSource(self.config)
        # source._pool is None initially

        # We expect connect() to be called within get_connection()
        # which will then call mock_pool_constructor_outer
        with source.get_connection() as conn:
             self.assertEqual(conn, mock_conn_outer)

        mock_pool_constructor_outer.assert_called_once() # connect() was called
        self.assertIsNotNone(source._pool)


    @patch('mysql.connector.pooling.MySQLConnectionPool')
    def test_get_connection_pool_error_on_get(self, mock_pool_constructor):
        mock_pool_instance = MagicMock(spec=mysql.connector.pooling.MySQLConnectionPool)
        # First call to get_connection (in connect()) works
        mock_conn_init = MagicMock(spec=mysql.connector.connection.MySQLConnection)
        mock_pool_instance.get_connection.return_value = mock_conn_init
        mock_pool_constructor.return_value = mock_pool_instance

        source = MySQLSource(self.config)
        source.connect()

        # Subsequent call to get_connection (in context manager) fails
        mock_pool_instance.get_connection.side_effect = mysql.connector.errors.PoolError("Pool exhausted")

        with self.assertRaisesRegex(mysql.connector.errors.PoolError, "Pool exhausted"):
            with source.get_connection():
                pass # Should not reach here

        # Initial conn was closed, get_connection for context manager failed, so no new close.
        mock_conn_init.close.assert_called_once()


    @patch('mysql.connector.pooling.MySQLConnectionPool')
    def test_close_clears_pool_reference(self, mock_pool_constructor):
        mock_pool_instance = MagicMock()
        mock_pool_constructor.return_value = mock_pool_instance

        source = MySQLSource(self.config)
        source.connect() # Sets up the pool
        self.assertIsNotNone(source._pool)

        source.close()
        self.assertIsNone(source._pool) # Check that the reference is cleared

    def test_close_no_pool(self):
        source = MySQLSource(self.config) # Pool is None
        with patch.object(source.logger, 'info') as mock_logger:
            source.close()
            mock_logger.assert_any_call(f"MySQL source '{source.name}' pool already considered closed or not initialized.")
        self.assertIsNone(source._pool)


if __name__ == '__main__':
    unittest.main()
