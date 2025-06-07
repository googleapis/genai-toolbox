import unittest
from unittest.mock import patch, MagicMock
from neo4j import GraphDatabase, Auth, exceptions as neo4j_exceptions # For types and exceptions

try:
    from py_toolbox.internal.sources.neo4j_source import Neo4jConfig, Neo4jSource, SOURCE_KIND
    IMPORT_SUCCESSFUL = True
except ImportError as e:
    print(f"Import error in test_neo4j_source.py: {e}. Ensure py_toolbox is in PYTHONPATH.")
    IMPORT_SUCCESSFUL = False
    class Neo4jConfig: pass
    class Neo4jSource: pass
    SOURCE_KIND = "neo4j"


@unittest.skipIf(not IMPORT_SUCCESSFUL, "Skipping Neo4j source tests due to import error")
class TestNeo4jConfig(unittest.TestCase):
    def test_config_creation_minimal(self):
        data = {"kind": SOURCE_KIND, "uri": "neo4j://localhost:7687", "user": "neo4j", "password": "password"}
        config = Neo4jConfig.from_dict("neo4j_min", data)
        self.assertEqual(config.name, "neo4j_min")
        self.assertEqual(config.uri, "neo4j://localhost:7687")
        self.assertEqual(config.user, "neo4j")
        self.assertEqual(config.password, "password")
        self.assertIsNone(config.database) # Default database is None
        self.assertEqual(config.source_config_kind(), SOURCE_KIND)

    def test_config_creation_with_database_and_extras(self):
        data = {
            "kind": SOURCE_KIND,
            "uri": "bolt://db.example.com",
            "user": "admin",
            "password": "secure",
            "database": "customdb",
            "max_connection_lifetime": 3600 # Extra driver option
        }
        config = Neo4jConfig.from_dict("neo4j_full", data)
        self.assertEqual(config.uri, "bolt://db.example.com")
        self.assertEqual(config.database, "customdb")
        self.assertTrue(hasattr(config, "max_connection_lifetime"))
        self.assertEqual(getattr(config, "max_connection_lifetime"), 3600)

    def test_config_missing_required_field(self):
        data = {"kind": SOURCE_KIND, "uri": "uri", "user": "user"} # Missing 'password'
        with self.assertRaisesRegex(ValueError, "missing required field: 'password'"):
            Neo4jConfig.from_dict("neo4j_fail_pass", data)

    def test_config_empty_uri(self): # Constructor check, not from_dict
        with self.assertRaisesRegex(ValueError, "requires 'uri', 'user', and 'password'"):
            Neo4jConfig(name="test", kind=SOURCE_KIND, uri="", user="u", password="p")


