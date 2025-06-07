from abc import ABC, abstractmethod
from typing import Any, Dict, List, TypeVar, Type, Mapping
from py_toolbox.internal.sources.base import Source # Re-importing for type hinting
from py_toolbox.internal.core.logging import get_logger
from pydantic import BaseModel, Field # For Manifests

logger = get_logger(__name__)

T = TypeVar('T')

# Manifest Structures (similar to Go project)
class ParameterManifest(BaseModel):
    name: str
    type: str # e.g., "string", "integer", "boolean"
    description: str
    required: bool = False

class Manifest(BaseModel):
    description: str
    parameters: List[ParameterManifest] = Field(default_factory=list)
    auth_required: List[str] = Field(default_factory=list) # List of source kinds or auth groups

class McpInputSchema(BaseModel):
    type: str = "object"
    properties: Dict[str, Dict[str, Any]] = Field(default_factory=dict) # JSON Schema properties
    required: List[str] = Field(default_factory=list)

class McpManifest(BaseModel):
    name: str
    description: str = ""
    input_schema: McpInputSchema = Field(default_factory=McpInputSchema)


class Tool(ABC):
    """Base class for a tool."""
    def __init__(self, name: str, kind: str):
        self.name = name
        self.kind = kind

    @abstractmethod
    def tool_kind(self) -> str:
        """Returns the kind of the tool."""
        pass

    @abstractmethod
    def invoke(self, params: Dict[str, Any]) -> Any: # Return type can be complex
        """Executes the tool's logic with the given parameters."""
        pass

    @abstractmethod
    def get_manifest(self) -> Manifest:
        """Returns the tool's manifest for client SDKs."""
        pass

    @abstractmethod
    def get_mcp_manifest(self) -> McpManifest:
        """Returns the tool's manifest for the MCP."""
        pass

    @abstractmethod
    def is_authorized(self, verified_auth_services: List[str]) -> bool:
        """Checks if the tool is authorized based on provided services."""
        pass


class ToolConfig(ABC):
    """Base class for tool configuration."""
    def __init__(self, name: str, kind: str, description: str = "", **kwargs: Any):
        self.name = name
        self.kind = kind
        self.description = description
        # Store additional arguments not explicitly defined in constructor
        for key, value in kwargs.items():
            setattr(self, key, value)


    @abstractmethod
    def tool_config_kind(self) -> str:
        """Returns the kind of the tool config."""
        pass

    @abstractmethod
    def initialize(self, sources: Mapping[str, Source]) -> Tool:
        """Initializes and returns a Tool instance, using provided sources."""
        pass

    @classmethod
    def from_dict(cls: Type[T], name: str, data: Dict[str, Any]) -> T:
        """Creates a config instance from a dictionary."""
        kind = data.get("kind")
        if not kind:
            raise ValueError(f"Tool config for '{name}' is missing 'kind' field.")

        # Pass all data items to the constructor
        logger.debug(f"Creating tool config for '{name}' of kind '{kind}' using {cls.__name__} with data: {data}")
        return cls(name=name, **data)
