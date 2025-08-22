import uuid
import os
import asyncio

from toolbox_core import auth_methods
from toolbox_llamaindex import ToolboxClient

from llama_index.core.workflow import Context
from llama_index.core.agent.workflow import AgentWorkflow, FunctionAgent
from llama_index.llms.google_genai import GoogleGenAI

TOOLBOX_URL = "http://127.0.0.1:5000"

CS_PROMPT = """
You are a friendly and professional customer service agent for Cymbal Travel Agency, a travel booking service.

Your primary responsibilities are to:
1.  **Welcome users** and greet them warmly.
2.  **Answer general knowledge questions** related to travel (e.g. questions about airports or specific airport such as SFO). If you are unsure of a specific information or do not have tool to access it, you should let the user know.
3.  **Manage conversational flow** and ask clarifying questions to understand the user's intent.

**Specialized Agent:**
- **flight agent**: If a user asks a question specifically about **searching, listing and booking a flight**.
- **hotel agent**: If a user asks a question specifically about **searching, listing and booking a hotel**.

**You can call the specialized agents** to help you with certain tasks. You can do this however many times you want until you have all the informations needed.

**If you already have the informamtion you need, feel free to response directly to the user instead of calling another agent. If you're unsure, please check with the other agents.**
"""

HOTEL_PROMPT = """
You are the dedicated hotel specialist. Your expertise is dedicated exclusively to hotel and accommodation services. The customer service agent will reach out to you regarding to question around your specialties.

Your primary responsibilities are to:
1.  **Search for hotels** based on specific criteria (hotel name, city, rating, price range).
2.  **Book or reserve hotel rooms**.
3.  **List bookings** that are under a specific name.

You **must** focus solely on hotel and accommodation-related tasks. Do not answer questions about flights, rental cars, activities, or general travel knowledge.

**Your communication style should be helpful and detailed, providing rich information to help the customer service agent choose the best accommodation.**
"""

FLIGHT_PROMPT = """
You are the dedicated flight specialist. Your expertise is dedicated exclusively to flights. The customer service agent will reach out to you regarding to questions around your specialties.

Your primary responsibilities are to:
1.  **Search for flights** based on user criteria (origin, destination, dates).
2.  **Book or reserve flight tickets** on behalf of the user.
3.  **Provide detailed information about flights** (e.g., flight numbers, departure/arrival times, layovers, airline, and fare rules).
4.  **List flight tickets** that are under user's name.
5.  **Answer questions on Cymbal Air Flight's policy**.

You **must** focus solely on flight-related tasks. Do not answer questions about hotels, rental cars, activities, or general travel knowledge.

**Your communication style should be efficient and informative, directly addressing the customer service agent's flight-related questions.**
"""

async def run_app():
    # load model
    llm = GoogleGenAI(
        model="gemini-2.5-flash",
        vertexai_config={"project": "project-id", "location": "us-central1"},
    )

    # Alternatively, you can also load the gemini model using google api key
    # llm = GoogleGenAI(
    #     model="gemini-2.5-flash",
    #     api_key=os.getenv("GOOGLE_API_KEY"),
    # )

    # load tools from Toolbox
    general_tools, hotel_tools, flight_tools = await get_tools()

    # build agents
    customer_service_agent = FunctionAgent(
        name="CustomerServiceAgent",
        description="Answer user's queries and route to the right agents for flight and hotel related queries",
        system_prompt=CS_PROMPT,
        llm=llm,
        tools=general_tools,
    )

    hotel_agent = FunctionAgent(
        name="HotelAgent",
        description="Handles hotel and accommodation services, including searching, booking and list bookings of hotels",
        system_prompt=HOTEL_PROMPT,
        llm=llm,
        tools=hotel_tools,
    )

    flight_agent = FunctionAgent(
        name="FlightAgent",
        description="Handles flights-related services, including searching, booking and list tickets of flights",
        system_prompt=FLIGHT_PROMPT,
        llm=llm,
        tools=flight_tools,
    )

    # set up agent workflow
    agent_workflow = AgentWorkflow(
        agents = [customer_service_agent, hotel_agent, flight_agent],
        root_agent=customer_service_agent.name,
        initial_state={},
    )

    # use Context to maintain state between runs
    ctx = Context(agent_workflow)

    # start application
    print("\nCymbal Travel Agency: What question do you have?")
    while True:
        user_input = input("\nUser: ")
        resp = await agent_workflow.run(user_msg=user_input, ctx=ctx)
        print("\nCymbal Travel Agency:", resp)
    
async def get_tools():
    """
    This function grab tools from Toolbox.
    """
    auth_token_provider = auth_methods.aget_google_id_token(TOOLBOX_URL)
    client = ToolboxClient(TOOLBOX_URL, client_headers={"Authorization": auth_token_provider})
    general_tools = await client.aload_toolset("general_tools")
    hotel_tools = await client.aload_toolset("hotel_tools")
    flight_tools = await client.aload_toolset("flight_tools")
    return (general_tools, hotel_tools, flight_tools)
    
if __name__ == "__main__":
    asyncio.run(run_app())
