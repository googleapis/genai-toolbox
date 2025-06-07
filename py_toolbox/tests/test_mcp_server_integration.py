import unittest
import subprocess
import json
import sys
import os
import time
from threading import Thread
from queue import Queue, Empty
from typing import Optional # Added for type hint, though not strictly necessary for script to run

# Adjust path to import py_toolbox.main if necessary, or run from parent dir
# This assumes tests are run from the parent directory of py_toolbox,
# or py_toolbox is in PYTHONPATH.

# Determine BASE_DIR relative to this test file's location
# __file__ is py_toolbox/tests/test_mcp_server_integration.py
# os.path.dirname(__file__) is py_toolbox/tests
# os.path.dirname(os.path.dirname(__file__)) is py_toolbox
BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
MAIN_PY_PATH = os.path.join(BASE_DIR, "main.py")
TEST_CONFIG_PATH = os.path.join(BASE_DIR, "tests", "mcp_test_tools.yaml")


class MCPTestClient:
    def __init__(self, config_path=TEST_CONFIG_PATH):
        self.process: Optional[subprocess.Popen] = None
        self.request_id_counter: int = 0
        self.config_path: str = config_path
        self._stdout_queue: Queue = Queue()
        self._stderr_queue: Queue = Queue()
        self._stdout_thread: Optional[Thread] = None
        self._stderr_thread: Optional[Thread] = None

    def _enqueue_output(self, pipe, queue: Queue):
        try:
            for line in iter(pipe.readline, b''):
                queue.put(line.decode('utf-8'))
        except ValueError: # Pipe closed before readline could finish (e.g. process killed)
            pass
        finally:
            if pipe and not pipe.closed:
                pipe.close()

    def start_server(self):
        if self.process and self.process.poll() is None:
            print("Server already running.")
            return

        python_executable = sys.executable
        # Ensure paths are absolute or correctly relative if main.py uses relative paths itself
        command = [python_executable, MAIN_PY_PATH, "--config", self.config_path, "--log-level", "DEBUG", "mcp-serve"]

        print(f"Starting MCP server with command: {' '.join(command)}")
        # Set working directory to BASE_DIR so that main.py can find tools.yaml if relative paths are used in it.
        # And also so that py_toolbox can be imported correctly by main.py
        self.process = subprocess.Popen(
            command,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            bufsize=1, # Line buffered
            cwd=os.path.dirname(BASE_DIR) # Run from parent of py_toolbox for `python py_toolbox/main.py`
        )

        self._stdout_queue = Queue()
        self._stderr_queue = Queue()

        self._stdout_thread = Thread(target=self._enqueue_output, args=(self.process.stdout, self._stdout_queue), daemon=True)
        self._stderr_thread = Thread(target=self._enqueue_output, args=(self.process.stderr, self._stderr_queue), daemon=True)
        self._stdout_thread.start()
        self._stderr_thread.start()

        time.sleep(1.0) # Increased sleep for server to fully initialize, especially if loading many components
        if self.process.poll() is not None:
            stderr_output = self.read_all_stderr(timeout=0.2) # Try to get more stderr
            stdout_output = self.read_all_stdout(timeout=0.2)
            raise RuntimeError(f"MCP Server failed to start. Exit code: {self.process.returncode}.\nSTDOUT:\n{stdout_output}\nSTDERR:\n{stderr_output}")
        print("MCP Server started.")


    def stop_server(self):
        if self.process and self.process.poll() is None:
            print("Stopping MCP server...")
            if self.process.stdin and not self.process.stdin.closed:
                try:
                    self.process.stdin.close()
                except BrokenPipeError:
                    print("Stdin already closed (server might have shut down).")

            try:
                self.process.wait(timeout=3)
            except subprocess.TimeoutExpired:
                print("Server did not terminate gracefully, killing.")
                self.process.kill()
                self.process.wait() # Ensure process is reaped after kill
            print(f"MCP Server stopped. Exit code: {self.process.returncode}")

        # Wait for threads to finish
        if self._stdout_thread and self._stdout_thread.is_alive():
            self._stdout_thread.join(timeout=1)
        if self._stderr_thread and self._stderr_thread.is_alive():
            self._stderr_thread.join(timeout=1)

        # Ensure pipes are closed if not already
        if self.process:
            if self.process.stdout and not self.process.stdout.closed: self.process.stdout.close()
            if self.process.stderr and not self.process.stderr.closed: self.process.stderr.close()

        self.process = None


    def read_all_stdout(self, timeout=0.1):
        lines = []
        while True:
            try:
                lines.append(self._stdout_queue.get(timeout=timeout))
            except Empty:
                break
        return "".join(lines)

    def read_all_stderr(self, timeout=0.1):
        lines = []
        while True:
            try:
                lines.append(self._stderr_queue.get(timeout=timeout))
            except Empty:
                break
        return "".join(lines)

    def send_request(self, method: str, params: Optional[dict] = None, is_notification: bool = False) -> Optional[str]:
        if not self.process or self.process.poll() is not None:
            raise ConnectionError("MCP Server is not running.")

        self.request_id_counter += 1
        request_obj: Dict[str, Any] = { # Ensure type for request_obj
            "jsonrpc": "2.0",
            "method": method,
        }
        if params is not None:
            request_obj["params"] = params

        current_request_id_str: Optional[str] = None
        if not is_notification:
            current_request_id_str = str(self.request_id_counter)
            request_obj["id"] = current_request_id_str # JSON-RPC ID can be string or number

        request_str = json.dumps(request_obj) + "\\n"
        print(f"Client > Server: {request_str.strip()}")
        try:
            if self.process.stdin and not self.process.stdin.closed:
                self.process.stdin.write(request_str.encode('utf-8'))
                self.process.stdin.flush()
            else:
                raise ConnectionError("Stdin is closed, cannot send request.")
        except (BrokenPipeError, ValueError) as e:
            raise ConnectionError(f"Failed to send request to MCP server: {e}. Server might have crashed.")

        return current_request_id_str

    def receive_response(self, timeout=3.0) -> Optional[dict]: # Increased timeout
        if not self.process or self.process.poll() is not None:
            stderr = self.read_all_stderr(timeout=0.1)
            if stderr: print(f"Server stderr before ConnectionError in receive_response: {stderr}")
            raise ConnectionError("MCP Server is not running or pipe broken.")

        response_line = ""
        try:
            response_line = self._stdout_queue.get(timeout=timeout)
            if not response_line: # Should not happen with current _enqueue_output
                return None

            print(f"Client < Server: {response_line.strip()}")
            return json.loads(response_line)
        except Empty:
            print(f"Timeout waiting for response (waited {timeout}s).")
            stderr_output = self.read_all_stderr(timeout=0.1)
            if stderr_output:
                print(f"Server stderr during timeout in receive_response:\n{stderr_output}")
            return None
        except json.JSONDecodeError as e:
            print(f"Failed to parse JSON response: {response_line.strip()}. Error: {e}")
            # Return a dict that mimics an error structure for assertResponseError to parse
            return {"jsonrpc": "2.0", "error": {"code": -32001, "message": "JSONDecodeError from client", "data": response_line.strip()}, "id": None}


