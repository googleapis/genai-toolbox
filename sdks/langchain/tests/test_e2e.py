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

"""End-to-end tests for the toolbox SDK interacting with the toolbox server.

This file covers the following use cases:

1. Loading a tool.
2. Loading a specific toolset.
3. Loading the default toolset (contains all tools).
4. Running a tool with no required auth, with auth provided.
5. Running a tool with required auth:
    a. No auth provided.
    b. Wrong auth provided: The tool requires a different authentication
                            than the one provided.
    c. Correct auth provided.
6. Running a tool with a parameter that requires auth:
    a. No auth provided.
    b. Correct auth provided.
    c. Auth provided does not contain the required claim.
"""

import pytest
import pytest_asyncio
from aiohttp import ClientResponseError

from toolbox_langchain_sdk.client import ToolboxClient


@pytest.mark.asyncio
@pytest.mark.usefixtures("toolbox_server")
class TestE2EClient:
    @pytest_asyncio.fixture(scope="function")
    async def toolbox(self):
        """Provides a ToolboxClient instance for each test."""
        toolbox = ToolboxClient("http://localhost:5000")
        yield toolbox
        await toolbox.close()

    #### Basic e2e tests
    @pytest.mark.asyncio
    async def test_load_tool(self, toolbox):
        tool = await toolbox.load_tool("get-n-rows")
        response = await tool.ainvoke({"num_rows": "2"})
        result = response["result"]

        assert "row1" in result
        assert "row2" in result
        assert "row3" not in result

    @pytest.mark.asyncio
    async def test_load_toolset_specific(self, toolbox):
        toolset = await toolbox.load_toolset("my-toolset")
        assert len(toolset) == 1
        assert toolset[0].name == "get-row-by-id"

        toolset = await toolbox.load_toolset("my-toolset-2")
        assert len(toolset) == 2
        tool_names = ["get-n-rows", "get-row-by-id"]
        assert toolset[0].name in tool_names
        assert toolset[1].name in tool_names

    @pytest.mark.asyncio
    async def test_load_toolset_all(self, toolbox):
        toolset = await toolbox.load_toolset()
        assert len(toolset) == 5
        tool_names = [
            "get-n-rows",
            "get-row-by-id",
            "get-row-by-id-auth",
            "get-row-by-email-auth",
            "get-row-by-content-auth",
        ]
        for tool in toolset:
            assert tool.name in tool_names

    ##### Auth tests
    @pytest.mark.asyncio
    @pytest.mark.skip(reason="b/389574566")
    async def test_run_tool_unauth_with_auth(self, toolbox, auth_token2):
        """Tests running a tool that doesn't require auth, with auth provided."""
        tool = await toolbox.load_tool(
            "get-row-by-id", auth_tokens={"my-test-auth": lambda: auth_token2}
        )
        response = await tool.ainvoke({"id": "2"})
        assert "row2" in response["result"]

    @pytest.mark.asyncio
    async def test_run_tool_no_auth(self, toolbox):
        """Tests running a tool requiring auth without providing auth."""
        tool = await toolbox.load_tool(
            "get-row-by-id-auth",
        )
        with pytest.raises(ClientResponseError, match="401, message='Unauthorized'"):
            await tool.ainvoke({"id": "2"})

    @pytest.mark.asyncio
    async def test_run_tool_wrong_auth(self, toolbox, auth_token2):
        """Tests running a tool with incorrect auth."""
        tool = await toolbox.load_tool(
            "get-row-by-id-auth",
        )
        tool.add_auth_token("my-test-auth", lambda: auth_token2)
        # TODO: Fix error message (b/389577313)
        with pytest.raises(ClientResponseError, match="400, message='Bad Request'"):
            await tool.ainvoke({"id": "2"})

    @pytest.mark.asyncio
    async def test_run_tool_auth(self, toolbox, auth_token1):
        """Tests running a tool with correct auth."""
        tool = await toolbox.load_tool(
            "get-row-by-id-auth",
        )
        tool.add_auth_token("my-test-auth", lambda: auth_token1)
        response = await tool.ainvoke({"id": "2"})
        assert "row2" in response["result"]

    @pytest.mark.asyncio
    async def test_run_tool_param_auth_no_auth(self, toolbox):
        """Tests running a tool with a param requiring auth, without auth."""
        tool = await toolbox.load_tool("get-row-by-email-auth")
        with pytest.raises(
            PermissionError,
            match="Parameter(s) `email` of tool get-row-by-email-auth require authentication, but no valid authentication sources are registered. Please register the required sources before use.",
        ):
            await tool.ainvoke({})

    @pytest.mark.asyncio
    async def test_run_tool_param_auth(self, toolbox, auth_token1):
        """Tests running a tool with a param requiring auth, with correct auth."""
        tool = await toolbox.load_tool(
            "get-row-by-email-auth", auth_tokens={"my-test-auth": lambda: auth_token1}
        )
        response = await tool.ainvoke({})
        result = response["result"]
        assert "row4" in result
        assert "row5" in result
        assert "row6" in result

    @pytest.mark.asyncio
    async def test_run_tool_param_auth_no_field(self, toolbox, auth_token1):
        """Tests running a tool with a param requiring auth, with insufficient auth."""
        tool = await toolbox.load_tool(
            "get-row-by-content-auth", auth_tokens={"my-test-auth": lambda: auth_token1}
        )
        with pytest.raises(ClientResponseError, match="400, message='Bad Request'"):
            await tool.ainvoke({})
