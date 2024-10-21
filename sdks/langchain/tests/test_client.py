from unittest.mock import Mock, call, patch

import pytest
import requests
from langchain_core.tools import StructuredTool
from pydantic import ValidationError
from toolbox_langchain_sdk import ToolboxClient


@patch("toolbox_langchain_sdk.utils.requests.get")
def test_load_tool_manifest(mock_get):
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
                    description: Parameter 1
                param2:
                    type: integer
                    description: Parameter 2
    """
    mock_response.raise_for_status = Mock()
    mock_get.return_value = mock_response

    client._load_tool_manifest("test_tool")
    mock_get.assert_called_once_with("https://my-toolbox.com/api/tool/test_tool")
    assert client._manifest["serverVersion"] == "0.0.1"
    assert "tools" in client._manifest
    assert len(client._manifest["tools"].keys()) == 1
    assert "test_tool" in client._manifest["tools"]

    tool = client._manifest["tools"]["test_tool"]
    assert "summary" in tool
    assert "description" in tool
    assert "parameters" in tool
    assert tool["summary"] == "Test Tool"
    assert tool["description"] == "This is a test tool."
    assert len(tool["parameters"].keys()) == 2

    assert "param1" in tool["parameters"]
    assert "type" in tool["parameters"]["param1"]
    assert "description" in tool["parameters"]["param1"]
    assert tool["parameters"]["param1"]["type"] == "string"
    assert tool["parameters"]["param1"]["description"] == "Parameter 1"

    assert "param2" in tool["parameters"]
    assert "type" in tool["parameters"]["param2"]
    assert "description" in tool["parameters"]["param2"]
    assert tool["parameters"]["param2"]["type"] == "integer"
    assert tool["parameters"]["param2"]["description"] == "Parameter 2"


@patch("toolbox_langchain_sdk.utils.requests.get")
def test_load_multiple_tool_manifest(mock_get):
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
                    description: Parameter 1
                param2:
                    type: integer
                    description: Parameter 2
    """
    mock_response.raise_for_status = Mock()
    mock_get.return_value = mock_response

    client._load_tool_manifest("test_tool1")
    mock_get.assert_called_once_with("https://my-toolbox.com/api/tool/test_tool1")
    assert client._manifest["serverVersion"] == "0.0.1"
    assert "tools" in client._manifest
    assert len(client._manifest["tools"].keys()) == 1
    assert "test_tool1" in client._manifest["tools"]

    tool1 = client._manifest["tools"]["test_tool1"]
    assert "summary" in tool1
    assert "description" in tool1
    assert "parameters" in tool1
    assert tool1["summary"] == "Test Tool 1"
    assert tool1["description"] == "This is a test tool 1."
    assert len(tool1["parameters"].keys()) == 2

    assert "param1" in tool1["parameters"]
    assert "type" in tool1["parameters"]["param1"]
    assert "description" in tool1["parameters"]["param1"]
    assert tool1["parameters"]["param1"]["type"] == "string"
    assert tool1["parameters"]["param1"]["description"] == "Parameter 1"

    assert "param2" in tool1["parameters"]
    assert "type" in tool1["parameters"]["param2"]
    assert "description" in tool1["parameters"]["param2"]
    assert tool1["parameters"]["param2"]["type"] == "integer"
    assert tool1["parameters"]["param2"]["description"] == "Parameter 2"

    mock_response = Mock()
    mock_response.text = """
        serverVersion: 0.0.1
        tools:
          test_tool2:
            summary: Test Tool 2
            description: This is a test tool 2.
            parameters:
                param1:
                    type: integer
                    description: Parameter 1
                param2:
                    type: string
                    description: Parameter 2
    """
    mock_response.raise_for_status = Mock()
    mock_get.return_value = mock_response

    client._load_tool_manifest("test_tool2")
    assert mock_get.call_count == 2
    mock_get.assert_has_calls(
        [
            call("https://my-toolbox.com/api/tool/test_tool1"),
            call().raise_for_status(),
            call("https://my-toolbox.com/api/tool/test_tool2"),
            call().raise_for_status(),
        ]
    )
    assert client._manifest["serverVersion"] == "0.0.1"
    assert "tools" in client._manifest
    assert len(client._manifest["tools"].keys()) == 2

    assert "test_tool1" in client._manifest["tools"]
    tool1 = client._manifest["tools"]["test_tool1"]
    assert "summary" in tool1
    assert "description" in tool1
    assert "parameters" in tool1
    assert tool1["summary"] == "Test Tool 1"
    assert tool1["description"] == "This is a test tool 1."
    assert len(tool1["parameters"].keys()) == 2
    assert "param1" in tool1["parameters"]
    assert "type" in tool1["parameters"]["param1"]
    assert "description" in tool1["parameters"]["param1"]
    assert tool1["parameters"]["param1"]["type"] == "string"
    assert tool1["parameters"]["param1"]["description"] == "Parameter 1"
    assert "param2" in tool1["parameters"]
    assert "type" in tool1["parameters"]["param2"]
    assert "description" in tool1["parameters"]["param2"]
    assert tool1["parameters"]["param2"]["type"] == "integer"
    assert tool1["parameters"]["param2"]["description"] == "Parameter 2"

    assert "test_tool2" in client._manifest["tools"]
    tool2 = client._manifest["tools"]["test_tool2"]
    assert "summary" in tool2
    assert "description" in tool2
    assert "parameters" in tool2
    assert tool2["summary"] == "Test Tool 2"
    assert tool2["description"] == "This is a test tool 2."
    assert len(tool2["parameters"].keys()) == 2
    assert "param1" in tool2["parameters"]
    assert "type" in tool2["parameters"]["param1"]
    assert "description" in tool2["parameters"]["param1"]
    assert tool2["parameters"]["param1"]["type"] == "integer"
    assert tool2["parameters"]["param1"]["description"] == "Parameter 1"
    assert "param2" in tool2["parameters"]
    assert "type" in tool2["parameters"]["param2"]
    assert "description" in tool2["parameters"]["param2"]
    assert tool2["parameters"]["param2"]["type"] == "string"
    assert tool2["parameters"]["param2"]["description"] == "Parameter 2"


