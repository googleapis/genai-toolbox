from typing import Optional, Type

from langchain_core.tools import StructuredTool
from pydantic import BaseModel

from .utils import _call_tool_api, _load_yaml, _schema_to_model


# Helps validate the schema of the tool manifest.
class ToolSchema(BaseModel):
    summary: str
    description: str
    parameters: dict


class ToolboxClient:
    def __init__(self, url: str):
        """
        Initializes the ToolboxClient for the Toolbox service at the given URL.

        Args:
            url: The base URL of the Toolbox service.
        """
        self._url: str = url
        self._manifest: dict = {}
        self._tools: list[StructuredTool] = []

    async def _load_tool_manifest(self, tool_name: str) -> None:
        """
        Fetches and parses the YAML manifest for the given tool from the Toolbox service.

        Args:
            tool_name: The name of the tool to load.
        """
        url = f"{self._url}/api/tool/{tool_name}"

        yaml = await _load_yaml(url)

        if "tools" in self._manifest and "tools" in yaml and tool_name in yaml["tools"]:
            self._manifest["tools"][tool_name] = yaml["tools"][tool_name]
        else:
            self._manifest = yaml

    async def _load_toolset_manifest(self, toolset_name: Optional[str] = None) -> None:
        """
        Fetches and parses the YAML manifest from the Toolbox service.

        Args:
            toolset_name: The name of the toolset to load.
                Default: None. If not provided, then all the tools are loaded.
        """
        url = (
            f"{self._url}/api/toolset/{toolset_name}"
            if toolset_name
            else f"{self._url}/api/toolset"
        )
        self._manifest = await _load_yaml(url)

    def _generate_tool(self, tool_name: str) -> None:
        """
        Creates a StructuredTool object and a dynamically generated BaseModel for the given tool.

        Args:
            tool_name: The name of the tool.
        """
        tool_schema: dict = self._manifest["tools"][tool_name]

        ToolSchema(**tool_schema)

        tool_model: Type[BaseModel] = _schema_to_model(
            model_name=tool_name, schema=tool_schema["parameters"]
        )

        async def _tool_func(**kwargs) -> dict:
            return await _call_tool_api(self._url, tool_name, kwargs)

        tool: StructuredTool = StructuredTool.from_function(
            func=_tool_func,
            name=tool_schema["summary"],
            description=tool_schema["description"],
            args_schema=tool_model,
        )

        self._tools.append(tool)

    async def load_tool(self, tool_name: str) -> StructuredTool:
        """
        Loads tools from the Toolbox service, optionally filtered by toolset name.

        Args:
            toolset_name: The name of the toolset to load.
                Default: None. If not provided, then all the tools are loaded.

        Returns:
            A tool loaded from the Toolbox
        """
        await self._load_tool_manifest(tool_name)
        self._generate_tool(tool_name)
        return self._tools[-1]

    async def load_toolset(
        self, toolset_name: Optional[str] = None
    ) -> list[StructuredTool]:
        """
        Loads tools from the Toolbox service, optionally filtered by toolset name.

        Args:
            toolset_name: The name of the toolset to load.
                Default: None. If not provided, then all the tools are loaded.

        Returns:
            A list of all tools loaded from the Toolbox.
        """
        self._manifest = {}
        self._tools = []
        await self._load_toolset_manifest(toolset_name)
        for tool_name in self._manifest["tools"]:
            self._generate_tool(tool_name)
        return self._tools
