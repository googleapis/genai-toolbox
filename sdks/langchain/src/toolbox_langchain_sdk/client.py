# Copyright 2024 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import asyncio
from typing import Any, Callable, Optional, Union
from warnings import warn

from aiohttp import ClientSession

from .tools import ToolboxTool
from .utils import ManifestSchema, _load_manifest


class ToolboxClient:
    def __init__(self, url: str, session: Optional[ClientSession] = None):
        """
        Initializes the ToolboxClient for the Toolbox service at the given URL.

        Args:
            url: The base URL of the Toolbox service.
            session: An optional HTTP client session. If not provided, a new
                session will be created.
        """
        self._url: str = url
        self._should_close_session: bool = session is None
        self._session: ClientSession = session or ClientSession()

    async def close(self) -> None:
        """
        Closes the HTTP client session if it was created by this client.
        """
        # We check whether _should_close_session is set or not since we do not
        # want to close the session in case the user had passed their own
        # ClientSession object, since then we expect the user to be owning its
        # lifecycle.
        if self._session and self._should_close_session:
            await self._session.close()

    def __del__(self):
        """
        Ensures the HTTP client session is closed when the client is garbage
        collected.
        """
        try:
            loop = asyncio.get_event_loop()
            if loop.is_running():
                loop.create_task(self.close())
            else:
                loop.run_until_complete(self.close())
        except Exception:
            # We "pass" assuming that the exception is thrown because the event
            # loop is no longer running, but at that point the Session should
            # have been closed already anyway.
            pass

    async def _load_tool_manifest(self, tool_name: str) -> ManifestSchema:
        """
        Fetches and parses the manifest schema for the given tool from the
        Toolbox service.

        Args:
            tool_name: The name of the tool to load.

        Returns:
            The parsed Toolbox manifest.
        """
        url = f"{self._url}/api/tool/{tool_name}"
        return await _load_manifest(url, self._session)

    async def _load_toolset_manifest(
        self, toolset_name: Optional[str] = None
    ) -> ManifestSchema:
        """
        Fetches and parses the manifest schema from the Toolbox service.

        Args:
            toolset_name: The name of the toolset to load. If not provided,
                the manifest for all available tools is loaded.

        Returns:
            The parsed Toolbox manifest.
        """
        url = f"{self._url}/api/toolset/{toolset_name or ''}"
        return await _load_manifest(url, self._session)

    async def load_tool(
        self,
        tool_name: str,
        auth_tokens: dict[str, Callable[[], str]] = {},
        auth_headers: Optional[dict[str, Callable[[], str]]] = None,
    ) -> ToolboxTool:
        """
        Loads the tool with the given tool name from the Toolbox service.

        Args:
            tool_name: The name of the tool to load.
            auth_tokens: An optional mapping of authentication source names to
                functions that retrieve ID tokens.
            auth_headers: Deprecated. Use `auth_tokens` instead.

        Returns:
            A tool loaded from the Toolbox.
        """
        if auth_headers:
            if auth_tokens:
                warn(
                    "Both `auth_tokens` and `auth_headers` are provided. `auth_headers` is deprecated, and `auth_tokens` will be used.",
                    DeprecationWarning,
                )
            else:
                warn(
                    "Argument `auth_headers` is deprecated. Use `auth_tokens` instead.",
                    DeprecationWarning,
                )
                auth_tokens = auth_headers

        manifest: ManifestSchema = await self._load_tool_manifest(tool_name)
        return ToolboxTool(
            tool_name,
            manifest.tools[tool_name],
            self._url,
            self._session,
            auth_tokens,
        )

    async def load_toolset(
        self,
        toolset_name: Optional[str] = None,
        auth_tokens: dict[str, Callable[[], str]] = {},
        auth_headers: Optional[dict[str, Callable[[], str]]] = None,
    ) -> list[ToolboxTool]:
        """
        Loads tools from the Toolbox service, optionally filtered by toolset
        name.

        Args:
            toolset_name: The name of the toolset to load. If not provided,
                all tools are loaded.
            auth_tokens: An optional mapping of authentication source names to
                functions that retrieve ID tokens.
            auth_headers: Deprecated. Use `auth_tokens` instead.

        Returns:
            A list of all tools loaded from the Toolbox.
        """
        if auth_headers:
            if auth_tokens:
                warn(
                    "Both `auth_tokens` and `auth_headers` are provided. `auth_headers` is deprecated, and `auth_tokens` will be used.",
                    DeprecationWarning,
                )
            else:
                warn(
                    "Argument `auth_headers` is deprecated. Use `auth_tokens` instead.",
                    DeprecationWarning,
                )
                auth_tokens = auth_headers

        tools: list[ToolboxTool] = []
        manifest: ManifestSchema = await self._load_toolset_manifest(toolset_name)

        for tool_name, tool_schema in manifest.tools.items():
            tools.append(
                ToolboxTool(
                    tool_name,
                    tool_schema,
                    self._url,
                    self._session,
                    auth_tokens,
                )
            )
        return tools
