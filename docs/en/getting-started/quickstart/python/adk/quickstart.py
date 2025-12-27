from google.adk import Agent
from google.adk.apps import App
from google.adk.tools.toolbox_toolset import ToolboxToolset

# TODO(developer): update the TOOLBOX_URL to your toolbox endpoint
toolset = ToolboxToolset(
    server_url="http://127.0.0.1:5000",
    toolset_name="my-toolset"
)

root_agent = Agent(
    name='root_agent',
    model='gemini-2.5-flash',
    instruction="You are a helpful AI assistant designed to provide accurate and useful information.",
    tools=[toolset],
)

app = App(root_agent=root_agent, name="my_agent")
