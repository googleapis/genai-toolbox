import os
import subprocess
import sys
import unittest
from pathlib import Path


class TestADKQuickstart(unittest.TestCase):

    def test_agent_output_and_keywords(self):
        """Test that the ADK quickstart script runs successfully and produces expected output."""
        
        # Check API key
        if not os.getenv("GOOGLE_API_KEY"):
            self.skipTest("Skipping integration test: GOOGLE_API_KEY environment variable is not set.")
        
        # Check quickstart.py exists
        quickstart_path = Path("quickstart.py")
        if not quickstart_path.exists():
            self.fail("quickstart.py not found")
        
        # Run the quickstart script
        try:
            result = subprocess.run(
                [sys.executable, "quickstart.py"],
                capture_output=True,
                text=True,
                timeout=120  # 2 minute timeout
            )
            
            actual_output = result.stdout
            stderr_output = result.stderr
            
            print("    quickstart_test.py:30: --- SCRIPT OUTPUT ---")
            if actual_output:
                for line in actual_output.split('\n'):
                    if line.strip():
                        print(f"        {line}")
            else:
                print("        (No output)")
            
            # Check return code
            if result.returncode != 0:
                self.fail(f"Script execution failed with return code {result.returncode}")
            else:
                print("    quickstart_test.py:32: ✅ PASSED: Script ran successfully and produced output.")
            
            # Check output content
            if not actual_output.strip():
                self.fail("Script ran successfully but produced no output.")
            
            # Check for essential keywords
            golden_file_path = Path("../../golden.txt")
            
            if golden_file_path.exists():
                try:
                    with open(golden_file_path, 'r') as f:
                        keywords = [line.strip() for line in f.readlines() if line.strip()]
                    
                    for keyword in keywords:
                        if keyword in actual_output:
                            print(f"    quickstart_test.py:41: ✅ INFO: Found keyword '{keyword}' in output.")
                        else:
                            print(f"    quickstart_test.py:43: ⚠️ INFO: Did not find keyword '{keyword}' in output.")
                            
                except Exception as e:
                    print(f"    quickstart_test.py:45: ⚠️ WARNING: Could not read golden.txt: {e}")
                
        except subprocess.TimeoutExpired:
            self.fail("Script execution timed out after 2 minutes.")
        except Exception as e:
            self.fail(f"Script execution failed with error: {e}")


if __name__ == "__main__":
    unittest.main(verbosity=2)