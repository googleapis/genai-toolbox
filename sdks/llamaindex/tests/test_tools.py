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
from llama_index.core.tools import ToolOutput
from pydantic import ValidationError

from toolbox_llamaindex_sdk.tools import ToolboxTool


@pytest.fixture
def tool_schema():
    return {
        "description": "Test Tool Description",
        "parameters": [
            {"name": "param1", "type": "string", "description": "Param 1"},
            {"name": "param2", "type": "integer", "description": "Param 2"},
        ],
    }


@pytest.fixture
def auth_tool_schema():
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


@pytest.fixture
@patch("aiohttp.ClientSession")
async def toolbox_tool(MockClientSession, tool_schema):
    mock_session = MockClientSession.return_value
    mock_session.post.return_value.__aenter__.return_value.raise_for_status = Mock()
    mock_session.post.return_value.__aenter__.return_value.json = AsyncMock(
        return_value={"result": "test-result"}
    )
    tool = ToolboxTool(
        name="test_tool",
        schema=tool_schema,
        url="https://test-url",
        session=mock_session,
    )
    yield tool


@pytest.fixture
@patch("aiohttp.ClientSession")
async def auth_toolbox_tool(MockClientSession, auth_tool_schema):
    mock_session = MockClientSession.return_value
    mock_session.post.return_value.__aenter__.return_value.raise_for_status = Mock()
    mock_session.post.return_value.__aenter__.return_value.json = AsyncMock(
        return_value={"result": "test-result"}
    )
    with pytest.warns(
        UserWarning,
        match=r"Parameter\(s\) \`param1\` of tool test_tool require authentication\, but no valid authentication sources are registered\. Please register the required sources before use\.",
    ):
        tool = ToolboxTool(
            name="test_tool",
            schema=auth_tool_schema,
            url="https://test-url",
            session=mock_session,
        )
    yield tool


@pytest.mark.asyncio
@patch("toolbox_llamaindex_sdk.client.ClientSession")
async def test_toolbox_tool_init(MockClientSession, tool_schema):
    mock_session = MockClientSession.return_value
    tool = ToolboxTool(
        name="test_tool",
        schema=tool_schema,
        url="https://test-url",
        session=mock_session,
    )
    assert tool.metadata.name == "test_tool"
    assert tool.metadata.description == "Test Tool Description"


@pytest.mark.asyncio
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
async def test_toolbox_tool_bind_params(toolbox_tool, params, expected_bound_params):
    async for tool in toolbox_tool:
        tool.bind_params(params)
        for key, value in expected_bound_params.items():
            if callable(value):
                assert value() == tool._bound_params[key]()
            else:
                assert value == tool._bound_params[key]


@pytest.mark.asyncio
@pytest.mark.parametrize("strict", [True, False])
async def test_toolbox_tool_bind_params_invalid(toolbox_tool, strict):
    async for tool in toolbox_tool:
        if strict:
            with pytest.raises(ValueError) as e:
                tool.bind_params({"param3": "bound-value"}, strict=strict)
            assert "Parameter(s) `param3` not existing in tool `test_tool`." in str(
                e.value
            )
        else:
            with pytest.warns(UserWarning) as record:
                tool.bind_params({"param3": "bound-value"}, strict=strict)
            assert len(record) == 1
            assert "Parameter(s) `param3` not existing in tool `test_tool`." in str(
                record[0].message
            )


@pytest.mark.asyncio
async def test_toolbox_tool_bind_params_duplicate(toolbox_tool):
    async for tool in toolbox_tool:
        tool.bind_params({"param1": "bound-value"})
        with pytest.raises(ValueError) as e:
            tool.bind_params({"param1": "bound-value"})
        assert "Parameter(s) `param1` already bound in tool `test_tool`." in str(
            e.value
        )


@pytest.mark.asyncio
async def test_toolbox_tool_bind_params_invalid_params(auth_toolbox_tool):
    async for tool in auth_toolbox_tool:
        with pytest.raises(ValueError) as e:
            tool.bind_params({"param1": "bound-value"})
        assert (
            "Value(s) for `param1` automatically handled by authentication source(s) and cannot be modified."
            in str(e.value)
        )


@pytest.mark.asyncio
async def test_toolbox_tool_bind_param(toolbox_tool):
    async for tool in toolbox_tool:
        tool.bind_param("param1", "bound-value")
        assert tool._bound_params == {"param1": "bound-value"}


@pytest.mark.asyncio
@pytest.mark.parametrize("strict", [True, False])
async def test_toolbox_tool_bind_param_invalid(toolbox_tool, strict):
    async for tool in toolbox_tool:
        if strict:
            with pytest.raises(ValueError) as e:
                tool.bind_param("param3", "bound-value", strict=strict)
            assert "Parameter(s) `param3` not existing in tool `test_tool`." in str(
                e.value
            )
        else:
            with pytest.warns(UserWarning) as record:
                tool.bind_param("param3", "bound-value", strict=strict)
            assert len(record) == 1
            assert "Parameter(s) `param3` not existing in tool `test_tool`." in str(
                record[0].message
            )


