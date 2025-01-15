from typing import Any, Callable
from warnings import warn

from aiohttp import ClientSession
from langchain_core.tools import StructuredTool
from pydantic import BaseModel

from .utils import ParameterSchema, ToolSchema, _invoke_tool, _schema_to_model


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

        super().__init__(
            coroutine=self._tool_func,
            func=None,
            name=name,
            description=schema.description,
            args_schema=BaseModel,
        )

        self._name: str = name
        self._schema: ToolSchema = schema
        self._url: str = url
        self._session: ClientSession = session
        self._auth_tokens: dict[str, Callable[[], str]] = {}
        self._auth_params: dict[str, list[str]] = {}

        self.add_auth_tokens(auth_tokens)
        self.__validate_auth(strict=False)

    def _update_params(self, params: list[ParameterSchema]) -> None:
        """
        Updates the tool's schema with the given parameters and regenerates the
        args schema.

        Args:
            params: The new list of parameters.
        """
        self._schema.parameters = params

        self.args_schema = _schema_to_model(
            model_name=self._name, schema=self._schema.parameters
        )

    async def _tool_func(self, **kwargs: Any) -> dict:
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

        self._update_params(non_auth_params)

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
        unauth_params: list[str] = []

        for param_name, auth_sources in self._auth_params.items():
            found_match = False
            for registered_auth_source in self._auth_tokens:
                if registered_auth_source in auth_sources:
                    found_match = True
                    break
            if not found_match:
                unauth_params.append(param_name)

        if unauth_params:
            message = f"Parameter(s) `{', '.join(unauth_params)}` of tool {self._name} require authentication, but no valid authentication sources are registered. Please register the required sources before use."

            if strict:
                raise PermissionError(message)
            warn(message)

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
