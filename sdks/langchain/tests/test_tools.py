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

import asyncio
from threading import Thread
from unittest.mock import AsyncMock, Mock, patch

import pytest
from pydantic import ValidationError

from toolbox_langchain_sdk.background_loop import _BackgroundLoop
from toolbox_langchain_sdk.tools import ToolboxTool


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
        url="http://test_url",
        session=mock_session,
    )
    yield tool


@pytest.fixture
@patch("aiohttp.ClientSession")
def sync_toolbox_tool(MockClientSession, tool_schema):
    mock_session = MockClientSession.return_value
    mock_session.post.return_value.__aenter__.return_value.raise_for_status = Mock()
    mock_session.post.return_value.__aenter__.return_value.json = AsyncMock(
        return_value={"result": "test-result"}
    )
    loop = asyncio.new_event_loop()
    thread = Thread(target=loop.run_forever, daemon=True)
    thread.start()
    tool = ToolboxTool(
        name="test_tool",
        schema=tool_schema,
        url="http://test_url",
        session=mock_session,
        bg_loop=_BackgroundLoop(loop, thread),
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
        match="Parameter\(s\) \`param1\` of tool test_tool require authentication\, but no valid authentication sources are registered\. Please register the required sources before use\.",
    ):
        tool = ToolboxTool(
            name="test_tool",
            schema=auth_tool_schema,
            url="https://test-url",
            session=mock_session,
        )
    yield tool


@pytest.mark.asyncio
@patch("aiohttp.ClientSession")
async def test_toolbox_tool_init(MockClientSession, tool_schema):
    mock_session = MockClientSession.return_value
    tool = ToolboxTool(
        name="test_tool",
        schema=tool_schema,
        url="https://test-url",
        session=mock_session,
    )
    assert tool.name == "test_tool"
    assert tool.description == "Test Tool Description"


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
        tool = tool.bind_params(params)
        for key, value in expected_bound_params.items():
            if callable(value):
                assert value() == tool._ToolboxTool__bound_params[key]()
            else:
                assert value == tool._ToolboxTool__bound_params[key]


@pytest.mark.asyncio
@pytest.mark.parametrize("strict", [True, False])
async def test_toolbox_tool_bind_params_invalid(toolbox_tool, strict):
    async for tool in toolbox_tool:
        if strict:
            with pytest.raises(ValueError) as e:
                tool = tool.bind_params({"param3": "bound-value"}, strict=strict)
            assert "Parameter(s) param3 missing and cannot be bound." in str(e.value)
        else:
            with pytest.warns(UserWarning) as record:
                tool = tool.bind_params({"param3": "bound-value"}, strict=strict)
            assert len(record) == 1
            assert "Parameter(s) param3 missing and cannot be bound." in str(
                record[0].message
            )


@pytest.mark.asyncio
async def test_toolbox_tool_bind_params_duplicate(toolbox_tool):
    async for tool in toolbox_tool:
        tool = tool.bind_params({"param1": "bound-value"})
        with pytest.raises(ValueError) as e:
            tool = tool.bind_params({"param1": "bound-value"})
        assert "Parameter(s) `param1` already bound in tool `test_tool`." in str(
            e.value
        )


@pytest.mark.asyncio
async def test_toolbox_tool_bind_params_invalid_params(auth_toolbox_tool):
    async for tool in auth_toolbox_tool:
        with pytest.raises(ValueError) as e:
            tool = tool.bind_params({"param1": "bound-value"})
        assert "Parameter(s) param1 already authenticated and cannot be bound." in str(
            e.value
        )


@pytest.mark.asyncio
async def test_toolbox_tool_bind_param(toolbox_tool):
    async for tool in toolbox_tool:
        tool = tool.bind_param("param1", "bound-value")
        assert tool._ToolboxTool__bound_params == {"param1": "bound-value"}


@pytest.mark.asyncio
@pytest.mark.parametrize("strict", [True, False])
async def test_toolbox_tool_bind_param_invalid(toolbox_tool, strict):
    async for tool in toolbox_tool:
        if strict:
            with pytest.raises(ValueError) as e:
                tool = tool.bind_param("param3", "bound-value", strict=strict)
            assert "Parameter(s) param3 missing and cannot be bound." in str(e.value)
        else:
            with pytest.warns(UserWarning) as record:
                tool = tool.bind_param("param3", "bound-value", strict=strict)
            assert len(record) == 1
            assert "Parameter(s) param3 missing and cannot be bound." in str(
                record[0].message
            )


