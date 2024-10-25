from typing import Optional

from aiohttp import ClientSession
from langchain_core.tools import StructuredTool
from pydantic import BaseModel

from .utils import ToolSchema, _call_tool_api, _load_yaml, _schema_to_model


class ToolboxClient:
    def __init__(self, url: str, session: ClientSession):
        """
        Initializes the ToolboxClient for the Toolbox service at the given URL.

        Args:
            url: The base URL of the Toolbox service.
            session: The HTTP client session.
        """
        self._url: str = url
        self._session = session

    async def _load_tool_manifest(self, tool_name: str) -> dict:
        """
        Fetches and parses the YAML manifest for the given tool from the Toolbox service.

        Args:
            tool_name: The name of the tool to load.

        Returns:
            The parsed YAML manifest.
        """
        url = f"{self._url}/api/tool/{tool_name}"
        return await _load_yaml(url, self._session)

    async def _load_toolset_manifest(self, toolset_name: Optional[str] = None) -> dict:
        """
        Fetches and parses the YAML manifest from the Toolbox service.

        Args:
            toolset_name: The name of the toolset to load.
                Default: None. If not provided, then all the available tools are loaded.

        Returns:
            The parsed YAML manifest.
        """
        url = f"{self._url}/api/toolset/{toolset_name or ''}"
        return await _load_yaml(url, self._session)

    def _generate_tool(self, tool_name: str, manifest: dict) -> StructuredTool:
        """
        Creates a StructuredTool object and a dynamically generated BaseModel for the given tool.

        Args:
            tool_name: The name of the tool to generate.
            manifest: The parsed Toolbox manifest.

        Returns:
            The generated tool.
        """
        tool_schema = ToolSchema(**manifest["tools"][tool_name])
        tool_model: BaseModel = _schema_to_model(
            model_name=tool_name, schema=tool_schema.parameters
        )

        async def _tool_func(**kwargs) -> dict:
            return await _call_tool_api(self._url, self._session, tool_name, kwargs)

        return StructuredTool.from_function(
            coroutine=_tool_func,
            name=tool_name,
            description=tool_schema.description,
            args_schema=tool_model,
        )

    async def load_tool(self, tool_name: str) -> StructuredTool:
        """
        Loads tools from the Toolbox service, optionally filtered by toolset name.

        Args:
            toolset_name: The name of the toolset to load.
                Default: None. If not provided, then all the tools are loaded.

        Returns:
            A tool loaded from the Toolbox
        """
        manifest: dict = await self._load_tool_manifest(tool_name)
        return self._generate_tool(tool_name, manifest)

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
        tools: list[StructuredTool] = []
        manifest: dict = await self._load_toolset_manifest(toolset_name)
        for tool_name in manifest["tools"]:
            tools.append(self._generate_tool(tool_name, manifest))
        return tools
