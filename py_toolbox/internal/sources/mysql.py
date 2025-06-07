import mysql.connector
from mysql.connector import pooling
from typing import Any, Dict, Optional
from contextlib import contextmanager

from py_toolbox.internal.sources.base import Source, SourceConfig
from py_toolbox.internal.core.logging import get_logger
# Assuming SourceRegistry will be passed to the registration function
# from py_toolbox.internal.core.registry import SourceRegistry

logger = get_logger(__name__)

SOURCE_KIND = "mysql"

class MySQLConfig(SourceConfig):
    def __init__(self, name: str, kind: str, host: str, port: int, user: str, password: str, database: str, pool_name: str = "mysql_pool", pool_size: int = 5, **kwargs):
        super().__init__(name, kind, **kwargs) # Pass kwargs to parent for storage
        if kind != SOURCE_KIND:
            raise ValueError(f"Kind mismatch for MySQLConfig. Expected '{SOURCE_KIND}', got '{kind}'")
        self.host = host
        self.port = port
        self.user = user
        self.password = password
        self.database = database
        self.pool_name = pool_name
        self.pool_size = pool_size
        # self.extra_kwargs are already stored by super().__init__

    def source_config_kind(self) -> str:
        return SOURCE_KIND

    def initialize(self) -> Source:
        logger.info(f"Initializing MySQL source: {self.name}")
        return MySQLSource(self)

    @classmethod
    def from_dict(cls, name: str, data: Dict[str, Any]) -> 'MySQLConfig':
        required_fields = ['host', 'port', 'user', 'password', 'database']
        for field in required_fields:
            if field not in data:
                raise ValueError(f"MySQL config for '{name}' is missing required field: '{field}'")

        # Collect extra arguments not explicitly in constructor or already handled
        current_fields = ['name', 'kind', 'host', 'port', 'user', 'password', 'database', 'pool_name', 'pool_size']
        extra_kwargs = {k: v for k, v in data.items() if k not in current_fields}

        return cls(
            name=name,
            kind=data.get("kind", SOURCE_KIND),
            host=data['host'],
            port=int(data['port']),
            user=data['user'],
            password=data['password'],
            database=data['database'],
            pool_name=data.get('pool_name', f"{name}_mysql_pool"), # Default pool_name based on source name
            pool_size=int(data.get('pool_size', 5)),
            **extra_kwargs # Pass other mysql.connector options
        )

class MySQLSource(Source):
    def __init__(self, config: MySQLConfig):
        super().__init__(config.name, config.source_config_kind())
        self.config = config
        self._pool: Optional[mysql.connector.pooling.MySQLConnectionPool] = None

    def source_kind(self) -> str:
        return SOURCE_KIND

    def connect(self) -> None:
        if self._pool:
            logger.info(f"MySQL source '{self.name}' connection pool '{self.config.pool_name}' already initialized.")
            return

        logger.info(f"Connecting MySQL source: {self.name} to {self.config.host}:{self.config.port}/{self.config.database} using pool '{self.config.pool_name}'")
        try:
            # Retrieve extra_kwargs from config object (stored by base SourceConfig via **kwargs)
            # These are connection-specific arguments like 'auth_plugin', 'ssl_ca', etc.
            extra_conn_params = {k: getattr(self.config, k) for k in self.config.__dict__.keys()
                                 if k not in ['name', 'kind', 'host', 'port', 'user', 'password', 'database', 'pool_name', 'pool_size']}


            pool_config = {
                "pool_name": self.config.pool_name,
                "pool_size": self.config.pool_size,
                "host": self.config.host,
                "port": self.config.port,
                "user": self.config.user,
                "password": self.config.password,
                "database": self.config.database,
                **extra_conn_params
            }
            self._pool = mysql.connector.pooling.MySQLConnectionPool(**pool_config)

            # Test connection by getting and returning a connection
            conn_test = self._pool.get_connection()
            logger.info(f"Successfully connected to MySQL source: {self.name} and pool '{self.config.pool_name}' created.")
            conn_test.close()
        except mysql.connector.Error as e:
            logger.error(f"Failed to connect to MySQL source '{self.name}' or create pool '{self.config.pool_name}': {e}")
            self._pool = None # Ensure pool is None if connection failed
            raise # Re-raise the exception

    def close(self) -> None:
        # MySQL pooling does not have an explicit close for the pool itself.
        # Connections are returned to the pool. If the pool object is dereferenced, it gets GC'd.
        # We can set self._pool to None to signify it's no longer managed by this Source instance.
        if self._pool:
            logger.info(f"MySQL source '{self.name}' (pool '{self.config.pool_name}') close called. The pool itself will be garbage collected if no other references exist. Active connections will remain open until returned and closed.")
            self._pool = None
        else:
            logger.info(f"MySQL source '{self.name}' pool already considered closed or not initialized.")


    @contextmanager
    def get_connection(self):
        if not self._pool:
            # Attempt to reconnect if pool is None (e.g., after a close() or failed initial connect())
            logger.info(f"Pool for MySQL source '{self.name}' (pool '{self.config.pool_name}') is not available. Attempting to connect.")
            self.connect()

        if not self._pool: # If still no pool after connect attempt
             raise ConnectionError(f"MySQL source '{self.name}' (pool '{self.config.pool_name}') is not connected.")

        conn = None
        try:
            conn = self._pool.get_connection()
            yield conn
        except mysql.connector.Error as e: # This includes PoolError
            logger.error(f"Error getting connection from MySQL pool '{self.config.pool_name}' for source '{self.name}': {e}")
            # Check if connection was retrieved but is broken
            if conn and not conn.is_connected():
                 logger.warning(f"MySQL connection for source '{self.name}' from pool '{self.config.pool_name}' was found to be broken.")
            # Specific check for pool exhaustion or other pool errors
            if isinstance(e, mysql.connector.errors.PoolError):
                logger.error(f"MySQL connection pool '{self.config.pool_name}' exhausted or other pool error: {e}")
            # No need to manually return/close 'conn' here if get_connection() failed, as it wasn't successfully yielded.
            raise # Re-raise the original error
        finally:
            # This block executes regardless of whether an exception occurred in 'yield conn'
            if conn and conn.is_connected(): # If 'conn' was successfully yielded and is still connected
                try:
                    conn.close() # Returns the connection to the pool
                except mysql.connector.Error as e_close:
                    logger.error(f"Error returning connection to MySQL pool '{self.config.pool_name}' for source '{self.name}': {e_close}")


# Registration function for this source (to be called by main.py or similar)
def register_source(registry: Any): # Using Any for registry type to avoid direct SourceRegistry import if problematic
    registry.register(SOURCE_KIND, MySQLConfig.from_dict)
    logger.info(f"MySQL source kind '{SOURCE_KIND}' registration function called.")
