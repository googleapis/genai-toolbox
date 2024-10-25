from typing import Any, Type

import yaml
from aiohttp import ClientSession
from pydantic import BaseModel, Field, create_model


class ParameterSchema(BaseModel):
    name: str
    type: str
    description: str


class ToolSchema(BaseModel):
    description: str
    parameters: list[ParameterSchema]


async def _load_yaml(url: str, session: ClientSession) -> dict:
    """
    Asynchronously fetches and parses the YAML data from the given URL.

    Args:
        url: The base URL to fetch the YAML from.
        session: The HTTP client session

    Returns:
        A dictionary representing the parsed YAML data.
    """
    async with session.get(url) as response:
        response.raise_for_status()
        return yaml.safe_load(await response.text())


def _schema_to_model(model_name: str, schema: list[ParameterSchema]) -> Type[BaseModel]:
    """
    Converts a schema (from the YAML manifest) to a Pydantic BaseModel class.

    Args:
        model_name: The name of the model to create.
        schema: The schema to convert.

    Returns:
        A Pydantic BaseModel class.
    """
    field_definitions = {}
    for field in schema:
        field_definitions[field.name] = (
            _parse_type(field.type),
            Field(description=field.description),
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


async def _call_tool_api(
    url: str, session: ClientSession, tool_name: str, data: dict
) -> dict:
    """
    Asynchronously makes an API call to the Toolbox service to execute a tool.

    Args:
        url: The base URL of the Toolbox service.
        session: The HTTP client session.
        tool_name: The name of the tool to execute.
        data: The input data for the tool.

    Returns:
        A dictionary containing the response from the Toolbox service.
    """
    url = f"{url}/api/tool/{tool_name}/invoke"
    async with session.post(url, json=_filter_none_values(data)) as response:
        response.raise_for_status()
        return await response.json()


def _filter_none_values(params: dict) -> dict:
    return {key: value for key, value in params.items() if value is not None}
