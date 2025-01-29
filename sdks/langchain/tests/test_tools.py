# Copyright 2025 Google LLC
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

import pytest
from pydantic import ValidationError

from toolbox_langchain_sdk.tools import ToolboxTool
from toolbox_langchain_sdk.async_tools import AsyncToolboxTool


class TestToolboxTool:
    @pytest.fixture
    def tool_schema(self):
        return {
            "description": "Test Tool Description",
            "name": "test_tool",
            "parameters": [
                {"name": "param1", "type": "string", "description": "Param 1"},
                {"name": "param2", "type": "integer", "description": "Param 2"},
            ],
        }

    @pytest.fixture
    def auth_tool_schema(self):
        return {
            "description": "Test Tool Description",
            "name": "test_tool",
            "parameters": [
                {
                    "name": "param1",
                    "type": "string",
                    "description": "Param 1",
                    "authSources": ["test-auth-source"],
                },
                {"name": "param2", "type": "integer", "description": "Param 2"},
            ],
        }

    @pytest.fixture
    def toolbox_tool(self, tool_schema):
        tool = ToolboxTool(
            name="test_tool",
            schema=tool_schema,
            url="http://test_url",
            session=Mock(),
        )
        yield tool
        ToolboxTool._ToolboxTool__bg_loop = None

    @pytest.fixture(scope="function")
    def mock_async_tool(self, tool_schema):
        mock_async_tool = Mock(spec=AsyncToolboxTool)
        mock_async_tool._AsyncToolboxTool__name = "test_tool"
        mock_async_tool._AsyncToolboxTool__schema = tool_schema
        mock_async_tool._AsyncToolboxTool__url = "http://test_url"
        mock_async_tool._AsyncToolboxTool__session = Mock()
        mock_async_tool._AsyncToolboxTool__auth_tokens = {}
        mock_async_tool._AsyncToolboxTool__bound_params = {}
        return mock_async_tool

    @pytest.fixture
    def auth_toolbox_tool(self, auth_tool_schema):
        with pytest.warns(
            UserWarning,
            match=r"Parameter\(s\) `param1` of tool test_tool require authentication",
        ):
            tool = ToolboxTool(
                name="test_tool",
                schema=auth_tool_schema,
                url="https://test-url",
                session=Mock(),  # Simple Mock session
            )
        yield tool
        ToolboxTool._ToolboxTool__bg_loop = None

    def test_toolbox_tool_init(self, tool_schema):
        tool = ToolboxTool(
            name="test_tool",
            schema=tool_schema,
            url="https://test-url",
            session=Mock(),
        )
        async_tool = tool._ToolboxTool__async_tool
        assert async_tool._AsyncToolboxTool__name == "test_tool"

    @pytest.mark.parametrize(
        "params, expected_bound_params",
        [
            ({"param1": "bound-value"}, {"param1": "bound-value"}),
            ({"param1": lambda: "bound-value"}, {"param1": lambda: "bound-value"}),
            (
                {"param1": "bound-value", "param2": 123},
                {"param1": "bound-value", "param2": 123},
            ),
        ],
    )
    @patch("toolbox_langchain_sdk.tools.AsyncToolboxTool.bind_params")
    def test_toolbox_tool_bind_params(
        self,
        mock_bind_params,
        toolbox_tool,
        mock_async_tool,
        params,
        expected_bound_params,
    ):
        mock_async_tool._AsyncToolboxTool__bound_params = expected_bound_params
        mock_bind_params.return_value = mock_async_tool

        tool = toolbox_tool.bind_params(params)
        mock_bind_params.assert_called_once_with(params, True)
        assert isinstance(tool, ToolboxTool)

        for key, value in expected_bound_params.items():
            async_tool_bound_param_val = (
                tool._ToolboxTool__async_tool._AsyncToolboxTool__bound_params[key]
            )
            if callable(value):
                assert value() == async_tool_bound_param_val()
            else:
                assert value == async_tool_bound_param_val

    @patch("toolbox_langchain_sdk.tools.AsyncToolboxTool.bind_params")
    def test_toolbox_tool_bind_param(self, mock_bind_params, mock_async_tool, toolbox_tool):
        expected_bound_param = {"param1": "bound-value"}
        mock_async_tool._AsyncToolboxTool__bound_params = expected_bound_param
        mock_bind_params.return_value = mock_async_tool
        
        tool = toolbox_tool.bind_param("param1", "bound-value")
        mock_bind_params.assert_called_once_with(expected_bound_param, True)
        
        assert tool._ToolboxTool__async_tool._AsyncToolboxTool__bound_params == expected_bound_param
        assert isinstance(tool, ToolboxTool)

    @pytest.mark.parametrize(
        "auth_tokens, expected_auth_tokens",
        [
            (
                {"test-auth-source": lambda: "test-token"},
                {"test-auth-source": lambda: "test-token"},
            ),
            (
                {
                    "test-auth-source": lambda: "test-token",
                    "another-auth-source": lambda: "another-token",
                },
                {
                    "test-auth-source": lambda: "test-token",
                    "another-auth-source": lambda: "another-token",
                },
            ),
        ],
    )
    @patch("toolbox_langchain_sdk.tools.AsyncToolboxTool.add_auth_tokens")
    def test_toolbox_tool_add_auth_tokens(
        self, mock_add_auth_tokens, mock_async_tool, auth_toolbox_tool, auth_tokens, expected_auth_tokens
    ):
        mock_async_tool._AsyncToolboxTool__auth_tokens = expected_auth_tokens
        mock_add_auth_tokens.return_value = mock_async_tool
        tool = auth_toolbox_tool.add_auth_tokens(auth_tokens)
        mock_add_auth_tokens.assert_called_once_with(auth_tokens, True)
        
        for source, getter in expected_auth_tokens.items():
            assert tool._ToolboxTool__async_tool._AsyncToolboxTool__auth_tokens[source]() == getter()
        assert isinstance(tool, ToolboxTool)

    # @patch("toolbox_langchain_sdk.tools.AsyncToolboxTool.add_auth_tokens")
    # def test_toolbox_tool_add_auth_token(self, mock_add_auth_tokens, mock_async_tool, auth_toolbox_tool):
    #     expected_auth_tokens = {"test-auth-source": lambda: "test-token"}
    #     mock_async_tool._AsyncToolboxTool__auth_tokens = expected_auth_tokens
    #     mock_add_auth_tokens.return_value = mock_async_tool
       
    #     tool = auth_toolbox_tool.add_auth_token("test-auth-source", lambda: "test-token")
    #     mock_add_auth_tokens.assert_called_once_with({"test-auth-source":  lambda: "test-token"}, True)
        
    #     assert tool._async_tool._AsyncToolboxTool__auth_tokens["test-auth-source"]() == "test-token"
    #     assert isinstance(tool, ToolboxTool)

    @patch("toolbox_langchain_sdk.tools.AsyncToolboxTool._arun")
    def test_toolbox_tool_validate_auth_strict(self, mock_arun, auth_toolbox_tool):
        mock_arun.side_effect = PermissionError("Parameter(s) `param1` of tool test_tool require authentication")
        with pytest.raises(PermissionError) as e:
            auth_toolbox_tool._ToolboxTool__async_tool._AsyncToolboxTool__validate_auth(strict=True)
        assert (
            "Parameter(s) `param1` of tool test_tool require authentication"
            in str(e.value)
        )

    # @patch("toolbox_langchain_sdk.tools.AsyncToolboxTool._arun")
    # def test_toolbox_tool_call_with_callable_bound_params(self, mock_arun, toolbox_tool):
    #     mock_arun.return_value = {"result": "test-result"}
    #     tool = toolbox_tool.bind_param("param1", lambda: "bound-value")
    #     result = tool.invoke({"param2": 123})
    #     assert result == {"result": "test-result"}
    #     mock_arun.assert_called_once_with(param2=123, param1="bound-value")

    @patch("toolbox_langchain_sdk.tools.AsyncToolboxTool._arun")
    def test_toolbox_tool_call(self, mock_arun, toolbox_tool):
        mock_arun.return_value = {"result": "test-result"}
        result = toolbox_tool.invoke({"param1": "test-value", "param2": 123})
        assert result == {"result": "test-result"}
        mock_arun.assert_called_once_with(param1="test-value", param2=123)

    # @patch("toolbox_langchain_sdk.tools.AsyncToolboxTool._arun")
    # def test_toolbox_tool_call_with_bound_params(self, mock_arun, toolbox_tool):
    #     mock_arun.return_value = {"result": "test-result"}
    #     tool = toolbox_tool.bind_params({"param1": "bound-value"})
    #     result = tool.invoke({"param2": 123})
    #     assert result == {"result": "test-result"}
    #     mock_arun.assert_called_once_with(param1="bound-value", param2=123)

    # @patch("toolbox_langchain_sdk.tools.AsyncToolboxTool._arun")
    # def test_toolbox_tool_call_with_auth_tokens(self, mock_arun, auth_toolbox_tool):
    #     mock_arun.return_value = {"result": "test-result"}
    #     tool = auth_toolbox_tool.add_auth_tokens({"test-auth-source": lambda: "test-token"})
    #     result = tool.invoke({"param2": 123})
    #     assert result == {"result": "test-result"}
    #     mock_arun.assert_called_once_with(param2=123)
