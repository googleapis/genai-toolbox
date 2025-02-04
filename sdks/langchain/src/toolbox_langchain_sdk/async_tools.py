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
from typing import Any, Callable, TypeVar, Union
from warnings import warn

from aiohttp import ClientSession
from langchain_core.tools import BaseTool

from .utils import (
    ToolSchema,
    _find_auth_params,
    _find_bound_params,
    _invoke_tool,
    _schema_to_model,
)

T = TypeVar("T")


class AsyncToolboxTool(BaseTool):
    """
    A subclass of LangChain's BaseTool that supports features specific to
    Toolbox, like bound parameters and authenticated tools.
    """

    def __init__(
        self,
        name: str,
        schema: ToolSchema,
        url: str,
        session: ClientSession,
        auth_tokens: dict[str, Callable[[], str]] = {},
        bound_params: dict[str, Union[Any, Callable[[], Any]]] = {},
        strict: bool = True,
    ) -> None:
        """
        Initializes an AsyncToolboxTool instance.

        Args:
            name: The name of the tool.
            schema: The tool schema.
            url: The base URL of the Toolbox service.
            session: The HTTP client session.
            auth_tokens: A mapping of authentication source names to functions
                that retrieve ID tokens.
            bound_params: A mapping of parameter names to their bound
                values.
            strict: If True, raises a ValueError if any of the given bound
                parameters are missing from the schema or require
                authentication. If False, only issues a warning.
        """

        # If the schema is not already a ToolSchema instance, we create one from
        # its attributes. This allows flexibility in how the schema is provided,
        # accepting both a ToolSchema object and a dictionary of schema
        # attributes.
        if not isinstance(schema, ToolSchema):
            schema = ToolSchema(**schema)

        auth_params, non_auth_params = _find_auth_params(schema.parameters)
        non_auth_bound_params, non_auth_non_bound_params = _find_bound_params(
            non_auth_params, list(bound_params)
        )

        # Check if the user is trying to bind a param that is authenticated or
        # is missing from the given schema.
        auth_bound_params: list[str] = []
        missing_bound_params: list[str] = []
        for bound_param in bound_params:
            if bound_param in [param.name for param in auth_params]:
                auth_bound_params.append(bound_param)
            elif bound_param not in [param.name for param in non_auth_params]:
                missing_bound_params.append(bound_param)

        # Create error messages for any params that are found to be
        # authenticated or missing.
        messages: list[str] = []
        if auth_bound_params:
            messages.append(
                f"Parameter(s) {', '.join(auth_bound_params)} already authenticated and cannot be bound."
            )
        if missing_bound_params:
            messages.append(
                f"Parameter(s) {', '.join(missing_bound_params)} missing and cannot be bound."
            )

        # Join any error messages and raise them as an error or warning,
        # depending on the value of the strict flag.
        if messages:
            message = "\n\n".join(messages)
            if strict:
                raise ValueError(message)
            warn(message)

        # Bind values for parameters present in the schema that don't require
        # authentication.
        bound_params = {
            param_name: param_value
            for param_name, param_value in bound_params.items()
            if param_name in [param.name for param in non_auth_bound_params]
        }

        # Update the tools schema to validate only the presence of parameters
        # that neither require authentication nor are bound.
        schema.parameters = non_auth_non_bound_params

        # Due to how pydantic works, we must initialize the underlying
        # StructuredTool class before assigning values to member variables.
        super().__init__(
            name=name,
            description=schema.description,
            args_schema=_schema_to_model(model_name=name, schema=schema.parameters),
        )

        self.__name = name
        self.__schema = schema
        self.__url = url
        self.__session = session
        self.__auth_tokens = auth_tokens
        self.__auth_params = auth_params
        self.__bound_params = bound_params

        # Warn users about any missing authentication so they can add it before
        # tool invocation.
        self.__validate_auth(strict=False)

    def _run(self, **kwargs: Any) -> dict[str, Any]:
        raise NotImplementedError("Synchronous methods not supported by async tools.")

    async def _arun(self, **kwargs: Any) -> dict[str, Any]:
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

        # Evaluate dynamic parameter values if any
        evaluated_params = {}
        for param_name, param_value in self.__bound_params.items():
            if callable(param_value):
                evaluated_params[param_name] = param_value()
            else:
                evaluated_params[param_name] = param_value

        # Merge bound parameters with the provided arguments
        kwargs.update(evaluated_params)

        return await _invoke_tool(
            self.__url, self.__session, self.__name, kwargs, self.__auth_tokens
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
        for param in self.__auth_params:
            assert param.authSources is not None
            has_auth = False
            for src in param.authSources:

                # Find first auth source that is specified
                if src in self.__auth_tokens:
                    has_auth = True
                    break
            if not has_auth:
                params_missing_auth.append(param.name)

        if params_missing_auth:
            message = f"Parameter(s) `{', '.join(params_missing_auth)}` of tool {self.__name} require authentication, but no valid authentication sources are registered. Please register the required sources before use."

            if strict:
                raise PermissionError(message)
            warn(message)

    def __create_copy(
        self,
        *,
        auth_tokens: dict[str, Callable[[], str]] = {},
        bound_params: dict[str, Union[Any, Callable[[], Any]]] = {},
        strict: bool,
    ) -> "AsyncToolboxTool":
        """
        Creates a copy of the current AsyncToolboxTool instance, allowing for
        modification of auth tokens and bound params.

        This method enables the creation of new tool instances with inherited
        properties from the current instance, while optionally updating the auth
        tokens and bound params. This is useful for creating variations of the
        tool with additional auth tokens or bound params without modifying the
        original instance, ensuring immutability.

        Args:
            auth_tokens: A dictionary of auth source names to functions that
                retrieve ID tokens. These tokens will be merged with the
                existing auth tokens.
            bound_params: A dictionary of parameter names to their
                bound values or functions to retrieve the values. These params
                will be merged with the existing bound params.
            strict: If True, raises a ValueError if any of the given bound
                parameters are missing from the schema or require
                authentication. If False, only issues a warning.

        Returns:
            A new AsyncToolboxTool instance that is a deep copy of the current
            instance, with added auth tokens or bound params.
        """
        new_schema = deepcopy(self.__schema)

        # Reconstruct the complete parameter schema by merging the auth
        # parameters back with the non-auth parameters. This is necessary to
        # accurately validate the new combination of auth tokens and bound
        # params in the constructor of the new AsyncToolboxTool instance, ensuring
        # that any overlaps or conflicts are correctly identified and reported
        # as errors or warnings, depending on the given `strict` flag.
        new_schema.parameters += self.__auth_params
        return AsyncToolboxTool(
            name=self.__name,
            schema=new_schema,
            url=self.__url,
            session=self.__session,
            auth_tokens={**self.__auth_tokens, **auth_tokens},
            bound_params={**self.__bound_params, **bound_params},
            strict=strict,
        )

    def add_auth_tokens(
        self, auth_tokens: dict[str, Callable[[], str]], strict: bool = True
    ) -> "AsyncToolboxTool":
        """
        Registers functions to retrieve ID tokens for the corresponding
        authentication sources.

        Args:
            auth_tokens: A dictionary of authentication source names to the
                functions that return corresponding ID token.
            strict: If True, a ValueError is raised if any of the provided auth
                tokens are already bound. If False, only a warning is issued.

        Returns:
            A new AsyncToolboxTool instance that is a deep copy of the current
            instance, with added auth tokens.

        Raises:
            ValueError: If the provided auth tokens are already registered.
            ValueError: If the provided auth tokens are already bound and strict
                is True.
        """

        # Check if the authentication source is already registered.
        dupe_tokens: list[str] = []
        for auth_token, _ in auth_tokens.items():
            if auth_token in self.__auth_tokens:
                dupe_tokens.append(auth_token)

        if dupe_tokens:
            raise ValueError(
                f"Authentication source(s) `{', '.join(dupe_tokens)}` already registered in tool `{self.__name}`."
            )

        return self.__create_copy(auth_tokens=auth_tokens, strict=strict)

    def add_auth_token(
        self, auth_source: str, get_id_token: Callable[[], str], strict: bool = True
    ) -> "AsyncToolboxTool":
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
    ) -> "AsyncToolboxTool":
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
            A new AsyncToolboxTool instance that is a deep copy of the current
            instance, with added bound params.

        Raises:
            ValueError: If the provided bound params are already bound.
            ValueError: if the provided bound params are not defined in the tool's schema, or require
                authentication, and strict is True.
        """

        # Check if the parameter is already bound.
        dupe_params: list[str] = []
        for param_name, _ in bound_params.items():
            if param_name in self.__bound_params:
                dupe_params.append(param_name)

        if dupe_params:
            raise ValueError(
                f"Parameter(s) `{', '.join(dupe_params)}` already bound in tool `{self.__name}`."
            )

        return self.__create_copy(bound_params=bound_params, strict=strict)

    def bind_param(
        self,
        param_name: str,
        param_value: Union[Any, Callable[[], Any]],
        strict: bool = True,
    ) -> "AsyncToolboxTool":
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
