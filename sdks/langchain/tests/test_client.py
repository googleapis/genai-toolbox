import asyncio
import warnings
from unittest.mock import AsyncMock, Mock, call, patch

import aiohttp
import pytest
from langchain_core.tools import StructuredTool
from pydantic import BaseModel

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
@patch("toolbox_langchain_sdk.client._load_manifest")
async def test_load_tool_manifest_success(mock_load_manifest):
    client = ToolboxClient("https://my-toolbox.com", session=aiohttp.ClientSession())
    mock_load_manifest.return_value = ManifestSchema(**manifest_data)

    result = await client._load_tool_manifest("test_tool")
    assert result == ManifestSchema(**manifest_data)
    mock_load_manifest.assert_called_once_with(
        "https://my-toolbox.com/api/tool/test_tool", client._session
    )


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client._load_manifest")
async def test_load_tool_manifest_failure(mock_load_manifest):
    client = ToolboxClient("https://my-toolbox.com", session=aiohttp.ClientSession())
    mock_load_manifest.side_effect = Exception("Failed to load manifest")

    with pytest.raises(Exception) as e:
        await client._load_tool_manifest("test_tool")
    assert str(e.value) == "Failed to load manifest"


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client._load_manifest")
async def test_load_toolset_manifest_success(mock_load_manifest):
    client = ToolboxClient("https://my-toolbox.com", session=aiohttp.ClientSession())
    mock_load_manifest.return_value = ManifestSchema(**manifest_data)

    # Test with toolset name
    result = await client._load_toolset_manifest(toolset_name="test_toolset")
    assert result == ManifestSchema(**manifest_data)
    mock_load_manifest.assert_called_once_with(
        "https://my-toolbox.com/api/toolset/test_toolset", client._session
    )
    mock_load_manifest.reset_mock()

    # Test without toolset name
    result = await client._load_toolset_manifest()
    assert result == ManifestSchema(**manifest_data)
    mock_load_manifest.assert_called_once_with(
        "https://my-toolbox.com/api/toolset/", client._session
    )


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client._load_manifest")
async def test_load_toolset_manifest_failure(mock_load_manifest):
    client = ToolboxClient("https://my-toolbox.com", session=aiohttp.ClientSession())
    mock_load_manifest.side_effect = Exception("Failed to load manifest")

    with pytest.raises(Exception) as e:
        await client._load_toolset_manifest(toolset_name="test_toolset")
    assert str(e.value) == "Failed to load manifest"


