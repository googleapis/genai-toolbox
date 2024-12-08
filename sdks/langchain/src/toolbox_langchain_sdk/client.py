import asyncio
import warnings
from typing import Callable, Optional, Type

from aiohttp import ClientSession
from langchain_core.tools import StructuredTool
from pydantic import BaseModel

from .utils import ManifestSchema, _invoke_tool, _load_yaml, _schema_to_model


class ToolboxClient:
    def __init__(self, url: str, session: Optional[ClientSession] = None):
        """
        Initializes the ToolboxClient for the Toolbox service at the given URL.

        Args:
            url: The base URL of the Toolbox service.
            session: The HTTP client session.
                Default: None
        """
        self._url: str = url
        self._should_close_session: bool = session is None
        self._id_token_getters: dict[str, Callable[[], str]] = {}
        self._auth_tools: dict[str, set[str]] = {}
        self._session: ClientSession = session or ClientSession()

    async def close(self) -> None:
        """
        Close the Toolbox client and its tools.
        """
        # We check whether _should_close_session is set or not since we do not
        # want to close the session in case the user had passed their own
        # ClientSession object, since then we expect the user to be owning its
        # lifecycle.
        if self._session and self._should_close_session:
            await self._session.close()

    def __del__(self):
        try:
            loop = asyncio.get_event_loop()
            if loop.is_running():
                loop.create_task(self.close())
            else:
                loop.run_until_complete(self.close())
        except Exception:
            # We "pass" assuming that the exception is thrown because  the event
            # loop is no longer running, but at that point the Session should
            # have been closed already anyway.
            pass

    async def _load_tool_manifest(self, tool_name: str) -> ManifestSchema:
        """
        Fetches and parses the YAML manifest for the given tool from the Toolbox
        service.

        Args:
            tool_name: The name of the tool to load.

        Returns:
            The parsed Toolbox manifest.
        """
        url = f"{self._url}/api/tool/{tool_name}"
        return await _load_yaml(url, self._session)

    async def _load_toolset_manifest(
        self, toolset_name: Optional[str] = None
    ) -> ManifestSchema:
        """
        Fetches and parses the YAML manifest from the Toolbox service.

        Args:
            toolset_name: The name of the toolset to load.
                Default: None. If not provided, then all the available tools are
                loaded.

        Returns:
            The parsed Toolbox manifest.
        """
        url = f"{self._url}/api/toolset/{toolset_name or ''}"
        return await _load_yaml(url, self._session)

    def _validate_tool_auth(self, tool_name: str) -> None:
        """
        Validates that all the authentication sources that are required to call
        the given tool are registered.

        Args:
            tool_name: The name of the tool to validate.

        Raises:
            PermissionError: If any of the required authentication sources are
            not registered.
        """
        if tool_name in self._auth_tools:
            missing_auth = []

            # If the tool had parameters that require authentication, then right
            # before invoking that tool, we validate whether all these required
            # authentication sources are registered or not.
            for auth_source in self._auth_tools[tool_name]:
                if auth_source not in self._id_token_getters:
                    missing_auth.append(auth_source)
            if missing_auth:
                raise PermissionError(f"Login required before invoking {tool_name}.")

    def _generate_tool(
        self, tool_name: str, manifest: ManifestSchema
    ) -> StructuredTool:
        """
        Creates a StructuredTool object and a dynamically generated BaseModel
        for the given tool.

        Args:
            tool_name: The name of the tool to generate.
            manifest: The parsed Toolbox manifest.

        Returns:
            The generated tool.
        """
        tool_schema = manifest.tools[tool_name]
        tool_model: Type[BaseModel] = _schema_to_model(
            model_name=tool_name, schema=tool_schema.parameters
        )

        async def _tool_func(**kwargs) -> dict:
            self._validate_tool_auth(tool_name)
            return await _invoke_tool(
                self._url, self._session, tool_name, kwargs, self._id_token_getters
            )

        return StructuredTool.from_function(
            coroutine=_tool_func,
            name=tool_name,
            description=tool_schema.description,
            args_schema=tool_model,
        )

    def _process_auth_params(self, manifest: ManifestSchema) -> None:
        """
        First validates that each auth parameter of each tool in the given
        manifest has at least one auth source that is registered. Then it
        filters out and stores parameters with auth sources from the manifest.

        Args:
            manifest: The manifest to validate and modify.

        Warns:
            UserWarning: If a parameter in the manifest has no authSources that
                         are registered.
        """
        for tool_name, tool_schema in manifest.tools.items():
            non_auth_params = []
            for param in tool_schema.parameters:

                # Identify tools in the manifest that require authentication.
                # Remove these tools from the manifest and store their required
                # auth sources in a map called `_auth_tools`. This allows for
                # efficient validation of auth sources immediately before tool
                # invocation in the `_validate_tool_auth` function.
                if param.authSources:

                    # If all permitted auth sources for a parameter are not yet
                    # registered, raise a warning message to the user.
                    if all(
                        auth_source not in self._id_token_getters
                        for auth_source in param.authSources
                    ):
                        warnings.warn(
                            f"""The parameter {param.name} of tool {tool_name}
                             requires authentication, but none of its permitted
                             auth sources are currently registered. Please
                             ensure the necessary auth source is registered
                             before invoking this tool."""
                        )

                    if tool_name not in self._auth_tools:
                        self._auth_tools[tool_name] = set()
                    self._auth_tools[tool_name].update(param.authSources)
                else:
                    non_auth_params.append(param)
            tool_schema.parameters = non_auth_params

    def add_auth_header(
        self, auth_source: str, get_id_token: Callable[[], str]
    ) -> None:
        """
        Registers a function to retrieve an ID token for a given authentication
        source.

        Args:
            auth_source : The name of the authentication source.
            get_id_token: A function that returns the ID token.
        """
        self._id_token_getters[auth_source] = get_id_token

    async def load_tool(
        self, tool_name: str, auth_headers: dict[str, Callable[[], str]] = {}
    ) -> StructuredTool:
        """
        Loads the tool, with the given tool name, from the Toolbox service.

        Args:
            tool_name: The name of the tool to load.
            auth_headers: A mapping of authentication source names to
                functions that retrieve ID tokens. If provided, these will
                override or be added to the existing ID token getters.
                Default: Empty.

        Returns:
            A tool loaded from the Toolbox
        """
        for auth_source, get_id_token in auth_headers.items():
            self.add_auth_header(auth_source, get_id_token)

        manifest: ManifestSchema = await self._load_tool_manifest(tool_name)

        self._process_auth_params(manifest)

        return self._generate_tool(tool_name, manifest)

    async def load_toolset(
        self,
        toolset_name: Optional[str] = None,
        auth_headers: dict[str, Callable[[], str]] = {},
    ) -> list[StructuredTool]:
        """
        Loads tools from the Toolbox service, optionally filtered by toolset
        name.

        Args:
            toolset_name: The name of the toolset to load.
                Default: None. If not provided, then all the tools are loaded.
            auth_headers: A mapping of authentication source names to
                functions that retrieve ID tokens. If provided, these will
                override or be added to the existing ID token getters.
                Default: Empty.

        Returns:
            A list of all tools loaded from the Toolbox.
        """
        for auth_source, get_id_token in auth_headers.items():
            self.add_auth_header(auth_source, get_id_token)

        tools: list[StructuredTool] = []
        manifest: ManifestSchema = await self._load_toolset_manifest(toolset_name)

        self._process_auth_params(manifest)

        for tool_name in manifest.tools:
            tools.append(self._generate_tool(tool_name, manifest))
        return tools
