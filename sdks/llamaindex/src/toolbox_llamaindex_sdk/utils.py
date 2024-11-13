from typing import Any, Optional, Type

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


class ManifestSchema(BaseModel):
    serverVersion: str
    tools: dict[str, ToolSchema]


async def _load_yaml(url: str, session: ClientSession) -> ManifestSchema:
    """
    Asynchronously fetches and parses the YAML data from the given URL.

    Args:
        url: The base URL to fetch the YAML from.
        session: The HTTP client session

    Returns:
        The parsed Toolbox manifest.
    """
    async with session.get(url) as response:
        response.raise_for_status()
        parsed_yaml = yaml.safe_load(await response.text())
        return ManifestSchema(**parsed_yaml)


# def _schema_to_model(model_name: str, schema: list[ParameterSchema]) -> Type[BaseModel]:
#     """
#     Converts a schema (from the YAML manifest) to a Pydantic BaseModel class.

#     Args:
#         model_name: The name of the model to create.
#         schema: The schema to convert.

#     Returns:
#         A Pydantic BaseModel class.
#     """
#     field_definitions = {}
#     for field in schema:
#         field_definitions[field.name] = (
#             # TODO: Remove the hardcoded optional types once optional fields are supported by Toolbox.
#             Optional[_parse_type(field.type)],
#             Field(description=field.description),
#         )

#     return create_model(model_name=model_name, **field_definitions)


# def _parse_type(type_: str) -> Any:
#     """
#     Converts a schema type to a JSON type.

#     Args:
#         type_: The type name to convert.

#     Returns:
#         A valid JSON type.
#     """

#     if type_ == "string":
#         return str
#     elif type_ == "integer":
#         return int
#     elif type_ == "number":
#         return float
#     elif type_ == "boolean":
#         return bool
#     elif type_ == "array":
#         return list
#     else:
#         raise ValueError(f"Unsupported schema type: {type_}")


# async def _invoke_tool(
#     url: str, session: ClientSession, tool_name: str, data: dict
# ) -> dict:
#     """
#     Asynchronously makes an API call to the Toolbox service to invoke a tool.

#     Args:
#         url: The base URL of the Toolbox service.
#         session: The HTTP client session.
#         tool_name: The name of the tool to invoke.
#         data: The input data for the tool.

#     Returns:
#         A dictionary containing the parsed JSON response from the tool invocation.
#     """
#     url = f"{url}/api/tool/{tool_name}/invoke"
#     async with session.post(url, json=_convert_none_to_empty_string(data)) as response:
#         response.raise_for_status()
#         return await response.json()


# # TODO: Remove this temporary fix once optional fields are supported by Toolbox.
# def _convert_none_to_empty_string(input_dict):
#     new_dict = {}
#     for key, value in input_dict.items():
#         if value is None:
#             new_dict[key] = ""
#         else:
#             new_dict[key] = value
#     return new_dict
