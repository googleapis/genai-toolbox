# Copyright 2025 Google LLC
# 
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# 
#      http://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os
import pytest
from pathlib import Path
import asyncio
import sys
import importlib.util

ORCH_NAME = os.environ.get("ORCH_NAME")
module_path = f"python.{ORCH_NAME}.quickstart"
quickstart = importlib.import_module(module_path)


@pytest.fixture(scope="module")
def golden_keywords():
    """Loads expected keywords from the golden.txt file."""
    golden_file_path = Path("../golden.txt")
    if not golden_file_path.exists():
        pytest.fail(f"Golden file not found: {golden_file_path}")
    try:
        with open(golden_file_path, 'r') as f:
            return [line.strip() for line in f.readlines() if line.strip()]
    except Exception as e:
        pytest.fail(f"Could not read golden.txt: {e}")


# --- Execution Tests ---
class TestExecution:
    """Test framework execution and output validation."""

    @pytest.fixture(scope="function")
    def script_output(self, capsys):
        """Run the quickstart function and return its output."""

        # TODO: Add better validation for ADK once we have a way to capture its
        # output.
        if ORCH_NAME == "adk":
            return quickstart.app.root_agent.name
        else:
            asyncio.run(quickstart.main())

        return capsys.readouterr()

    def test_script_runs_without_errors(self, script_output):
        """Test that the script runs and produces no stderr."""
        if ORCH_NAME == "adk":
            return
        assert script_output.err == "", f"Script produced stderr: {script_output.err}"

    def test_keywords_in_output(self, script_output, golden_keywords):
        """Test that expected keywords are present in the script's output."""
        
        if ORCH_NAME == "adk":
            assert script_output == "root_agent"
            return
        output = script_output.out
        missing_keywords = [kw for kw in golden_keywords if kw not in output]
        assert not missing_keywords, f"Missing keywords in output: {missing_keywords}"
