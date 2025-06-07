from neo4j import GraphDatabase, Auth, exceptions as neo4j_exceptions
from typing import Any, Dict, Optional
from contextlib import contextmanager

from py_toolbox.internal.sources.base import Source, SourceConfig
from py_toolbox.internal.core.logging import get_logger

logger = get_logger(__name__)

SOURCE_KIND = "neo4j"

class Neo4jConfig(SourceConfig):
    def __init__(self, name: str, kind: str, uri: str, user: str, password: str, database: Optional[str] = None, **kwargs):
        super().__init__(name, kind, **kwargs) # Pass kwargs to parent
        if kind != SOURCE_KIND:
            raise ValueError(f"Kind mismatch for Neo4jConfig. Expected '{SOURCE_KIND}', got '{kind}'")
        if not uri or not user or not password: # Basic check
            raise ValueError(f"Neo4j config for '{name}' requires 'uri', 'user', and 'password'.")
        self.uri = uri
        self.user = user
        self.password = password
        # For Neo4j, the database can be None, and the driver will use the default database.
        # The constant for default database in the driver is neo4j.DEFAULT_DATABASE but it's often just None.
        self.database = database
        # self.extra_driver_config stored by super().__init__(**kwargs)

    def source_config_kind(self) -> str:
        return SOURCE_KIND

    def initialize(self) -> Source:
        # Use self.database which might be None (driver default) or a specific name.
        db_for_log = self.database if self.database is not None else "default"
        logger.info(f"Initializing Neo4j source: {self.name} for URI '{self.uri}' and database '{db_for_log}'")
        return Neo4jSource(self)

    @classmethod
    def from_dict(cls, name: str, data: Dict[str, Any]) -> 'Neo4jConfig':
        required_fields = ['uri', 'user', 'password']
        for field in required_fields:
            if field not in data:
                raise ValueError(f"Neo4j config for '{name}' is missing required field: '{field}'")

        current_fields = ['name', 'kind', 'uri', 'user', 'password', 'database']
        extra_kwargs = {k: v for k, v in data.items() if k not in current_fields}

        return cls(
            name=name,
            kind=data.get("kind", SOURCE_KIND),
            uri=data['uri'],
            user=data['user'],
            password=data['password'],
            database=data.get('database'), # Optional, driver handles default if None
            **extra_kwargs # For Neo4j driver options like max_connection_lifetime, etc.
        )

class Neo4jSource(Source):
    def __init__(self, config: Neo4jConfig):
        super().__init__(config.name, config.source_config_kind())
        self.config = config
        self._driver: Optional[GraphDatabase.driver] = None

    def source_kind(self) -> str:
        return SOURCE_KIND

    def connect(self) -> None:
        if self._driver:
            logger.info(f"Neo4j source '{self.name}' driver already initialized.")
            return

        db_for_log = self.config.database if self.config.database is not None else "default"
        logger.info(f"Connecting Neo4j source: {self.name} to URI '{self.config.uri}' for database '{db_for_log}'")
        try:
            auth = Auth("basic", self.config.user, self.config.password)
            # Retrieve extra_driver_config from the config object (stored by base SourceConfig)
            driver_options = {k: getattr(self.config, k) for k in self.config.__dict__.keys()
                              if k not in ['name', 'kind', 'uri', 'user', 'password', 'database']}

            self._driver = GraphDatabase.driver(self.config.uri, auth=auth, **driver_options)
            self._driver.verify_connectivity() # Checks if server is available and auth is okay
            logger.info(f"Successfully connected and verified Neo4j source: {self.name}")
        except neo4j_exceptions.ServiceUnavailable as e:
            logger.error(f"Failed to connect to Neo4j source '{self.name}' at URI '{self.config.uri}': Service unavailable. {e}")
            self._driver = None
            raise
        except neo4j_exceptions.AuthError as e:
            logger.error(f"Authentication failed for Neo4j source '{self.name}': {e}")
            self._driver = None
            raise
        except Exception as e:
            logger.error(f"An unexpected error occurred while connecting Neo4j source '{self.name}': {e}")
            self._driver = None
            raise

    def close(self) -> None:
        if self._driver:
            logger.info(f"Closing Neo4j driver for source '{self.name}'.")
            self._driver.close()
            self._driver = None
        else:
            logger.info(f"Neo4j driver for source '{self.name}' already closed or not initialized.")

    @property
    def driver(self) -> GraphDatabase.driver: # Type hint for clarity
        if not self._driver:
            logger.info(f"Neo4j driver for '{self.name}' not initialized. Attempting to connect.")
            self.connect()
        if not self._driver:
            raise ConnectionError(f"Neo4j source '{self.name}' is not connected. Driver unavailable.")
        return self._driver

    @contextmanager
    def get_session(self, **kwargs):
        # Default to the database specified in config if not overridden in kwargs
        # If self.config.database is None, the driver will use the user's default database.
        db_name_to_use = kwargs.pop('database', self.config.database)

        session = None
        try:
            # Pass db_name_to_use which might be None
            session = self.driver.session(database=db_name_to_use, **kwargs)
            log_db_name = db_name_to_use if db_name_to_use is not None else "user's default"
            logger.debug(f"Neo4j session acquired for database '{log_db_name}' on source '{self.name}'.")
            yield session
        finally:
            if session:
                session.close()
                log_db_name = db_name_to_use if db_name_to_use is not None else "user's default"
                logger.debug(f"Neo4j session closed for database '{log_db_name}' on source '{self.name}'.")

# Registration function
def register_source(registry: Any): # Using Any for registry type
    registry.register(SOURCE_KIND, Neo4jConfig.from_dict)
    logger.info(f"Neo4j source kind '{SOURCE_KIND}' registration function called.")
