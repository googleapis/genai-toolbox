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
from typing import Any, Callable, Optional, TypeVar, Union

from aiohttp import ClientSession
from langchain_core.tools import BaseTool

from .async_tools import AsyncToolboxTool
from .background_loop import _BackgroundLoop
from .utils import ToolSchema, _schema_to_model

T = TypeVar("T")


class ToolboxTool(BaseTool):
    """
    A subclass of LangChain's BaseTool that supports features specific to
    Toolbox, like bound parameters and authenticated tools.
    """

    __bg_loop: Optional[_BackgroundLoop] = None

    def __init__(
        self,
        name: str,
        schema: Union[ToolSchema, dict[str, Any]],
        url: str,
        session: ClientSession,
        bg_loop: Optional[_BackgroundLoop] = None,
        auth_tokens: dict[str, Callable[[], str]] = {},
        bound_params: dict[str, Union[Any, Callable[[], Any]]] = {},
        strict: bool = True,
    ) -> None:
        """
        Initializes a ToolboxTool instance.

        Args:
            name: The name of the tool.
            schema: The tool schema.
            url: The base URL of the Toolbox service.
            session: The HTTP client session.
            bg_loop: Optional background async event loop.
            auth_tokens: A mapping of authentication source names to functions
                that retrieve ID tokens.
            bound_params: A mapping of parameter names to their bound
                values.
            strict: If True, raises a ValueError if any of the given bound
                parameters are missing from the schema or require
                authentication. If False, only issues a warning.
        """
        if not isinstance(schema, ToolSchema):
            schema = ToolSchema(**schema)

        super().__init__(
            name=name,
            description=schema.description,
            args_schema=_schema_to_model(model_name=name, schema=schema.parameters),
        )
        if not bg_loop:
            if not self.__class__.__bg_loop:
                loop = asyncio.new_event_loop()
                thread = Thread(target=loop.run_forever, daemon=True)
                thread.start()
                bg_loop = _BackgroundLoop(loop, thread)
            else:
                bg_loop = self.__class__.__bg_loop
        self.__async_tool = AsyncToolboxTool(
            name, schema, url, session, auth_tokens, bound_params, strict
        )
        self.__class__.__bg_loop = bg_loop

    def __from_async_tool(
        self, async_tool: AsyncToolboxTool, strict: bool = False
    ) -> "ToolboxTool":
        """Creates a ToolboxTool from an AsyncToolboxTool (factory method)."""
        return ToolboxTool(
            name=async_tool._name,
            schema=async_tool._schema,
            url=async_tool._url,
            session=async_tool._session,
            auth_tokens=async_tool._auth_tokens,
            bound_params=async_tool._bound_params,
            strict=strict,
        )

    def _run(self, **kwargs: Any) -> dict[str, Any]:
        """Synchronous tool invocation."""
        loop = self.__class__.__bg_loop
        print(f"DEBUG: Trying to invoke a sync tool with name {self.__async_tool._name}")
        if loop is None:
            raise RuntimeError("Background loop is not running.")
        return loop.run_as_sync(self.__async_tool._arun(**kwargs))

    async def _arun(self, **kwargs: Any) -> Any:
        """async tool invocation."""
        loop = self.__class__.__bg_loop
        if loop is None:
            raise RuntimeError("Background loop is not running.")
        return await loop.run_as_async(self.__async_tool._arun(**kwargs))

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

        loop = self.__class__.__bg_loop
        if loop is None:
            raise RuntimeError("Background loop is not running.")
        async_tool = self.__async_tool.add_auth_tokens(auth_tokens, strict)
        return self.__from_async_tool(async_tool)

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
        return self.add_auth_tokens({auth_source: get_id_token}, strict=strict)

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

        loop = self.__class__.__bg_loop
        if loop is None:
            raise RuntimeError("Background loop is not running.")
        async_tool = self.__async_tool.bind_params(bound_params, strict)
        return self.__from_async_tool(async_tool)

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
        return self.bind_params({param_name: param_value}, strict)
