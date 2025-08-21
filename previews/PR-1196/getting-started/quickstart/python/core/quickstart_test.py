import os
import subprocess
import sys
import pytest
import ast
from pathlib import Path


# --- Core Framework Configuration ---
CORE_CONFIG = {
    "name": "Google GenAI Core",
    "expected_imports": {
        'google': ['genai'],
        'google.genai.types': ['Content', 'FunctionDeclaration', 'GenerateContentConfig', 'Part', 'Tool'],
        'toolbox_core': ['ToolboxClient'],
        'asyncio': []
    },
    "required_packages": ['google-genai', 'toolbox-core', 'pytest']
}


# --- Shared Fixtures ---
@pytest.fixture(scope="module")
def quickstart_path():
    """Provides the path to the quickstart.py script."""
    path = Path("quickstart.py")
    if not path.exists():
        pytest.fail(f"Core quickstart script not found: {path}")
    return path

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


# --- Import Tests ---
class TestCoreImports:
    """Test Core framework imports and dependencies."""

    def test_required_imports_present(self, quickstart_path):
        """Test that all required Core imports are present."""
        with open(quickstart_path, 'r') as f:
            tree = ast.parse(f.read())
        
        # Extract imports
        imports = []
        for node in ast.walk(tree):
            if isinstance(node, ast.Import):
                for alias in node.names:
                    imports.append((alias.name, []))
            elif isinstance(node, ast.ImportFrom):
                module = node.module or ''
                names = [alias.name for alias in node.names]
                imports.append((module, names))
        
        found_imports = {module: names for module, names in imports}
        
        # Check each expected import
        missing_imports = []
        for expected_module, expected_names in CORE_CONFIG["expected_imports"].items():
            if expected_module not in found_imports:
                missing_imports.append(f"Module '{expected_module}' not imported")
                continue
            
            if expected_names:
                found_names = found_imports[expected_module]
                for expected_name in expected_names:
                    if expected_name not in found_names:
                        missing_imports.append(f"'{expected_name}' not imported from '{expected_module}'")
        
        assert not missing_imports, f"Missing Core imports: {missing_imports}"


# --- Execution Tests ---
class TestCoreExecution:
    """Test Core framework execution and output validation."""

    @pytest.fixture(autouse=True)
    def check_prerequisites(self):
        """Check for required environment variables."""
        if not os.getenv("GOOGLE_API_KEY"):
            pytest.skip("GOOGLE_API_KEY environment variable is not set.")

    @pytest.fixture(scope="function")
    def script_output(self, quickstart_path):
        """Run the Core quickstart script and return output."""
        try:
            result = subprocess.run(
                [sys.executable, str(quickstart_path)],
                capture_output=True,
                text=True,
                timeout=120
            )
            return result
        except subprocess.TimeoutExpired:
            pytest.fail("Core script execution timed out after 2 minutes")

    def test_script_execution_success(self, script_output):
        """Test that the Core script runs without errors."""
        assert script_output.returncode == 0, f"Core script failed with return code {script_output.returncode}. stderr: {script_output.stderr}"

    def test_script_produces_output(self, script_output):
        """Test that the Core script produces meaningful output."""
        assert script_output.stdout.strip(), "Core script produced no output"
        
        # Check output length
        output_lines = [line.strip() for line in script_output.stdout.split('\n') if line.strip()]
        assert len(output_lines) >= 4, f"Core script produced insufficient output ({len(output_lines)} lines)"

    def test_hotel_keywords_in_output(self, script_output, golden_keywords):
        """Test that expected hotel keywords appear in Core output."""
        actual_output = script_output.stdout
        found_keywords = [keyword for keyword in golden_keywords if keyword in actual_output]
        missing_keywords = [keyword for keyword in golden_keywords if keyword not in actual_output]
        
        # Require all keywords to be present
        assert not missing_keywords, f"Core script: Missing required keywords: {missing_keywords}. Found: {found_keywords}"


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
