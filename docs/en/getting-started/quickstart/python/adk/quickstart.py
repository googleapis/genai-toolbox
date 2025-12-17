from google.adk import Agent
from google.adk.apps import App
from toolbox_core import ToolboxSyncClient

# TODO(developer): update the TOOLBOX_URL to your toolbox endpoint
client = ToolboxSyncClient("http://127.0.0.1:5000")

root_agent = Agent(
    name='root_agent',
    model='gemini-2.5-flash',
    instruction="You are a helpful AI assistant designed to provide accurate and useful information.",
    tools=client.load_toolset(),
)

app = App(root_agent=root_agent, name="my_agent")
