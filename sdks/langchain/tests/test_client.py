from unittest.mock import patch, Mock
from langchain_core.tools import StructuredTool
import pytest
import requests
from toolbox_langchain_sdk import ToolboxClient
from pydantic import ValidationError


@patch("toolbox_langchain_sdk.client.requests.get")
def test_load_manifest(mock_get):
    client = ToolboxClient("https://my-toolbox.com")
    mock_response = Mock()
    mock_response.text = """
        serverVersion: 0.0.1
        tools:
          test_tool:
            summary: Test Tool
            description: This is a test tool.
            parameters:
              param1:
                type: string
    """
    mock_response.raise_for_status = Mock()
    mock_get.return_value = mock_response

    client._load_manifest("test_toolset")

    assert client._manifest["serverVersion"] == "0.0.1"
    assert "test_tool" in client._manifest["tools"]


@patch("toolbox_langchain_sdk.client.requests.get")
def test_load_manifest_single_toolset(mock_get):
    client = ToolboxClient("https://my-toolbox.com")
    mock_response = Mock()
    mock_response.text = """
        serverVersion: 0.0.1
        tools:
          test_tool:
            summary: Test Tool
            description: This is a test tool.
            parameters:
              param1:
                type: string
    """
    mock_response.raise_for_status = Mock()
    mock_get.return_value = mock_response

    client._load_manifest("test_toolset")

    assert client._manifest["serverVersion"] == "0.0.1"
    assert "test_tool" in client._manifest["tools"]


@patch("toolbox_langchain_sdk.client.requests.get")
def test_load_manifest_all_toolsets(mock_get):
    client = ToolboxClient("https://my-toolbox.com")
    mock_response = Mock()
    mock_response.text = """
        serverVersion: 0.0.1
        tools:
          test_tool1:
            summary: Test Tool 1
            description: This is a test tool 1.
            parameters:
                param1:
                type: string
          test_tool2:
            summary: Test Tool 2
            description: This is a test tool 2.
            parameters:
                param2:
                type: integer
    """
    mock_response.raise_for_status = Mock()
    mock_get.return_value = mock_response

    client._load_manifest()

    assert len(client._manifest) == 2
    assert client._manifest["serverVersion"] == "0.0.1"
    assert "test_tool1" in client._manifest["tools"]
    assert "test_tool2" in client._manifest["tools"]


@patch("toolbox_langchain_sdk.client.requests.get")
def test_load_manifest_invalid_yaml(mock_get):
    client = ToolboxClient("https://my-toolbox.com")
    mock_response = Mock()
    mock_response.text = "invalid yaml"
    mock_response.raise_for_status = Mock()
    mock_get.return_value = mock_response

    client._load_manifest("test_toolset")
    assert client._manifest == "invalid yaml"


@patch("toolbox_langchain_sdk.client.requests.get")
def test_load_manifest_api_error(mock_get):
    client = ToolboxClient("https://my-toolbox.com")
    mock_response = Mock()
    mock_response.raise_for_status = Mock(side_effect=requests.exceptions.HTTPError)
    mock_get.return_value = mock_response

    with pytest.raises(requests.exceptions.HTTPError):
        client._load_manifest("test_toolset")


@patch("toolbox_langchain_sdk.utils.requests.post")
def test_generate_tool_success(mock_post):
    client = ToolboxClient("https://my-toolbox.com")
    client._manifest = {
        "test_tool": {
            "summary": "Test Tool",
            "description": "This is a test tool.",
            "parameters": {
                "param1": {"type": "string", "description": "Parameter 1"},
                "param2": {"type": "integer", "description": "Parameter 2"},
            },
        }
    }

    mock_response = Mock()
    mock_response.status_code = 200
    mock_response.json.return_value = {"result": "some_result"}
    mock_post.return_value = mock_response

    client._generate_tool("test_tool")
    assert len(client._tools) == 1
    tool = client._tools[0]

    assert isinstance(tool, StructuredTool)
    assert tool.name == "Test Tool"
    assert tool.description == "This is a test tool."
    assert tool.args_schema.model_fields.keys() == {"param1", "param2"}
    assert tool.args_schema.model_fields["param1"].annotation == str
    assert tool.args_schema.model_fields["param2"].annotation == int
    assert tool.args_schema.model_fields["param1"].description == "Parameter 1"
    assert tool.args_schema.model_fields["param2"].description == "Parameter 2"

    result = tool.func(param1="test", param2=123)

    mock_post.assert_called_once_with(
        "https://my-toolbox.com/api/tool/test_tool",
        json={"param1": "test", "param2": 123},
    )

    assert result == mock_response.json.return_value


