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
4. Running a tool with
    a. Missing params.
    b. Wrong param type.
5. Running a tool with no required auth, with auth provided.
6. Running a tool with required auth:
    a. No auth provided.
    b. Wrong auth provided: The tool requires a different authentication
                            than the one provided.
    c. Correct auth provided.
7. Running a tool with a parameter that requires auth:
    a. No auth provided.
    b. Correct auth provided.
    c. Auth provided does not contain the required claim.
"""

import pytest
import pytest_asyncio
from aiohttp import ClientResponseError
from pydantic import ValidationError

from toolbox_langchain_sdk.client import ToolboxClient

@pytest.mark.asyncio
@pytest.mark.usefixtures("toolbox_server")
class TestE2EClientAsync:
    @pytest.fixture(scope="function")
    def toolbox(self):
        """Provides a ToolboxClient instance for each test."""
        toolbox = ToolboxClient("http://localhost:5000")
        return toolbox

    @pytest_asyncio.fixture(scope="function")
    async def get_n_rows_tool(self, toolbox):
        tool = await toolbox.aload_tool("get-n-rows")
        assert tool._ToolboxTool__async_tool._name == "get-n-rows"
        return tool

    #### Basic e2e tests
    @pytest.mark.parametrize(
        "toolset_name, expected_length, expected_tools",
        [
            ("my-toolset", 1, ["get-row-by-id"]),
            ("my-toolset-2", 2, ["get-n-rows", "get-row-by-id"]),
        ],
    )
    async def test_aload_toolset_specific(
        self, toolbox, toolset_name, expected_length, expected_tools
    ):
        toolset = await toolbox.aload_toolset(toolset_name)
        assert len(toolset) == expected_length
        for tool in toolset:
            name = tool._ToolboxTool__async_tool._name
            assert name in expected_tools

    async def test_aload_toolset_all(self, toolbox):
        toolset = await toolbox.aload_toolset()
        assert len(toolset) == 5
        tool_names = [
            "get-n-rows",
            "get-row-by-id",
            "get-row-by-id-auth",
            "get-row-by-email-auth",
            "get-row-by-content-auth",
        ]
        for tool in toolset:
            name = tool._ToolboxTool__async_tool._name
            assert name in tool_names

    async def test_run_tool_async(self, get_n_rows_tool):
        response = await get_n_rows_tool.ainvoke({"num_rows": "2"})
        result = response["result"]

        assert "row1" in result
        assert "row2" in result
        assert "row3" not in result

    async def test_run_tool_sync(self, get_n_rows_tool):
        response = get_n_rows_tool.invoke({"num_rows": "2"})
        result = response["result"]

        assert "row1" in result
        assert "row2" in result
        assert "row3" not in result

    async def test_run_tool_missing_params(self, get_n_rows_tool):
        with pytest.raises(ValidationError, match="Field required"):
            await get_n_rows_tool.ainvoke({})

    async def test_run_tool_wrong_param_type(self, get_n_rows_tool):
        with pytest.raises(ValidationError, match="Input should be a valid string"):
            await get_n_rows_tool.ainvoke({"num_rows": 2})

    ##### Auth tests
    @pytest.mark.skip(reason="b/389574566")
    async def test_run_tool_unauth_with_auth(self, toolbox, auth_token2):
        """Tests running a tool that doesn't require auth, with auth provided."""
        tool = await toolbox.aload_tool(
            "get-row-by-id", auth_tokens={"my-test-auth": lambda: auth_token2}
        )
        response = await tool.ainvoke({"id": "2"})
        assert "row2" in response["result"]

    async def test_run_tool_no_auth(self, toolbox):
        """Tests running a tool requiring auth without providing auth."""
        tool = await toolbox.aload_tool(
            "get-row-by-id-auth",
        )
        with pytest.raises(ClientResponseError, match="401, message='Unauthorized'"):
            await tool.ainvoke({"id": "2"})

    async def test_run_tool_wrong_auth(self, toolbox, auth_token2):
        """Tests running a tool with incorrect auth."""
        tool = await toolbox.aload_tool(
            "get-row-by-id-auth",
        )
        auth_tool = tool.add_auth_token("my-test-auth", lambda: auth_token2)
        # TODO: Fix error message (b/389577313)
        with pytest.raises(ClientResponseError, match="400, message='Bad Request'"):
            await auth_tool.ainvoke({"id": "2"})

    async def test_run_tool_auth(self, toolbox, auth_token1):
        """Tests running a tool with correct auth."""
        tool = await toolbox.aload_tool(
            "get-row-by-id-auth",
        )
        auth_tool = tool.add_auth_token("my-test-auth", lambda: auth_token1)
        response = await auth_tool.ainvoke({"id": "2"})
        assert "row2" in response["result"]

    async def test_run_tool_param_auth_no_auth(self, toolbox):
        """Tests running a tool with a param requiring auth, without auth."""
        tool = await toolbox.aload_tool("get-row-by-email-auth")
        with pytest.raises(
            PermissionError,
            match="Parameter\(s\) `email` of tool get-row-by-email-auth require authentication\, but no valid authentication sources are registered\. Please register the required sources before use\.",
        ):
            await tool.ainvoke({})

    async def test_run_tool_param_auth(self, toolbox, auth_token1):
        """Tests running a tool with a param requiring auth, with correct auth."""
        tool = await toolbox.aload_tool(
            "get-row-by-email-auth", auth_tokens={"my-test-auth": lambda: auth_token1}
        )
        response = await tool.ainvoke({})
        result = response["result"]
        assert "row4" in result
        assert "row5" in result
        assert "row6" in result

    async def test_run_tool_param_auth_no_field(self, toolbox, auth_token1):
        """Tests running a tool with a param requiring auth, with insufficient auth."""
        tool = await toolbox.aload_tool(
            "get-row-by-content-auth", auth_tokens={"my-test-auth": lambda: auth_token1}
        )
        with pytest.raises(ClientResponseError, match="400, message='Bad Request'"):
            await tool.ainvoke({})


