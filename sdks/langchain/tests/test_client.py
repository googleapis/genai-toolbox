from unittest.mock import AsyncMock, Mock, call, patch

import aiohttp
import pytest
from langchain_core.tools import StructuredTool

from toolbox_langchain_sdk import ToolboxClient
from toolbox_langchain_sdk.utils import ManifestSchema, ParameterSchema, ToolSchema

# Sample manifest data for testing
manifest_data = {
    "serverVersion": "0.0.1",
    "tools": {
        "test_tool": ToolSchema(
            description="This is test tool.",
            parameters=[
                ParameterSchema(
                    name="param1", type="string", description="Parameter 1"
                ),
                ParameterSchema(
                    name="param2", type="integer", description="Parameter 2"
                ),
            ],
        ),
        "test_tool2": ToolSchema(
            description="This is test tool 2.",
            parameters=[
                ParameterSchema(
                    name="param3", type="string", description="Parameter 3"
                ),
            ],
        ),
    },
}


@pytest.mark.asyncio
async def test_close_session_success():
    mock_session = Mock(spec=aiohttp.ClientSession)
    client = ToolboxClient(url="test_url")
    client._session = mock_session
    client._should_close_session = True

    await client.close()

    mock_session.close.assert_awaited_once()


@pytest.mark.asyncio
async def test_close_no_session():
    client = ToolboxClient(url="test_url")
    client._session = None
    client._should_close_session = True

    await client.close()  # Should not raise any errors


@pytest.mark.asyncio
async def test_close_not_closing_session():
    """Test that the session is not closed when _should_close_session is False."""
    mock_session = Mock(spec=aiohttp.ClientSession)
    client = ToolboxClient(url="test_url")
    client._session = mock_session
    client._should_close_session = False

    await client.close()

    mock_session.close.assert_not_awaited()


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client._load_yaml")
async def test_load_tool_manifest_success(mock_load_yaml):
    client = ToolboxClient("https://my-toolbox.com", session=aiohttp.ClientSession())
    mock_load_yaml.return_value = ManifestSchema(**manifest_data)

    result = await client._load_tool_manifest("test_tool")
    assert result == ManifestSchema(**manifest_data)
    mock_load_yaml.assert_called_once_with(
        "https://my-toolbox.com/api/tool/test_tool", client._session
    )


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client._load_yaml")
async def test_load_tool_manifest_failure(mock_load_yaml):
    client = ToolboxClient("https://my-toolbox.com", session=aiohttp.ClientSession())
    mock_load_yaml.side_effect = Exception("Failed to load YAML")

    with pytest.raises(Exception) as e:
        await client._load_tool_manifest("test_tool")
    assert str(e.value) == "Failed to load YAML"


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client._load_yaml")
async def test_load_toolset_manifest_success(mock_load_yaml):
    client = ToolboxClient("https://my-toolbox.com", session=aiohttp.ClientSession())
    mock_load_yaml.return_value = ManifestSchema(**manifest_data)

    # Test with toolset name
    result = await client._load_toolset_manifest(toolset_name="test_toolset")
    assert result == ManifestSchema(**manifest_data)
    mock_load_yaml.assert_called_once_with(
        "https://my-toolbox.com/api/toolset/test_toolset", client._session
    )
    mock_load_yaml.reset_mock()

    # Test without toolset name
    result = await client._load_toolset_manifest()
    assert result == ManifestSchema(**manifest_data)
    mock_load_yaml.assert_called_once_with(
        "https://my-toolbox.com/api/toolset/", client._session
    )


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client._load_yaml")
async def test_load_toolset_manifest_failure(mock_load_yaml):
    client = ToolboxClient("https://my-toolbox.com", session=aiohttp.ClientSession())
    mock_load_yaml.side_effect = Exception("Failed to load YAML")

    with pytest.raises(Exception) as e:
        await client._load_toolset_manifest(toolset_name="test_toolset")
    assert str(e.value) == "Failed to load YAML"


@pytest.mark.asyncio
async def test_generate_tool_success():
    client = ToolboxClient("https://my-toolbox.com", session=aiohttp.ClientSession())
    tool = client._generate_tool("test_tool", ManifestSchema(**manifest_data))

    assert isinstance(tool, StructuredTool)
    assert tool.name == "test_tool"
    assert tool.description == "This is test tool."
    assert tool.args_schema is not None  # Check if args_schema is generated


