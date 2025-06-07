from abc import ABC, abstractmethod
import yaml
from typing import Any, Dict, TypeVar, Type
from py_toolbox.internal.core.logging import get_logger

logger = get_logger(__name__)

T = TypeVar('T')

class Source(ABC):
    """Base class for a data source."""
    def __init__(self, name: str, kind: str):
        self.name = name
        self.kind = kind

    @abstractmethod
    def source_kind(self) -> str:
        """Returns the kind of the source."""
        pass

    @abstractmethod
    def connect(self) -> None:
        """Establishes a connection to the source if applicable."""
        pass

    @abstractmethod
    def close(self) -> None:
        """Closes the connection to the source if applicable."""
        pass

class SourceConfig(ABC):
    """Base class for source configuration."""
    def __init__(self, name: str, kind: str, **kwargs: Any): # Added **kwargs to accept extra fields
        self.name = name
        self.kind = kind
        # Store additional arguments not explicitly defined in constructor
        for key, value in kwargs.items():
            setattr(self, key, value)

    @abstractmethod
    def source_config_kind(self) -> str:
        """Returns the kind of the source config."""
        pass

    @abstractmethod
    def initialize(self) -> Source:
        """Initializes and returns a Source instance."""
        pass

    @classmethod
    def from_dict(cls: Type[T], name: str, data: Dict[str, Any]) -> T:
        """Creates a config instance from a dictionary.
        This method should be overridden by subclasses if they have complex initialization.
        """
        kind = data.get("kind")
        if not kind:
            raise ValueError(f"Source config for '{name}' is missing 'kind' field.")

        # Pass all data items to the constructor, which will handle them via **kwargs
        # or specific parameters if overridden.
        logger.debug(f"Creating config for source '{name}' of kind '{kind}' using {cls.__name__} with data: {data}")
        return cls(name=name, **data)