class TestMCPServerIntegration(unittest.TestCase):
    client: Optional[MCPTestClient] = None # Class attribute for client

    @classmethod
    def setUpClass(cls):
        print(f"Using MAIN_PY_PATH: {MAIN_PY_PATH}")
        print(f"Using TEST_CONFIG_PATH: {TEST_CONFIG_PATH}")
        if not os.path.exists(MAIN_PY_PATH):
             raise FileNotFoundError(f"main.py not found at {MAIN_PY_PATH}")
        if not os.path.exists(TEST_CONFIG_PATH):
             raise FileNotFoundError(f"Test config {TEST_CONFIG_PATH} not found.")

        cls.client = MCPTestClient(config_path=TEST_CONFIG_PATH)
        try:
            cls.client.start_server()
        except Exception as e:
            if cls.client: # Attempt to get more info if client was partially initialized
                stderr_output = cls.client.read_all_stderr(timeout=0.5)
                stdout_output = cls.client.read_all_stdout(timeout=0.5)
                cls.client.stop_server()
                print(f"MCP Server setup failed.\nSTDOUT:\n{stdout_output}\nSTDERR:\n{stderr_output}")
            raise RuntimeError(f"Failed to set up MCP server for tests: {e}")


    @classmethod
    def tearDownClass(cls):
        if cls.client:
            cls.client.stop_server()
            # Read any remaining stderr/stdout after server stop
            time.sleep(0.1) # Give queues a moment
            stderr_output = cls.client.read_all_stderr(timeout=0.2)
            stdout_output = cls.client.read_all_stdout(timeout=0.2)
            if stderr_output: print(f"Final server STDERR:\n{stderr_output}")
            if stdout_output: print(f"Final server STDOUT:\n{stdout_output}")


    def assertResponseSuccess(self, response: Optional[dict], expected_id: Optional[str]):
        self.assertIsNotNone(response, "Response should not be None")
        assert response is not None # For type checker
        self.assertEqual(response.get("jsonrpc"), "2.0")
        if expected_id is not None: # Notifications won't have ID in response
            self.assertEqual(str(response.get("id")), expected_id)
        self.assertIn("result", response)
        self.assertNotIn("error", response, f"Response was an error: {response.get('error')}")
        return response["result"]

    def assertResponseError(self, response: Optional[dict], expected_id: Optional[str], expected_code: Optional[int] = None):
        self.assertIsNotNone(response, "Response should not be None")
        assert response is not None # For type checker
        self.assertEqual(response.get("jsonrpc"), "2.0")
        # For some errors (like parse error), ID might be null even if request had one.
        # So, only assert ID if it's present in the error response.
        if response.get("id") is not None and expected_id is not None:
            self.assertEqual(str(response.get("id")), expected_id)

        self.assertNotIn("result", response)
        self.assertIn("error", response, "Response missing 'error' field")
        error_obj = response["error"]
        self.assertIsInstance(error_obj, dict, "'error' field should be an object")
        self.assertIn("code", error_obj)
        self.assertIn("message", error_obj)
        if expected_code is not None:
            self.assertEqual(error_obj["code"], expected_code)
        return error_obj


    def test_01_list_tools(self):
        print("\\n--- Test: list_tools ---")
        assert self.client is not None # For type checker
        req_id = self.client.send_request("list_tools")
        response = self.client.receive_response()
        result = self.assertResponseSuccess(response, req_id)
        self.assertIsInstance(result, list)
        self.assertEqual(len(result), 1)
        self.assertEqual(result[0]["name"], "ask_sqlite_version")
        self.assertIn("SQLite version", result[0]["description"])

    def test_02_get_tool_description(self):
        print("\\n--- Test: get_tool_description ---")
        assert self.client is not None
        req_id = self.client.send_request("get_tool_description", {"tool_name": "ask_sqlite_version"})
        response = self.client.receive_response()
        result = self.assertResponseSuccess(response, req_id)
        self.assertEqual(result["name"], "ask_sqlite_version")
        self.assertIn("input_schema", result)
        self.assertIn("statement", result["input_schema"]["properties"])

    def test_03_invoke_tool_success(self):
        print("\\n--- Test: invoke_tool_success ---")
        assert self.client is not None
        params = {
            "tool_name": "ask_sqlite_version",
            "invoke_params": {
                "statement": "SELECT sqlite_version() AS version;"
                # args is optional and not needed for this query
            }
        }
        req_id = self.client.send_request("invoke_tool", params)
        response = self.client.receive_response()
        result = self.assertResponseSuccess(response, req_id)
        self.assertIsInstance(result, list)
        self.assertEqual(len(result), 1)
        self.assertIn("version", result[0])
        print(f"SQLite version from tool: {result[0]['version']}")

    def test_04_invoke_tool_not_found(self):
        print("\\n--- Test: invoke_tool_not_found ---")
        assert self.client is not None
        params = {"tool_name": "non_existent_tool", "invoke_params": {}}
        req_id = self.client.send_request("invoke_tool", params)
        response = self.client.receive_response()
        self.assertResponseError(response, req_id, expected_code=-32601) # JSONRPC_METHOD_NOT_FOUND


    def test_05_invalid_json_request(self):
        print("\\n--- Test: invalid_json_request ---")
        assert self.client is not None
        if not self.client.process or self.client.process.poll() is not None:
            self.fail("MCP Server is not running for invalid_json_request test")

        invalid_json_str = "{'jsonrpc': '2.0', 'method': 'foo', 'id': 'err-req-1'}\\n"
        self.client.process.stdin.write(invalid_json_str.encode('utf-8'))
        self.client.process.stdin.flush()

        response = self.client.receive_response()
        # JSONRPC_PARSE_ERROR = -32700. ID is null for parse error.
        error_obj = self.assertResponseError(response, None, expected_code=-32700)
        self.assertIn("Failed to parse JSON request", error_obj["message"])

    def test_06_invalid_jsonrpc_structure(self):
        print("\\n--- Test: invalid_jsonrpc_structure ---")
        assert self.client is not None
        # Missing 'jsonrpc: "2.0"'
        invalid_rpc_str = json.dumps({"method": "list_tools", "id": "err-req-2"}) + "\\n"
        self.client.process.stdin.write(invalid_rpc_str.encode('utf-8'))
        self.client.process.stdin.flush()

        response = self.client.receive_response()
        self.assertResponseError(response, "err-req-2", expected_code=-32600) # JSONRPC_INVALID_REQUEST

if __name__ == '__main__':
    print(f"Executing MCP Integration Tests...")
    print(f"Python Executable: {sys.executable}")
    print(f"Current Working Directory: {os.getcwd()}")
    print(f"Calculated BASE_DIR: {BASE_DIR}")
    print(f"MAIN_PY_PATH for testing: {MAIN_PY_PATH}")
    print(f"TEST_CONFIG_PATH for testing: {TEST_CONFIG_PATH}")

    if not os.path.exists(MAIN_PY_PATH):
        print(f"CRITICAL ERROR: main.py not found at {MAIN_PY_PATH}")
        sys.exit(1)
    if not os.path.exists(TEST_CONFIG_PATH):
        print(f"CRITICAL ERROR: Test config {TEST_CONFIG_PATH} not found.")
        sys.exit(1)

    unittest.main()

EOL_MCP_TEST_SCRIPT
echo "Created py_toolbox/tests/test_mcp_server_integration.py"
