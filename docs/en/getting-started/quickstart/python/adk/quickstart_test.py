import os
import pytest
from pathlib import Path
import asyncio
from quickstart import main

@pytest.fixture(scope="module")
def golden_keywords():
    """Loads expected keywords from the golden.txt file."""
    golden_file_path = Path("../../golden.txt")
    if not golden_file_path.exists():
        pytest.fail(f"Golden file not found: {golden_file_path}")
    try:
        with open(golden_file_path, 'r') as f:
            return [line.strip() for line in f.readlines() if line.strip()]
    except Exception as e:
        pytest.fail(f"Could not read golden.txt: {e}")


# --- Execution Tests ---
class TestADKExecution:
    """Test ADK framework execution and output validation."""

    @pytest.fixture(autouse=True)
    def check_prerequisites(self):
        """Check for required environment variables."""
        if not os.getenv("GOOGLE_API_KEY"):
            pytest.skip("GOOGLE_API_KEY environment variable is not set.")

    @pytest.fixture(scope="function")
    def script_output(self, capsys):
        """Run the quickstart function and return its output."""
        asyncio.run(main())
        return capsys.readouterr()

    def test_script_runs_without_errors(self, script_output):
        """Test that the script runs and produces no stderr."""
        assert script_output.err == "", f"Script produced stderr: {script_output.err}"

    def test_script_produces_output(self, script_output):
        """Test that the script produces some output to stdout."""
        assert script_output.out.strip(), "Script produced no stdout"

    def test_keywords_in_output(self, script_output, golden_keywords):
        """Test that expected keywords are present in the script's output."""
        output = script_output.out
        missing_keywords = [kw for kw in golden_keywords if kw not in output]
        assert not missing_keywords, f"Missing keywords in output: {missing_keywords}"
