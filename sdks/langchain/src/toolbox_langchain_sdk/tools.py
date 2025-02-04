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
from asyncio import AbstractEventLoop
from threading import Thread
from typing import Any, Awaitable, Callable, TypeVar, Union

from langchain_core.tools import BaseTool

from .async_tools import AsyncToolboxTool

T = TypeVar("T")


class ToolboxTool(BaseTool):
    """
    A subclass of LangChain's BaseTool that supports features specific to
    Toolbox, like bound parameters and authenticated tools.
    """

    def __init__(
        self,
        async_tool: AsyncToolboxTool,
        loop: AbstractEventLoop,
        thread: Thread,
    ) -> None:
        """
        Initializes a ToolboxTool instance.

        Args:
            async_tool: The underlying AsyncToolboxTool instance.
            loop: The event loop used to run asynchronous tasks.
            thread: The thread to run blocking operations in.
        """

        # Due to how pydantic works, we must initialize the underlying
        # StructuredTool class before assigning values to member variables.
        super().__init__(
            name=async_tool.name,
            description=async_tool.description,
            args_schema=async_tool.args_schema,
        )

        self.__async_tool = async_tool
        self.__loop = loop
        self.__thread = thread

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

    def _run(self, **kwargs: Any) -> dict[str, Any]:
        return self.__run_as_sync(self.__async_tool._arun(**kwargs))

    async def _arun(self, **kwargs: Any) -> dict[str, Any]:
        return await self.__run_as_async(self.__async_tool._arun(**kwargs))

    def add_auth_tokens(
        self, auth_tokens: dict[str, Callable[[], str]], strict: bool = True
    ) -> "ToolboxTool":
        """
        Registers functions to retrieve ID tokens for the corresponding
        authentication sources.

        Args:
            auth_tokens: A dictionary of authentication source names to the
                functions that return corresponding ID token.
            strict: If True, a ValueError is raised if any of the provided auth
                tokens are already bound. If False, only a warning is issued.

        Returns:
            A new ToolboxTool instance that is a deep copy of the current
            instance, with added auth tokens.

        Raises:
            ValueError: If the provided auth tokens are already registered.
            ValueError: If the provided auth tokens are already bound and strict
                is True.
        """
        return ToolboxTool(
            self.__async_tool.add_auth_tokens(auth_tokens, strict),
            self.__loop,
            self.__thread,
        )

    def add_auth_token(
        self, auth_source: str, get_id_token: Callable[[], str], strict: bool = True
    ) -> "ToolboxTool":
        """
        Registers a function to retrieve an ID token for a given authentication
        source.

        Args:
            auth_source: The name of the authentication source.
            get_id_token: A function that returns the ID token.
            strict: If True, a ValueError is raised if any of the provided auth
                token is already bound. If False, only a warning is issued.

        Returns:
            A new ToolboxTool instance that is a deep copy of the current
            instance, with added auth token.

        Raises:
            ValueError: If the provided auth token is already registered.
            ValueError: If the provided auth token is already bound and strict
                is True.
        """
        return ToolboxTool(
            self.__async_tool.add_auth_token(auth_source, get_id_token, strict),
            self.__loop,
            self.__thread,
        )

    def bind_params(
        self,
        bound_params: dict[str, Union[Any, Callable[[], Any]]],
        strict: bool = True,
    ) -> "ToolboxTool":
        """
        Registers values or functions to retrieve the value for the
        corresponding bound parameters.

        Args:
            bound_params: A dictionary of the bound parameter name to the
                value or function of the bound value.
            strict: If True, a ValueError is raised if any of the provided bound
                params are not defined in the tool's schema, or require
                authentication. If False, only a warning is issued.

        Returns:
            A new ToolboxTool instance that is a deep copy of the current
            instance, with added bound params.

        Raises:
            ValueError: If the provided bound params are already bound.
            ValueError: if the provided bound params are not defined in the tool's schema, or require
                authentication, and strict is True.
        """
        return ToolboxTool(
            self.__async_tool.bind_params(bound_params, strict),
            self.__loop,
            self.__thread,
        )

    def bind_param(
        self,
        param_name: str,
        param_value: Union[Any, Callable[[], Any]],
        strict: bool = True,
    ) -> "ToolboxTool":
        """
        Registers a value or a function to retrieve the value for a given bound
        parameter.

        Args:
            param_name: The name of the bound parameter. param_value: The value
            of the bound parameter, or a callable that
                returns the value.
            strict: If True, a ValueError is raised if any of the provided bound
                params is not defined in the tool's schema, or requires
                authentication. If False, only a warning is issued.

        Returns:
            A new ToolboxTool instance that is a deep copy of the current
            instance, with added bound param.

        Raises:
            ValueError: If the provided bound param is already bound.
            ValueError: if the provided bound param is not defined in the tool's
                schema, or requires authentication, and strict is True.
        """
        return ToolboxTool(
            self.__async_tool.bind_param(param_name, param_value, strict),
            self.__loop,
            self.__thread,
        )
