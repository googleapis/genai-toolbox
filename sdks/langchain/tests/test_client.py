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

from unittest.mock import AsyncMock, Mock, patch

import pytest
from aiohttp import ClientSession

from toolbox_langchain_sdk.client import ToolboxClient
from toolbox_langchain_sdk.utils import ManifestSchema


@pytest.fixture
def manifest_schema():
    return ManifestSchema(
        **{
            "serverVersion": "1.0.0",
            "tools": {
                "test_tool_1": {
                    "description": "Test Tool 1 Description",
                    "parameters": [
                        {"name": "param1", "type": "string", "description": "Param 1"}
                    ],
                },
                "test_tool_2": {
                    "description": "Test Tool 2 Description",
                    "parameters": [
                        {"name": "param2", "type": "integer", "description": "Param 2"}
                    ],
                },
            },
        }
    )


@pytest.fixture
def mock_auth_tokens():
    return {"test-auth-source": lambda: "test-token"}


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client.ClientSession")
async def test_toolbox_client_init(mock_client):
    client = ToolboxClient(url="https://test-url", session=mock_client)
    assert client._url == "https://test-url"
    assert client._session == mock_client


@pytest.fixture(params=[True, False])
@patch("toolbox_langchain_sdk.client.ClientSession")
def toolbox_client(MockClientSession, request):
    """
    Fixture to provide a ToolboxClient with and without a provided session.
    """
    if request.param:
        # Client with a provided session
        session = MockClientSession.return_value
        client = ToolboxClient(url="https://test-url", session=session)
        yield client
    else:
        # Client that creates its own session
        client = ToolboxClient(url="https://test-url")
        yield client


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client.ClientSession")
async def test_toolbox_client_close(MockClientSession, toolbox_client):
    MockClientSession.return_value.close = AsyncMock()
    for client in toolbox_client:
        assert not client._session.close.called
        await client.close()
        if client._should_close_session:
            # Assert session is closed only if it was created by the client
            assert client._session.closed
        else:
            # Assert session is NOT closed if it was provided
            assert not client._session.close.called


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client.ClientSession")
async def test_toolbox_client_del(MockClientSession, toolbox_client):
    MockClientSession.return_value.close = AsyncMock()
    for client in toolbox_client:
        client_session = client._session
        assert not client_session.close.called
        client.__del__()
        assert not client_session.close.called


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client._load_manifest")
async def test_toolbox_client_load_tool_manifest(mock_load_manifest):
    mock_load_manifest.return_value = AsyncMock(
        return_value={"tools": {"test_tool": {"description": "Test Tool Description"}}}
    )
    async with ClientSession() as session:
        client = ToolboxClient(url="https://test-url", session=session)
        manifest = await client._load_tool_manifest("test_tool")
        assert manifest == (  # Call the mock object to get its return value
            mock_load_manifest.return_value  # This will return the dictionary
        )
        mock_load_manifest.assert_called_once_with(
            "https://test-url/api/tool/test_tool", session
        )


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client._load_manifest")
async def test_toolbox_client_load_toolset_manifest(mock_load_manifest):
    mock_load_manifest.return_value = AsyncMock(
        return_value={"tools": {"test_tool": {"description": "Test Tool Description"}}}
    )
    async with ClientSession() as session:
        client = ToolboxClient(url="https://test-url", session=session)
        manifest = await client._load_toolset_manifest("test_toolset")
        assert manifest == (  # Call the mock object to get its return value
            mock_load_manifest.return_value  # This will return the dictionary
        )
        mock_load_manifest.assert_called_once_with(
            "https://test-url/api/toolset/test_toolset", session
        )


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client._load_manifest")
async def test_toolbox_client_load_toolset_manifest_no_toolset(mock_load_manifest):
    mock_load_manifest.return_value = AsyncMock(
        return_value={"tools": {"test_tool": {"description": "Test Tool Description"}}}
    )
    async with ClientSession() as session:
        client = ToolboxClient(url="https://test-url", session=session)
        manifest = await client._load_toolset_manifest()
        assert manifest == (  # Call the mock object to get its return value
            mock_load_manifest.return_value  # This will return the dictionary
        )
        mock_load_manifest.assert_called_once_with(
            "https://test-url/api/toolset/", session
        )


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client.ToolboxTool")
@patch("toolbox_langchain_sdk.client._load_manifest")
async def test_toolbox_client_load_tool(mock_load_manifest, MockToolboxTool):
    mock_load_manifest.return_value = AsyncMock(
        return_value={"tools": {"test_tool": {"description": "Test Tool Description"}}}
    )
    async with ClientSession() as session:
        client = ToolboxClient(url="https://test-url", session=session)
        tool = await client.load_tool("test_tool")
        assert tool == MockToolboxTool.return_value
        MockToolboxTool.assert_called_once_with(
            "test_tool",
            mock_load_manifest.return_value.tools.__getitem__(
                "test_tool"
            ),  # Correctly access the tool schema
            "https://test-url",
            session,
            {},
        )


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client.ToolboxTool")
@patch("toolbox_langchain_sdk.client._load_manifest")
async def test_toolbox_client_load_tool_with_auth(
    mock_load_manifest, MockToolboxTool, mock_auth_tokens
):
    mock_load_manifest.return_value = AsyncMock(
        return_value={"tools": {"test_tool": {"description": "Test Tool Description"}}}
    )
    async with ClientSession() as session:
        client = ToolboxClient(url="https://test-url", session=session)
        tool = await client.load_tool("test_tool", auth_tokens=mock_auth_tokens)
        assert tool == MockToolboxTool.return_value
        MockToolboxTool.assert_called_once_with(
            "test_tool",
            mock_load_manifest.return_value.tools.__getitem__("test_tool"),
            "https://test-url",
            session,
            mock_auth_tokens,
        )


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client.ToolboxTool")
@patch("toolbox_langchain_sdk.client._load_manifest")
async def test_toolbox_client_load_tool_with_auth_headers(
    mock_load_manifest, MockToolboxTool, mock_auth_tokens
):
    mock_load_manifest.return_value = AsyncMock(
        return_value={"tools": {"test_tool": {"description": "Test Tool Description"}}}
    )
    async with ClientSession() as session:
        client = ToolboxClient(url="https://test-url", session=session)
        with pytest.warns(
            DeprecationWarning,
            match="Argument `auth_headers` is deprecated. Use `auth_tokens` instead.",
        ):
            tool = await client.load_tool("test_tool", auth_headers=mock_auth_tokens)
        assert tool == MockToolboxTool.return_value
        MockToolboxTool.assert_called_once_with(
            "test_tool",
            mock_load_manifest.return_value.tools.__getitem__("test_tool"),
            "https://test-url",
            session,
            mock_auth_tokens,
        )


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client.ToolboxTool")
@patch("toolbox_langchain_sdk.client._load_manifest")
async def test_toolbox_client_load_tool_with_auth_and_headers(
    mock_load_manifest, MockToolboxTool, mock_auth_tokens
):
    mock_load_manifest.return_value = AsyncMock(
        return_value={"tools": {"test_tool": {"description": "Test Tool Description"}}}
    )
    async with ClientSession() as session:
        client = ToolboxClient(url="https://test-url", session=session)
        with pytest.warns(
            DeprecationWarning,
            match="Both `auth_tokens` and `auth_headers` are provided. `auth_headers` is deprecated, and `auth_tokens` will be used.",
        ):
            tool = await client.load_tool(
                "test_tool", auth_tokens=mock_auth_tokens, auth_headers=mock_auth_tokens
            )
        assert tool == MockToolboxTool.return_value
        MockToolboxTool.assert_called_once_with(
            "test_tool",
            mock_load_manifest.return_value.tools.__getitem__("test_tool"),
            "https://test-url",
            session,
            mock_auth_tokens,
        )


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client._load_manifest")
async def test_toolbox_client_load_toolset(
    mock_load_manifest, toolbox_client, manifest_schema
):
    mock_load_manifest.return_value = manifest_schema
    for client in toolbox_client:
        tools = await client.load_toolset()
        assert [tool._schema for tool in tools] == list(manifest_schema.tools.values())


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client.ToolboxTool")
@patch("toolbox_langchain_sdk.client._load_manifest")
async def test_toolbox_client_load_toolset_with_auth(
    mock_load_manifest,
    mock_toolbox_tool,
    toolbox_client,
    manifest_schema,
    mock_auth_tokens,
):
    mock_load_manifest.return_value = manifest_schema
    for client in toolbox_client:
        tools = await client.load_toolset(auth_tokens=mock_auth_tokens)

        for i, (tool_name, tool_schema) in enumerate(manifest_schema.tools.items()):
            call_args, _ = mock_toolbox_tool.call_args_list[i]
            assert call_args[0] == tool_name
            assert call_args[1] == tool_schema
            assert call_args[2] == client._url
            assert call_args[3] == client._session
            assert call_args[4] == mock_auth_tokens

        assert len(tools) == len(manifest_schema.tools)


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client.ToolboxTool")
@patch("toolbox_langchain_sdk.client._load_manifest")
async def test_toolbox_client_load_toolset_with_auth_headers(
    mock_load_manifest,
    mock_toolbox_tool,
    toolbox_client,
    manifest_schema,
    mock_auth_tokens,
):
    mock_load_manifest.return_value = manifest_schema
    for client in toolbox_client:
        with pytest.warns(
            DeprecationWarning,
            match="Argument `auth_headers` is deprecated. Use `auth_tokens` instead.",
        ):
            tools = await client.load_toolset(auth_headers=mock_auth_tokens)

        for i, (tool_name, tool_schema) in enumerate(manifest_schema.tools.items()):
            call_args, _ = mock_toolbox_tool.call_args_list[i]
            assert call_args[0] == tool_name
            assert call_args[1] == tool_schema
            assert call_args[2] == client._url
            assert call_args[3] == client._session
            assert call_args[4] == mock_auth_tokens

        assert len(tools) == len(manifest_schema.tools)