@pytest.mark.asyncio
async def test_toolbox_tool_bind_param_duplicate(toolbox_tool):
    async for tool in toolbox_tool:
        tool.bind_param("param1", "bound-value")
        with pytest.raises(ValueError) as e:
            tool.bind_param("param1", "bound-value")
        assert "Parameter(s) `param1` already bound in tool `test_tool`." in str(
            e.value
        )


@pytest.mark.asyncio
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
    auth_toolbox_tool, auth_tokens, expected_auth_tokens
):
    async for tool in auth_toolbox_tool:
        tool.add_auth_tokens(auth_tokens)
        for source, getter in expected_auth_tokens.items():
            assert tool._auth_tokens[source]() == getter()


@pytest.mark.asyncio
async def test_toolbox_tool_add_auth_tokens_duplicate(auth_toolbox_tool):
    async for tool in auth_toolbox_tool:
        tool.add_auth_tokens({"test-auth-source": lambda: "test-token"})
        with pytest.raises(ValueError) as e:
            tool.add_auth_tokens({"test-auth-source": lambda: "test-token"})
        assert (
            "Authentication source(s) `test-auth-source` already registered in tool `test_tool`."
            in str(e.value)
        )


@pytest.mark.asyncio
async def test_toolbox_tool_add_auth_token(auth_toolbox_tool):
    async for tool in auth_toolbox_tool:
        tool.add_auth_token("test-auth-source", lambda: "test-token")
        assert tool._auth_tokens["test-auth-source"]() == "test-token"


@pytest.mark.asyncio
async def test_toolbox_tool_validate_auth_strict(auth_toolbox_tool):
    async for tool in auth_toolbox_tool:
        with pytest.raises(PermissionError) as e:
            tool._validate_auth(strict=True)
        assert (
            "Parameter(s) `param1` of tool test_tool require authentication, but no valid authentication sources are registered. Please register the required sources before use."
            in str(e.value)
        )


@pytest.mark.asyncio
async def test_toolbox_tool_call_with_callable_bound_params(toolbox_tool):
    async for tool in toolbox_tool:
        tool.bind_param("param1", lambda: "bound-value")
        result = await tool.acall(param2=123)
        assert result == ToolOutput(
            content="{'result': 'test-result'}",
            tool_name="test_tool",
            raw_input={"args": (), "kwargs": {"param2": 123}},
            raw_output={"result": "test-result"},
            is_error=False,
        )


@pytest.mark.asyncio
async def test_toolbox_tool_call(toolbox_tool):
    async for tool in toolbox_tool:
        result = await tool.acall(param1="test-value", param2=123)
        assert result == ToolOutput(
            content="{'result': 'test-result'}",
            tool_name="test_tool",
            raw_input={"args": (), "kwargs": {"param1": "test-value", "param2": 123}},
            raw_output={"result": "test-result"},
            is_error=False,
        )


@pytest.mark.asyncio
async def test_toolbox_tool_call_with_bound_params(toolbox_tool):
    async for tool in toolbox_tool:
        tool.bind_params({"param1": "bound-value"})
        result = await tool.acall(param2=123)
        assert result == ToolOutput(
            content="{'result': 'test-result'}",
            tool_name="test_tool",
            raw_input={"args": (), "kwargs": {"param2": 123}},
            raw_output={"result": "test-result"},
            is_error=False,
        )


@pytest.mark.asyncio
async def test_toolbox_tool_call_with_auth_tokens(auth_toolbox_tool):
    async for tool in auth_toolbox_tool:
        tool.add_auth_tokens({"test-auth-source": lambda: "test-token"})
        result = await tool.acall(param2=123)
        assert result == ToolOutput(
            content="{'result': 'test-result'}",
            tool_name="test_tool",
            raw_input={"args": (), "kwargs": {"param2": 123}},
            raw_output={"result": "test-result"},
            is_error=False,
        )


@pytest.mark.asyncio
async def test_toolbox_tool_call_with_auth_tokens_insecure(auth_toolbox_tool):
    async for tool in auth_toolbox_tool:
        with pytest.warns(
            UserWarning,
            match="Sending ID token over HTTP. User data may be exposed. Use HTTPS for secure communication.",
        ):
            tool._url = "http://test-url"
            tool.add_auth_tokens({"test-auth-source": lambda: "test-token"})
            result = await tool.acall(param2=123)
            assert result == ToolOutput(
                content="{'result': 'test-result'}",
                tool_name="test_tool",
                raw_input={"args": (), "kwargs": {"param2": 123}},
                raw_output={"result": "test-result"},
                is_error=False,
            )


@pytest.mark.asyncio
async def test_toolbox_tool_call_with_invalid_input(toolbox_tool):
    async for tool in toolbox_tool:
        with pytest.raises(ValidationError) as e:
            await tool.acall(param1=123, param2="invalid")
        assert "2 validation errors for test_tool" in str(e.value)
        assert "param1\n  Input should be a valid string" in str(e.value)
        assert "param2\n  Input should be a valid integer" in str(e.value)


@pytest.mark.asyncio
async def test_toolbox_tool_call_with_empty_input(toolbox_tool):
    async for tool in toolbox_tool:
        with pytest.raises(ValidationError) as e:
            await tool.acall()
        assert "2 validation errors for test_tool" in str(e.value)
        assert "param1\n  Field required" in str(e.value)
        assert "param2\n  Field required" in str(e.value)
