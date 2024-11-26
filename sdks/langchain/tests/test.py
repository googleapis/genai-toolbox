import asyncio

from aiohttp import ClientSession

import vertexai
from toolbox_langchain_sdk import ToolboxClient

from langchain_google_vertexai import ChatVertexAI
from langchain_google_vertexai.llms import
from langchain.agents import initialize_agent


"""
Manual tests

Load toolset:
1. Single tool -> Done
2. Multiple tools -> Done

Load tool:
1. Single tool -> Done

Call tool (Get appropriate response) -> Done

For README
Use tools in agents -> Done

"""

"""
Run toolbox locally:

go build -o toolbox
./toolbox
"""

import nest_asyncio

nest_asyncio.apply() 


# these tools can be passed to your application!
async def run_test():
    session = ClientSession()
    # update the url to point to your server
    toolbox = ToolboxClient("http://localhost:5000", session=session)
    tools = await toolbox.load_toolset()
    vertexai.init()
    model = ChatVertexAI(project="twisha-dev", location="us-central1")
    agent = initialize_agent(tools, model)
    agent.run("What are the airports present in Canada?")

if __name__ == "__main__":
    asyncio.run(run_test())