@unittest.skipIf(not IMPORT_SUCCESSFUL, "Skipping Neo4j source tests due to import error")
class TestNeo4jSource(unittest.TestCase):
    def setUp(self):
        self.base_config_data = {"uri": "neo4j://dummyserver:7687", "user": "testuser", "password": "testpassword"}
        # Create a full config object for source instantiation
        self.config = Neo4jConfig.from_dict("neo4j_src_test", {**self.base_config_data, "kind": SOURCE_KIND, "database": "testgraph", "encrypted": False})

    @patch('neo4j.GraphDatabase.driver') # Patch the driver constructor
    def test_source_connect_success(self, mock_driver_constructor):
        mock_driver_instance = MagicMock(spec=GraphDatabase.driver) # Mock the driver instance
        mock_driver_constructor.return_value = mock_driver_instance

        source = Neo4jSource(self.config)
        source.connect()

        expected_driver_options = {'encrypted': False} # From self.config extra kwargs
        mock_driver_constructor.assert_called_once_with(
            self.config.uri,
            auth=unittest.mock.ANY, # Check specific auth details below
            **expected_driver_options
        )
        # Check auth object details
        _, kwargs_used = mock_driver_constructor.call_args
        auth_param = kwargs_used['auth']
        self.assertIsInstance(auth_param, Auth)
        self.assertEqual(auth_param.scheme, "basic")
        self.assertEqual(auth_param.principal, self.config.user)
        self.assertEqual(auth_param.credentials, self.config.password)

        mock_driver_instance.verify_connectivity.assert_called_once()
        self.assertIsNotNone(source._driver)
        self.assertEqual(source._driver, mock_driver_instance)
        self.assertEqual(source.source_kind(), SOURCE_KIND)


    @patch('neo4j.GraphDatabase.driver')
    def test_source_connect_service_unavailable(self, mock_driver_constructor):
        mock_driver_instance = MagicMock()
        mock_driver_instance.verify_connectivity.side_effect = neo4j_exceptions.ServiceUnavailable("Cannot connect")
        mock_driver_constructor.return_value = mock_driver_instance

        source = Neo4jSource(self.config)
        with self.assertRaisesRegex(neo4j_exceptions.ServiceUnavailable, "Cannot connect"):
            source.connect()
        self.assertIsNone(source._driver)

    @patch('neo4j.GraphDatabase.driver')
    def test_source_connect_auth_error(self, mock_driver_constructor):
        mock_driver_instance = MagicMock()
        mock_driver_instance.verify_connectivity.side_effect = neo4j_exceptions.AuthError("Invalid credentials")
        mock_driver_constructor.return_value = mock_driver_instance

        source = Neo4jSource(self.config)
        with self.assertRaisesRegex(neo4j_exceptions.AuthError, "Invalid credentials"):
            source.connect()
        self.assertIsNone(source._driver)

    def test_driver_property_connects_if_not_connected(self):
        source = Neo4jSource(self.config) # _driver is None initially
        mock_driver_instance = MagicMock(spec=GraphDatabase.driver)

        # Mock the connect method of this specific source instance
        with patch.object(source, 'connect', wraps=source.connect) as mock_instance_connect:
            # Temporarily replace the _driver that connect() would set, to simulate successful connection
            # This is a bit complex because driver property calls connect which sets _driver.
            # So, we mock connect to also set the _driver to our mock_driver_instance
            def side_effect_connect():
                source._driver = mock_driver_instance
            mock_instance_connect.side_effect = side_effect_connect

            retrieved_driver = source.driver # Access property

            mock_instance_connect.assert_called_once() # connect() should have been called
            self.assertEqual(retrieved_driver, mock_driver_instance)


    def test_driver_property_raises_if_connect_fails(self):
        source = Neo4jSource(self.config)
        with patch.object(source, 'connect', side_effect=neo4j_exceptions.ServiceUnavailable("Failed")) as mock_instance_connect:
            with self.assertRaises(ConnectionError): # Property should raise ConnectionError
                _ = source.driver
            mock_instance_connect.assert_called_once()


    def test_get_session_uses_config_database_by_default(self):
        source = Neo4jSource(self.config) # config has database="testgraph"
        mock_driver_val = MagicMock(spec=GraphDatabase.driver)
        mock_session_val = MagicMock(spec=neo4j_exceptions.BoltStatementResult) # Placeholder for session type
        mock_driver_val.session.return_value = mock_session_val
        source._driver = mock_driver_val # Manually set driver to avoid connect complexity here

        with source.get_session() as session:
            self.assertEqual(session, mock_session_val)

        mock_driver_val.session.assert_called_once_with(database="testgraph") # From self.config
        mock_session_val.close.assert_called_once()

    def test_get_session_overrides_database(self):
        source = Neo4jSource(self.config)
        mock_driver_val = MagicMock(spec=GraphDatabase.driver)
        mock_session_val = MagicMock()
        mock_driver_val.session.return_value = mock_session_val
        source._driver = mock_driver_val

        with source.get_session(database="overridedb") as session:
            self.assertEqual(session, mock_session_val)

        mock_driver_val.session.assert_called_once_with(database="overridedb") # Overridden
        mock_session_val.close.assert_called_once()

    def test_get_session_default_database_if_config_db_is_none(self):
        config_no_db = Neo4jConfig.from_dict("neo_no_db", {**self.base_config_data, "kind": SOURCE_KIND, "database": None})
        source = Neo4jSource(config_no_db)
        mock_driver_val = MagicMock(spec=GraphDatabase.driver)
        mock_session_val = MagicMock()
        mock_driver_val.session.return_value = mock_session_val
        source._driver = mock_driver_val

        with source.get_session() as session: # Should pass database=None to driver.session
            self.assertEqual(session, mock_session_val)
        mock_driver_val.session.assert_called_once_with(database=None)
        mock_session_val.close.assert_called_once()


    def test_close_closes_driver(self):
        source = Neo4jSource(self.config)
        mock_driver_val = MagicMock(spec=GraphDatabase.driver)
        source._driver = mock_driver_val # Simulate active driver

        source.close()
        mock_driver_val.close.assert_called_once()
        self.assertIsNone(source._driver)

    def test_close_no_driver(self):
        source = Neo4jSource(self.config) # _driver is None
        with patch.object(source.logger, 'info') as mock_logger:
            source.close()
            mock_logger.assert_any_call(f"Neo4j driver for source '{source.name}' already closed or not initialized.")
        self.assertIsNone(source._driver)


if __name__ == '__main__':
    unittest.main()
