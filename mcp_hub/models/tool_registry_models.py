from sqlalchemy import Column, Integer, String, DateTime, Text, JSON, UniqueConstraint
from sqlalchemy.sql import func # For server_default=func.now()
from mcp_hub.db.database import Base # Import Base from our database setup
from pydantic import BaseModel, Field # For API data validation
from typing import Optional, Dict, Any, List # For Pydantic models
import datetime

# --- SQLAlchemy Model ---
class RegisteredTool(Base):
    __tablename__ = "registered_tools"

    id = Column(Integer, primary_key=True, index=True, autoincrement=True)
    # tool_id from microservice could be different, this is our DB's primary key.

    tool_name = Column(String, nullable=False, index=True)
    microservice_id = Column(String, nullable=False, index=True) # ID of the microservice instance
    description = Column(Text, nullable=True)

    # Store complex objects as JSON strings in SQLite.
    # For other DBs, SQLAlchemy might have native JSON types.
    invocation_info = Column(JSON, nullable=False) # How to call the microservice for this tool
    mcp_manifest = Column(JSON, nullable=False)    # Tool's McpManifest (params, schema)

    registered_at = Column(DateTime(timezone=True), server_default=func.now())
    last_heartbeat_at = Column(DateTime(timezone=True), nullable=True, onupdate=func.now())

    __table_args__ = (UniqueConstraint('microservice_id', 'tool_name', name='_microservice_tool_uc'),)

    def __repr__(self):
        return f"<RegisteredTool(id={self.id}, name='{self.tool_name}', microservice='{self.microservice_id}')>"


# --- Pydantic Models for API Data Validation ---
# These will be used by FastAPI for request and response bodies.

# Pydantic model for the McpManifest (as stored/expected by Hub)
# This mirrors the structure from py_toolbox.internal.tools.base.McpManifest
# but defined here for Hub's API contract.
class McpInputSchema(BaseModel):
    type: str = "object"
    properties: Dict[str, Dict[str, Any]] = Field(default_factory=dict)
    required: List[str] = Field(default_factory=list)

class McpManifestModel(BaseModel):
    name: str # Tool's own name, might differ from Hub's registered name if namespacing
    description: str = ""
    input_schema: McpInputSchema = Field(default_factory=McpInputSchema)

# Model for registering a new tool
class ToolRegistrationRequest(BaseModel):
    tool_name: str = Field(..., examples=["my_pg_query_tool"])
    microservice_id: str = Field(..., examples=["pg_microservice_instance_1"])
    description: Optional[str] = Field(None, examples=["Executes a specific query on PG DB alpha."])
    invocation_info: Dict[str, Any] = Field(..., examples=[{"type": "mcp", "command": "python py_toolbox/main.py --config specific.yaml mcp-serve", "mcp_method_for_tool": "invoke_tool"}])
    mcp_manifest: McpManifestModel # The full McpManifest from the tool

class ToolRegistrationResponse(BaseModel):
    id: int # The Hub's DB ID for the registration
    tool_name: str
    microservice_id: str
    registered_at: datetime.datetime
    # Add other fields as necessary, e.g., a URL to the tool's detail endpoint in the Hub

# Model for displaying a tool from the Hub's registry (e.g., in a list)
class ToolDisplay(BaseModel):
    id: int
    tool_name: str
    microservice_id: str
    description: Optional[str] = None
    # invocation_info: Dict[str, Any] # Usually not in list view, but in detail view
    # mcp_manifest: McpManifestModel # Usually not in list view, but in detail view
    registered_at: datetime.datetime
    last_heartbeat_at: Optional[datetime.datetime] = None

    class Config: # Pydantic V1 syntax, for Pydantic V2 use model_config = {}
        from_attributes = True # Enable ORM mode for SQLAlchemy model conversion (Pydantic V1)
        # For Pydantic V2, it would be: model_config = {"from_attributes": True}

# Model for detailed view of a tool
class ToolDetail(ToolDisplay):
    invocation_info: Dict[str, Any]
    mcp_manifest: McpManifestModel
