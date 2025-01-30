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

import json
from typing import Any, Callable, Optional, Type, cast
from warnings import warn

from aiohttp import ClientSession
from deprecated import deprecated
from pydantic import BaseModel, Field, create_model


class ParameterSchema(BaseModel):
    """
    Schema for a tool parameter.
    """

    name: str
    type: str
    description: str
    authSources: Optional[list[str]] = None


class ToolSchema(BaseModel):
    """
    Schema for a tool.
    """

    description: str
    parameters: list[ParameterSchema]


class ManifestSchema(BaseModel):
    """
    Schema for the Toolbox manifest.
    """

    serverVersion: str
    tools: dict[str, ToolSchema]


async def _load_manifest(url: str, session: ClientSession) -> ManifestSchema:
    """
    Asynchronously fetches and parses the JSON manifest schema from the given
    URL.

    Args:
        url: The URL to fetch the JSON from.
        session: The HTTP client session.

    Returns:
        The parsed Toolbox manifest.

    Raises:
        json.JSONDecodeError: If the response is not valid JSON.
        ValueError: If the response is not a valid manifest.
    """
    async with session.get(url) as response:
        # TODO: Remove as it masks error messages.
        response.raise_for_status()
        try:
            # TODO: Simply use response.json()
            parsed_json = json.loads(await response.text())
        except json.JSONDecodeError as e:
            raise json.JSONDecodeError(
                f"Failed to parse JSON from {url}: {e}", e.doc, e.pos
            ) from e
        try:
            return ManifestSchema(**parsed_json)
        except ValueError as e:
            raise ValueError(f"Invalid JSON data from {url}: {e}") from e


def _schema_to_model(model_name: str, schema: list[ParameterSchema]) -> Type[BaseModel]:
    """
    Converts the given manifest schema to a Pydantic BaseModel class.

    Args:
        model_name: The name of the model to create.
        schema: The schema to convert.

    Returns:
        A Pydantic BaseModel class.
    """
    field_definitions = {}
    for field in schema:
        field_definitions[field.name] = cast(
            Any,
            (
                # TODO: Remove the hardcoded optional types once optional fields
                # are supported by Toolbox.
                Optional[_parse_type(field.type)],
                Field(description=field.description),
            ),
        )

    return create_model(model_name, **field_definitions)


def _parse_type(type_: str) -> Any:
    """
    Converts a schema type to a JSON type.

    Args:
        type_: The type name to convert.

    Returns:
        A valid JSON type.

    Raises:
        ValueError: If the given type is not supported.
    """

    if type_ == "string":
        return str
    elif type_ == "integer":
        return int
    elif type_ == "number":
        return float
    elif type_ == "boolean":
        return bool
    elif type_ == "array":
        return list
    else:
        raise ValueError(f"Unsupported schema type: {type_}")


@deprecated("Please use `_get_auth_tokens` instead.")
def _get_auth_headers(id_token_getters: dict[str, Callable[[], str]]) -> dict[str, str]:
    """
    Deprecated. Use `_get_auth_tokens` instead.
    """
    return _get_auth_tokens(id_token_getters)


def _get_auth_tokens(id_token_getters: dict[str, Callable[[], str]]) -> dict[str, str]:
    """
    Gets ID tokens for the given auth sources in the getters map and returns
    tokens to be included in tool invocation.

    Args:
        id_token_getters: A dict that maps auth source names to the functions
            that return its ID token.

    Returns:
        A dictionary of tokens to be included in the tool invocation.
    """
    auth_tokens = {}
    for auth_source, get_id_token in id_token_getters.items():
        auth_tokens[f"{auth_source}_token"] = get_id_token()
    return auth_tokens


async def _invoke_tool(
    url: str,
    session: ClientSession,
    tool_name: str,
    data: dict,
    id_token_getters: dict[str, Callable[[], str]],
) -> dict:
    """
    Asynchronously makes an API call to the Toolbox service to invoke a tool.

    Args:
        url: The base URL of the Toolbox service.
        session: The HTTP client session.
        tool_name: The name of the tool to invoke.
        data: The input data for the tool.
        id_token_getters: A dict that maps auth source names to the functions
            that return its ID token.

    Returns:
        A dictionary containing the parsed JSON response from the tool
        invocation.
    """
    url = f"{url}/api/tool/{tool_name}/invoke"
    auth_tokens = _get_auth_tokens(id_token_getters)

    # ID tokens contain sensitive user information (claims). Transmitting these
    # over HTTP exposes the data to interception and unauthorized access. Always
    # use HTTPS to ensure secure communication and protect user privacy.
    if auth_tokens and not url.startswith("https://"):
        warn(
            "Sending ID token over HTTP. User data may be exposed. Use HTTPS for secure communication."
        )

    async with session.post(
        url,
        json=_convert_none_to_empty_string(data),
        headers=auth_tokens,
    ) as response:
        # TODO: Remove as it masks error messages.
        print("DEBUG: Sending post invoke request to tool with headers", auth_tokens)
        response.raise_for_status()
        return await response.json()


def _convert_none_to_empty_string(input_dict):
    """
    Temporary fix to convert None values to empty strings in the input data.
    This is needed because the current version of the Toolbox service does not
    support optional fields.

    TODO: Remove this once optional fields are supported by Toolbox.

    Args:
        input_dict: The input data dictionary.

    Returns:
        A new dictionary with None values replaced by empty strings.
    """
    new_dict = {}
    for key, value in input_dict.items():
        if value is None:
            new_dict[key] = ""
        else:
            new_dict[key] = value
    return new_dict


def _find_auth_params(
    params: list[ParameterSchema],
) -> tuple[list[ParameterSchema], list[ParameterSchema]]:
    """
    Separates parameters into those that are authenticated and those that are not.

    Args:
        params: A list of ParameterSchema objects.

    Returns:
        A tuple containing two lists:
            - auth_params: A list of ParameterSchema objects that require authentication.
            - non_auth_params: A list of ParameterSchema objects that do not require authentication.
    """
    _auth_params: list[ParameterSchema] = []
    _non_auth_params: list[ParameterSchema] = []

    for param in params:
        if param.authSources:
            _auth_params.append(param)
        else:
            _non_auth_params.append(param)

    return (_auth_params, _non_auth_params)


def _find_bound_params(
    params: list[ParameterSchema], bound_params: list[str]
) -> tuple[list[ParameterSchema], list[ParameterSchema]]:
    """
    Separates parameters into those that are bound and those that are not.

    Args:
        params: A list of ParameterSchema objects.
        bound_params: A list of parameter names that are bound.

    Returns:
        A tuple containing two lists:
            - bound_params: A list of ParameterSchema objects whose names are in the bound_params list.
            - non_bound_params: A list of ParameterSchema objects whose names are not in the bound_params list.
    """

    _bound_params: list[ParameterSchema] = []
    _non_bound_params: list[ParameterSchema] = []

    for param in params:
        if param.name in bound_params:
            _bound_params.append(param)
        else:
            _non_bound_params.append(param)

    return (_bound_params, _non_bound_params)
