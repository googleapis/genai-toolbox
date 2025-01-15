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

from typing import Any, Callable, Union
from warnings import warn

from aiohttp import ClientSession
from llama_index.core.tools import FunctionTool, ToolMetadata

from .utils import ParameterSchema, ToolSchema, _invoke_tool, _schema_to_model


class ToolboxTool(FunctionTool):
    """
    A subclass of LlamaIndex's FunctionTool that supports features specific to
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
            bound_params: A mapping of parameter names to their bound
                values.
        """
        # If the schema is not already a ToolSchema instance, we create one from
        # its attributes. This allows flexibility in how the schema is provided,
        # accepting both a ToolSchema object and a dictionary of schema
        # attributes.
        if not isinstance(schema, ToolSchema):
            schema = ToolSchema(**schema)

        super().__init__(
            async_fn=self.__tool_func,
            metadata=ToolMetadata(
                name=name,
                description=schema.description,
            ),
        )

        self._name: str = name
        self._schema: ToolSchema = schema
        self._url: str = url
        self._session: ClientSession = session
        self._auth_tokens: dict[str, Callable[[], str]] = {}
        self._auth_params: dict[str, list[str]] = {}
        self._bound_params: dict[str, Union[Any, Callable[[], Any]]] = {}

        self.bind_params(bound_params, strict=False)
        self.add_auth_tokens(auth_tokens)
        self.__validate_auth(strict=False)

    def __update_params(self, params: list[ParameterSchema]) -> None:
        """
        Updates the tool's schema with the given parameters and regenerates the
        args schema.

        Args:
            params: The new list of parameters.
        """
        self._schema.parameters = params

        self.metadata.fn_schema = _schema_to_model(
            model_name=self._name, schema=self._schema.parameters
        )

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

        # Evaluate dynamic parameter values if any
        evaluated_params = {}
        for param_name, param_value in self._bound_params.items():
            if callable(param_value):
                evaluated_params[param_name] = param_value()
            else:
                evaluated_params[param_name] = param_value

        # Merge bound parameters with the provided arguments
        kwargs.update(evaluated_params)

        # To ensure data integrity, we added input validation against the
        # function schema, as this is not currently performed by the underlying
        # `FunctionTool`.
        if self.metadata.fn_schema is not None:
            self.metadata.fn_schema.model_validate(kwargs)

        return await _invoke_tool(
            self._url, self._session, self._name, kwargs, self._auth_tokens
        )

    def __process_auth_params(self) -> None:
        """
        Extracts parameters requiring authentication from the schema.

        These parameters are removed from the schema to prevent data validation
        errors since their values are inferred by the Toolbox service, not
        provided by the user.

        The permitted authentication sources for each parameter are stored in
        `_auth_params` for efficient validation in `__validate_auth`.
        """
        non_auth_params: list[ParameterSchema] = []
        for param in self._schema.parameters:
            if param.authSources:
                self._auth_params[param.name] = param.authSources
            else:
                non_auth_params.append(param)

        self.__update_params(non_auth_params)

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

        # check each parameter for at least 1 required auth_source
        for param_name, required_auth in self._auth_params.items():
            has_auth = False
            for src in required_auth:
                # find first auth_source that is specified
                if src in self._auth_tokens:
                    has_auth = True
                    break
            if not has_auth:
                params_missing_auth.append(param_name)

        if params_missing_auth:
            message = f"Parameter(s) `{', '.join(params_missing_auth)}` of tool {self._name} require authentication, but no valid authentication sources are registered. Please register the required sources before use."

            if strict:
                raise PermissionError(message)
            warn(message)

    def __remove_bound_params(self) -> None:
        """
        Removes parameters bound to a value from the tool schema.

        This is to prevent data validation errors since their values are
        inferred by the SDK, not provided by the user.

        Raises:
            ValueError: If attempting to bind a value on an authenticated tool.
        """
        non_bound_params: list[ParameterSchema] = []
        for param in self._schema.parameters:
            if param.name not in self._bound_params:
                non_bound_params.append(param)

        self.__update_params(non_bound_params)

    def add_auth_tokens(self, auth_tokens: dict[str, Callable[[], str]]) -> None:
        """
        Registers functions to retrieve ID tokens for the corresponding
        authentication sources.

        Args:
            auth_tokens: A dictionary of authentication source names to the
                functions that return corresponding ID token.

        Raises:
            ValueError: If any of the given authentication sources are already
                registered.
        """
        dupe_sources: list[str] = []
        for auth_source, get_id_token in auth_tokens.items():

            # Check if the authentication source is already registered.
            if auth_source in self._auth_tokens:
                dupe_sources.append(auth_source)
                continue

            self._auth_tokens[auth_source] = get_id_token

        # Remove auth params from the schema to prevent data validation errors
        # since their values are inferred by the Toolbox service, not provided
        # by the user.
        self.__process_auth_params()

        if dupe_sources:
            raise ValueError(
                f"Authentication source(s) `{', '.join(dupe_sources)}` already registered in tool `{self._name}`."
            )

    def add_auth_token(self, auth_source: str, get_id_token: Callable[[], str]) -> None:
        """
        Registers a function to retrieve an ID token for a given authentication
        source.

        Args:
            auth_source: The name of the authentication source.
            get_id_token: A function that returns the ID token.

        Raises:
            ValueError: If the given authentication source is already
                registered.
        """
        return self.add_auth_tokens({auth_source: get_id_token})

    def bind_params(
        self,
        bound_params: dict[str, Union[Any, Callable[[], Any]]],
        strict: bool = True,
    ) -> None:
        """
        Registers values or functions to retrieve the value for the
        corresponding bound parameters.

        Args:
            bound_params: A dictionary of the bound parameter name to the
                value or function of the bound value.
            strict: If True, raises a ValueError if the parameter is not
                present in the tool's schema.

        Raises:
            ValueError: If the given parameter is already bound, or if strict
                is True and the parameter is not present in the tool's schema.
        """
        dupe_params: list[str] = []
        invalid_params: list[str] = []
        missing_params: list[str] = []
        for param_name, param_value in bound_params.items():

            # Check if the parameter is already bound.
            if param_name in self._bound_params:
                dupe_params.append(param_name)
                continue

            # Check if the parameter has authSources set OR is already present
            # in _auth_params.
            if param_name in self._auth_params or any(
                param.authSources
                for param in self._schema.parameters
                if param.name == param_name
            ):
                invalid_params.append(param_name)
                continue

            # Check if the parameter is missing from the tool schema.
            if not param_name in [param.name for param in self._schema.parameters]:
                missing_params.append(param_name)
                continue

            self._bound_params[param_name] = param_value

        # Bound parameters are handled internally, so remove them from the
        # schema to prevent validation errors and present a cleaner schema in
        # the tool.
        self.__remove_bound_params()

        if dupe_params:
            raise ValueError(
                f"Parameter(s) `{', '.join(dupe_params)}` already bound in tool `{self._name}`."
            )
        if invalid_params:
            raise ValueError(
                f"Value(s) for `{', '.join(invalid_params)}` automatically handled by authentication source(s) and cannot be modified."
            )
        if missing_params:
            message = f"Parameter(s) `{', '.join(missing_params)}` not existing in tool `{self._name}`."
            if strict:
                raise ValueError(message)
            warn(message)

    def bind_param(
        self,
        param_name: str,
        param_value: Union[Any, Callable[[], Any]],
        strict: bool = True,
    ) -> None:
        """
        Registers a value or a function to retrieve the value for a given
        bound parameter.

        Args:
            param_name: The name of the bound parameter.
            param_value: The value of the bound parameter, or a callable
                that returns the value.
            strict: If True, raises a ValueError if the parameter is not
                present in the tool's schema.

        Raises:
            ValueError: If the given parameter is already bound, or if strict
                is True and the parameter is not present in the tool's schema.
        """
        return self.bind_params({param_name: param_value}, strict)