@patch("toolbox_langchain_sdk.utils.requests.get")
def test_load_tool_manifest_invalid_yaml(mock_get):
    client = ToolboxClient("https://my-toolbox.com")
    mock_response = Mock()
    mock_response.text = "invalid yaml"
    mock_response.raise_for_status = Mock()
    mock_get.return_value = mock_response

    client._load_tool_manifest("test_tool")
    mock_get.assert_called_once_with("https://my-toolbox.com/api/tool/test_tool")
    assert client._manifest == "invalid yaml"


@patch("toolbox_langchain_sdk.utils.requests.get")
def test_load_tool_manifest_api_error(mock_get):
    client = ToolboxClient("https://my-toolbox.com")
    mock_response = Mock()
    mock_response.raise_for_status = Mock(side_effect=requests.exceptions.HTTPError)
    mock_get.return_value = mock_response

    with pytest.raises(requests.exceptions.HTTPError):
        client._load_tool_manifest("test_tool")
    mock_get.assert_called_once_with("https://my-toolbox.com/api/tool/test_tool")


@patch("toolbox_langchain_sdk.utils.requests.get")
def test_load_tool_manifest_valid_then_invalid_yaml(mock_get):
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
                    type: integer
                    description: Parameter 1
                param2:
                    type: string
                    description: Parameter 2
    """
    mock_response.raise_for_status = Mock()
    mock_get.return_value = mock_response

    client._load_tool_manifest("test_tool1")
    mock_get.assert_called_once_with("https://my-toolbox.com/api/tool/test_tool1")
    assert client._manifest["serverVersion"] == "0.0.1"
    assert "tools" in client._manifest
    assert len(client._manifest["tools"].keys()) == 1
    assert "test_tool1" in client._manifest["tools"]

    tool1 = client._manifest["tools"]["test_tool1"]
    assert "summary" in tool1
    assert "description" in tool1
    assert "parameters" in tool1
    assert tool1["summary"] == "Test Tool 1"
    assert tool1["description"] == "This is a test tool 1."
    assert len(tool1["parameters"].keys()) == 2

    assert "param1" in tool1["parameters"]
    assert "type" in tool1["parameters"]["param1"]
    assert "description" in tool1["parameters"]["param1"]
    assert tool1["parameters"]["param1"]["type"] == "integer"
    assert tool1["parameters"]["param1"]["description"] == "Parameter 1"

    assert "param2" in tool1["parameters"]
    assert "type" in tool1["parameters"]["param2"]
    assert "description" in tool1["parameters"]["param2"]
    assert tool1["parameters"]["param2"]["type"] == "string"
    assert tool1["parameters"]["param2"]["description"] == "Parameter 2"

    mock_response = Mock()
    mock_response.text = "invalid yaml"
    mock_response.raise_for_status = Mock()
    mock_get.return_value = mock_response

    client._load_tool_manifest("test_tool2")
    assert mock_get.call_count == 2
    mock_get.assert_has_calls(
        [
            call("https://my-toolbox.com/api/tool/test_tool1"),
            call().raise_for_status(),
            call("https://my-toolbox.com/api/tool/test_tool2"),
            call().raise_for_status(),
        ]
    )
    assert client._manifest == "invalid yaml"


@patch("toolbox_langchain_sdk.utils.requests.get")
def test_load_toolset_manifest(mock_get):
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
                    description: Parameter 1
                param2:
                    type: integer
                    description: Parameter 2
    """
    mock_response.raise_for_status = Mock()
    mock_get.return_value = mock_response

    client._load_toolset_manifest("test_toolset")
    mock_get.assert_called_once_with("https://my-toolbox.com/api/toolset/test_toolset")
    assert client._manifest["serverVersion"] == "0.0.1"
    assert "tools" in client._manifest
    assert len(client._manifest["tools"].keys()) == 1
    assert "test_tool" in client._manifest["tools"]

    tool = client._manifest["tools"]["test_tool"]
    assert "summary" in tool
    assert "description" in tool
    assert "parameters" in tool
    assert tool["summary"] == "Test Tool"
    assert tool["description"] == "This is a test tool."
    assert len(tool["parameters"].keys()) == 2

    assert "param1" in tool["parameters"]
    assert "type" in tool["parameters"]["param1"]
    assert "description" in tool["parameters"]["param1"]
    assert tool["parameters"]["param1"]["type"] == "string"
    assert tool["parameters"]["param1"]["description"] == "Parameter 1"

    assert "param2" in tool["parameters"]
    assert "type" in tool["parameters"]["param2"]
    assert "description" in tool["parameters"]["param2"]
    assert tool["parameters"]["param2"]["type"] == "integer"
    assert tool["parameters"]["param2"]["description"] == "Parameter 2"