@pytest.mark.usefixtures("toolbox_server")
class TestE2EClientSync:
    @pytest.fixture(scope="session")
    def toolbox(self):
        """Provides a ToolboxClient instance for each test."""
        toolbox = ToolboxClient("http://localhost:5000")
        return toolbox

    @pytest.fixture(scope="function")
    def get_n_rows_tool(self, toolbox):
        tool = toolbox.load_tool("get-n-rows")
        assert tool._ToolboxTool__async_tool._name == "get-n-rows"
        return tool

    #### Basic e2e tests
    @pytest.mark.parametrize(
        "toolset_name, expected_length, expected_tools",
        [
            ("my-toolset", 1, ["get-row-by-id"]),
            ("my-toolset-2", 2, ["get-n-rows", "get-row-by-id"]),
        ],
    )
    def test_load_toolset_specific(
        self, toolbox, toolset_name, expected_length, expected_tools
    ):
        toolset = toolbox.load_toolset(toolset_name)
        assert len(toolset) == expected_length
        for tool in toolset:
            name = tool._ToolboxTool__async_tool._name
            assert name in expected_tools

    def test_aload_toolset_all(self, toolbox):
        toolset = toolbox.load_toolset()
        assert len(toolset) == 5
        tool_names = [
            "get-n-rows",
            "get-row-by-id",
            "get-row-by-id-auth",
            "get-row-by-email-auth",
            "get-row-by-content-auth",
        ]
        for tool in toolset:
            name = tool._ToolboxTool__async_tool._name
            assert name in tool_names

    @pytest.mark.asyncio
    async def test_run_tool_async(self, get_n_rows_tool):
        response = await get_n_rows_tool.ainvoke({"num_rows": "2"})
        result = response["result"]

        assert "row1" in result
        assert "row2" in result
        assert "row3" not in result

    def test_run_tool_sync(self, get_n_rows_tool):
        response = get_n_rows_tool.invoke({"num_rows": "2"})
        result = response["result"]

        assert "row1" in result
        assert "row2" in result
        assert "row3" not in result

    def test_run_tool_missing_params(self, get_n_rows_tool):
        with pytest.raises(ValidationError, match="Field required"):
            get_n_rows_tool.invoke({})

    def test_run_tool_wrong_param_type(self, get_n_rows_tool):
        with pytest.raises(ValidationError, match="Input should be a valid string"):
            get_n_rows_tool.invoke({"num_rows": 2})

    ##### Auth tests
    @pytest.mark.skip(reason="b/389574566")
    def test_run_tool_unauth_with_auth(self, toolbox, auth_token2):
        """Tests running a tool that doesn't require auth, with auth provided."""
        tool = toolbox.load_tool(
            "get-row-by-id", auth_tokens={"my-test-auth": lambda: auth_token2}
        )
        response = tool.invoke({"id": "2"})
        assert "row2" in response["result"]

    def test_run_tool_no_auth(self, toolbox):
        """Tests running a tool requiring auth without providing auth."""
        tool = toolbox.load_tool(
            "get-row-by-id-auth",
        )
        with pytest.raises(ClientResponseError, match="401, message='Unauthorized'"):
            tool.invoke({"id": "2"})

    def test_run_tool_wrong_auth(self, toolbox, auth_token2):
        """Tests running a tool with incorrect auth."""
        tool = toolbox.load_tool(
            "get-row-by-id-auth",
        )
        auth_tool = tool.add_auth_token("my-test-auth", lambda: auth_token2)
        # TODO: Fix error message (b/389577313)
        with pytest.raises(ClientResponseError, match="400, message='Bad Request'"):
            auth_tool.invoke({"id": "2"})

    def test_run_tool_auth(self, toolbox, auth_token1):
        """Tests running a tool with correct auth."""
        tool = toolbox.load_tool(
            "get-row-by-id-auth",
        )
        auth_tool = tool.add_auth_token("my-test-auth", lambda: auth_token1)
        response = auth_tool.invoke({"id": "2"})
        assert "row2" in response["result"]

    def test_run_tool_param_auth_no_auth(self, toolbox):
        """Tests running a tool with a param requiring auth, without auth."""
        tool = toolbox.load_tool("get-row-by-email-auth")
        with pytest.raises(
            PermissionError,
            match="Parameter\(s\) `email` of tool get-row-by-email-auth require authentication\, but no valid authentication sources are registered\. Please register the required sources before use\.",
        ):
            tool.invoke({})

    def test_run_tool_param_auth(self, toolbox, auth_token1):
        """Tests running a tool with a param requiring auth, with correct auth."""
        tool = toolbox.load_tool(
            "get-row-by-email-auth", auth_tokens={"my-test-auth": lambda: auth_token1}
        )
        response = tool.invoke({})
        result = response["result"]
        assert "row4" in result
        assert "row5" in result
        assert "row6" in result

    def test_run_tool_param_auth_no_field(self, toolbox, auth_token1):
        """Tests running a tool with a param requiring auth, with insufficient auth."""
        tool = toolbox.load_tool(
            "get-row-by-content-auth", auth_tokens={"my-test-auth": lambda: auth_token1}
        )
        with pytest.raises(ClientResponseError, match="400, message='Bad Request'"):
            tool.invoke({})