@pytest.mark.asyncio
async def test_generate_tool_success():
    client = ToolboxClient("https://my-toolbox.com", session=aiohttp.ClientSession())
    tool = client._generate_tool("test_tool", ManifestSchema(**manifest_data))

    assert isinstance(tool, StructuredTool)
    assert tool.name == "test_tool"
    assert tool.description == "This is test tool."
    assert tool.args is not None

    assert "param1" in tool.args
    assert tool.args["param1"]["title"] == "Param1"
    assert tool.args["param1"]["description"] == "Parameter 1"
    assert tool.args["param1"]["anyOf"] == [{"type": "string"}, {"type": "null"}]

    assert "param2" in tool.args
    assert tool.args["param2"]["title"] == "Param2"
    assert tool.args["param2"]["description"] == "Parameter 2"
    assert tool.args["param2"]["anyOf"] == [{"type": "integer"}, {"type": "null"}]


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
        args_schema=BaseModel,
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
            args_schema=BaseModel,
            coroutine=AsyncMock(),
        ),
        StructuredTool(
            name="test_tool2",
            description="This is test tool 2.",
            args_schema=BaseModel,
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
    result = await tool.ainvoke({"param1": "test_value", "param2": 123})

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


@pytest.mark.asyncio
@pytest.mark.parametrize(
    "tool_param_auth, id_token_getters, expected_result",
    [
        ({}, {}, True),  # No auth required
        (
            {"tool_name": {"param1": ["auth_source1"]}},
            {"auth_source1": lambda: "test_token"},
            True,
        ),  # Auth required and satisfied (single param)
        (
            {"tool_name": {"param1": ["auth_source1"]}},
            {},
            False,
        ),  # Auth required but not satisfied (single param)
        (
            {"tool_name": {"param1": ["auth_source1", "auth_source2"]}},
            {"auth_source2": lambda: "test_token"},
            True,
        ),  # Multiple auth sources, one satisfied (single param)
        (
            {
                "tool_name": {
                    "param1": ["auth_source1"],
                    "param2": ["auth_source2"],
                }
            },
            {
                "auth_source1": lambda: "test_token1",
                "auth_source2": lambda: "test_token2",
            },
            True,
        ),  # Multiple params, auth satisfied
        (
            {
                "tool_name": {
                    "param1": ["auth_source1"],
                    "param2": ["auth_source2"],
                }
            },
            {"auth_source1": lambda: "test_token1"},
            False,
        ),  # Multiple params, one auth missing
        (
            {
                "tool_name": {
                    "param1": ["auth_source1", "auth_source3"],
                    "param2": ["auth_source2"],
                }
            },
            {
                "auth_source2": lambda: "test_token2",
                "auth_source3": lambda: "test_token3",
            },
            True,
        ),  # Multiple params, multiple auth sources, satisfied
    ],
)
async def test_validate_auth(tool_param_auth, id_token_getters, expected_result):
    """Test _validate_auth with different auth scenarios."""
    client = ToolboxClient("http://test-url")
    client._tool_param_auth = tool_param_auth
    for auth_source, get_id_token in id_token_getters.items():
        client.add_auth_token(auth_source, get_id_token)
    assert client._validate_auth("tool_name") == expected_result


@pytest.mark.asyncio
@pytest.mark.parametrize(
    "manifest, id_token_getters, expected_tool_param_auth, expected_warning",
    [
        (
            ManifestSchema(
                serverVersion="1.0",
                tools={
                    "tool_name": ToolSchema(
                        description="Test tool",
                        parameters=[
                            ParameterSchema(
                                name="param1", type="string", description="Test param"
                            )
                        ],
                    )
                },
            ),
            {},
            {},
            None,
        ),  # No auth params, no warning
        (
            ManifestSchema(
                serverVersion="1.0",
                tools={
                    "tool_name": ToolSchema(
                        description="Test tool",
                        parameters=[
                            ParameterSchema(
                                name="param1",
                                type="string",
                                description="Test param",
                                authSources=["auth_source1"],
                            ),
                            ParameterSchema(
                                name="param2", type="string", description="Test param"
                            ),
                        ],
                    )
                },
            ),
            {},
            {"tool_name": {"param1": ["auth_source1"]}},
            "Some parameters of tool tool_name require authentication, but no valid auth sources are registered. Please register the required sources before use.",
        ),  # With auth params, auth not satisfied, warning expected
        (
            ManifestSchema(
                serverVersion="1.0",
                tools={
                    "tool_name": ToolSchema(
                        description="Test tool",
                        parameters=[
                            ParameterSchema(
                                name="param1",
                                type="string",
                                description="Test param",
                                authSources=["auth_source1"],
                            ),
                            ParameterSchema(
                                name="param2", type="string", description="Test param"
                            ),
                        ],
                    )
                },
            ),
            {"auth_source1": lambda: "test_token"},
            {"tool_name": {"param1": ["auth_source1"]}},
            None,
        ),  # With auth params, auth satisfied, no warning expected
        (
            ManifestSchema(
                serverVersion="1.0",
                tools={
                    "tool_name": ToolSchema(
                        description="Test tool",
                        parameters=[
                            ParameterSchema(
                                name="param1",
                                type="string",
                                description="Test param",
                                authSources=["auth_source1"],
                            ),
                            ParameterSchema(
                                name="param2", type="string", description="Test param"
                            ),
                            ParameterSchema(
                                name="param3",
                                type="string",
                                description="Test param",
                                authSources=[
                                    "auth_source1",
                                    "auth_source2",
                                ],
                            ),
                            ParameterSchema(
                                name="param4",
                                type="string",
                                description="Test param",
                            ),
                            ParameterSchema(
                                name="param5",
                                type="string",
                                description="Test param",
                                authSources=[
                                    "auth_source3",
                                    "auth_source2",
                                ],
                            ),  # more parameters with and without authSources
                        ],
                    )
                },
            ),
            {
                "auth_source2": lambda: "test_token",
                "auth_source3": lambda: "test_token",
            },
            {
                "tool_name": {
                    "param1": ["auth_source1"],
                    "param3": ["auth_source1", "auth_source2"],
                    "param5": ["auth_source3", "auth_source2"],
                }
            },
            "Some parameters of tool tool_name require authentication, but no valid auth sources are registered. Please register the required sources before use.",
        ),  # With multiple auth params, auth not satisfied, warning expected
        (
            ManifestSchema(
                serverVersion="1.0",
                tools={
                    "tool_name": ToolSchema(
                        description="Test tool",
                        parameters=[
                            ParameterSchema(
                                name="param1",
                                type="string",
                                description="Test param",
                                authSources=["auth_source1"],
                            ),
                            ParameterSchema(
                                name="param2", type="string", description="Test param"
                            ),
                            ParameterSchema(
                                name="param3",
                                type="string",
                                description="Test param",
                                authSources=[
                                    "auth_source1",
                                    "auth_source2",
                                ],
                            ),
                            ParameterSchema(
                                name="param4",
                                type="string",
                                description="Test param",
                            ),
                            ParameterSchema(
                                name="param5",
                                type="string",
                                description="Test param",
                                authSources=[
                                    "auth_source3",
                                    "auth_source2",
                                ],
                            ),  # more parameters with and without authSources
                        ],
                    )
                },
            ),
            {
                "auth_source1": lambda: "test_token",
                "auth_source3": lambda: "test_token",
            },
            {
                "tool_name": {
                    "param1": ["auth_source1"],
                    "param3": ["auth_source1", "auth_source2"],
                    "param5": ["auth_source3", "auth_source2"],
                }
            },
            None,
        ),  # With multiple auth params, auth satisfied, warning not expected
    ],
)
async def test_process_auth_params(
    manifest, id_token_getters, expected_tool_param_auth, expected_warning
):
    """Test _process_auth_params with and without auth params."""
    client = ToolboxClient("http://test-url")
    client._id_token_getters = id_token_getters
    if expected_warning:
        with pytest.warns(UserWarning, match=expected_warning):
            client._process_auth_params(manifest)
    else:
        with warnings.catch_warnings():
            warnings.simplefilter("error")
            client._process_auth_params(manifest)
    assert client._tool_param_auth == expected_tool_param_auth


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client._load_manifest")
@pytest.mark.parametrize(
    "params, auth_tokens, expected_tool_param_auth",
    [
        (
            [
                ParameterSchema(name="param1", type="string", description="Test param"),
                ParameterSchema(name="param2", type="string", description="Test param"),
            ],
            {},
            {},
        ),  # No auth tokens
        (
            [
                ParameterSchema(name="param1", type="string", description="Test param"),
                ParameterSchema(
                    name="param2",
                    type="string",
                    description="Test param",
                    authSources=["auth_source1"],
                ),
            ],
            {"auth_source1": lambda: "test_token"},
            {"tool_name": {"param2": ["auth_source1"]}},
        ),  # With auth tokens
    ],
)
async def test_load_tool(
    mock_load_manifest, params, auth_tokens, expected_tool_param_auth
):
    """Test load_tool with and without auth tokens."""
    client = ToolboxClient("http://test-url")

    # Replace with your desired mock manifest data
    mock_load_manifest.return_value = ManifestSchema(
        serverVersion="1.0",
        tools={
            "tool_name": ToolSchema(
                description="Test tool",
                parameters=params,
            )
        },
    )

    tool = await client.load_tool("tool_name", auth_tokens)

    assert isinstance(tool, StructuredTool)
    assert tool.name == "tool_name"
    assert "param1" in tool.args
    assert client._tool_param_auth == expected_tool_param_auth


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client._load_manifest")
@pytest.mark.parametrize(
    "params, auth_tokens, expected_tool_param_auth, expected_num_tools",
    [
        (
            [
                ParameterSchema(name="param1", type="string", description="Test param"),
                ParameterSchema(name="param2", type="string", description="Test param"),
            ],
            {},
            {},
            1,
        ),  # No auth tokens
        (
            [
                ParameterSchema(name="param1", type="string", description="Test param"),
                ParameterSchema(
                    name="param2",
                    type="string",
                    description="Test param",
                    authSources=["auth_source1"],
                ),
            ],
            {"auth_source1": lambda: "test_token"},
            {"tool_name": {"param2": ["auth_source1"]}},
            1,
        ),  # With auth tokens
    ],
)
async def test_load_toolset(
    mock_load_manifest,
    params,
    auth_tokens,
    expected_tool_param_auth,
    expected_num_tools,
):
    """Test load_toolset with and without toolset name and auth tokens."""
    client = ToolboxClient("http://test-url")

    # Replace with your desired mock manifest data
    mock_load_manifest.return_value = ManifestSchema(
        serverVersion="1.0",
        tools={
            "tool_name": ToolSchema(
                description="Test tool",
                parameters=params,
            )
        },
    )

    tools = await client.load_toolset("toolset_name", auth_tokens)

    assert isinstance(tools, list)
    assert len(tools) == expected_num_tools
    assert all(isinstance(tool, StructuredTool) for tool in tools)
    assert client._tool_param_auth == expected_tool_param_auth


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client._invoke_tool")
@pytest.mark.parametrize(
    "manifest, tool_param_auth, id_token_getters, expected_invoke_tool_call",
    [
        (
            ManifestSchema(
                serverVersion="1.0",
                tools={
                    "tool_name": ToolSchema(
                        description="Test tool description",
                        parameters=[
                            ParameterSchema(
                                name="param1",
                                type="string",
                                description="Test param",
                            )
                        ],
                    )
                },
            ),
            {},
            {},
            True,  # _invoke_tool should be called
        ),  # Basic tool schema, no auth
        (
            ManifestSchema(
                serverVersion="1.0",
                tools={
                    "tool_name": ToolSchema(
                        description="Test tool description",
                        parameters=[
                            ParameterSchema(
                                name="param1",
                                type="string",
                                description="Test param",
                                authSources=["auth_source1"],
                            )
                        ],
                    )
                },
            ),
            {"tool_name": {"param1": ["auth_source1"]}},
            {},
            False,  # _invoke_tool should not be called (auth missing)
        ),  # Tool schema with auth, auth missing
        (
            ManifestSchema(
                serverVersion="1.0",
                tools={
                    "tool_name": ToolSchema(
                        description="Test tool description",
                        parameters=[
                            ParameterSchema(
                                name="param1",
                                type="string",
                                description="Test param",
                                authSources=["auth_source1"],
                            )
                        ],
                    )
                },
            ),
            {"tool_name": {"param1": ["auth_source1"]}},
            {"auth_source1": lambda: "test_token"},
            True,  # _invoke_tool should be called
        ),  # Tool schema with auth, auth present
    ],
)
async def test_generate_tool(
    mock_invoke_tool,
    manifest,
    tool_param_auth,
    id_token_getters,
    expected_invoke_tool_call,
):
    """Test _generate_tool with different tool schemas and auth scenarios."""
    client = ToolboxClient("http://test-url")
    client._tool_param_auth = tool_param_auth
    for auth_source, get_id_token in id_token_getters.items():
        client.add_auth_token(auth_source, get_id_token)

    tool = client._generate_tool("tool_name", manifest)

    assert isinstance(tool, StructuredTool)
    assert tool.name == "tool_name"
    assert tool.description == "Test tool description"
    assert "param1" in tool.args
    assert tool.args["param1"]["title"] == "Param1"
    assert tool.args["param1"]["description"] == "Test param"
    assert tool.args["param1"]["anyOf"] == [{"type": "string"}, {"type": "null"}]

    # Call the tool function to check if _invoke_tool is called
    if expected_invoke_tool_call:
        await tool.coroutine(param1="test_value")
        mock_invoke_tool.assert_called_once()
    else:
        with pytest.raises(
            PermissionError, match="Login required before invoking tool_name."
        ):
            await tool.coroutine(param1="test_value")
        mock_invoke_tool.assert_not_called()


@pytest.mark.asyncio
@patch("aiohttp.ClientSession.close")
async def test_del_closes_session(mock_close):
    """Test that __del__ closes the session when the event loop is running."""
    client = ToolboxClient("http://test-url")

    # Simulate event loop running
    loop = asyncio.get_event_loop()
    loop.create_task(asyncio.sleep(0))

    del client

    # Give the event loop a chance to process the close task
    await asyncio.sleep(0.1)

    mock_close.assert_called_once()


@pytest.mark.asyncio
@patch("aiohttp.ClientSession.close")
async def test_del_closes_session_not_running(mock_close):
    """Test that __del__ closes the session when the event loop is not running."""
    client = ToolboxClient("http://test-url")

    # Keep a reference to the session
    session = client._session

    del client
    import gc

    gc.collect()

    # Now explicitly close the session
    await session.close()

    mock_close.assert_called_once()


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client.asyncio.get_event_loop")
@patch("aiohttp.ClientSession.close")
async def test_del_handles_exception(mock_close, mock_get_event_loop):
    """Test that __del__ handles exceptions gracefully."""
    client = ToolboxClient("http://test-url")

    # Simulate an exception when getting the event loop
    mock_get_event_loop.side_effect = Exception("Test exception")

    del client

    # close should not be called because of the exception
    mock_close.assert_not_called()


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client.asyncio.get_event_loop")
async def test_del_loop_not_running(mock_get_event_loop):
    """Test that __del__ handles the case where the loop is not running."""

    loop = asyncio.new_event_loop()
    asyncio.set_event_loop(loop)

    client = ToolboxClient("http://test-url")
    mock_loop = Mock()
    mock_loop.is_running.return_value = False
    mock_get_event_loop.return_value = mock_loop

    del client
    import gc

    gc.collect()  # Force garbage collection

    # Add a small delay to allow the event loop to process the close coroutine
    await asyncio.sleep(0.1)

    loop.close()
