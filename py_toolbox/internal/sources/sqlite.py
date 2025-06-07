import sqlite3
from typing import Any, Dict, Optional
from contextlib import contextmanager

from py_toolbox.internal.sources.base import Source, SourceConfig
from py_toolbox.internal.core.logging import get_logger

logger = get_logger(__name__)

SOURCE_KIND = "sqlite"

class SQLiteConfig(SourceConfig):
    def __init__(self, name: str, kind: str, database_file: str, **kwargs):
        super().__init__(name, kind, **kwargs) # Pass kwargs to parent
        if kind != SOURCE_KIND:
            raise ValueError(f"Kind mismatch for SQLiteConfig. Expected '{SOURCE_KIND}', got '{kind}'")
        if not database_file: # :memory: is allowed, but empty string is not.
            raise ValueError(f"SQLite config for '{name}' requires 'database_file' path.")
        self.database_file = database_file
        # self.extra_kwargs are stored by super().__init__

    def source_config_kind(self) -> str:
        return SOURCE_KIND

    def initialize(self) -> Source:
        logger.info(f"Initializing SQLite source: {self.name} with file '{self.database_file}'")
        return SQLiteSource(self)

    @classmethod
    def from_dict(cls, name: str, data: Dict[str, Any]) -> 'SQLiteConfig':
        # Allow 'database' as an alias for 'database_file' for consistency
        db_file = data.get('database_file', data.get('database'))

        if not db_file:
            raise ValueError(f"SQLite config for '{name}' is missing required field: 'database_file' or 'database'")

        current_fields = ['name', 'kind', 'database_file', 'database']
        extra_kwargs = {k: v for k, v in data.items() if k not in current_fields}

        return cls(
            name=name,
            kind=data.get("kind", SOURCE_KIND),
            database_file=db_file,
            **extra_kwargs # For sqlite3.connect options like timeout, check_same_thread
        )

class SQLiteSource(Source):
    def __init__(self, config: SQLiteConfig):
        super().__init__(config.name, config.source_config_kind())
        self.config = config
        self._connection: Optional[sqlite3.Connection] = None

    def source_kind(self) -> str:
        return SOURCE_KIND

    def connect(self) -> None:
        logger.info(f"SQLite source '{self.name}': connect() called. Validating database file access.")
        try:
            # Do not use self.config.extra_kwargs for this pre-check if they might prevent connection
            # (e.g. check_same_thread=True in a different thread).
            # This is just a basic accessibility check.
            if self.config.database_file != ":memory:": # Cannot pre-check in-memory db this way
                conn_test = sqlite3.connect(self.config.database_file) # Minimal connect
                conn_test.close()
            logger.info(f"SQLite source '{self.name}': Database file '{self.config.database_file}' seems accessible.")
        except sqlite3.Error as e:
            logger.error(f"Failed to test SQLite database file '{self.config.database_file}' for source '{self.name}': {e}")
            raise

    def close(self) -> None:
        if self._connection:
            logger.info(f"Closing persistent SQLite connection for source '{self.name}'.")
            self._connection.close()
            self._connection = None
        else:
            logger.info(f"SQLite source '{self.name}': close() called. No persistent connection to close or connection is managed by context.")

    @contextmanager
    def get_connection(self):
        conn = None
        try:
            logger.debug(f"SQLite source '{self.name}': Opening connection to '{self.config.database_file}'.")
            # Retrieve extra_kwargs from config object
            extra_conn_params = {k: getattr(self.config, k) for k in self.config.__dict__.keys()
                                 if k not in ['name', 'kind', 'database_file']}
            conn = sqlite3.connect(self.config.database_file, **extra_conn_params)
            conn.row_factory = sqlite3.Row
            yield conn
        except sqlite3.Error as e:
            logger.error(f"Error connecting to SQLite database '{self.config.database_file}' for source '{self.name}': {e}")
            raise
        finally:
            if conn:
                logger.debug(f"SQLite source '{self.name}': Closing connection context to '{self.config.database_file}'.")
                conn.close()

# Registration function for this source
def register_source(registry: Any): # Using Any for registry type
    registry.register(SOURCE_KIND, SQLiteConfig.from_dict)
    logger.info(f"SQLite source kind '{SOURCE_KIND}' registration function called.")
