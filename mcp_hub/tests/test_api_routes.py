import unittest # Using unittest for structure, but could also be pytest style
from fastapi.testclient import TestClient
from sqlalchemy.orm import Session # For type hinting if needed by test logic
import datetime
from typing import Dict, Any # For payload type hints

# Adjust import path based on how tests are run (e.g. from mcp_hub parent dir)
from mcp_hub.main import app # The FastAPI app instance
from mcp_hub.db.database import Base, get_db as get_main_db # Main DB and Base
from mcp_hub.tests.test_database import TestingSessionLocal, override_get_db, create_test_tables, drop_test_tables
from mcp_hub.models import tool_registry_models as models # Pydantic and SQLAlchemy models

# Override the get_db dependency for the FastAPI app during tests
app.dependency_overrides[get_main_db] = override_get_db

class TestToolRegistryAPI(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        """ Create database tables before all tests in this class. """
        create_test_tables()
        cls.client = TestClient(app) # Create TestClient once per class

    @classmethod
    def tearDownClass(cls):
        """ Drop database tables after all tests in this class. """
        drop_test_tables()

    def setUp(self):
        """ Before each test, clear data from tables to ensure test isolation. """
        with TestingSessionLocal() as db:
            # Order of deletion might matter if there are foreign key constraints.
            # For RegisteredTool, it's standalone for now.
            db.query(models.RegisteredTool).delete()
            db.commit()


    def test_01_register_new_tool(self):
        print("\\n--- Test: Register New Tool ---")
        payload: Dict[str, Any] = {
            "tool_name": "my_test_tool_1",
            "microservice_id": "ms_1",
            "description": "A test tool for registration",
            "invocation_info": {"type": "mcp", "command": "run_ms_1"},
            "mcp_manifest": {"name": "my_test_tool_1_manifest", "description": "manifest_desc", "input_schema": {"type": "object", "properties": {}}}
        }
        response = self.client.post("/api/v1/tools", json=payload)
        self.assertEqual(response.status_code, 201, f"Response JSON: {response.json()}")
        data = response.json()
        self.assertEqual(data["tool_name"], payload["tool_name"])
        self.assertEqual(data["microservice_id"], payload["microservice_id"])
        self.assertIn("id", data)
        self.assertIn("registered_at", data)
        # self.tool_id_1 = data["id"] # If needed for chained tests, manage carefully due to test isolation

    def test_02_register_existing_tool_updates(self):
        print("\\n--- Test: Register Existing Tool (Update) ---")
        initial_payload: Dict[str, Any] = {
            "tool_name": "updatable_tool",
            "microservice_id": "ms_update",
            "description": "Initial description",
            "invocation_info": {"type": "mcp", "command": "run_ms_update_v1"},
            "mcp_manifest": {"name": "updatable_tool_m", "description": "v1 manifest", "input_schema": {}}
        }
        reg_response = self.client.post("/api/v1/tools", json=initial_payload)
        self.assertEqual(reg_response.status_code, 201, f"Initial registration failed: {reg_response.json()}")
        reg_data = reg_response.json()

        update_payload: Dict[str, Any] = {
            "tool_name": "updatable_tool",
            "microservice_id": "ms_update",
            "description": "Updated description",
            "invocation_info": {"type": "mcp", "command": "run_ms_update_v2"},
            "mcp_manifest": {"name": "updatable_tool_m", "description": "v2 manifest", "input_schema": {"properties":{"new_param":{"type":"string"}}}}
        }
        response = self.client.post("/api/v1/tools", json=update_payload)
        self.assertEqual(response.status_code, 200, f"Update failed: {response.json()}")
        data = response.json()
        self.assertEqual(data["tool_name"], update_payload["tool_name"])

        detail_response = self.client.get(f"/api/v1/tools/{reg_data['id']}")
        self.assertEqual(detail_response.status_code, 200)
        detail_data = detail_response.json()
        self.assertEqual(detail_data["description"], "Updated description")
        self.assertEqual(detail_data["invocation_info"]["command"], "run_ms_update_v2")
        self.assertEqual(detail_data["mcp_manifest"]["description"], "v2 manifest")


    def test_03_list_tools_empty(self):
        print("\\n--- Test: List Tools (Empty) ---")
        response = self.client.get("/api/v1/tools")
        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json(), [])

    def test_04_list_tools_with_data_and_filter(self):
        print("\\n--- Test: List Tools (With Data & Filter) ---")
        payload1: Dict[str, Any] = {"tool_name": "list_tool_1", "microservice_id": "ms_list_A", "description": "Tool for listing A",
                   "invocation_info": {}, "mcp_manifest": {"name":"lt1_m", "description":"d_lt1", "input_schema":{}}}
        self.client.post("/api/v1/tools", json=payload1)
        payload2: Dict[str, Any] = {"tool_name": "list_tool_2", "microservice_id": "ms_list_B", "description": "Tool for listing B",
                   "invocation_info": {}, "mcp_manifest": {"name":"lt2_m", "description":"d_lt2", "input_schema":{}}}
        self.client.post("/api/v1/tools", json=payload2)

        # List all
        response_all = self.client.get("/api/v1/tools")
        self.assertEqual(response_all.status_code, 200)
        data_all = response_all.json()
        self.assertEqual(len(data_all), 2)

        # Filter by microservice_id
        response_filtered = self.client.get("/api/v1/tools?microservice_id=ms_list_A")
        self.assertEqual(response_filtered.status_code, 200)
        data_filtered = response_filtered.json()
        self.assertEqual(len(data_filtered), 1)
        self.assertEqual(data_filtered[0]["tool_name"], "list_tool_1")
        self.assertEqual(data_filtered[0]["microservice_id"], "ms_list_A")
        self.assertNotIn("invocation_info", data_filtered[0])
        self.assertNotIn("mcp_manifest", data_filtered[0])


    def test_05_get_tool_detail_success(self):
        print("\\n--- Test: Get Tool Detail (Success) ---")
        payload: Dict[str, Any] = {"tool_name": "detail_tool", "microservice_id": "ms_detail", "description": "Detail desc",
                   "invocation_info": {"cmd":"detail_cmd"}, "mcp_manifest": {"name":"detail_m", "description":"d_detail", "input_schema":{}}}
        reg_resp = self.client.post("/api/v1/tools", json=payload).json()
        tool_id = reg_resp["id"]

        response = self.client.get(f"/api/v1/tools/{tool_id}")
        self.assertEqual(response.status_code, 200)
        data = response.json()
        self.assertEqual(data["id"], tool_id)
        self.assertEqual(data["tool_name"], "detail_tool")
        self.assertEqual(data["description"], "Detail desc")
        self.assertEqual(data["invocation_info"]["cmd"], "detail_cmd")
        self.assertIn("mcp_manifest", data)
        self.assertEqual(data["mcp_manifest"]["name"], "detail_m")

    def test_06_get_tool_detail_not_found(self):
        print("\\n--- Test: Get Tool Detail (Not Found) ---")
        response = self.client.get("/api/v1/tools/99999")
        self.assertEqual(response.status_code, 404)

    def test_07_get_tool_by_lookup_success(self):
        print("\\n--- Test: Get Tool by Lookup (Success) ---")
        payload: Dict[str, Any] = {"tool_name": "lookup_tool", "microservice_id": "ms_lookup", "description": "Lookup desc",
                   "invocation_info": {"cmd":"lookup_cmd"}, "mcp_manifest": {"name":"lookup_m", "description":"d_lookup", "input_schema":{}}}
        self.client.post("/api/v1/tools", json=payload)

        response = self.client.get(f"/api/v1/tools/lookup?microservice_id=ms_lookup&tool_name=lookup_tool")
        self.assertEqual(response.status_code, 200)
        data = response.json()
        self.assertEqual(data["tool_name"], "lookup_tool")
        self.assertEqual(data["microservice_id"], "ms_lookup")

    def test_08_get_tool_by_lookup_not_found(self):
        print("\\n--- Test: Get Tool by Lookup (Not Found) ---")
        response = self.client.get(f"/api/v1/tools/lookup?microservice_id=ms_nonexist&tool_name=tool_nonexist")
        self.assertEqual(response.status_code, 404)

    def test_09_delete_tool_success(self):
        print("\\n--- Test: Delete Tool (Success) ---")
        payload: Dict[str, Any] = {"tool_name": "delete_me", "microservice_id": "ms_delete", "description": "To be deleted",
                   "invocation_info": {}, "mcp_manifest": {"name":"delete_m", "description":"d_delete", "input_schema":{}}}
        reg_resp = self.client.post("/api/v1/tools", json=payload).json()
        tool_id = reg_resp["id"]

        delete_response = self.client.delete(f"/api/v1/tools/{tool_id}")
        self.assertEqual(delete_response.status_code, 204)

        get_response = self.client.get(f"/api/v1/tools/{tool_id}")
        self.assertEqual(get_response.status_code, 404)

    def test_10_delete_tool_not_found(self):
        print("\\n--- Test: Delete Tool (Not Found) ---")
        delete_response = self.client.delete("/api/v1/tools/99998")
        self.assertEqual(delete_response.status_code, 404)

    def test_11_heartbeat_success(self):
        print("\\n--- Test: Heartbeat (Success) ---")
        payload: Dict[str, Any] = {"tool_name": "heartbeat_tool", "microservice_id": "ms_hb", "description": "HB desc",
                   "invocation_info": {}, "mcp_manifest": {"name":"hb_m", "description":"d_hb", "input_schema":{}}}
        reg_resp = self.client.post("/api/v1/tools", json=payload).json()
        tool_id = reg_resp["id"]

        initial_detail_resp = self.client.get(f"/api/v1/tools/{tool_id}")
        self.assertEqual(initial_detail_resp.status_code, 200)
        initial_heartbeat_at_str = initial_detail_resp.json().get("last_heartbeat_at")
        self.assertIsNotNone(initial_heartbeat_at_str, "Heartbeat should be set on creation/update.")
        initial_heartbeat_at = datetime.datetime.fromisoformat(initial_heartbeat_at_str.replace("Z", "+00:00"))


        time.sleep(0.05) # Ensure time difference for test

        hb_response = self.client.post(f"/api/v1/tools/heartbeat/ms_hb/heartbeat_tool")
        self.assertEqual(hb_response.status_code, 200)
        hb_data = hb_response.json()
        self.assertEqual(hb_data["message"], "Heartbeat received") # Corrected expected message

        updated_detail_resp = self.client.get(f"/api/v1/tools/{tool_id}")
        self.assertEqual(updated_detail_resp.status_code, 200)
        updated_heartbeat_at_str = updated_detail_resp.json().get("last_heartbeat_at")
        self.assertIsNotNone(updated_heartbeat_at_str)
        updated_heartbeat_at = datetime.datetime.fromisoformat(updated_heartbeat_at_str.replace("Z", "+00:00"))

        self.assertGreater(updated_heartbeat_at, initial_heartbeat_at)

    def test_12_heartbeat_not_found(self):
        print("\\n--- Test: Heartbeat (Not Found) ---")
        hb_response = self.client.post(f"/api/v1/tools/heartbeat/ms_nonexist_hb/tool_nonexist_hb")
        self.assertEqual(hb_response.status_code, 404)


if __name__ == "__main__":
    unittest.main()