@patch("toolbox_langchain_sdk.utils.requests.get")
def test_load_toolset_manifest_all_toolsets(mock_get):
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
                    description: Parameter 1
          test_tool2:
            summary: Test Tool 2
            description: This is a test tool 2.
            parameters:
                param2:
                    type: integer
                    description: Parameter 2
    """
    mock_response.raise_for_status = Mock()
    mock_get.return_value = mock_response

    client._load_toolset_manifest()
    mock_get.assert_called_once_with("https://my-toolbox.com/api/toolset")
    assert client._manifest["serverVersion"] == "0.0.1"
    assert "tools" in client._manifest
    assert len(client._manifest["tools"].keys()) == 2
    assert "test_tool1" in client._manifest["tools"]
    assert "test_tool2" in client._manifest["tools"]

    tool1 = client._manifest["tools"]["test_tool1"]
    assert "summary" in tool1
    assert "description" in tool1
    assert "parameters" in tool1
    assert tool1["summary"] == "Test Tool 1"
    assert tool1["description"] == "This is a test tool 1."
    assert len(tool1["parameters"].keys()) == 1
    assert "param1" in tool1["parameters"]
    assert "type" in tool1["parameters"]["param1"]
    assert "description" in tool1["parameters"]["param1"]
    assert tool1["parameters"]["param1"]["type"] == "string"
    assert tool1["parameters"]["param1"]["description"] == "Parameter 1"

    tool2 = client._manifest["tools"]["test_tool2"]
    assert "summary" in tool2
    assert "description" in tool2
    assert "parameters" in tool2
    assert tool2["summary"] == "Test Tool 2"
    assert tool2["description"] == "This is a test tool 2."
    assert len(tool2["parameters"].keys()) == 1
    assert "param2" in tool2["parameters"]
    assert "type" in tool2["parameters"]["param2"]
    assert "description" in tool2["parameters"]["param2"]
    assert tool2["parameters"]["param2"]["type"] == "integer"
    assert tool2["parameters"]["param2"]["description"] == "Parameter 2"


@patch("toolbox_langchain_sdk.utils.requests.get")
def test_load_toolset_manifest_invalid_yaml(mock_get):
    client = ToolboxClient("https://my-toolbox.com")
    mock_response = Mock()
    mock_response.text = "invalid yaml"
    mock_response.raise_for_status = Mock()
    mock_get.return_value = mock_response

    client._load_toolset_manifest("test_toolset")
    mock_get.assert_called_once_with("https://my-toolbox.com/api/toolset/test_toolset")
    assert client._manifest == "invalid yaml"


@patch("toolbox_langchain_sdk.utils.requests.get")
def test_load_toolset_manifest_api_error(mock_get):
    client = ToolboxClient("https://my-toolbox.com")
    mock_response = Mock()
    mock_response.raise_for_status = Mock(side_effect=requests.exceptions.HTTPError)
    mock_get.return_value = mock_response

    with pytest.raises(requests.exceptions.HTTPError):
        client._load_toolset_manifest("test_toolset")
    mock_get.assert_called_once_with("https://my-toolbox.com/api/toolset/test_toolset")


@patch("toolbox_langchain_sdk.utils.requests.post")
def test_generate_tool_success(mock_post):
    client = ToolboxClient("https://my-toolbox.com")
    client._manifest = {
        "tools": {
            "test_tool": {
                "summary": "Test Tool",
                "description": "This is a test tool.",
                "parameters": {
                    "param1": {"type": "string", "description": "Parameter 1"},
                    "param2": {"type": "integer", "description": "Parameter 2"},
                },
            },
        },
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
        "tools": {
            "test_tool": {
                "summary": "Test Tool",
                "description": "This is a test tool.",
                "parameters": {
                    "param1": {"type": "string", "description": "Parameter 1"},
                    "param2": {"type": "integer", "description": "Parameter 2"},
                },
            },
        },
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
    client._manifest = {"tools": {"test_tool": {"summary": "Test Tool"}}}
    with pytest.raises(ValidationError) as exc_info:
        client._generate_tool("test_tool")
    errors = exc_info.value.errors()
    assert len(errors) == 2
    assert errors[0]["input"] == client._manifest["tools"]["test_tool"]
    assert errors[0]["loc"] == ("description",)
    assert errors[0]["msg"] == "Field required"
    assert errors[0]["type"] == "missing"
    assert errors[1]["input"] == client._manifest["tools"]["test_tool"]
    assert errors[1]["loc"] == ("parameters",)
    assert errors[1]["msg"] == "Field required"
    assert errors[1]["type"] == "missing"


def test_generate_tool_invalid_schema_types():
    client = ToolboxClient("https://my-toolbox.com")
    client._manifest = {
        "tools": {
            "test_tool": {
                "summary": 123,
                "description": "This is a test tool.",
                "parameters": {
                    "param1": {"type": "string", "description": "Parameter 1"},
                    "param2": {"type": "integer", "description": "Parameter 2"},
                },
            },
        },
    }
    with pytest.raises(ValidationError) as exc_info:
        client._generate_tool("test_tool")
    errors = exc_info.value.errors()
    assert len(errors) == 1
    assert errors[0]["loc"] == ("summary",)
    assert errors[0]["input"] == 123
    assert errors[0]["msg"] == "Input should be a valid string"


@patch("toolbox_langchain_sdk.utils.requests.post")
def test_generate_tool_invalid_parameter_types(mock_post):
    client = ToolboxClient("https://my-toolbox.com")
    client._manifest = {
        "tools": {
            "test_tool": {
                "summary": "Test Tool",
                "description": "This is a test tool.",
                "parameters": {
                    "param1": {"type": "string", "description": "Parameter 1"},
                    "param2": {"type": "integer", "description": "Parameter 2"},
                },
            },
        },
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
    errors = exc_info.value.errors()
    assert len(errors) == 1
    assert errors[0]["loc"] == ("param2",)
    assert errors[0]["input"] == "abc"
    assert (
        errors[0]["msg"]
        == "Input should be a valid integer, unable to parse string as an integer"
    )


@patch.object(ToolboxClient, "_load_tool_manifest")
def test_load_tool(mock_load_tool_manifest):
    client = ToolboxClient("https://my-toolbox.com")

    mock_load_tool_manifest.side_effect = lambda _: setattr(
        client,
        "_manifest",
        {
            "serverVersion": "0.0.1",
            "tools": {
                "test_tool": {
                    "summary": "Test Tool",
                    "description": "This is a test tool.",
                    "parameters": {
                        "param1": {"type": "string", "description": "Parameter 1"},
                        "param2": {"type": "integer", "description": "Parameter 2"},
                    },
                },
            },
        },
    )

    tool = client.load_tool("test_tool")
    mock_load_tool_manifest.assert_called_once_with("test_tool")
    assert isinstance(tool, StructuredTool)
    assert tool.name == "Test Tool"
    assert tool.description == "This is a test tool."
    assert tool.args == {
        "param1": {"title": "Param1", "description": "Parameter 1", "type": "string"},
        "param2": {"title": "Param2", "description": "Parameter 2", "type": "integer"},
    }


@patch.object(ToolboxClient, "_load_tool_manifest")
def test_load_multiple_tools(mock_load_tool_manifest):
    client = ToolboxClient("https://my-toolbox.com")

    mock_load_tool_manifest.side_effect = lambda _: setattr(
        client,
        "_manifest",
        {
            "serverVersion": "0.0.1",
            "tools": {
                "test_tool1": {
                    "summary": "Test Tool 1",
                    "description": "This is a test tool 1.",
                    "parameters": {
                        "param1": {"type": "string", "description": "Parameter 1"},
                        "param2": {"type": "integer", "description": "Parameter 2"},
                    },
                },
            },
        },
    )

    tool1 = client.load_tool("test_tool1")
    mock_load_tool_manifest.assert_called_once_with("test_tool1")
    assert isinstance(tool1, StructuredTool)
    assert tool1.name == "Test Tool 1"
    assert tool1.description == "This is a test tool 1."
    assert tool1.args == {
        "param1": {"title": "Param1", "description": "Parameter 1", "type": "string"},
        "param2": {"title": "Param2", "description": "Parameter 2", "type": "integer"},
    }

    mock_load_tool_manifest.side_effect = lambda _: setattr(
        client,
        "_manifest",
        {
            "serverVersion": "0.0.1",
            "tools": {
                "test_tool1": {
                    "summary": "Test Tool 1",
                    "description": "This is a test tool 1.",
                    "parameters": {
                        "param1": {"type": "string", "description": "Parameter 1"},
                        "param2": {"type": "integer", "description": "Parameter 2"},
                    },
                },
                "test_tool2": {
                    "summary": "Test Tool 2",
                    "description": "This is a test tool 2.",
                    "parameters": {
                        "param1": {"type": "integer", "description": "Parameter 1"},
                        "param2": {"type": "string", "description": "Parameter 2"},
                    },
                },
            },
        },
    )

    tool2 = client.load_tool("test_tool2")
    mock_load_tool_manifest.assert_called_with("test_tool2")
    assert isinstance(tool2, StructuredTool)
    assert tool2.name == "Test Tool 2"
    assert tool2.description == "This is a test tool 2."
    assert tool2.args == {
        "param1": {"title": "Param1", "description": "Parameter 1", "type": "integer"},
        "param2": {"title": "Param2", "description": "Parameter 2", "type": "string"},
    }

    assert client._tools == [tool1, tool2]


@patch.object(ToolboxClient, "_load_toolset_manifest")
def test_load_toolset(mock_load_toolset_manifest):
    client = ToolboxClient("https://my-toolbox.com")

    mock_load_toolset_manifest.side_effect = lambda _: setattr(
        client,
        "_manifest",
        {
            "serverVersion": "0.0.1",
            "tools": {
                "test_tool1": {
                    "summary": "Test Tool 1",
                    "description": "This is a test tool 1.",
                    "parameters": {
                        "param1": {"type": "string", "description": "Parameter 1"},
                        "param2": {"type": "integer", "description": "Parameter 2"},
                    },
                },
                "test_tool2": {
                    "summary": "Test Tool 2",
                    "description": "This is a test tool 2.",
                    "parameters": {
                        "param1": {"type": "integer", "description": "Parameter 1"},
                        "param2": {"type": "string", "description": "Parameter 2"},
                    },
                },
            },
        },
    )

    [tool1, tool2] = client.load_toolset("test_toolset")
    mock_load_toolset_manifest.assert_called_once_with("test_toolset")
    assert isinstance(tool1, StructuredTool)
    assert isinstance(tool2, StructuredTool)
    assert tool1.name == "Test Tool 1"
    assert tool1.description == "This is a test tool 1."
    assert tool1.args == {
        "param1": {"title": "Param1", "description": "Parameter 1", "type": "string"},
        "param2": {"title": "Param2", "description": "Parameter 2", "type": "integer"},
    }
    assert tool2.name == "Test Tool 2"
    assert tool2.description == "This is a test tool 2."
    assert tool2.args == {
        "param1": {"title": "Param1", "description": "Parameter 1", "type": "integer"},
        "param2": {"title": "Param2", "description": "Parameter 2", "type": "string"},
    }


@patch.object(ToolboxClient, "_load_toolset_manifest")
def test_load_default_toolset(mock_load_toolset_manifest):
    client = ToolboxClient("https://my-toolbox.com")

    mock_load_toolset_manifest.side_effect = lambda _: setattr(
        client,
        "_manifest",
        {
            "serverVersion": "0.0.1",
            "tools": {
                "test_tool1": {
                    "summary": "Test Tool 1",
                    "description": "This is a test tool 1.",
                    "parameters": {
                        "param1": {"type": "string", "description": "Parameter 1"},
                        "param2": {"type": "integer", "description": "Parameter 2"},
                    },
                },
                "test_tool2": {
                    "summary": "Test Tool 2",
                    "description": "This is a test tool 2.",
                    "parameters": {
                        "param1": {"type": "integer", "description": "Parameter 1"},
                        "param2": {"type": "string", "description": "Parameter 2"},
                    },
                },
            },
        },
    )

    [tool1, tool2] = client.load_toolset()
    mock_load_toolset_manifest.assert_called_once_with(None)
    assert isinstance(tool1, StructuredTool)
    assert isinstance(tool2, StructuredTool)
    assert tool1.name == "Test Tool 1"
    assert tool1.description == "This is a test tool 1."
    assert tool1.args == {
        "param1": {"title": "Param1", "description": "Parameter 1", "type": "string"},
        "param2": {"title": "Param2", "description": "Parameter 2", "type": "integer"},
    }
    assert tool2.name == "Test Tool 2"
    assert tool2.description == "This is a test tool 2."
    assert tool2.args == {
        "param1": {"title": "Param1", "description": "Parameter 1", "type": "integer"},
        "param2": {"title": "Param2", "description": "Parameter 2", "type": "string"},
    }
