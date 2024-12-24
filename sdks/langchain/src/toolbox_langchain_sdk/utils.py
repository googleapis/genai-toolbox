import json
import warnings
from typing import Any, Callable, Optional, Type, cast

from aiohttp import ClientSession
from pydantic import BaseModel, Field, create_model


class ParameterSchema(BaseModel):
    name: str
    type: str
    description: str
    authSources: Optional[list[str]] = None


class ToolSchema(BaseModel):
    description: str
    parameters: list[ParameterSchema]


class ManifestSchema(BaseModel):
    serverVersion: str
    tools: dict[str, ToolSchema]


async def _load_manifest(url: str, session: ClientSession) -> ManifestSchema:
    """
    Asynchronously fetches and parses the JSON manifest schema from the given
    URL.

    Args:
        url: The base URL to fetch the JSON from.
        session: The HTTP client session

    Returns:
        The parsed Toolbox manifest.
    """
    async with session.get(url) as response:
        response.raise_for_status()
        try:
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


def _get_auth_headers(id_token_getters: dict[str, Callable[[], str]]) -> dict[str, str]:
    """
    Gets id tokens for the given auth sources in the getters map and returns
    headers to be included in tool invocation.

    Args:
        id_token_getters: A dict that maps auth source names to the functions
        that return its ID token.

    Returns:
        A dictionary of headers to be included in the tool invocation.
    """
    auth_headers = {}
    for auth_source, get_id_token in id_token_getters.items():
        auth_headers[f"{auth_source}_token"] = get_id_token()
    return auth_headers


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
    auth_headers = _get_auth_headers(id_token_getters)

    # ID tokens contain sensitive user information (claims). Transmitting these
    # over HTTP exposes the data to interception and unauthorized access. Always
    # use HTTPS to ensure secure communication and protect user privacy.
    if auth_headers and not url.startswith("https://"):
        warnings.warn(
            "Sending ID token over HTTP. User data may be exposed. Use HTTPS for secure communication."
        )

    async with session.post(
        url,
        json=_convert_none_to_empty_string(data),
        headers=auth_headers,
    ) as response:
        response.raise_for_status()
        return await response.json()


# TODO: Remove this temporary fix once optional fields are supported by Toolbox.
def _convert_none_to_empty_string(input_dict):
    new_dict = {}
    for key, value in input_dict.items():
        if value is None:
            new_dict[key] = ""
        else:
            new_dict[key] = value
    return new_dict
