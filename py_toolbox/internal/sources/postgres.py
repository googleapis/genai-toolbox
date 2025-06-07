import psycopg2
from psycopg2 import pool
from psycopg2 import extras # For RealDictCursor
from typing import Any, Dict, Optional
from contextlib import contextmanager

from py_toolbox.internal.sources.base import Source, SourceConfig
# Ensure SourceRegistry is imported correctly for registration
from py_toolbox.internal.core import SourceRegistry
from py_toolbox.internal.core.logging import get_logger

logger = get_logger(__name__)

SOURCE_KIND = "postgres"

class PostgresConfig(SourceConfig):
    def __init__(self, name: str, kind: str, host: str, port: int, user: str, password: str, database: str, pool_min_conn: int = 1, pool_max_conn: int = 10, **kwargs):
        super().__init__(name, kind, **kwargs) # Pass kwargs to parent
        if kind != SOURCE_KIND:
            raise ValueError(f"Kind mismatch for PostgresConfig. Expected '{SOURCE_KIND}', got '{kind}'")
        self.host = host
        self.port = port
        self.user = user
        self.password = password
        self.database = database
        self.pool_min_conn = pool_min_conn
        self.pool_max_conn = pool_max_conn
        # self.extra_kwargs are already handled by super().__init__ if not explicitly defined

    def source_config_kind(self) -> str:
        return SOURCE_KIND

    def initialize(self) -> Source:
        logger.info(f"Initializing PostgreSQL source: {self.name}")
        return PostgresSource(self)

    @classmethod
    def from_dict(cls, name: str, data: Dict[str, Any]) -> 'PostgresConfig':
        required_fields = ['host', 'port', 'user', 'password', 'database']
        for field in required_fields:
            if field not in data:
                raise ValueError(f"PostgreSQL config for '{name}' is missing required field: '{field}'")

        # Collect extra arguments not explicitly in constructor
        current_fields = ['name', 'kind', 'host', 'port', 'user', 'password', 'database', 'pool_min_conn', 'pool_max_conn']
        extra_kwargs = {k: v for k, v in data.items() if k not in current_fields}

        return cls(
            name=name,
            kind=data.get("kind", SOURCE_KIND),
            host=data['host'],
            port=int(data['port']),
            user=data['user'],
            password=data['password'],
            database=data['database'],
            pool_min_conn=int(data.get('pool_min_conn', 1)),
            pool_max_conn=int(data.get('pool_max_conn', 10)),
            **extra_kwargs
        )

class PostgresSource(Source):
    def __init__(self, config: PostgresConfig):
        super().__init__(config.name, config.source_config_kind())
        self.config = config
        self._pool: Optional[psycopg2.pool.ThreadedConnectionPool] = None

    def source_kind(self) -> str:
        return SOURCE_KIND

    def connect(self) -> None:
        if self._pool:
            logger.info(f"PostgreSQL source '{self.name}' connection pool already initialized.")
            return

        logger.info(f"Connecting PostgreSQL source: {self.name} to {self.config.host}:{self.config.port}/{self.config.database}")
        try:
            # Retrieve extra_kwargs from config object if they were stored there by base SourceConfig
            extra_conn_params = {k: getattr(self.config, k) for k in self.config.__dict__.keys() if k not in ['name', 'kind', 'host', 'port', 'user', 'password', 'database', 'pool_min_conn', 'pool_max_conn']}

            self._pool = psycopg2.pool.ThreadedConnectionPool(
                minconn=self.config.pool_min_conn,
                maxconn=self.config.pool_max_conn,
                user=self.config.user,
                password=self.config.password,
                host=self.config.host,
                port=self.config.port,
                database=self.config.database,
                **extra_conn_params # Pass additional psycopg2 specific params
            )
            conn = self._pool.getconn()
            logger.info(f"Successfully connected to PostgreSQL source: {self.name}")
            self._pool.putconn(conn)
        except psycopg2.Error as e:
            logger.error(f"Failed to connect to PostgreSQL source '{self.name}': {e}")
            self._pool = None
            raise
        except AttributeError as e: # Catch if some expected extra_kwargs were not set on config
            logger.error(f"Configuration error for PostgreSQL source '{self.name}': {e}")
            self._pool = None
            raise ValueError(f"Configuration error for source {self.name}: {e}")


    def close(self) -> None:
        if self._pool:
            logger.info(f"Closing PostgreSQL source connection pool: {self.name}")
            self._pool.closeall()
            self._pool = None
        else:
            logger.info(f"PostgreSQL source '{self.name}' connection pool already closed or not initialized.")

    @contextmanager
    def get_connection(self):
        if not self._pool:
            self.connect()

        if not self._pool:
             raise ConnectionError(f"PostgreSQL source '{self.name}' is not connected.")

        conn = None
        try:
            conn = self._pool.getconn()
            yield conn
        except psycopg2.Error as e:
            logger.error(f"Error getting connection from pool for '{self.name}': {e}")
            # TODO: Potentially handle broken connections or pool errors here
            # For now, re-raise the error.
            if conn: # If conn was retrieved but something went wrong using it (e.g. closed by server)
                # Return it to the pool, which might discard it if it's broken
                self._pool.putconn(conn, close=True) # Close this specific connection
            raise
        finally:
            if conn: # Ensure connection is returned if it was successfully retrieved
                 self._pool.putconn(conn)


# Registration function for this source

# Registration function for this source
def register_source(registry: SourceRegistry):
    # This function is called by the main application to register the source
    registry.register(SOURCE_KIND, PostgresConfig.from_dict)
    logger.info(f"PostgreSQL source kind '{SOURCE_KIND}' registration function called by main app.")

# Example of how to call it (application entry point would manage registries)
# if __name__ == '__main__':
#     # This is illustrative; actual registry would be managed by the application's core setup
#     # from py_toolbox.internal.core.registry import SourceRegistry
#     # global_source_registry = SourceRegistry()
#     # register_postgres_source(global_source_registry)
#     pass