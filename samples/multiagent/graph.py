from typing import Annotated, Sequence, TypedDict
from langchain_google_vertexai import ChatVertexAI
from langchain_core.runnables import RunnableLambda, RunnableConfig
from langchain_core.messages import AIMessage, BaseMessage, ToolMessage
from langgraph.graph import END, StateGraph
from langgraph.graph.message import add_messages
from toolbox_langchain import ToolboxTool

MODEL = "gemini-2.0-flash-001"

class State(TypedDict):
    messages: Annotated[Sequence[BaseMessage], add_messages]
    agent: str

async def create_graph(
    tools,
    cs_tools,
    hotel_tools,
    flight_tools,
    cs_prompt,
    hotel_prompt,
    flight_prompt,
    checkpointer,
    debug,
):

    # set up the customer service agent
    cs_model = ChatVertexAI(max_output_tokens=512, model_name=MODEL, temperature=0.0)
    cs_with_tools = cs_model.bind_tools(cs_tools)
    cs_model_runnable = cs_prompt | cs_with_tools 
    async def acall_cs_model(state: State, config: RunnableConfig):
        """
        The node representing async function that calls the model.
        After invoking model, it will return AIMessage back to the user.
        """
        print("INFO: calling customer service agent")
        messages = state["messages"]
        res = await cs_model_runnable.ainvoke({"messages": messages}, config)
        c = res.content
        next_agent = ""
        if "hotel_agent" in c:
            next_agent = "hotel_agent"
        if "flight_agent" in c:
            next_agent = "flight_agent"
        return {"messages": [res], "agent": next_agent}

    # set up the hotel agent
    hotel_model = ChatVertexAI(max_output_tokens=512, model_name=MODEL, temperature=0.0)
    hotel_with_tools = hotel_model.bind_tools(hotel_tools)
    hotel_model_runnable = hotel_prompt | hotel_with_tools 
    async def acall_hotel_model(state: State, config: RunnableConfig):
        """
        The node representing async function that calls the model.
        After invoking model, it will return AIMessage back to the user.
        """
        print("INFO: calling hotel agent")
        messages = state["messages"]
        res = await hotel_model_runnable.ainvoke({"messages": messages}, config)
        next_agent = ""
        if "customer_service_agent" in res.content:
            next_agent = "customer_service_agent"

        return {"messages": [res], "agent": next_agent}

    # set up the flight agent
    flight_model = ChatVertexAI(max_output_tokens=512, model_name=MODEL, temperature=0.0)
    flight_with_tools = flight_model.bind_tools(flight_tools)
    flight_model_runnable = flight_prompt | flight_with_tools
    async def acall_flight_model(state: State, config: RunnableConfig):
        """
        The node representing async function that calls the model.
        After invoking model, it will return AIMessage back to the user.
        """
        print("INFO: calling flight agent")
        messages = state["messages"]
        res = await flight_model_runnable.ainvoke({"messages": messages}, config)
        next_agent = ""
        if "customer_service_agent" in res.content:
            next_agent = "customer_service_agent"

        return {"messages": [res], "agent": next_agent}

    # customize tool node 
    async def tool_node(state: State, config: RunnableConfig):
        last_message = state["messages"][-1]
        tool_messages = []

        if not hasattr(last_message, "tool_calls"):
            return {"messages": []}

        for tool_call in last_message.tool_calls:
            tool_name = tool_call["name"]
            print("INFO: invoking tool: ", tool_name)
            # Find the corresponding tool from the provided list
            selected_tool = next((t for t in tools if t.name == tool_name), None)

            if not selected_tool:
                # Handle case where the model hallucinates a tool name
                output = f"Error: Tool '{tool_name}' not found."
            else:
                try:
                    # Manually invoke the tool with its arguments
                    output = await selected_tool.ainvoke(tool_call["args"])
                except Exception as e:
                    output = f"Error executing tool {tool_name}: {e}"

            if output == "":
                output = "operation success."
            # Create a ToolMessage with the result and original tool_call_id
            tool_messages.append(
                ToolMessage(
                    name=tool_name,
                    content=output,
                    tool_call_id=tool_call["id"],
                )
            )

        return {"messages": tool_messages}

    # constants for nodes
    CUSTOMER_SERVICE_AGENT_NODE = "customer_service_agent"
    HOTEL_AGENT_NODE = "hotel_agent"
    FLIGHT_AGENT_NODE = "flight_agent"
    TOOL_NODE = "tools"

    # create a state graph and add nodes
    graph = StateGraph(State)
    graph.add_node(CUSTOMER_SERVICE_AGENT_NODE, RunnableLambda(acall_cs_model))
    graph.add_node(HOTEL_AGENT_NODE, RunnableLambda(acall_hotel_model))
    graph.add_node(FLIGHT_AGENT_NODE, RunnableLambda(acall_flight_model))
    graph.add_node(TOOL_NODE, tool_node)

    # set entry point
    graph.set_entry_point(CUSTOMER_SERVICE_AGENT_NODE)

    # define conditional edges functions
    def customer_service_next_steps(state: State, config: RunnableConfig):
        if state["agent"] == HOTEL_AGENT_NODE:
            return "hotel_agent"
        if state["agent"] == FLIGHT_AGENT_NODE:
            return "flight_agent"
        return "end"

    def agent_should_continue(state: State, config: RunnableConfig):
        if state["agent"] == CUSTOMER_SERVICE_AGENT_NODE:
            return "customer_service_agent"

        messages = state["messages"]
        last_message = messages[-1]

        # First check if the last message has tool calls.
        if not hasattr(last_message, "tool_calls") or len(last_message.tool_calls) == 0:
            return "end"

        return "continue"

    # set edges
    graph.add_conditional_edges(
        CUSTOMER_SERVICE_AGENT_NODE,
        customer_service_next_steps,
        {
            "hotel_agent": HOTEL_AGENT_NODE,
            "flight_agent": FLIGHT_AGENT_NODE,
            "end": END,
        },
    )

    graph.add_conditional_edges(
        HOTEL_AGENT_NODE,
        agent_should_continue,
        {
            "continue": TOOL_NODE,
            "customer_service_agent": CUSTOMER_SERVICE_AGENT_NODE,
            "end": END,
        }
    )
    graph.add_conditional_edges(
        FLIGHT_AGENT_NODE,
        agent_should_continue,
        {
            "continue": TOOL_NODE,
            "customer_service_agent": CUSTOMER_SERVICE_AGENT_NODE,
            "end": END,
        }
    )
    graph.add_edge(TOOL_NODE, CUSTOMER_SERVICE_AGENT_NODE)

    # compile langgraph app
    app = graph.compile(checkpointer=checkpointer, debug=debug)
    return app