@patch("toolbox_langchain_sdk.utils.requests.post")
def test_generate_tool_api_error(mock_post):
    client = ToolboxClient("https://my-toolbox.com")
    client._manifest = {
        "test_tool": {
            "summary": "Test Tool",
            "description": "This is a test tool.",
            "parameters": {
                "param1": {"type": "string", "description": "Parameter 1"},
                "param2": {"type": "integer", "description": "Parameter 2"},
            },
        }
    }

    mock_post.side_effect = requests.exceptions.HTTPError("Simulated HTTP Error")

    client._generate_tool("test_tool")
    assert len(client._tools) == 1
    tool = client._tools[0]

    assert isinstance(tool, StructuredTool)
    assert tool.name == "Test Tool"
    assert tool.description == "This is a test tool."
    assert tool.args_schema.model_fields.keys() == {"param1", "param2"}
    assert tool.args_schema.model_fields["param1"].annotation == str
    assert tool.args_schema.model_fields["param2"].annotation == int
    assert tool.args_schema.model_fields["param1"].description == "Parameter 1"
    assert tool.args_schema.model_fields["param2"].description == "Parameter 2"

    with pytest.raises(requests.exceptions.HTTPError) as exc_info:
        tool.func(param1="test", param2=123)

        mock_post.assert_called_once_with(
            "https://my-toolbox.com/api/tool/test_tool",
            json={"param1": "test", "param2": 123},
        )

    assert exc_info.value == mock_post.side_effect


def test_generate_tool_missing_schema_fields():
    client = ToolboxClient("https://my-toolbox.com")
    client._manifest = {"test_tool": {"summary": "Test Tool"}}
    with pytest.raises(ValueError) as exc_info:
        client._generate_tool("test_tool")
    assert "Missing required fields in tool schema: parameters, description" == str(
        exc_info.value
    )


def test_generate_tool_invalid_schema_types():
    client = ToolboxClient("https://my-toolbox.com")
    client._manifest = {
        "test_tool": {
            "summary": 123,
            "description": "This is a test tool.",
            "parameters": {
                "param1": {"type": "string", "description": "Parameter 1"},
                "param2": {"type": "integer", "description": "Parameter 2"},
            },
        }
    }
    with pytest.raises(ValidationError) as exc_info:
        client._generate_tool("test_tool")
    assert len(exc_info.value.errors()) == 1
    assert len(exc_info.value.errors()[0]["loc"]) == 1
    assert exc_info.value.errors()[0]["loc"][0] == "name"
    assert exc_info.value.errors()[0]["input"] == 123
    assert exc_info.value.errors()[0]["msg"] == "Input should be a valid string"


@patch("toolbox_langchain_sdk.utils.requests.post")
def test_generate_tool_invalid_parameter_types(mock_post):
    client = ToolboxClient("https://my-toolbox.com")
    client._manifest = {
        "test_tool": {
            "summary": "Test Tool",
            "description": "This is a test tool.",
            "parameters": {
                "param1": {"type": "string", "description": "Parameter 1"},
                "param2": {"type": "integer", "description": "Parameter 2"},
            },
        }
    }

    client._generate_tool("test_tool")
    assert len(client._tools) == 1
    tool = client._tools[0]

    assert isinstance(tool, StructuredTool)
    assert tool.name == "Test Tool"
    assert tool.description == "This is a test tool."
    assert tool.args_schema.model_fields.keys() == {"param1", "param2"}
    assert tool.args_schema.model_fields["param1"].annotation == str
    assert tool.args_schema.model_fields["param2"].annotation == int
    assert tool.args_schema.model_fields["param1"].description == "Parameter 1"
    assert tool.args_schema.model_fields["param2"].description == "Parameter 2"

    with pytest.raises(ValidationError) as exc_info:
        tool.run({"param1": "test", "param2": "abc"})

    mock_post.assert_not_called()
    assert len(exc_info.value.errors()) == 1
    assert len(exc_info.value.errors()[0]["loc"]) == 1
    assert exc_info.value.errors()[0]["loc"][0] == "param2"
    assert exc_info.value.errors()[0]["input"] == "abc"
    assert (
        exc_info.value.errors()[0]["msg"]
        == "Input should be a valid integer, unable to parse string as an integer"
    )