@pytest.mark.asyncio
async def test_generate_tool_missing_tool():
    client = ToolboxClient("https://my-toolbox.com", session=aiohttp.ClientSession())

    with pytest.raises(KeyError) as e:
        client._generate_tool("missing_tool", ManifestSchema(**manifest_data))
    assert str(e.value) == "'missing_tool'"


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client.ToolboxClient._load_tool_manifest")
@patch("toolbox_langchain_sdk.client.ToolboxClient._generate_tool")
async def test_load_tool_success(mock_generate_tool, mock_load_manifest):
    client = ToolboxClient("https://my-toolbox.com", session=aiohttp.ClientSession())
    mock_load_manifest.return_value = ManifestSchema(**manifest_data)
    mock_generate_tool.return_value = StructuredTool(
        name="test_tool",
        description="This is test tool.",
        args_schema=None,
        coroutine=AsyncMock(),
    )

    tool = await client.load_tool("test_tool")

    assert isinstance(tool, StructuredTool)
    assert tool.name == "test_tool"
    mock_load_manifest.assert_called_once_with("test_tool")
    mock_generate_tool.assert_called_once_with(
        "test_tool", ManifestSchema(**manifest_data)
    )


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client.ToolboxClient._load_tool_manifest")
async def test_load_tool_failure(mock_load_manifest):
    client = ToolboxClient("https://my-toolbox.com", session=aiohttp.ClientSession())
    mock_load_manifest.side_effect = Exception("Failed to load manifest")

    with pytest.raises(Exception) as e:
        await client.load_tool("test_tool")
    assert str(e.value) == "Failed to load manifest"


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client.ToolboxClient._load_toolset_manifest")
@patch("toolbox_langchain_sdk.client.ToolboxClient._generate_tool")
async def test_load_toolset_success(mock_generate_tool, mock_load_manifest):
    client = ToolboxClient("https://my-toolbox.com", session=aiohttp.ClientSession())
    mock_load_manifest.return_value = ManifestSchema(**manifest_data)
    mock_generate_tool.side_effect = [
        StructuredTool(
            name="test_tool",
            description="This is test tool.",
            args_schema=None,
            coroutine=AsyncMock(),
        ),
        StructuredTool(
            name="test_tool2",
            description="This is test tool 2.",
            args_schema=None,
            coroutine=AsyncMock(),
        ),
    ] * 2

    # Test with toolset name
    tools = await client.load_toolset(toolset_name="test_toolset")
    assert len(tools) == 2
    assert isinstance(tools[0], StructuredTool)
    assert tools[0].name == "test_tool"
    assert isinstance(tools[1], StructuredTool)
    assert tools[1].name == "test_tool2"
    mock_load_manifest.assert_called_once_with("test_toolset")
    mock_generate_tool.assert_has_calls(
        [
            call("test_tool", ManifestSchema(**manifest_data)),
            call("test_tool2", ManifestSchema(**manifest_data)),
        ]
    )
    mock_load_manifest.reset_mock()
    mock_generate_tool.reset_mock()

    # Test without toolset name
    tools = await client.load_toolset()
    assert len(tools) == 2
    assert isinstance(tools[0], StructuredTool)
    assert tools[0].name == "test_tool"
    assert isinstance(tools[1], StructuredTool)
    assert tools[1].name == "test_tool2"
    mock_load_manifest.assert_called_once_with(None)
    mock_generate_tool.assert_has_calls(
        [
            call("test_tool", ManifestSchema(**manifest_data)),
            call("test_tool2", ManifestSchema(**manifest_data)),
        ]
    )


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client.ToolboxClient._load_toolset_manifest")
async def test_load_toolset_failure(mock_load_manifest):
    """Test handling of _load_toolset_manifest failure."""
    client = ToolboxClient("https://my-toolbox.com", session=aiohttp.ClientSession())
    mock_load_manifest.side_effect = Exception("Failed to load manifest")

    with pytest.raises(Exception) as e:
        await client.load_toolset(toolset_name="test_toolset")
    assert str(e.value) == "Failed to load manifest"


@pytest.mark.asyncio
@patch(
    "toolbox_langchain_sdk.client._invoke_tool", return_value={"result": "test_result"}
)
async def test_generate_tool_invoke(mock_invoke_tool):
    """Test invoking the tool function generated by _generate_tool."""
    mock_session = Mock(spec=aiohttp.ClientSession)
    client = ToolboxClient("https://my-toolbox.com", session=mock_session)
    tool = client._generate_tool("test_tool", ManifestSchema(**manifest_data))

    # Call the tool function with some arguments
    result = await tool.coroutine(param1="test_value", param2=123)

    # Assert that _invoke_tool was called with the correct parameters
    mock_invoke_tool.assert_called_once_with(
        "https://my-toolbox.com",
        client._session,
        "test_tool",
        {"param1": "test_value", "param2": 123},
        {},
    )

    # Assert that the result from _invoke_tool is returned
    assert result == {"result": "test_result"}