@pytest.mark.asyncio
@patch("toolbox_langchain_sdk.client.ToolboxTool")
@patch("toolbox_langchain_sdk.client._load_manifest")
async def test_toolbox_client_load_toolset_with_auth_and_headers(
    mock_load_manifest,
    mock_toolbox_tool,
    toolbox_client,
    manifest_schema,
    mock_auth_tokens,
):
    mock_load_manifest.return_value = manifest_schema
    for client in toolbox_client:
        with pytest.warns(
            DeprecationWarning,
            match="Both `auth_tokens` and `auth_headers` are provided. `auth_headers` is deprecated, and `auth_tokens` will be used.",
        ):
            tools = await client.load_toolset(
                auth_tokens=mock_auth_tokens, auth_headers=mock_auth_tokens
            )

        for i, (tool_name, tool_schema) in enumerate(manifest_schema.tools.items()):
            call_args, _ = mock_toolbox_tool.call_args_list[i]
            assert call_args[0] == tool_name
            assert call_args[1] == tool_schema
            assert call_args[2] == client._url
            assert call_args[3] == client._session
            assert call_args[4] == mock_auth_tokens

        assert len(tools) == len(manifest_schema.tools)



@pytest.mark.asyncio
async def test_toolbox_client_del_loop_not_running():
    """Test __del__ when the loop is not running."""
    mock_loop = Mock()
    mock_loop.is_running.return_value = False
    mock_close = Mock(spec=ToolboxClient.close)

    with patch("asyncio.get_event_loop", return_value=mock_loop):
        client = ToolboxClient(url="https://test-url")
        client.close = mock_close
        client.__del__()


@pytest.mark.asyncio
async def test_toolbox_client_del_exception():
    """Test __del__ when an exception occurs."""
    mock_loop = Mock()
    mock_loop.is_running.return_value = True
    mock_loop.create_task.side_effect = Exception("Test Exception")

    with patch("asyncio.get_event_loop", return_value=mock_loop):
        client = ToolboxClient(url="https://test-url")
        client.__del__()

    # Assert that create_task was called (despite the exception)
    mock_loop.create_task.assert_called_once()
