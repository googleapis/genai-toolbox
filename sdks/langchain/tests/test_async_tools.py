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

from unittest.mock import AsyncMock, Mock, patch

import pytest
import pytest_asyncio
from pydantic import ValidationError

from toolbox_langchain_sdk.async_tools import AsyncToolboxTool


@pytest.mark.asyncio
class TestAsyncToolboxTool:
    @pytest.fixture
    def tool_schema(self):
        return {
            "description": "Test Tool Description",
            "parameters": [
                {"name": "param1", "type": "string", "description": "Param 1"},
                {"name": "param2", "type": "integer", "description": "Param 2"},
            ],
        }

    @pytest.fixture
    def auth_tool_schema(self):
        return {
            "description": "Test Tool Description",
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

    @pytest_asyncio.fixture
    @patch("aiohttp.ClientSession")
    async def toolbox_tool(self, MockClientSession, tool_schema):
        mock_session = MockClientSession.return_value
        mock_session.post.return_value.__aenter__.return_value.raise_for_status = Mock()
        mock_session.post.return_value.__aenter__.return_value.json = AsyncMock(
            return_value={"result": "test-result"}
        )
        tool = AsyncToolboxTool(
            name="test_tool",
            schema=tool_schema,
            url="http://test_url",
            session=mock_session,
        )
        return tool

    @pytest_asyncio.fixture
    @patch("aiohttp.ClientSession")
    async def auth_toolbox_tool(self, MockClientSession, auth_tool_schema):
        mock_session = MockClientSession.return_value
        mock_session.post.return_value.__aenter__.return_value.raise_for_status = Mock()
        mock_session.post.return_value.__aenter__.return_value.json = AsyncMock(
            return_value={"result": "test-result"}
        )
        with pytest.warns(
            UserWarning,
            match=r"Parameter\(s\) `param1` of tool test_tool require authentication",
        ):
            tool = AsyncToolboxTool(
                name="test_tool",
                schema=auth_tool_schema,
                url="https://test-url",
                session=mock_session,
            )
        return tool

    @patch("aiohttp.ClientSession")
    async def test_toolbox_tool_init(self, MockClientSession, tool_schema):
        mock_session = MockClientSession.return_value
        tool = AsyncToolboxTool(
            name="test_tool",
            schema=tool_schema,
            url="https://test-url",
            session=mock_session,
        )
        assert tool.name == "test_tool"
        assert tool.description == "Test Tool Description"

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
    async def test_toolbox_tool_bind_params(
        self, toolbox_tool, params, expected_bound_params
    ):
        tool = toolbox_tool.bind_params(params)
        for key, value in expected_bound_params.items():
            if callable(value):
                assert value() == tool._AsyncToolboxTool__bound_params[key]()
            else:
                assert value == tool._AsyncToolboxTool__bound_params[key]

    @pytest.mark.parametrize("strict", [True, False])
    async def test_toolbox_tool_bind_params_invalid(self, toolbox_tool, strict):
        if strict:
            with pytest.raises(ValueError) as e:
                tool = toolbox_tool.bind_params(
                    {"param3": "bound-value"}, strict=strict
                )
            assert "Parameter(s) param3 missing and cannot be bound." in str(e.value)
        else:
            with pytest.warns(UserWarning) as record:
                tool = toolbox_tool.bind_params(
                    {"param3": "bound-value"}, strict=strict
                )
            assert len(record) == 1
            assert "Parameter(s) param3 missing and cannot be bound." in str(
                record[0].message
            )

    async def test_toolbox_tool_bind_params_duplicate(self, toolbox_tool):
        tool = toolbox_tool.bind_params({"param1": "bound-value"})
        with pytest.raises(ValueError) as e:
            tool = tool.bind_params({"param1": "bound-value"})
        assert "Parameter(s) `param1` already bound in tool `test_tool`." in str(
            e.value
        )

    async def test_toolbox_tool_bind_params_invalid_params(self, auth_toolbox_tool):
        with pytest.raises(ValueError) as e:
            auth_toolbox_tool.bind_params({"param1": "bound-value"})
        assert "Parameter(s) param1 already authenticated and cannot be bound." in str(
            e.value
        )

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
    async def test_toolbox_tool_add_auth_tokens(
        self, auth_toolbox_tool, auth_tokens, expected_auth_tokens
    ):
        tool = auth_toolbox_tool.add_auth_tokens(auth_tokens)
        for source, getter in expected_auth_tokens.items():
            assert tool._AsyncToolboxTool__auth_tokens[source]() == getter()

    async def test_toolbox_tool_add_auth_tokens_duplicate(self, auth_toolbox_tool):
        tool = auth_toolbox_tool.add_auth_tokens(
            {"test-auth-source": lambda: "test-token"}
        )
        with pytest.raises(ValueError) as e:
            tool = tool.add_auth_tokens({"test-auth-source": lambda: "test-token"})
        assert (
            "Authentication source(s) `test-auth-source` already registered in tool `test_tool`."
            in str(e.value)
        )

    async def test_toolbox_tool_validate_auth_strict(self, auth_toolbox_tool):
        with pytest.raises(PermissionError) as e:
            auth_toolbox_tool._AsyncToolboxTool__validate_auth(strict=True)
        assert "Parameter(s) `param1` of tool test_tool require authentication" in str(
            e.value
        )

    async def test_toolbox_tool_call(self, toolbox_tool):
        result = await toolbox_tool.ainvoke({"param1": "test-value", "param2": 123})
        assert result == {"result": "test-result"}
        toolbox_tool._AsyncToolboxTool__session.post.assert_called_once_with(
            "http://test_url/api/tool/test_tool/invoke",
            json={"param1": "test-value", "param2": 123},
            headers={},
        )

    async def test_toolbox_tool_call_with_bound_params(self, toolbox_tool):
        tool = toolbox_tool.bind_params({"param1": "bound-value"})
        result = await tool.ainvoke({"param2": 123})
        assert result == {"result": "test-result"}
        toolbox_tool._AsyncToolboxTool__session.post.assert_called_once_with(
            "http://test_url/api/tool/test_tool/invoke",
            json={"param1": "bound-value", "param2": 123},
            headers={},
        )

    async def test_toolbox_tool_call_with_auth_tokens(self, auth_toolbox_tool):
        tool = auth_toolbox_tool.add_auth_tokens(
            {"test-auth-source": lambda: "test-token"}
        )
        result = await tool.ainvoke({"param2": 123})
        assert result == {"result": "test-result"}
        auth_toolbox_tool._AsyncToolboxTool__session.post.assert_called_once_with(
            "https://test-url/api/tool/test_tool/invoke",
            json={"param2": 123},
            headers={"test-auth-source_token": "test-token"},
        )

    async def test_toolbox_tool_call_with_auth_tokens_insecure(self, auth_toolbox_tool):
        with pytest.warns(
            UserWarning,
            match="Sending ID token over HTTP. User data may be exposed. Use HTTPS for secure communication.",
        ):
            auth_toolbox_tool._AsyncToolboxTool__url = "http://test-url"
            tool = auth_toolbox_tool.add_auth_tokens(
                {"test-auth-source": lambda: "test-token"}
            )
            result = await tool.ainvoke({"param2": 123})
            assert result == {"result": "test-result"}
            auth_toolbox_tool._AsyncToolboxTool__session.post.assert_called_once_with(
                "http://test-url/api/tool/test_tool/invoke",
                json={"param2": 123},
                headers={"test-auth-source_token": "test-token"},
            )

    async def test_toolbox_tool_call_with_invalid_input(self, toolbox_tool):
        with pytest.raises(ValidationError) as e:
            await toolbox_tool.ainvoke({"param1": 123, "param2": "invalid"})
        assert "2 validation errors for test_tool" in str(e.value)
        assert "param1\n  Input should be a valid string" in str(e.value)
        assert "param2\n  Input should be a valid integer" in str(e.value)

    async def test_toolbox_tool_call_with_empty_input(self, toolbox_tool):
        with pytest.raises(ValidationError) as e:
            await toolbox_tool.ainvoke({})
        assert "2 validation errors for test_tool" in str(e.value)
        assert "param1\n  Field required" in str(e.value)
        assert "param2\n  Field required" in str(e.value)

    async def test_toolbox_tool_run_not_implemented(self, toolbox_tool):
        with pytest.raises(NotImplementedError):
            toolbox_tool._run()
