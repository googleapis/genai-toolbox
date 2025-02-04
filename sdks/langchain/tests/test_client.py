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

from unittest.mock import Mock, patch
from pydantic import BaseModel

import pytest

from toolbox_langchain_sdk.client import ToolboxClient
from toolbox_langchain_sdk.tools import ToolboxTool

URL = "http://test_url"


class TestToolboxClient:
    @pytest.fixture()
    def toolbox_client(self):
        client = ToolboxClient(URL)
        assert isinstance(client, ToolboxClient)
        assert client._ToolboxClient__async_client is not None

        # Check that the background loop was created and started
        assert client._ToolboxClient__loop is not None
        assert client._ToolboxClient__loop.is_running()

        return client

    @patch("toolbox_langchain_sdk.client.AsyncToolboxClient.aload_tool")
    def test_load_tool(self, mock_aload_tool, toolbox_client):
        mock_tool = Mock(spec=ToolboxTool)
        mock_tool.name="mock-tool"
        mock_tool.description="mock description"
        mock_tool.args_schema = BaseModel
        mock_aload_tool.return_value = mock_tool

        tool = toolbox_client.load_tool("test_tool")

        assert tool.name == mock_tool.name
        assert tool.description == mock_tool.description
        assert tool.args_schema == mock_tool.args_schema
        mock_aload_tool.assert_called_once_with("test_tool", {}, None, {}, True)

    @patch("toolbox_langchain_sdk.client.AsyncToolboxClient.aload_toolset")
    def test_load_toolset(self, mock_aload_toolset, toolbox_client):
        mock_tools = [Mock(spec=ToolboxTool), Mock(spec=ToolboxTool)]
        mock_tools[0].name="mock-tool-0"
        mock_tools[0].description="mock description 0"
        mock_tools[0].args_schema = BaseModel
        mock_tools[1].name="mock-tool-1"
        mock_tools[1].description="mock description 1"
        mock_tools[1].args_schema = BaseModel
        mock_aload_toolset.return_value = mock_tools

        tools = toolbox_client.load_toolset()
        assert len(tools) == len(mock_tools)
        assert all(a.name == b.name and a.description == b.description and a.args_schema == b.args_schema for a,b in zip(tools, mock_tools))
        mock_aload_toolset.assert_called_once_with(None, {}, None, {}, True)

    @pytest.mark.asyncio
    @patch("toolbox_langchain_sdk.client.AsyncToolboxClient.aload_tool")
    async def test_aload_tool(self, mock_aload_tool, toolbox_client):
        mock_tool = Mock(spec=ToolboxTool)
        mock_tool.name="mock-tool"
        mock_tool.description="mock description"
        mock_tool.args_schema = BaseModel
        mock_aload_tool.return_value = mock_tool

        tool = await toolbox_client.aload_tool("test_tool")
        assert tool.name == mock_tool.name
        assert tool.description == mock_tool.description
        assert tool.args_schema == mock_tool.args_schema
        mock_aload_tool.assert_called_once_with("test_tool", {}, None, {}, True)

    @pytest.mark.asyncio
    @patch("toolbox_langchain_sdk.client.AsyncToolboxClient.aload_toolset")
    async def test_aload_toolset(self, mock_aload_toolset, toolbox_client):
        mock_tools = [Mock(spec=ToolboxTool), Mock(spec=ToolboxTool)]
        mock_tools[0].name="mock-tool-0"
        mock_tools[0].description="mock description 0"
        mock_tools[0].args_schema = BaseModel
        mock_tools[1].name="mock-tool-1"
        mock_tools[1].description="mock description 1"
        mock_tools[1].args_schema = BaseModel
        mock_aload_toolset.return_value = mock_tools

        tools = await toolbox_client.aload_toolset()
        assert len(tools) == len(mock_tools)
        assert all(a.name == b.name and a.description == b.description and a.args_schema == b.args_schema for a,b in zip(tools, mock_tools))
        mock_aload_toolset.assert_called_once_with(None, {}, None, {}, True)

    @patch("toolbox_langchain_sdk.client.AsyncToolboxClient.aload_tool")
    def test_load_tool_with_args(self, mock_aload_tool, toolbox_client):
        mock_tool = Mock(spec=ToolboxTool)
        mock_tool.name="mock-tool"
        mock_tool.description="mock description"
        mock_tool.args_schema = BaseModel
        mock_aload_tool.return_value = mock_tool
        auth_tokens = {"token1": lambda: "value1"}
        auth_headers = {"header1": lambda: "value2"}
        bound_params = {"param1": "value3"}

        tool = toolbox_client.load_tool(
            "test_tool_name",
            auth_tokens=auth_tokens,
            auth_headers=auth_headers,
            bound_params=bound_params,
            strict=False,
        )

        assert tool.name == mock_tool.name
        assert tool.description == mock_tool.description
        assert tool.args_schema == mock_tool.args_schema
        mock_aload_tool.assert_called_once_with(
            "test_tool_name", auth_tokens, auth_headers, bound_params, False
        )

    @patch("toolbox_langchain_sdk.client.AsyncToolboxClient.aload_toolset")
    def test_load_toolset_with_args(self, mock_aload_toolset, toolbox_client):
        mock_tools = [Mock(spec=ToolboxTool), Mock(spec=ToolboxTool)]
        mock_tools[0].name="mock-tool-0"
        mock_tools[0].description="mock description 0"
        mock_tools[0].args_schema = BaseModel
        mock_tools[1].name="mock-tool-1"
        mock_tools[1].description="mock description 1"
        mock_tools[1].args_schema = BaseModel
        mock_aload_toolset.return_value = mock_tools

        auth_tokens = {"token1": lambda: "value1"}
        auth_headers = {"header1": lambda: "value2"}
        bound_params = {"param1": "value3"}

        tools = toolbox_client.load_toolset(
            toolset_name="my_toolset",
            auth_tokens=auth_tokens,
            auth_headers=auth_headers,
            bound_params=bound_params,
            strict=False,
        )

        assert len(tools) == len(mock_tools)
        assert all(a.name == b.name and a.description == b.description and a.args_schema == b.args_schema for a,b in zip(tools, mock_tools))
        mock_aload_toolset.assert_called_once_with(
            "my_toolset", auth_tokens, auth_headers, bound_params, False
        )

    @pytest.mark.asyncio
    @patch("toolbox_langchain_sdk.client.AsyncToolboxClient.aload_tool")
    async def test_aload_tool_with_args(self, mock_aload_tool, toolbox_client):
        mock_tool = Mock(spec=ToolboxTool)
        mock_tool.name="mock-tool"
        mock_tool.description="mock description"
        mock_tool.args_schema = BaseModel
        mock_aload_tool.return_value = mock_tool

        auth_tokens = {"token1": lambda: "value1"}
        auth_headers = {"header1": lambda: "value2"}
        bound_params = {"param1": "value3"}

        tool = await toolbox_client.aload_tool(
            "test_tool", auth_tokens, auth_headers, bound_params, False
        )
        assert tool.name == mock_tool.name
        assert tool.description == mock_tool.description
        assert tool.args_schema == mock_tool.args_schema
        mock_aload_tool.assert_called_once_with(
            "test_tool", auth_tokens, auth_headers, bound_params, False
        )

    @pytest.mark.asyncio
    @patch("toolbox_langchain_sdk.client.AsyncToolboxClient.aload_toolset")
    async def test_aload_toolset_with_args(self, mock_aload_toolset, toolbox_client):
        mock_tools = [Mock(spec=ToolboxTool), Mock(spec=ToolboxTool)]
        mock_tools[0].name="mock-tool-0"
        mock_tools[0].description="mock description 0"
        mock_tools[0].args_schema = BaseModel
        mock_tools[1].name="mock-tool-1"
        mock_tools[1].description="mock description 1"
        mock_tools[1].args_schema = BaseModel
        mock_aload_toolset.return_value = mock_tools

        auth_tokens = {"token1": lambda: "value1"}
        auth_headers = {"header1": lambda: "value2"}
        bound_params = {"param1": "value3"}

        tools = await toolbox_client.aload_toolset(
            "my_toolset", auth_tokens, auth_headers, bound_params, False
        )
        assert len(tools) == len(mock_tools)
        assert all(a.name == b.name and a.description == b.description and a.args_schema == b.args_schema for a,b in zip(tools, mock_tools))
        mock_aload_toolset.assert_called_once_with(
            "my_toolset", auth_tokens, auth_headers, bound_params, False
        )
