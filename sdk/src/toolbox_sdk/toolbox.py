from pydantic import BaseModel
import requests
import yaml
from langchain_core.tools import StructuredTool
from typing import List, Optional, Type

from .utils import (
    call_tool_api,
    schema_to_model,
)


def load_manifest(url: str, toolset_name: Optional[str] = None) -> dict:
    """
    Fetches and parses the YAML manifest from the Toolbox service.

    Args:
        url: The base URL of the Toolbox service.
        toolset_name: The name of the toolset to load.
          Default: None. If not provided, then all the tools are loaded.

    Returns:
        A dictionary representing the parsed YAML manifest.
    """
    if toolset_name:
        url = f"{url}/api/toolset/{toolset_name}"
    else:
        url = f"{url}/api/toolset"

    response = requests.get(url)
    response.raise_for_status()
    manifest = yaml.safe_load(response.text)
    return manifest


def generate_tool(tool_name: str, tool_schema: dict, url: str) -> StructuredTool:
    """
    Creates a StructuredTool object and a dynamically generated BaseModel based on the tool's schema from the manifest.

    Args:
        tool_name: The name of the tool.
        tool_schema: The schema definition of the tool.
        url: The base URL of the Toolbox service.

    Returns:
        A StructuredTool object.
    """
    if not isinstance(tool_schema, dict):
        raise ValueError("tool_schema must be a dictionary")

    required_keys = ["parameters", "summary", "description"]
    missing_keys: List[str] = []
    for key in required_keys:
        if key not in tool_schema:
            missing_keys.append(key)

    if missing_keys:
        raise ValueError(
            f"Missing required fields in tool schema: {', '.join(missing_keys)}"
        )

    tool_model: Type[BaseModel] = schema_to_model(
        model_name=tool_name, schema=tool_schema["parameters"]
    )

    tool = StructuredTool.from_function(
        func=lambda **kwargs: call_tool_api(url, tool_name, kwargs),
        name=tool_schema["summary"],
        description=tool_schema["description"],
        args_schema=tool_model,
    )
    return tool


def load_toolbox(url: str, toolset_name: str) -> List[StructuredTool]:
    """
    Loads all tools from the specified toolset in the Toolbox service.

    Args:
        url: The base URL of the Toolbox service.
        toolset_name: The name of the toolset to load.

    Returns:
        A list of StructuredTool objects.
    """
    manifest = load_manifest(url, toolset_name)
    tools = []
    for tool_name, tool_schema in manifest["tools"].items():
        tool = generate_tool(tool_name, tool_schema, url)
        tools.append(tool)
    return tools
