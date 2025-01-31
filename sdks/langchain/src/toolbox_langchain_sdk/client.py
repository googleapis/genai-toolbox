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
from typing import Any, Callable, Optional, TypeVar, Union, Awaitable

from .async_client import AsyncToolboxClient
from .tools import ToolboxTool

T = TypeVar("T")


class ToolboxClient:
    def __init__(
        self,
        url: str,
        loop: Optional[asyncio.AbstractEventLoop] = None,
    ) -> None:
        """
        Initializes the ToolboxClient for the Toolbox service at the given URL.

        Args:
            url: The base URL of the Toolbox service.
        """
        if loop is None:
            loop = asyncio.new_event_loop()
            thread = Thread(target=loop.run_forever, daemon=True)
            thread.start()

        self.__loop = loop

        async def init_client():
            return AsyncToolboxClient(url, None)

        coro = init_client()

        self.__async_client = asyncio.run_coroutine_threadsafe(
            coro, self.__loop
        ).result()

    async def _run_as_async(self, coro: Awaitable[T]) -> T:
        """Run an async coroutine asynchronously"""
        # If a loop has not been provided, attempt to run in current thread
        if not self.__loop:
            return await coro
        # Otherwise, run in the background thread
        return await asyncio.wrap_future(
            asyncio.run_coroutine_threadsafe(coro, self.__loop)
        )

    def _run_as_sync(self, coro: Awaitable[T]) -> T:
        """Run an async coroutine synchronously"""
        if not self.__loop:
            raise Exception(
                "Engine was initialized without a background loop and cannot call sync methods."
            )
        return asyncio.run_coroutine_threadsafe(coro, self.__loop).result()

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
        return await self._run_as_async(
            self.__async_client.aload_tool(
                tool_name, auth_tokens, auth_headers, bound_params, strict
            )
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
        return await self._run_as_async(
            self.__async_client.aload_toolset(
                toolset_name, auth_tokens, auth_headers, bound_params, strict
            )
        )

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
        return self._run_as_sync(
            self.__async_client.aload_tool(
                tool_name, auth_tokens, auth_headers, bound_params, strict
            )
        )

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
        return self._run_as_sync(
            self.__async_client.aload_toolset(
                toolset_name, auth_tokens, auth_headers, bound_params, strict
            )
        )
