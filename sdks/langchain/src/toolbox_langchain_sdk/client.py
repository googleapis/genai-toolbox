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
from threading import Thread
from typing import Any, Awaitable, Callable, Optional, TypeVar, Union

from aiohttp import ClientSession

from .async_client import AsyncToolboxClient
from .tools import ToolboxTool

T = TypeVar("T")


class ToolboxClient:
    __session: Optional[ClientSession] = None
    __loop: Optional[asyncio.AbstractEventLoop] = None
    __thread: Optional[Thread] = None

    def __init__(
        self,
        url: str,
    ) -> None:
        """
        Initializes the ToolboxClient for the Toolbox service at the given URL.

        Args:
            url: The base URL of the Toolbox service.
        """

        # Running a loop in a background thread allows us to support async
        # methods from non-async environments.
        if ToolboxClient.__loop is None:
            loop = asyncio.new_event_loop()
            thread = Thread(target=loop.run_forever, daemon=True)
            thread.start()
            ToolboxClient.__thread = thread
            ToolboxClient.__loop = loop

        async def __start_session(url: str) -> None:
            # Use a default session if none is provided. This leverages connection
            # pooling for better performance by reusing a single session throughout
            # the application's lifetime.
            if ToolboxClient.__session is None:
                ToolboxClient.__session = ClientSession()

        coro = __start_session(url)

        assert ToolboxClient.__loop
        asyncio.run_coroutine_threadsafe(coro, ToolboxClient.__loop).result()

        self.__async_client = AsyncToolboxClient(url, ToolboxClient.__session)

    def __run_as_sync(self, coro: Awaitable[T]) -> T:
        """Run an async coroutine synchronously"""
        if not self.__loop:
            raise Exception(
                "Cannot call synchronous methods before the background loop is initialized."
            )
        return asyncio.run_coroutine_threadsafe(coro, self.__loop).result()

    async def __run_as_async(self, coro: Awaitable[T]) -> T:
        """Run an async coroutine asynchronously"""

        # If a loop has not been provided, attempt to run in current thread.
        if not self.__loop:
            return await coro

        # Otherwise, run in the background thread.
        return await asyncio.wrap_future(
            asyncio.run_coroutine_threadsafe(coro, self.__loop)
        )

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
        async_tool = await self.__run_as_async(
            self.__async_client.aload_tool(
                tool_name, auth_tokens, auth_headers, bound_params, strict
            )
        )

        return ToolboxTool(async_tool, self.__loop, self.__thread)

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
        async_tools = await self.__run_as_async(
            self.__async_client.aload_toolset(
                toolset_name, auth_tokens, auth_headers, bound_params, strict
            )
        )

        tools: list[ToolboxTool] = []
        for async_tool in async_tools:
            tools.append(ToolboxTool(async_tool, self.__loop, self.__thread))
        return tools

    def load_tool(
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
        async_tool = self.__run_as_sync(
            self.__async_client.aload_tool(
                tool_name, auth_tokens, auth_headers, bound_params, strict
            )
        )

        return ToolboxTool(async_tool, self.__loop, self.__thread)

    def load_toolset(
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
        async_tools = self.__run_as_sync(
            self.__async_client.aload_toolset(
                toolset_name, auth_tokens, auth_headers, bound_params, strict
            )
        )

        tools: list[ToolboxTool] = []
        for async_tool in async_tools:
            tools.append(ToolboxTool(async_tool, self.__loop, self.__thread))
        return tools
