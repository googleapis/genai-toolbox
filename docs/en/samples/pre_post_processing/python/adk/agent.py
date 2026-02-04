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
from google.adk.runners import Runner
from google.adk.sessions.in_memory_session_service import InMemorySessionService
from google.genai import types
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
    """Enforces business logic: Max stay duration is 14 days."""
    tool_name = getattr(context, "name", "unknown_tool")    
    print(f"POLICY CHECK: Intercepting '{tool_name}'")
    
    if tool_name == "update-hotel" or ("checkin_date" in args and "checkout_date" in args):
        try:
            start = datetime.fromisoformat(args["checkin_date"])
            end = datetime.fromisoformat(args["checkout_date"])
            if (end - start).days > 14:
                print("BLOCKED: Stay too long")
                raise ValueError("Error: Maximum stay duration is 14 days.")
        except ValueError as e:
            if "Maximum stay duration" in str(e): raise
    return args

async def after_tool_callback(context: Any, args: Dict[str, Any], result: Any, error: Optional[Exception]) -> Awaitable[Any]:
    """Enriches response for successful bookings."""
    tool_name = getattr(context, "name", "unknown_tool")
    if error:
        print(f"[Tool-Level] Tool '{tool_name}' failed: {error}")
        return None

    if isinstance(result, str) and "Error" not in result:
        is_booking = tool_name == "book-hotel" or "booking" in str(result).lower()
        if is_booking:
             return f"Booking Confirmed!\n You earned 500 Loyalty Points with this stay.\n\nSystem Details: {result}"
    return result

async def run_turn(runner: Runner, user_id: str, session_id: str, text: str):
    """Helper to run a single turn."""
    print(f"\nUSER: '{text}'")
    response_text = ""
    async for event in runner.run_async(
        user_id=user_id, session_id=session_id,
        new_message=types.Content(role="user", parts=[types.Part(text=text)])
    ):
        if event.content and event.content.parts:
            for part in event.content.parts: 
                if part.text:
                    response_text += part.text
        pass
    print(f"AI: {response_text}")

queries = [
    "Book hotel with id 3.",
    "Update my hotel with id 3 with checkin date 2025-01-18 and checkout date 2025-02-10",
]

async def main():
    print("🚀 Initializing ADK Agent with Toolbox...")

    toolset = ToolboxToolset(
        server_url="http://127.0.0.1:5000",
        toolset_name="my-toolset",
        credentials=CredentialStrategy.toolbox_identity(),
        pre_hook=before_tool_callback,
        post_hook=after_tool_callback
    )

    app = App(
        root_agent=Agent(name='root_agent', model='gemini-2.5-flash', instruction=system_prompt, tools=[toolset]),
        name="my_agent"
    )
    runner = Runner(app=app, session_service=InMemorySessionService())
    
    user_id, session_id = "test-user", "test-session"
    await runner.session_service.create_session(app_name=app.name, user_id=user_id, session_id=session_id)

    for query in queries:
        await run_turn(runner, user_id, session_id, query)
        print("-" * 50)

if __name__ == "__main__":
    asyncio.run(main())
