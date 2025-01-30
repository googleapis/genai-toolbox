# # Copyright 2025 Google LLC
# #
# # Licensed under the Apache License, Version 2.0 (the "License");
# # you may not use this file except in compliance with the License.
# # You may obtain a copy of the License at
# #
# #     http://www.apache.org/licenses/LICENSE-2.0
# #
# # Unless required by applicable law or agreed to in writing, software
# # distributed under the License is distributed on an "AS IS" BASIS,
# # WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# # See the License for the specific language governing permissions and
# # limitations under the License.

# import asyncio
# from unittest.mock import AsyncMock, patch
# from warnings import catch_warnings, simplefilter

# import pytest
# from aiohttp import ClientSession

# from toolbox_langchain_sdk.async_client import AsyncToolboxClient
# from toolbox_langchain_sdk.tools import ToolboxTool
# from toolbox_langchain_sdk.utils import ManifestSchema

# URL = "http://test_url"
# MANIFEST_JSON = {
#     "serverVersion": "1.0.0",
#     "tools": {
#         "test_tool_1": {
#             "description": "Test Tool 1 Description",
#             "parameters": [
#                 {
#                     "name": "param1",
#                     "type": "string",
#                     "description": "Param 1",
#                 }
#             ],
#         },
#         "test_tool_2": {
#             "description": "Test Tool 2 Description",
#             "parameters": [
#                 {
#                     "name": "param2",
#                     "type": "integer",
#                     "description": "Param 2",
#                 }
#             ],
#         },
#     },
# }


# # Mock _BackgroundLoop for testing. A real one is needed for actual use.
# class MockBackgroundLoop:
#     def __init__(self):
#         self._loop = asyncio.new_event_loop()

#     def run_async(self, coro):
#         return asyncio.run(coro)


# @pytest.mark.asyncio
# class TestAsyncToolboxClient:
#     @pytest.fixture()
#     def manifest_schema(self):
#         return ManifestSchema(**MANIFEST_JSON)

#     @pytest.fixture()
#     def mock_session(self):
#         return AsyncMock(spec=ClientSession)

#     @pytest.fixture()
#     def mock_bg_loop(self):
#         return MockBackgroundLoop()

#     @pytest.fixture()
#     def mock_client(self, mock_session, mock_bg_loop):
#         return AsyncToolboxClient(URL, bg_loop=mock_bg_loop, session=mock_session)

#     async def test_create_with_existing_session(self, mock_client, mock_session):
#         assert mock_client._AsyncToolboxClient__session == mock_session

#     async def test_create_with_no_session(self, mock_bg_loop):
#         client = AsyncToolboxClient(URL, bg_loop=mock_bg_loop)
#         assert isinstance(client._AsyncToolboxClient__session, ClientSession)
#         await client._AsyncToolboxClient__session.close()  # Close to avoid warnings

#     @patch("toolbox_langchain_sdk.async_client._load_manifest")
#     async def test_aload_tool(
#         self, mock_load_manifest, mock_client, mock_session, manifest_schema
#     ):
#         tool_name = "test_tool_1"
#         mock_load_manifest.return_value = manifest_schema

#         tool = await mock_client.aload_tool(tool_name)

#         mock_load_manifest.assert_called_once_with(
#             f"{URL}/api/tool/{tool_name}", mock_session
#         )
#         assert isinstance(tool, ToolboxTool)
#         assert tool.name == tool_name

#     @patch("toolbox_langchain_sdk.async_client._load_manifest")
#     async def test_aload_tool_auth_headers_deprecated(
#         self, mock_load_manifest, mock_client, manifest_schema
#     ):
#         tool_name = "test_tool_1"
#         mock_manifest = manifest_schema
#         mock_load_manifest.return_value = mock_manifest
#         with catch_warnings(record=True) as w:
#             simplefilter("always")
#             await mock_client.aload_tool(
#                 tool_name, auth_headers={"Authorization": lambda: "Bearer token"}
#             )
#             assert len(w) == 1
#             assert issubclass(w[-1].category, DeprecationWarning)
#             assert "auth_headers" in str(w[-1].message)

