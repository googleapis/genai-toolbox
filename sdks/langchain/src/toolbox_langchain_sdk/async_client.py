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

from typing import Any, Callable, Optional, Union
from warnings import warn

from aiohttp import ClientSession

from .background_loop import _BackgroundLoop
from .tools import ToolboxTool
from .utils import ManifestSchema, _load_manifest


class AsyncToolboxClient:
    __default_session: Optional[ClientSession] = None
    __create_key = object()

    def __init__(
        self,
        key: object,
        url: str,
        session: ClientSession,
        bg_loop: Optional[_BackgroundLoop] = None,
    ):
        """
        Initializes the AsyncToolboxClient for the Toolbox service at the given URL.

        Args:
            key: Prevent direct constructor usage.
            url: The base URL of the Toolbox service.
            session: An HTTP client session.
            bg_loop: Optional background async event loop used to create ToolboxTool.
        """
        if key != AsyncToolboxClient.__create_key:
            raise Exception("Only create class through 'create' method!")

        self.__url = url
        self.__session = session
        self.__bg_loop = bg_loop

    @classmethod
    def create(
        cls: type["AsyncToolboxClient"],
        url: str,
        session: Optional[ClientSession] = None,
        bg_loop: Optional[_BackgroundLoop] = None,
    ) -> "AsyncToolboxClient":

        # Use a default session if none is provided. This leverages connection
        # pooling for better performance by reusing a single session throughout
        # the application's lifetime.
        if session is None:
            if cls.__default_session is None:
                cls.__default_session = ClientSession()
            session = cls.__default_session
        return AsyncToolboxClient(cls.__create_key, url, session, bg_loop)

    async def aload_tool(
        self,
        tool_name: str,
        auth_tokens: dict[str, Callable[[], str]] = {},
        auth_headers: Optional[dict[str, Callable[[], str]]] = None,
        bound_params: dict[str, Union[Any, Callable[[], Any]]] = {},
        strict: bool = True,
    ) -> ToolboxTool:
        """
        Loads the tool with the given tool name from the Toolbox service.

        Args:
            tool_name: The name of the tool to load.
            auth_tokens: An optional mapping of authentication source names to
                functions that retrieve ID tokens.
            auth_headers: Deprecated. Use `auth_tokens` instead.
            bound_params: An optional mapping of parameter names to their
                bound values.
            strict: If True, raises a ValueError if any of the given bound
                parameters are missing from the schema or require
                authentication. If False, only issues a warning.

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

        url = f"{self.__url}/api/tool/{tool_name}"
        manifest: ManifestSchema = await _load_manifest(url, self.__session)

        assert self.__bg_loop
        return ToolboxTool(
            tool_name,
            manifest.tools[tool_name],
            self.__url,
            self.__session,
            self.__bg_loop,
            auth_tokens,
            bound_params,
            strict,
        )

    async def aload_toolset(
        self,
        toolset_name: Optional[str] = None,
        auth_tokens: dict[str, Callable[[], str]] = {},
        auth_headers: Optional[dict[str, Callable[[], str]]] = None,
        bound_params: dict[str, Union[Any, Callable[[], Any]]] = {},
        strict: bool = True,
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
            bound_params: An optional mapping of parameter names to their
                bound values.
            strict: If True, raises a ValueError if any of the given bound
                parameters are missing from the schema or require
                authentication. If False, only issues a warning.

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

        url = f"{self.__url}/api/toolset/{toolset_name or ''}"
        manifest: ManifestSchema = await _load_manifest(url, self.__session)
        tools: list[ToolboxTool] = []

        assert self.__bg_loop
        for tool_name, tool_schema in manifest.tools.items():
            tools.append(
                ToolboxTool(
                    tool_name,
                    tool_schema,
                    self.__url,
                    self.__session,
                    self.__bg_loop,
                    auth_tokens,
                    bound_params,
                    strict,
                )
            )
        return tools

    def load_tool(
        self,
        tool_name: str,
        auth_tokens: dict[str, Callable[[], str]] = {},
        auth_headers: Optional[dict[str, Callable[[], str]]] = None,
        bound_params: dict[str, Union[Any, Callable[[], Any]]] = {},
        strict: bool = True,
    ) -> ToolboxTool:
        raise NotImplementedError(
            "You can use the ToolboxClient to call synchronous methods."
        )

    def load_toolset(
        self,
        toolset_name: Optional[str] = None,
        auth_tokens: dict[str, Callable[[], str]] = {},
        auth_headers: Optional[dict[str, Callable[[], str]]] = None,
        bound_params: dict[str, Union[Any, Callable[[], Any]]] = {},
        strict: bool = True,
    ) -> list[ToolboxTool]:
        raise NotImplementedError(
            "You can use the ToolboxClient to call synchronous methods."
        )
