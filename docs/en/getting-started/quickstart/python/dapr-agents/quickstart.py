import logging
import os
import requests
import threading
import time
from dapr_agents import DurableAgent, OpenAIChatClient
from dapr_agents.tool import AgentTool
from dapr_agents.agents.configs import (
    AgentMemoryConfig,
    AgentStateConfig,
)
from dapr_agents.memory import ConversationDaprStateMemory
from dapr_agents.storage.daprstores.stateservice import StateStoreService
from dapr_agents.workflow.runners import AgentRunner
from toolbox_core import ToolboxSyncClient

# TODO(developer): To run a DurableAgent you'll need to serve a durable statestore. See the "Dapr Agents Documentation" below for more details.

queries = [
    "Find hotels in Basel with Basel in its name.",
    "Please book the hotel Hilton Basel for me.",
    "This is too expensive. Please cancel it.",
    "Please book Hyatt Regency for me",
    "My check in dates for my booking would be from April 10, 2024 to April 19, 2024.",
]

def main() -> None:
    logging.basicConfig(level=logging.INFO)
    with ToolboxSyncClient("http://127.0.0.1:5000") as toolbox_client:
        agent = DurableAgent(
            name="TravelBuddy",
            role="Travel Agent",
            goal="Help users find and book hotels",
            instructions=[
                "You're a helpful hotel assistant.",
                "You handle hotel searching, booking and cancellations.",
                "When the user searches for a hotel, mention it's name, id, location and price tier.",
                "Always mention hotel id while performing any searches.",
                "This is very important for any operations.",
                "For any bookings or cancellations, please provide the appropriate confirmation.",
                "Be sure to update checkin or checkout dates if mentioned by the user.",
                "Don't ask for confirmations from the user.",
                "If at any point your tools return an error, correct the error based on the error message and try again.",
            ],
            tools=AgentTool.from_toolbox_many(toolbox_client.load_toolset("my-toolset")), # TODO(developer): Replace "my-toolset" with your actual toolset name.
            llm=OpenAIChatClient(model="gpt-4.1-2025-04-14", api_key=os.environ.get("OPENAI_API_KEY", "")),
            state=AgentStateConfig(
                store=StateStoreService(store_name="statestore"),
            ),
            memory=AgentMemoryConfig(
                store=ConversationDaprStateMemory(
                    store_name="statestore",
                    session_id="travel-buddy-session-001",
                )
            ),
        )

        runner = AgentRunner()

        try:
            # The below is for demonstration purposes only. Normally you'd run the DurableAgent like this:
            # runner.serve(agent, port=8001)
            server_thread = threading.Thread(target=lambda: runner.serve(agent, port=8001), daemon=True)
            server_thread.start()
            time.sleep(2)

            for query in queries:
                # Start the workflow task, this would normally be done by an external system.
                response = requests.post(
                    "http://localhost:8001/run",
                    json={"task": query}
                )
                result = response.json()
                logging.info(f"Task submitted: {result}")
                
                # We want to get the result of the workflow execution before moving on with the next query.
                instance_id = result.get("instance_id") or result.get("workflow_id") or result.get("id")
                if instance_id:
                    while True:
                        status_response = requests.get(f"http://localhost:8001/run/{instance_id}")
                        status = status_response.json()
                        workflow_status = status.get("status") or status.get("runtime_status")
                        
                        if workflow_status:
                            status_lower = workflow_status.lower()
                            if status_lower in ["completed", "finished"]:
                                logging.info(f"Workflow result: {status.get('result') or status.get('output')}")
                                break
                            elif status_lower in ["failed", "error"]:
                                logging.info(f"Workflow failed: {status}")
                                raise Exception(f"Workflow execution failed with status: {workflow_status}")
                        
                        time.sleep(2)
        finally:
            toolbox_client.close()
            runner.shutdown(agent)


main()

# TODO(developer): To run this example, you need to write the following file:
# components/statestore.yaml
# with the following content:
"""
apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: statestore
spec:
  type: state.in-memory
  version: v1
  metadata:
  - name: actorStateStore
    value: "true"
"""