#     @patch("toolbox_langchain_sdk.async_client._load_manifest")
#     async def test_aload_tool_auth_headers_and_tokens(
#         self, mock_load_manifest, mock_client, manifest_schema
#     ):
#         tool_name = "test_tool_1"
#         mock_manifest = manifest_schema
#         mock_load_manifest.return_value = mock_manifest
#         with catch_warnings(record=True) as w:
#             simplefilter("always")
#             await mock_client.aload_tool(
#                 tool_name,
#                 auth_headers={"Authorization": lambda: "Bearer token"},
#                 auth_tokens={"test": lambda: "token"},
#             )
#             assert len(w) == 1
#             assert issubclass(w[-1].category, DeprecationWarning)
#             assert "auth_headers" in str(w[-1].message)

#     @patch("toolbox_langchain_sdk.async_client._load_manifest")
#     async def test_aload_toolset(
#         self, mock_load_manifest, mock_client, mock_session, manifest_schema
#     ):
#         mock_manifest = manifest_schema
#         mock_load_manifest.return_value = mock_manifest
#         tools = await mock_client.aload_toolset()

#         mock_load_manifest.assert_called_once_with(f"{URL}/api/toolset/", mock_session)
#         assert len(tools) == 2
#         for tool in tools:
#             assert isinstance(tool, ToolboxTool)
#             assert tool.name in ["test_tool_1", "test_tool_2"]

#     @patch("toolbox_langchain_sdk.async_client._load_manifest")
#     async def test_aload_toolset_with_toolset_name(
#         self, mock_load_manifest, mock_client, mock_session, manifest_schema
#     ):
#         toolset_name = "test_toolset_1"
#         mock_manifest = manifest_schema
#         mock_load_manifest.return_value = mock_manifest
#         tools = await mock_client.aload_toolset(toolset_name=toolset_name)

#         mock_load_manifest.assert_called_once_with(
#             f"{URL}/api/toolset/{toolset_name}", mock_session
#         )
#         assert len(tools) == 2
#         for tool in tools:
#             assert isinstance(tool, ToolboxTool)
#             assert tool.name in ["test_tool_1", "test_tool_2"]

#     @patch("toolbox_langchain_sdk.async_client._load_manifest")
#     async def test_aload_toolset_auth_headers_deprecated(
#         self, mock_load_manifest, mock_client, manifest_schema
#     ):
#         mock_manifest = manifest_schema
#         mock_load_manifest.return_value = mock_manifest
#         with catch_warnings(record=True) as w:
#             simplefilter("always")
#             await mock_client.aload_toolset(
#                 auth_headers={"Authorization": lambda: "Bearer token"}
#             )
#             assert len(w) == 1
#             assert issubclass(w[-1].category, DeprecationWarning)
#             assert "auth_headers" in str(w[-1].message)

#     @patch("toolbox_langchain_sdk.async_client._load_manifest")
#     async def test_aload_toolset_auth_headers_and_tokens(
#         self, mock_load_manifest, mock_client, manifest_schema
#     ):
#         mock_manifest = manifest_schema
#         mock_load_manifest.return_value = mock_manifest
#         with catch_warnings(record=True) as w:
#             simplefilter("always")
#             await mock_client.aload_toolset(
#                 auth_headers={"Authorization": lambda: "Bearer token"},
#                 auth_tokens={"test": lambda: "token"},
#             )
#             assert len(w) == 1
#             assert issubclass(w[-1].category, DeprecationWarning)
#             assert "auth_headers" in str(w[-1].message)

#     async def test_load_tool_not_implemented(self, mock_client):
#         with pytest.raises(NotImplementedError) as excinfo:
#             mock_client.load_tool("test_tool")
#         assert "You can use the ToolboxClient to call synchronous methods." in str(
#             excinfo.value
#         )

#     async def test_load_toolset_not_implemented(self, mock_client):
#         with pytest.raises(NotImplementedError) as excinfo:
#             mock_client.load_toolset()
#         assert "You can use the ToolboxClient to call synchronous methods." in str(
#             excinfo.value
#         )
