import asyncio
from unittest.mock import Mock, patch

import pytest

from toolbox_langchain_sdk.client import ToolboxClient
from toolbox_langchain_sdk.tools import ToolboxTool

URL = "http://test_url"


class TestToolboxClient:
    @pytest.fixture()
    def toolbox_client(self):
        client = ToolboxClient.create(URL)
        assert isinstance(client, ToolboxClient)
        assert client._ToolboxClient__async_client is not None

        # Check that the background loop was created and started
        assert client._ToolboxClient__bg_loop is not None
        assert client._ToolboxClient__bg_loop._loop.is_running()

        return client

    def test_constructor_direct_access_raises_exception(self):
        with pytest.raises(
            Exception, match="Only create class through 'create' method!"
        ):
            ToolboxClient(object(), URL)

    @patch("toolbox_langchain_sdk.client.AsyncToolboxClient.aload_tool")
    def test_load_tool(self, mock_aload_tool, toolbox_client):
        mock_tool = Mock(spec=ToolboxTool)
        mock_aload_tool.return_value = asyncio.Future()
        mock_aload_tool.return_value.set_result(mock_tool)

        tool = toolbox_client.load_tool("test_tool").result()

        assert tool == mock_tool
        mock_aload_tool.assert_called_once_with("test_tool", {}, None, {}, True)

    @patch("toolbox_langchain_sdk.client.AsyncToolboxClient.aload_toolset")
    def test_load_toolset(self, mock_aload_toolset, toolbox_client):
        mock_tools = [Mock(spec=ToolboxTool), Mock(spec=ToolboxTool)]
        mock_aload_toolset.return_value = asyncio.Future()
        mock_aload_toolset.return_value.set_result(mock_tools)

        tools = toolbox_client.load_toolset().result()

        assert tools == mock_tools
        mock_aload_toolset.assert_called_once_with(None, {}, None, {}, True)

    @pytest.mark.asyncio
    @patch("toolbox_langchain_sdk.client.AsyncToolboxClient.aload_tool")
    async def test_aload_tool(self, mock_aload_tool, toolbox_client):
        mock_tool = Mock(spec=ToolboxTool)

        mock_aload_tool.return_value = asyncio.Future()
        mock_aload_tool.return_value.set_result(mock_tool)

        tool = await toolbox_client.aload_tool("test_tool")
        assert tool.result() == mock_tool
        mock_aload_tool.assert_called_once_with("test_tool", {}, None, {}, True)

    @pytest.mark.asyncio
    @patch("toolbox_langchain_sdk.client.AsyncToolboxClient.aload_toolset")
    async def test_aload_toolset(self, mock_aload_toolset, toolbox_client):
        mock_tools = [Mock(spec=ToolboxTool), Mock(spec=ToolboxTool)]
        mock_aload_toolset.return_value = asyncio.Future()
        mock_aload_toolset.return_value.set_result(mock_tools)

        tools = await toolbox_client.aload_toolset()
        assert tools.result() == mock_tools
        mock_aload_toolset.assert_called_once_with(None, {}, None, {}, True)

    @patch("toolbox_langchain_sdk.client.AsyncToolboxClient.aload_tool")
    def test_load_tool_with_args(self, mock_aload_tool, toolbox_client):
        mock_tool = Mock(spec=ToolboxTool)
        mock_aload_tool.return_value = asyncio.Future()
        mock_aload_tool.return_value.set_result(mock_tool)
        auth_tokens = {"token1": lambda: "value1"}
        auth_headers = {"header1": lambda: "value2"}
        bound_params = {"param1": "value3"}

        tool = toolbox_client.load_tool(
            "test_tool_name",
            auth_tokens=auth_tokens,
            auth_headers=auth_headers,
            bound_params=bound_params,
            strict=False,
        ).result()

        assert tool == mock_tool
        mock_aload_tool.assert_called_once_with(
            "test_tool_name", auth_tokens, auth_headers, bound_params, False
        )

    @patch("toolbox_langchain_sdk.client.AsyncToolboxClient.aload_toolset")
    def test_load_toolset_with_args(self, mock_aload_toolset, toolbox_client):
        mock_tools = [Mock(spec=ToolboxTool), Mock(spec=ToolboxTool)]
        mock_aload_toolset.return_value = asyncio.Future()
        mock_aload_toolset.return_value.set_result(mock_tools)

        auth_tokens = {"token1": lambda: "value1"}
        auth_headers = {"header1": lambda: "value2"}
        bound_params = {"param1": "value3"}

        tools = toolbox_client.load_toolset(
            toolset_name="my_toolset",
            auth_tokens=auth_tokens,
            auth_headers=auth_headers,
            bound_params=bound_params,
            strict=False,
        ).result()

        assert tools == mock_tools
        mock_aload_toolset.assert_called_once_with(
            "my_toolset", auth_tokens, auth_headers, bound_params, False
        )

    @pytest.mark.asyncio
    @patch("toolbox_langchain_sdk.client.AsyncToolboxClient.aload_tool")
    async def test_aload_tool_with_args(self, mock_aload_tool, toolbox_client):
        mock_tool = Mock(spec=ToolboxTool)
        mock_aload_tool.return_value = asyncio.Future()
        mock_aload_tool.return_value.set_result(mock_tool)

        auth_tokens = {"token1": lambda: "value1"}
        auth_headers = {"header1": lambda: "value2"}
        bound_params = {"param1": "value3"}

        tool = await toolbox_client.aload_tool(
            "test_tool", auth_tokens, auth_headers, bound_params, False
        )
        assert tool.result() == mock_tool
        mock_aload_tool.assert_called_once_with(
            "test_tool", auth_tokens, auth_headers, bound_params, False
        )

    @pytest.mark.asyncio
    @patch("toolbox_langchain_sdk.client.AsyncToolboxClient.aload_toolset")
    async def test_aload_toolset_with_args(self, mock_aload_toolset, toolbox_client):
        mock_tools = [Mock(spec=ToolboxTool), Mock(spec=ToolboxTool)]
        mock_aload_toolset.return_value = asyncio.Future()
        mock_aload_toolset.return_value.set_result(mock_tools)

        auth_tokens = {"token1": lambda: "value1"}
        auth_headers = {"header1": lambda: "value2"}
        bound_params = {"param1": "value3"}

        tools = await toolbox_client.aload_toolset(
            "my_toolset", auth_tokens, auth_headers, bound_params, False
        )
        assert tools.result() == mock_tools
        mock_aload_toolset.assert_called_once_with(
            "my_toolset", auth_tokens, auth_headers, bound_params, False
        )
