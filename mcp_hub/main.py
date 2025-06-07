from fastapi import FastAPI
from mcp_hub.db.database import init_db
from mcp_hub.api import routes as api_routes

app = FastAPI(
    title="MCP Hub",
    description="A central hub for discovering and managing MCP-enabled tools from various microservices.",
    version="0.1.0"
)

@app.on_event("startup")
async def startup_db_init():
    # This is a simple way to ensure tables are created.
    # For production, consider Alembic or more sophisticated migration tools.
    print("MCP Hub: Initializing database...")
    try:
        init_db()
        print("MCP Hub: Database initialization complete.")
    except Exception as e:
        print(f"MCP Hub: Database initialization failed: {e}")
        # Depending on severity, you might want to prevent app startup

@app.get("/")
async def read_root():
    return {"message": "Welcome to the MCP Hub!"}

# Further API endpoints will be added in api/routes.py and included here.
# For example:
app.include_router(api_routes.router, prefix="/api/v1")

# Database initialization could also be triggered here or in a startup event.
# from mcp_hub.db import database
# database.init_db() # Example call

if __name__ == "__main__":
    import uvicorn
    # This is for direct execution.
    # Typically, you'd run: uvicorn mcp_hub.main:app --reload
    uvicorn.run(app, host="0.0.0.0", port=8080) # Port 8080 for the hub, to differentiate from py_toolbox server if run locally