@pytest.mark.asyncio
async def test_toolbox_tool_bind_param_duplicate(toolbox_tool):
    async for tool in toolbox_tool:
        tool = tool.bind_param("param1", "bound-value")
        with pytest.raises(ValueError) as e:
            tool = tool.bind_param("param1", "bound-value")
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
        tool = tool.add_auth_tokens(auth_tokens)
        for source, getter in expected_auth_tokens.items():
            assert tool._ToolboxTool__auth_tokens[source]() == getter()


@pytest.mark.asyncio
async def test_toolbox_tool_add_auth_tokens_duplicate(auth_toolbox_tool):
    async for tool in auth_toolbox_tool:
        tool = tool.add_auth_tokens({"test-auth-source": lambda: "test-token"})
        with pytest.raises(ValueError) as e:
            tool = tool.add_auth_tokens({"test-auth-source": lambda: "test-token"})
        assert (
            "Authentication source(s) `test-auth-source` already registered in tool `test_tool`."
            in str(e.value)
        )


@pytest.mark.asyncio
async def test_toolbox_tool_add_auth_token(auth_toolbox_tool):
    async for tool in auth_toolbox_tool:
        tool = tool.add_auth_token("test-auth-source", lambda: "test-token")
        assert tool._ToolboxTool__auth_tokens["test-auth-source"]() == "test-token"


@pytest.mark.asyncio
async def test_toolbox_tool_validate_auth_strict(auth_toolbox_tool):
    async for tool in auth_toolbox_tool:
        with pytest.raises(PermissionError) as e:
            tool._ToolboxTool__validate_auth(strict=True)
        assert (
            "Parameter(s) `param1` of tool test_tool require authentication, but no valid authentication sources are registered. Please register the required sources before use."
            in str(e.value)
        )


@pytest.mark.asyncio
async def test_toolbox_tool_call_with_callable_bound_params(toolbox_tool):
    async for tool in toolbox_tool:
        tool = tool.bind_param("param1", lambda: "bound-value")
        result = await tool.ainvoke({"param2": 123})
        assert result == {"result": "test-result"}


@pytest.mark.asyncio
async def test_toolbox_tool_call(toolbox_tool):
    async for tool in toolbox_tool:
        result = await tool.ainvoke({"param1": "test-value", "param2": 123})
        assert result == {"result": "test-result"}


@pytest.mark.asyncio
async def test_toolbox_sync_tool_call(sync_toolbox_tool):
    for tool in sync_toolbox_tool:
        result = tool.invoke({"param1": "test-value", "param2": 123})
        assert result == {"result": "test-result"}


@pytest.mark.asyncio
async def test_toolbox_tool_call_with_bound_params(toolbox_tool):
    async for tool in toolbox_tool:
        tool = tool.bind_params({"param1": "bound-value"})
        result = await tool.ainvoke({"param2": 123})
        assert result == {"result": "test-result"}


@pytest.mark.asyncio
async def test_toolbox_tool_call_with_auth_tokens(auth_toolbox_tool):
    async for tool in auth_toolbox_tool:
        tool = tool.add_auth_tokens({"test-auth-source": lambda: "test-token"})
        result = await tool.ainvoke({"param2": 123})
        assert result == {"result": "test-result"}


@pytest.mark.asyncio
async def test_toolbox_tool_call_with_auth_tokens_insecure(auth_toolbox_tool):
    async for tool in auth_toolbox_tool:
        with pytest.warns(
            UserWarning,
            match="Sending ID token over HTTP. User data may be exposed. Use HTTPS for secure communication.",
        ):
            tool._ToolboxTool__url = "http://test-url"
            tool = tool.add_auth_tokens({"test-auth-source": lambda: "test-token"})
            result = await tool.ainvoke({"param2": 123})
            assert result == {"result": "test-result"}


@pytest.mark.asyncio
async def test_toolbox_tool_call_with_invalid_input(toolbox_tool):
    async for tool in toolbox_tool:
        with pytest.raises(ValidationError) as e:
            await tool.ainvoke({"param1": 123, "param2": "invalid"})
        assert "2 validation errors for test_tool" in str(e.value)
        assert "param1\n  Input should be a valid string" in str(e.value)
        assert "param2\n  Input should be a valid integer" in str(e.value)


@pytest.mark.asyncio
async def test_toolbox_tool_call_with_empty_input(toolbox_tool):
    async for tool in toolbox_tool:
        with pytest.raises(ValidationError) as e:
            await tool.ainvoke({})
        assert "2 validation errors for test_tool" in str(e.value)
        assert "param1\n  Field required" in str(e.value)
        assert "param2\n  Field required" in str(e.value)
