import asyncio
import os

from google import genai
from google.genai.types import (
    Content,
    FunctionDeclaration,
    GenerateContentConfig,
    Part,
    Tool,
)

from toolbox_core import ToolboxClient

prompt = """
  You're a helpful hotel assistant. You handle hotel searching, booking and
  cancellations. When the user searches for a hotel, mention it's name, id,
  location and price tier. Always mention hotel id while performing any
  searches. This is very important for any operations. For any bookings or
  cancellations, please provide the appropriate confirmation. Be sure to
  update checkin or checkout dates if mentioned by the user.
  Don't ask for confirmations from the user.
"""

queries = [
    "Find hotels in Basel with Basel in its name.",
    "Please book the hotel Hilton Basel for me.",
    "This is too expensive. Please cancel it.",
    "Please book Hyatt Regency for me",
    "My check in dates for my booking would be from April 10, 2024 to April 19, 2024.",
]

async def run_application():
    host = os.environ.get("TOOLBOX_HOST", "127.0.0.1")
    toolbox_url = f"http://{host}:5000"
    async with ToolboxClient(toolbox_url) as toolbox_client:

        # The toolbox_tools list contains Python callables (functions/methods) designed for LLM tool-use
        # integration. While this example uses Google's genai client, these callables can be adapted for
        # various function-calling or agent frameworks. For easier integration with supported frameworks
        # (https://github.com/googleapis/mcp-toolbox-python-sdk/tree/main/packages), use the
        # provided wrapper packages, which handle framework-specific boilerplate.
        toolbox_tools = await toolbox_client.load_toolset("my-toolset")
        genai_client = genai.Client(
            vertexai=True, project=os.environ.get("GCP_PROJECT"), location="us-central1"
        )

        genai_tools = [
            Tool(
                function_declarations=[
                    FunctionDeclaration.from_callable(tool)
                ]
            )
            for tool in toolbox_tools
        ]
        history = []
        for query in queries:
            user_prompt_content = Content(
                role="user",
                parts=[Part.from_text(text=query)],
            )
            history.append(user_prompt_content)

            response = genai_client.models.generate_content(
                model="gemini-2.0-flash-001",
                contents=history,
                config=GenerateContentConfig(
                    system_instruction=prompt,
                    tools=genai_tools,
                ),
            )
            history.append(response.candidates[0].content)
            function_response_parts = []
            for function_call in response.function_calls:
                fn_name = function_call.name
                # The tools are sorted alphabetically
                if fn_name == "search-hotels-by-name":
                    function_result = await toolbox_tools[3](**function_call.args)
                elif fn_name == "search-hotels-by-location":
                    function_result = await toolbox_tools[2](**function_call.args)
                elif fn_name == "book-hotel":
                    function_result = await toolbox_tools[0](**function_call.args)
                elif fn_name == "update-hotel":
                    function_result = await toolbox_tools[4](**function_call.args)
                elif fn_name == "cancel-hotel":
                    function_result = await toolbox_tools[1](**function_call.args)
                else:
                    raise ValueError("Function name not present.")
                function_response = {"result": function_result}
                function_response_part = Part.from_function_response(
                    name=function_call.name,
                    response=function_response,
                )
                function_response_parts.append(function_response_part)

            if function_response_parts:
                tool_response_content = Content(role="tool", parts=function_response_parts)
                history.append(tool_response_content)

            response2 = genai_client.models.generate_content(
                model="gemini-2.0-flash-001",
                contents=history,
                config=GenerateContentConfig(
                    tools=genai_tools,
                ),
            )
            final_model_response_content = response2.candidates[0].content
            history.append(final_model_response_content)
            print(response2.text)

asyncio.run(run_application())
