import asyncio
from unittest.mock import AsyncMock, Mock, call, patch

import aiohttp
import pytest
from langchain_core.tools import StructuredTool
from pydantic import ValidationError

from toolbox_langchain_sdk import ToolboxClient
from toolbox_langchain_sdk.utils import ParameterSchema, ToolSchema, _load_yaml

URL = "https://my-toolbox.com/test"
MOCK_MANIFEST = """
serverVersion: 0.0.1
tools:
    test_tool:
        summary: Test Tool
        description: This is a test tool.
        parameters:
          - name: param1
            type: string
            description: Parameter 1
          - name: param2
            type: integer
            description: Parameter 2
"""

@pytest.fixture(scope="module")
def mock_response():
    return aiohttp.ClientResponse(
        method="GET",
        url=aiohttp.client.URL(URL),
        writer=None,
        continue100=None,
        timer=None,
        request_info=None,
        traces=None,
        session=None,
        loop=asyncio.get_event_loop(),
    )

@pytest.mark.asyncio
@patch("aiohttp.ClientSession.get")
async def test_load_yaml(mock_get, mock_response):
    mock_response.raise_for_status = Mock()
    mock_response.text = AsyncMock(
        return_value=MOCK_MANIFEST
    )

    mock_get.return_value = mock_response
    session = aiohttp.ClientSession()
    manifest = await _load_yaml(URL, session)
    await session.close()
    mock_get.assert_called_once_with(URL)

    assert manifest.serverVersion == "0.0.1"
    assert len(manifest.tools) == 1

    tool = manifest.tools["test_tool"]
    assert tool.description == "This is a test tool."
    assert tool.parameters == [
        ParameterSchema(name="param1", type="string", description="Parameter 1"),
        ParameterSchema(name="param2", type="integer", description="Parameter 2"),
    ]

@pytest.mark.asyncio
@patch("aiohttp.ClientSession.get")
async def test_load_yaml_invalid_yaml(mock_get, mock_response):
    mock_response.raise_for_status = Mock()
    mock_response.text = AsyncMock(return_value="invalid yaml")
    mock_get.return_value = mock_response

    with pytest.raises(Exception):
        session = aiohttp.ClientSession()
        await _load_yaml(URL, session)
        await session.close()
        mock_get.assert_called_once_with(URL)


@pytest.mark.asyncio
@patch("aiohttp.ClientSession.get")
async def test_load_yaml_api_error(mock_get, mock_response):
    error = aiohttp.ClientError("Simulated HTTP Error")
    mock_response.raise_for_status = Mock()
    mock_response.text = AsyncMock(side_effect=error)
    mock_get.return_value = mock_response

    with pytest.raises(aiohttp.ClientError) as exc_info:
        session = aiohttp.ClientSession()
        await _load_yaml(URL, session)
        await session.close()
    mock_get.assert_called_once_with(URL)
    assert exc_info.value == error
