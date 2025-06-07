import unittest
import subprocess
import json
import sys
import os
import time
# import requests_mock # requests-mock won't work directly with subprocess, so removed.
import uuid
from threading import Thread
from queue import Queue, Empty
from typing import Optional, IO # For type hints

# Assuming this test file is in py_toolbox/tests/
BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__))) # py_toolbox dir
MAIN_PY_PATH = os.path.join(BASE_DIR, "main.py")
# Use the existing mcp_test_tools.yaml for this test as it defines known tools
TEST_CONFIG_PATH = os.path.join(BASE_DIR, "tests", "mcp_test_tools.yaml")

MOCK_HUB_URL = "http://mockhub.test/api/v1" # This URL will be intentionally unresolvable for the test
TEST_MICROSERVICE_ID = f"test_pytoolbox_ms_{str(uuid.uuid4())[:8]}"


class TestHubRegistration(unittest.TestCase):
    mcp_server_process: Optional[subprocess.Popen] = None
    stdout_queue: Optional[Queue] = None
    stderr_queue: Optional[Queue] = None
    stdout_thread: Optional[Thread] = None
    stderr_thread: Optional[Thread] = None

    @classmethod
    def _enqueue_output(cls, pipe: Optional[IO[bytes]], queue: Queue):
        if pipe is None:
            return
        try:
            for line in iter(pipe.readline, b''):
                queue.put(line.decode('utf-8', errors='replace')) # Add error handling for decode
        except ValueError: # Pipe closed
            pass
        finally:
            if pipe:
                try:
                    pipe.close()
                except Exception:
                    pass # Ignore errors on close if already closed or problematic

    @classmethod
    def setUpClass(cls):
        python_executable = sys.executable
        cls.env = os.environ.copy()
        cls.env["MCP_HUB_API_URL"] = MOCK_HUB_URL
        cls.env["PYTOOLBOX_MICROSERVICE_ID"] = TEST_MICROSERVICE_ID

        # Log paths for subprocess output capture
        cls.stdout_log_path = os.path.join(BASE_DIR, "tests", "hub_reg_stdout.log")
        cls.stderr_log_path = os.path.join(BASE_DIR, "tests", "hub_reg_stderr.log")

        command = [
            python_executable, MAIN_PY_PATH,
            "--config", TEST_CONFIG_PATH,
            "--log-level", "DEBUG",
            "mcp-serve"
        ]

        print(f"Starting py_toolbox (mcp-serve) for Hub registration test with command: {' '.join(command)}")
        print(f"Env for subprocess: MCP_HUB_API_URL={cls.env['MCP_HUB_API_URL']}, PYTOOLBOX_MICROSERVICE_ID={cls.env['PYTOOLBOX_MICROSERVICE_ID']}")
        print(f"Subprocess stdout will be logged to: {cls.stdout_log_path}")
        print(f"Subprocess stderr will be logged to: {cls.stderr_log_path}")

        # Open log files for stdout and stderr of the subprocess
        cls.stdout_log_file = open(cls.stdout_log_path, 'w')
        cls.stderr_log_file = open(cls.stderr_log_path, 'w')

        cls.mcp_server_process = subprocess.Popen(
            command,
            stdin=subprocess.PIPE,
            stdout=cls.stdout_log_file, # Redirect to file
            stderr=cls.stderr_log_file, # Redirect to file
            env=cls.env,
            bufsize=1,
            cwd=os.path.dirname(BASE_DIR) # Run from parent of py_toolbox
        )

        print(f"PID of mcp-serve: {cls.mcp_server_process.pid}")
        time.sleep(2.0) # Increased time for server startup and registration attempt

        if cls.mcp_server_process.poll() is not None:
            cls._cleanup_process_and_logs(read_logs=True) # Ensure logs are read before raising
            raise RuntimeError(f"py_toolbox (mcp-serve) failed to start. Exit code: {cls.mcp_server_process.returncode}.")
        print("py_toolbox (mcp-serve) started for Hub registration test.")

    @classmethod
    def tearDownClass(cls):
        cls._cleanup_process_and_logs(read_logs=False) # Logs should be read by tests or here if needed

    @classmethod
    def _cleanup_process_and_logs(cls, read_logs=False):
        if cls.mcp_server_process and cls.mcp_server_process.poll() is None:
            print("Stopping py_toolbox (mcp-serve)...")
            if cls.mcp_server_process.stdin and not cls.mcp_server_process.stdin.closed:
                try:
                    cls.mcp_server_process.stdin.close()
                except (OSError, ValueError): pass

            try:
                cls.mcp_server_process.terminate()
                cls.mcp_server_process.wait(timeout=3)
            except subprocess.TimeoutExpired:
                print("py_toolbox (mcp-serve) did not terminate gracefully, killing.")
                cls.mcp_server_process.kill()
                try:
                    cls.mcp_server_process.wait(timeout=3)
                except subprocess.TimeoutExpired:
                    print("Failed to kill process even after timeout.")
            if cls.mcp_server_process: # Check if it still exists
                 print(f"py_toolbox (mcp-serve) stopped. Exit code: {cls.mcp_server_process.returncode}")
        cls.mcp_server_process = None

        if cls.stdout_log_file and not cls.stdout_log_file.closed: cls.stdout_log_file.close()
        if cls.stderr_log_file and not cls.stderr_log_file.closed: cls.stderr_log_file.close()

        if read_logs: # For reading logs if startup failed for example
            if os.path.exists(cls.stdout_log_path):
                with open(cls.stdout_log_path, 'r') as f: print(f"STARTUP STDOUT:\n{f.read()}")
            if os.path.exists(cls.stderr_log_path):
                with open(cls.stderr_log_path, 'r') as f: print(f"STARTUP STDERR:\n{f.read()}")


    def get_log_content(self, log_path):
        # Ensure files are flushed before reading from them
        if self.stdout_log_file and not self.stdout_log_file.closed: self.stdout_log_file.flush()
        if self.stderr_log_file and not self.stderr_log_file.closed: self.stderr_log_file.flush()

        if os.path.exists(log_path):
            with open(log_path, 'r') as f:
                return f.read()
        return ""

    def test_tool_registration_on_startup(self):
        print("\\n--- Test: py_toolbox registers its tools with Hub on startup ---")

        expected_tool_name = "ask_sqlite_version"

        # The registration attempt happens on startup. We need to check the logs.
        # Give a little more time for logs to be flushed if there was any late registration activity.
        time.sleep(0.5)

        # Read logs from the files
        # Uvicorn and FastAPI might log to stdout, our application logger (set to DEBUG) might go to stderr by default
        # or stdout depending on Click/Python's handling in subprocess.
        # The Popen setup redirects both to separate files. Let's check both.

        stdout_output = self.get_log_content(self.stdout_log_path)
        stderr_output = self.get_log_content(self.stderr_log_path)

        full_log_output = stdout_output + stderr_output
        print(f"py_toolbox combined output during registration test run:\n{full_log_output}")

        # Check for the attempt to register
        self.assertIn(
            f"Attempting to register 1 tools with MCP Hub at {MOCK_HUB_URL} for microservice_id '{TEST_MICROSERVICE_ID}'",
            full_log_output,
            "Log message for starting registration attempt not found."
        )
        # Check for the specific tool being prepared for registration
        self.assertIn(
            f"Registering tool '{expected_tool_name}' with payload:",
            full_log_output,
            "Log message for preparing specific tool registration not found."
        )

        # Since MOCK_HUB_URL is not a real server, requests.post should fail.
        # We expect to see a log message indicating this failure.
        # The exact error message can vary depending on the OS and network stack.
        # Common messages include "Connection refused", "Name or service not known", "Temporary failure in name resolution".
        # We'll check for a generic part of the error log.
        self.assertIn(
            f"HTTP request error while registering tool '{expected_tool_name}' with MCP Hub",
            full_log_output,
            "Expected log message about HTTP request error not found."
        )
        # More specific check for connection failure, but make it broad enough
        self.assertTrue(
            "Failed to establish a new connection" in full_log_output or \
            "Connection refused" in full_log_output or \
            "Name or service not known" in full_log_output or \
            "Temporary failure in name resolution" in full_log_output or \
            "cannot resolve host" in full_log_output, # From some environments
            "Expected a connection-related error log when trying to register with the mock Hub URL."
        )


if __name__ == '__main__':
    if not os.path.exists(MAIN_PY_PATH):
        print(f"FATAL: main.py not found at {MAIN_PY_PATH}")
        sys.exit(1)
    if not os.path.exists(TEST_CONFIG_PATH):
        print(f"FATAL: Test config not found at {TEST_CONFIG_PATH}")
        sys.exit(1)
    unittest.main()

EOL_HUB_REG_TEST_PY
echo "Created py_toolbox/tests/test_hub_registration.py"
