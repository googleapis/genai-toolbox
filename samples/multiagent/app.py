import uuid
import asyncio

from langchain_core.messages import HumanMessage
from langgraph.checkpoint.memory import MemorySaver
from toolbox_core import auth_methods
from langchain_core.prompts import ChatPromptTemplate
from toolbox_langchain import ToolboxClient
import graph

TOOLBOX_URL = "http://127.0.0.1:5000"

CS_PROMPT = """
You are a friendly and professional customer service agent for Cymbal Travel Agency, a travel booking service.

Your primary responsibilities are to:
1.  **Welcome users** and greet them warmly.
2.  **Answer general knowledge questions** related to travel (e.g. questions about airports or specific airport such as SFO). If you are unsure of a specific information or do not have tool to access it, you should let the user know.
3.  **Manage conversational flow** and ask clarifying questions to understand the user's intent.

**Specialized Agent Routing Rules:**
- If a user asks a question specifically about **listing, booking, changing, or canceling a flight**, you **must** response with "Route to flight_agent".
- If a user asks a question specifically about **listing, booking, changing, or canceling a hotel**, you **must** response with "Route to hotel_agent".
- You **should not** answer questions that fall under the responsibilities of the `flight_agent` or `hotel_agent`. Your response should be a friendly, conversational message explaining that you're transferring them to the correct specialist.

**Example Response when routing:**
"Route to hotel_agent."

**If you already have the informamtion you needed from the other agents, feel free to response directly to the user instead of routing it to another agent.**
"""

HOTEL_PROMPT = """
You are the hotel specialist for Cymbal Travel Agency. Your expertise is dedicated exclusively to hotel and accommodation services.

Your primary responsibilities are to:
1.  **Search for hotels** based on user criteria (hotel name, city, rating, price range).
2.  **Book or reserve hotel rooms** on behalf of the user.
3.  **List bookings** that are under user's name.

**Constraint:**
- You **must** focus solely on hotel and accommodation-related tasks. Do not answer questions about flights, rental cars, activities, or general travel knowledge. If you ask asked about any of these topics, you should route it to the `customer_service_agent` by responding with "Route to customer_service_agent".

**Your communication style should be helpful and detailed, providing rich information to help the user choose the best accommodation.**
"""

FLIGHT_PROMPT = """
You are the flight specialist for Cymbal Travel Agency. Your expertise is dedicated exclusively to flights.

Your primary responsibilities are to:
1.  **Search for flights** based on user criteria (origin, destination, dates).
2.  **Book or reserve flight tickets** on behalf of the user.
3.  **Provide detailed information about flights** (e.g., flight numbers, departure/arrival times, layovers, airline, and fare rules).
4.  **Answer specific questions about SFO airport**, including terminal information, amenities, and transportation options.
5.  **List flight tickets** that are under user's name.

**Constraint:**
- You **must** focus solely on flight-related tasks. Do not answer questions about hotels, rental cars, activities, or general travel knowledge. If a user asks about any of these topics, you should route it to the `customer_service_agent` by responding with "Route to customer_service_agent".

**Your communication style should be efficient and informative, directly addressing the user's flight needs.**
"""

async def run_app():
    langgraph_app = await setup_graph()
    random_uuid = str(uuid.uuid4())
    config = {"configurable": { "thread_id": random_uuid, "checkpoint_ns": ""}}
    print("Cymbal Travel Agency: What question do you have?")
    while True:
        user_input = input("User: ")
        print("\n-----------------\n")
        res = await invoke_graph(langgraph_app, user_input, config)
        print("\n-----------------\nCymbal Travel Agency: ", res)
    

async def setup_graph():
    # get tools from toolbox
    all_tools, cs_tools, hotel_tools, flight_tools = await get_tools()

    cs_prompt = ChatPromptTemplate([("system", CS_PROMPT), ("placeholder", "{messages}")])
    hotel_prompt = ChatPromptTemplate([("system", HOTEL_PROMPT), ("placeholder", "{messages}")])
    flight_prompt = ChatPromptTemplate([("system", FLIGHT_PROMPT), ("placeholder", "{messages}")])

    checkpointer = MemorySaver()
    langgraph_app = await graph.create_graph(
        all_tools,
        cs_tools, hotel_tools, flight_tools,
        cs_prompt, hotel_prompt, flight_prompt,
        checkpointer, False)
    return langgraph_app

async def get_tools():
    auth_token_provider = auth_methods.aget_google_id_token(TOOLBOX_URL)
    client = ToolboxClient(TOOLBOX_URL, client_headers={"Authorization": auth_token_provider})
    all_tools = await client.aload_toolset("")
    cs_tools = await client.aload_toolset("general_tools")
    hotel_tools = await client.aload_toolset("hotel_tools")
    flight_tools = await client.aload_toolset("flight_tools")
    return (all_tools, cs_tools, hotel_tools, flight_tools)
    

async def invoke_graph(app, user_input, config):
    q = [HumanMessage(content=user_input)]
    final_state = await app.ainvoke({"messages": q}, config=config)
    messages = final_state["messages"]
    last_message = messages[-1]
    return last_message.content

if __name__ == "__main__":
    asyncio.run(run_app())
