# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import asyncio
from datetime import datetime
from typing import Any, Dict, Optional, Awaitable

from google.adk import Agent
from google.adk.apps import App
from google.adk.agents import InvocationContext
from toolbox_adk import ToolboxToolset, CredentialStrategy

system_prompt = """
  You're a helpful hotel assistant. You handle hotel searching, booking and
  cancellations. When the user searches for a hotel, mention it's name, id,
  location and price tier. Always mention hotel ids while performing any
  searches. This is very important for any operations. For any bookings or
  cancellations, please provide the appropriate confirmation. Be sure to
  update checkin or checkout dates if mentioned by the user.
  Don't ask for confirmations from the user.
"""

async def before_tool_callback(context: Any, args: Dict[str, Any]):
    """
    Callback fired before a tool is executed.
    Enforces business logic: Max stay duration is 14 days.
    """
    tool_name = getattr(context, "name", "unknown_tool")
    
    print(f"POLICY CHECK: Intercepting '{tool_name}'")
    if tool_name == "update-hotel" or ("checkin_date" in args and "checkout_date" in args):
        try:
            start = datetime.fromisoformat(args["checkin_date"])
            end = datetime.fromisoformat(args["checkout_date"])
            duration = (end - start).days

            if duration > 14:
                print("BLOCKED: Stay too long")
                raise ValueError("Error: Maximum stay duration is 14 days.")
        except ValueError as e:
            if "Maximum stay duration" in str(e):
                raise
            pass 
            
    return args

async def after_tool_callback(context: Any, args: Dict[str, Any], result: Any, error: Optional[Exception]) -> Awaitable[Any]:
    """
    Callback fired after a tool execution.
    Enriches response for successful bookings.
    """
    tool_name = getattr(context, "name", "unknown_tool")

    if error:
        print(f"[Tool-Level] after_tool_callback: Tool '{tool_name}' failed with error: {error}")
        return None
    if isinstance(result, str) and "Error" not in result:
        is_booking = tool_name == "book-hotel" or "booking" in str(result).lower() or "confirmed" in str(result).lower()
        
        if is_booking:
             loyalty_bonus = 500
             return f"Booking Confirmed!\n You earned {loyalty_bonus} Loyalty Points with this stay.\n\nSystem Details: {result}"
    return result

async def main():
    print("🚀 Initializing ADK Agent with Toolbox...")

    toolset = ToolboxToolset(
        server_url="http://127.0.0.1:5000",
        toolset_name="my-toolset",
        credentials=CredentialStrategy.toolbox_identity(),
        pre_hook=before_tool_callback,
        post_hook=after_tool_callback
    )

    root_agent = Agent(
        name='root_agent',
        model='gemini-2.5-flash',
        instruction=system_prompt,
        tools=[toolset],
    )

    app = App(root_agent=root_agent, name="my_agent")
    user_input = "Book hotel with id 3."
    print(f"\nUSER: '{user_input}'")
    
    # Note: run_async expects an InvocationContext and returns an async generator
    context = InvocationContext(text=user_input)
    response_text = ""
    async for chunk in root_agent.run_async(context):
        # Accumulate text from chunks
        text_chunk = getattr(chunk, 'text', str(chunk))
        if text_chunk:
             response_text += text_chunk

    print(f"AI: {response_text}")
    
    # Test Pre-processing
    print("-" * 50)
    user_input_2 = "Update my hotel with id 3 with checkin date 2025-01-18 and checkout date 2025-02-10" # > 14 days
    print(f"USER: '{user_input_2}'")
    
    context_2 = InvocationContext(text=user_input_2)
    response_text = ""
    async for chunk in root_agent.run_async(context_2):
        text_chunk = getattr(chunk, 'text', str(chunk))
        if text_chunk:
             response_text += text_chunk
             
    print(f"AI: {response_text}")

if __name__ == "__main__":
    asyncio.run(main())
