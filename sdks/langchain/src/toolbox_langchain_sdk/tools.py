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

from copy import deepcopy
from typing import Any, Callable
from warnings import warn

from aiohttp import ClientSession
from langchain_core.tools import StructuredTool
from typing_extensions import Self

from .utils import (
    ParameterSchema,
    ToolSchema,
    _find_auth_params,
    _invoke_tool,
    _schema_to_model,
)


class ToolboxTool(StructuredTool):
    """
    A subclass of LangChain's StructuredTool that supports features specific to
    Toolbox, like authenticated tools.
    """

    def __init__(
        self,
        name: str,
        schema: ToolSchema,
        url: str,
        session: ClientSession,
        auth_tokens: dict[str, Callable[[], str]] = {},
    ) -> None:
        """
        Initializes a ToolboxTool instance.

        Args:
            name: The name of the tool.
            schema: The tool schema.
            url: The base URL of the Toolbox service.
            session: The HTTP client session.
            auth_tokens: A mapping of authentication source names to functions
                that retrieve ID tokens.
        """

        # If the schema is not already a ToolSchema instance, we create one from
        # its attributes. This allows flexibility in how the schema is provided,
        # accepting both a ToolSchema object and a dictionary of schema
        # attributes.
        if not isinstance(schema, ToolSchema):
            schema = ToolSchema(**schema)

        auth_params, non_auth_params = _find_auth_params(schema.parameters)

        # Update the tools schema to validate only the presence of parameters
        # that do not require authentication.
        schema.parameters = non_auth_params

        # Due to how pydantic works, we must initialize the underlying
        # StructuredTool class before assigning values to member variables.
        super().__init__(
            coroutine=self.__tool_func,
            func=None,
            name=name,
            description=schema.description,
            args_schema=_schema_to_model(model_name=name, schema=schema.parameters),
        )

        self._name: str = name
        self._schema: ToolSchema = schema
        self._url: str = url
        self._session: ClientSession = session
        self._auth_tokens: dict[str, Callable[[], str]] = auth_tokens
        self._auth_params: list[ParameterSchema] = auth_params

        # Warn users about any missing authentication so they can add it before
        # tool invocation.
        self.__validate_auth(strict=False)

    async def __tool_func(self, **kwargs: Any) -> dict:
        """
        The coroutine that invokes the tool with the given arguments.

        Args:
            **kwargs: The arguments to the tool.

        Returns:
            A dictionary containing the parsed JSON response from the tool
            invocation.
        """

        # If the tool had parameters that require authentication, then right
        # before invoking that tool, we check whether all these required
        # authentication sources have been registered or not.
        self.__validate_auth()

        return await _invoke_tool(
            self._url, self._session, self._name, kwargs, self._auth_tokens
        )

    def __validate_auth(self, strict: bool = True) -> None:
        """
        Checks if a tool meets the authentication requirements.

        A tool is considered authenticated if all of its parameters meet at
        least one of the following conditions:

            * The parameter has at least one registered authentication source.
            * The parameter requires no authentication.

        Args:
            strict: If True, raises a PermissionError if any required
                authentication sources are not registered. If False, only issues
                a warning.

        Raises:
            PermissionError: If strict is True and any required authentication
                sources are not registered.
        """
        params_missing_auth: list[str] = []

        # Check each parameter for at least 1 required auth source
        for param in self._auth_params:
            assert param.authSources is not None
            has_auth = False
            for src in param.authSources:
                # Find first auth source that is specified
                if src in self._auth_tokens:
                    has_auth = True
                    break
            if not has_auth:
                params_missing_auth.append(param.name)

        if params_missing_auth:
            message = f"Parameter(s) `{', '.join(params_missing_auth)}` of tool {self._name} require authentication, but no valid authentication sources are registered. Please register the required sources before use."

            if strict:
                raise PermissionError(message)
            warn(message)

    def __create_copy(
        self,
        *,
        auth_tokens: dict[str, Callable[[], str]] = {},
    ) -> Self:
        """
        Creates a deep copy of the current ToolboxTool instance, allowing for
        modification of auth tokens.

        This method enables the creation of new tool instances with inherited
        properties from the current instance, while optionally updating the auth
        tokens. This is useful for creating variations of the tool with
        additional auth tokens without modifying the original instance, ensuring
        immutability.

        Args:
            auth_tokens: A dictionary of auth source names to functions that
                retrieve ID tokens. These tokens will be merged with the
                existing auth tokens.

        Returns:
            A new ToolboxTool instance that is a deep copy of the current
            instance, with optionally updated auth tokens.
        """
        return type(self)(
            name=self._name,
            schema=deepcopy(self._schema),
            url=self._url,
            session=self._session,
            auth_tokens={**self._auth_tokens, **auth_tokens},
        )

    def add_auth_tokens(self, auth_tokens: dict[str, Callable[[], str]]) -> Self:
        """
        Registers functions to retrieve ID tokens for the corresponding
        authentication sources.

        Args:
            auth_tokens: A dictionary of authentication source names to the
                functions that return corresponding ID token.

        Returns:
            A new ToolboxTool instance that is a deep copy of the current
            instance, with added auth tokens.
        """

        # Check if the authentication source is already registered.
        dupe_tokens: list[str] = []
        for auth_token, _ in auth_tokens.items():
            if auth_token in self._auth_tokens:
                dupe_tokens.append(auth_token)

        if dupe_tokens:
            raise ValueError(
                f"Authentication source(s) `{', '.join(dupe_tokens)}` already registered in tool `{self._name}`."
            )

        return self.__create_copy(auth_tokens=auth_tokens)

    def add_auth_token(self, auth_source: str, get_id_token: Callable[[], str]) -> Self:
        """
        Registers a function to retrieve an ID token for a given authentication
        source.

        Args:
            auth_source: The name of the authentication source.
            get_id_token: A function that returns the ID token.

        Returns:
            A new ToolboxTool instance that is a deep copy of the current
            instance, with added auth tokens.
        """
        return self.add_auth_tokens({auth_source: get_id_token})
