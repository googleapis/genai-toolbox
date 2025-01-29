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
from concurrent.futures import Future
from threading import Thread
from typing import Any, Callable, Optional, TypeVar, Union

from .async_client import AsyncToolboxClient
from .background_loop import _BackgroundLoop
from .tools import ToolboxTool

T = TypeVar("T")


class ToolboxClient:
    __bg_loop: Optional[_BackgroundLoop] = None
    __create_key = object()

    def __init__(
        self,
        key: object,
        url: str,
    ) -> None:
        """
        Initializes the ToolboxClient for the Toolbox service at the given URL.

        Args:
            key: Prevent direct constructor usage.
            url: The base URL of the Toolbox service.
        """

        if key != ToolboxClient.__create_key:
            raise Exception("Only create class through 'create' method!")

        # Rely on AsyncToolboxClient's default session for managing its own
        # connections.
        self.__async_client = AsyncToolboxClient.create(
            url, None, self.__class__.__bg_loop
        )

    @classmethod
    async def __create(cls: type["ToolboxClient"], url: str) -> "ToolboxClient":
        return cls(cls.__create_key, url)

    @classmethod
    def __start_background_loop(cls: type["ToolboxClient"], url: str) -> Future:

        # Running a loop in a background thread allows us to support async
        # methods from non-async environments.
        if cls.__bg_loop is None:
            loop = asyncio.new_event_loop()
            thread = Thread(target=loop.run_forever, daemon=True)
            thread.start()
            cls.__bg_loop = _BackgroundLoop(loop, thread)
        coro = cls.__create(url)

        if cls.__bg_loop._loop is None:
            raise RuntimeError(
                "Background loop not initialized. ToolboxClient was not properly initialized."
            )

        return asyncio.run_coroutine_threadsafe(coro, cls.__bg_loop._loop)

    @classmethod
    def create(cls: type["ToolboxClient"], url: str) -> "ToolboxClient":
        """
        Create a ToolboxClient for the Toolbox service at the given URL.

        Args:
            url: The base URL of the Toolbox service.

        Returns:
            ToolboxClient: A newly created ToolboxClient instance.
        """

        future = cls.__start_background_loop(url)
        return future.result()

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

        if self.__bg_loop is None:
            raise RuntimeError(
                "Background loop not initialized. ToolboxClient was not properly initialized."
            )
        return await self.__bg_loop.run_as_async(
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
        if self.__bg_loop is None:
            raise RuntimeError(
                "Background loop not initialized. ToolboxClient was not properly initialized."
            )
        return await self.__bg_loop.run_as_async(
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

        if self.__bg_loop is None:
            raise RuntimeError(
                "Background loop not initialized. ToolboxClient was not properly initialized."
            )

        return self.__bg_loop.run_as_sync(
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
        if self.__bg_loop is None:
            raise RuntimeError(
                "Background loop not initialized. ToolboxClient was not properly initialized."
            )

        return self.__bg_loop.run_as_sync(
            self.__async_client.aload_toolset(
                toolset_name, auth_tokens, auth_headers, bound_params, strict
            )
        )
