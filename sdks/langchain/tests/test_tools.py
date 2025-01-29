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

    # @pytest.mark.parametrize("strict", [True, False])
    # @patch("toolbox_langchain_sdk.tools.AsyncToolboxTool.bind_params")
    # def test_toolbox_tool_bind_params_invalid(self, mock_bind_params, mock_async_tool, toolbox_tool):
    #     strict = False
    #     mock_bind_params.return_value = mock_async_tool

    #     if strict:
    #         mock_bind_params.side_effect = ValueError("Parameter(s) param3 missing")
    #         with pytest.raises(ValueError) as e:
    #             toolbox_tool.bind_params({"param3": "bound-value"}, strict=strict)
    #         assert "Parameter(s) param3 missing" in str(e.value)
    #     else:
    #         mock_bind_params.side_effect = UserWarning("Parameter(s) param3 missing")
    #         with pytest.warns(UserWarning) as record:
    #             toolbox_tool.bind_params({"param3": "bound-value"}, strict=strict)
    #         assert len(record) == 1
    #         assert "Parameter(s) param3 missing" in str(record[0].message)
    #     mock_bind_params.assert_called_once_with({"param3": "bound-value"}, strict=strict)

    # @patch("toolbox_langchain_sdk.tools.AsyncToolboxTool.bind_params")
    # def test_toolbox_tool_bind_params_duplicate(self, mock_bind_params, toolbox_tool):
    #     mock_async_tool = Mock(spec=AsyncToolboxTool)
    #     mock_bind_params.return_value = mock_async_tool  # Correct return value
    #     mock_bind_params.side_effect = ValueError("Parameter(s) `param1` already bound")

    #     toolbox_tool.bind_params({"param1": "bound-value"})  # First call (doesn't raise)
    #     with pytest.raises(ValueError) as e:
    #         toolbox_tool.bind_params({"param1": "bound-value"}) #Second call that throws error
    #     assert "Parameter(s) `param1` already bound" in str(e.value)

    # @patch("toolbox_langchain_sdk.tools.AsyncToolboxTool.bind_params")
    # def test_toolbox_tool_bind_params_invalid_params(self, mock_bind_params, auth_toolbox_tool):
    #     mock_async_tool = Mock(spec=AsyncToolboxTool)
    #     mock_bind_params.return_value = mock_async_tool # Correct return value
    #     mock_bind_params.side_effect = ValueError("Parameter(s) param1 already authenticated")
    #     with pytest.raises(ValueError) as e:
    #         auth_toolbox_tool.bind_params({"param1": "bound-value"})
    #     assert "Parameter(s) param1 already authenticated" in str(e.value)
    #     mock_bind_params.assert_called_once_with({"param1": "bound-value"}, strict=True)

    # @patch("toolbox_langchain_sdk.tools.AsyncToolboxTool.bind_params")
    # def test_toolbox_tool_bind_param(self, mock_bind_params, toolbox_tool):
    #     mock_async_tool = Mock(spec=AsyncToolboxTool)
    #     mock_bind_params.return_value = mock_async_tool
    #     mock_async_tool._AsyncToolboxTool__bound_params = {"param1": "bound-value"}
    #     tool = toolbox_tool.bind_param("param1", "bound-value")
    #     mock_bind_params.assert_called_once_with("param1", "bound-value", strict=True)
    #     assert tool._async_tool._AsyncToolboxTool__bound_params == {"param1": "bound-value"}
    #     assert isinstance(tool, ToolboxTool)

    # @pytest.mark.parametrize("strict", [True, False])
    # @patch("toolbox_langchain_sdk.tools.AsyncToolboxTool.bind_params")
    # def test_toolbox_tool_bind_param_invalid(self, mock_bind_params, toolbox_tool, strict):
    #     mock_async_tool = Mock(spec=AsyncToolboxTool)
    #     mock_bind_params.return_value = mock_async_tool

    #     if strict:
    #         mock_bind_params.side_effect = ValueError("Parameter(s) param3 missing")
    #         with pytest.raises(ValueError) as e:
    #             toolbox_tool.bind_param("param3", "bound-value", strict=strict)
    #         assert "Parameter(s) param3 missing" in str(e.value)
    #     else:
    #         mock_bind_params.side_effect = UserWarning("Parameter(s) param3 missing")
    #         with pytest.warns(UserWarning) as record:
    #             toolbox_tool.bind_param("param3", "bound-value", strict=strict)
    #         assert len(record) == 1
    #         assert "Parameter(s) param3 missing" in str(record[0].message)
    #     mock_bind_params.assert_called_once_with("param3", "bound-value", strict=strict)

    # @patch("toolbox_langchain_sdk.tools.AsyncToolboxTool.bind_params")
    # def test_toolbox_tool_bind_param_duplicate(self, mock_bind_params, toolbox_tool):
    #     mock_async_tool = Mock(spec=AsyncToolboxTool)
    #     mock_bind_params.return_value = mock_async_tool
    #     mock_bind_params.side_effect = ValueError("Parameter(s) `param1` already bound")
    #     toolbox_tool.bind_param("param1", "bound-value")
    #     with pytest.raises(ValueError) as e:
    #         toolbox_tool.bind_param("param1", "bound-value")
    #     assert "Parameter(s) `param1` already bound" in str(e.value)

    # @pytest.mark.parametrize(
    #     "auth_tokens, expected_auth_tokens",
    #     [
    #         (
    #             {"test-auth-source": lambda: "test-token"},
    #             {"test-auth-source": lambda: "test-token"},
    #         ),
    #         (
    #             {
    #                 "test-auth-source": lambda: "test-token",
    #                 "another-auth-source": lambda: "another-token",
    #             },
    #             {
    #                 "test-auth-source": lambda: "test-token",
    #                 "another-auth-source": lambda: "another-token",
    #             },
    #         ),
    #     ],
    # )
    # @patch("toolbox_langchain_sdk.tools.AsyncToolboxTool.add_auth_tokens")
    # def test_toolbox_tool_add_auth_tokens(
    #     self, mock_add_auth_tokens, auth_toolbox_tool, auth_tokens, expected_auth_tokens
    # ):
    #     mock_async_tool = Mock(spec=AsyncToolboxTool)
    #     mock_add_auth_tokens.return_value = mock_async_tool
    #     mock_async_tool._AsyncToolboxTool__auth_tokens = expected_auth_tokens
    #     tool = auth_toolbox_tool.add_auth_tokens(auth_tokens)
    #     mock_add_auth_tokens.assert_called_once_with(auth_tokens, strict=True)
    #     for source, getter in expected_auth_tokens.items():
    #         assert tool._async_tool._AsyncToolboxTool__auth_tokens[source]() == getter()
    #     assert isinstance(tool, ToolboxTool)

    # @patch("toolbox_langchain_sdk.tools.AsyncToolboxTool.add_auth_tokens")
    # def test_toolbox_tool_add_auth_tokens_duplicate(self, mock_add_auth_tokens, auth_toolbox_tool):
    #     mock_async_tool = Mock(spec=AsyncToolboxTool)
    #     mock_add_auth_tokens.return_value = mock_async_tool
    #     mock_add_auth_tokens.side_effect = ValueError("Authentication source(s) `test-auth-source` already registered")

    #     auth_toolbox_tool.add_auth_tokens({"test-auth-source": lambda: "test-token"})
    #     with pytest.raises(ValueError) as e:
    #         auth_toolbox_tool.add_auth_tokens({"test-auth-source": lambda: "test-token"})
    #     assert (
    #         "Authentication source(s) `test-auth-source` already registered"
    #         in str(e.value)
    #     )

    # @patch("toolbox_langchain_sdk.tools.AsyncToolboxTool.add_auth_token")
    # def test_toolbox_tool_add_auth_token(self, mock_add_auth_token, auth_toolbox_tool):
    #     mock_async_tool = Mock(spec=AsyncToolboxTool)
    #     mock_add_auth_token.return_value = mock_async_tool
    #     mock_async_tool._AsyncToolboxTool__auth_tokens = {"test-auth-source": lambda: "test-token"}
    #     tool = auth_toolbox_tool.add_auth_token("test-auth-source", lambda: "test-token")
    #     mock_add_auth_token.assert_called_once_with("test-auth-source",  lambda: "test-token", strict=True)
    #     assert tool._async_tool._AsyncToolboxTool__auth_tokens["test-auth-source"]() == "test-token"
    #     assert isinstance(tool, ToolboxTool)

    # @patch("toolbox_langchain_sdk.tools.AsyncToolboxTool._arun")
    # def test_toolbox_tool_validate_auth_strict(self, mock_arun, auth_toolbox_tool):
    #     mock_arun.side_effect = PermissionError("Parameter(s) `param1` of tool test_tool require authentication")
    #     with pytest.raises(PermissionError) as e:
    #         auth_toolbox_tool._async_tool._AsyncToolboxTool__validate_auth(strict=True)  # Access private member
    #     assert (
    #         "Parameter(s) `param1` of tool test_tool require authentication"
    #         in str(e.value)
    #     )

    # @patch("toolbox_langchain_sdk.tools.AsyncToolboxTool._arun")
    # def test_toolbox_tool_call_with_callable_bound_params(self, mock_arun, toolbox_tool):
    #     mock_arun.return_value = {"result": "test-result"}  # Mock the result
    #     tool = toolbox_tool.bind_param("param1", lambda: "bound-value") #bind param
    #     result = tool.invoke({"param2": 123})  # Use invoke, not _run
    #     assert result == {"result": "test-result"}
    #     mock_arun.assert_called_once_with(param2=123, param1="bound-value")  # Check args passed to _arun

    # @patch("toolbox_langchain_sdk.tools.AsyncToolboxTool._arun")
    # def test_toolbox_tool_call(self, mock_arun, toolbox_tool):
    #     mock_arun.return_value = {"result": "test-result"}  # Mock the result
    #     result = tool.invoke({"param1": "test-value", "param2": 123})  # Use invoke
    #     assert result == {"result": "test-result"}
    #     mock_arun.assert_called_once_with(param1="test-value", param2=123)  # Check args

    # @patch("toolbox_langchain_sdk.tools.AsyncToolboxTool._arun")
    # def test_toolbox_tool_call_with_bound_params(self, mock_arun, toolbox_tool):
    #     mock_arun.return_value = {"result": "test-result"}
    #     tool = toolbox_tool.bind_params({"param1": "bound-value"}) #bind params
    #     result = tool.invoke({"param2": 123})  # Use invoke
    #     assert result == {"result": "test-result"}
    #     mock_arun.assert_called_once_with(param1="bound-value", param2=123)

    # @patch("toolbox_langchain_sdk.tools.AsyncToolboxTool._arun")
    # def test_toolbox_tool_call_with_auth_tokens(self, mock_arun, auth_toolbox_tool):
    #     mock_arun.return_value = {"result": "test-result"}
    #     tool = auth_toolbox_tool.add_auth_tokens({"test-auth-source": lambda: "test-token"}) #add auth tokens
    #     result = tool.invoke({"param2": 123})  # Use invoke
    #     assert result == {"result": "test-result"}
    #     mock_arun.assert_called_once_with(param2=123)  # Check args
