from typing import Any, Dict, List, Type

import requests
from pydantic import create_model, Field, BaseModel


def schema_to_model(model_name: str, schema: Dict[str, Any]) -> Type[BaseModel]:
    """
    Converts a schema (from the YAML manifest) to a Pydantic BaseModel class.

    Args:
        model_name: The name of the model to create.
        schema: The schema to convert.

    Returns:
        A Pydantic BaseModel class.
    """
    field_definitions = {}
    for name, property_ in schema.items():
        field_definitions[name] = (
            parse_type(property_["type"]),
            Field(description=property_.get("description")),
        )

    return create_model(model_name, **field_definitions)


def parse_type(type_: str) -> Any:
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
        return List
    elif type_ == "object":
        return Dict
    elif type_ == "null":
        return None
    else:
        raise ValueError(f"Unsupported schema type: {type_}")


def call_tool_api(url: str, tool_name: str, data: dict) -> dict:
    """
    Makes an API call to the Toolbox service to execute a tool.

    Args:
        url: The base URL of the Toolbox service.
        tool_name: The name of the tool to execute.
        data: The input data for the tool.

    Returns:
        A dictionary containing the response from the Toolbox service.
    """
    url = f"{url}/api/tool/{tool_name}"
    response = requests.post(url, json=_filter_none_values(data))
    response.raise_for_status()
    return response.json()


def _filter_none_values(params: dict) -> dict:
    return {key: value for key, value in params.items() if value is not None}
