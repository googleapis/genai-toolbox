from fastapi import APIRouter, Depends, HTTPException, status, Response # Added Response
from sqlalchemy.orm import Session
from typing import List, Optional

from mcp_hub.db import database
from mcp_hub.models import tool_registry_models as models
from sqlalchemy.exc import IntegrityError
from sqlalchemy.sql import func # For explicit heartbeat update if needed

router = APIRouter()

# Dependency for getting DB session
get_db = database.get_db

@router.post("/tools", response_model=models.ToolRegistrationResponse, status_code=status.HTTP_201_CREATED, tags=["Tool Registration"])
async def register_tool(
    tool_data: models.ToolRegistrationRequest,
    response: Response, # Inject Response object to modify status code
    db: Session = Depends(get_db)
):
    """
    Registers a new tool or updates an existing one based on microservice_id and tool_name.
    If the tool already exists, it's updated, and a 200 OK status is returned.
    If it's a new tool, it's created, and a 201 Created status is returned.
    """
    db_tool = db.query(models.RegisteredTool).filter(
        models.RegisteredTool.microservice_id == tool_data.microservice_id,
        models.RegisteredTool.tool_name == tool_data.tool_name
    ).first()

    current_time = func.now() # For consistent timestamping if needed for heartbeat

    if db_tool:
        # Update existing tool
        db_tool.description = tool_data.description
        db_tool.mcp_manifest = tool_data.mcp_manifest.model_dump(exclude_none=True) # Pydantic to dict
        db_tool.invocation_info = tool_data.invocation_info
        db_tool.last_heartbeat_at = current_time # Explicitly update heartbeat

        response.status_code = status.HTTP_200_OK # Set status to OK for update
        print(f"Updating existing tool: {tool_data.tool_name} from microservice {tool_data.microservice_id}")
    else:
        # Create new tool registration
        db_tool = models.RegisteredTool(
            tool_name=tool_data.tool_name,
            microservice_id=tool_data.microservice_id,
            description=tool_data.description,
            mcp_manifest=tool_data.mcp_manifest.model_dump(exclude_none=True), # Pydantic to dict
            invocation_info=tool_data.invocation_info,
            # registered_at is server_default
            last_heartbeat_at=current_time # Set initial heartbeat
        )
        db.add(db_tool)
        # response.status_code remains status.HTTP_201_CREATED (default for this path operation)
        print(f"Registering new tool: {tool_data.tool_name} from microservice {tool_data.microservice_id}")

    try:
        db.commit()
        db.refresh(db_tool)
    except IntegrityError:
        db.rollback()
        # This case should ideally be less frequent due to the initial query,
        # but it handles race conditions if two identical registrations arrive simultaneously.
        raise HTTPException(
            status_code=status.HTTP_409_CONFLICT,
            detail="A tool with this name and microservice ID already exists (concurrent registration attempt or unique constraint failed)."
        )
    except Exception as e:
        db.rollback()
        print(f"Error during tool registration commit: {e}") # For server logs
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"An internal error occurred during tool registration: {str(e)}"
        )

    # The response_model will shape the output.
    # The status code is set on the `response` object for updates.
    return models.ToolRegistrationResponse(
        id=db_tool.id,
        tool_name=db_tool.tool_name,
        microservice_id=db_tool.microservice_id,
        registered_at=db_tool.registered_at # This will be the original registration time
    )


@router.get("/tools", response_model=List[models.ToolDisplay], tags=["Tool Discovery"])
async def list_tools(
    microservice_id: Optional[str] = None, # Optional query parameter to filter by microservice
    skip: int = 0,
    limit: int = 100,
    db: Session = Depends(get_db)
):
    """
    Lists all registered tools with pagination.
    Optionally filters by `microservice_id`.
    """
    query = db.query(models.RegisteredTool)
    if microservice_id:
        query = query.filter(models.RegisteredTool.microservice_id == microservice_id)

    tools = query.offset(skip).limit(limit).all()
    return tools


@router.get("/tools/{tool_id}", response_model=models.ToolDetail, tags=["Tool Discovery"])
async def get_tool_detail(tool_id: int, db: Session = Depends(get_db)):
    """
    Retrieves detailed information for a specific tool by its Hub ID.
    """
    db_tool = db.query(models.RegisteredTool).filter(models.RegisteredTool.id == tool_id).first()
    if db_tool is None:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="Tool not found by Hub ID")
    return db_tool


@router.get("/tools/lookup", response_model=models.ToolDetail, tags=["Tool Discovery"])
async def get_tool_by_microservice_and_name(
    microservice_id: str,
    tool_name: str,
    db: Session = Depends(get_db)
):
    """
    Retrieves detailed information for a specific tool by its microservice ID and tool name.
    """
    db_tool = db.query(models.RegisteredTool).filter(
        models.RegisteredTool.microservice_id == microservice_id,
        models.RegisteredTool.tool_name == tool_name
    ).first()
    if db_tool is None:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail=f"Tool '{tool_name}' from microservice '{microservice_id}' not found.")
    return db_tool

@router.delete("/tools/{tool_id}", status_code=status.HTTP_204_NO_CONTENT, tags=["Tool Registration"])
async def delete_tool(tool_id: int, db: Session = Depends(get_db)):
    """
    Deletes a tool registration by its Hub ID.
    """
    db_tool = db.query(models.RegisteredTool).filter(models.RegisteredTool.id == tool_id).first()
    if db_tool is None:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="Tool not found for deletion.")

    db.delete(db_tool)
    db.commit()
    # No content is returned for 204, so just return Response or None
    return Response(status_code=status.HTTP_204_NO_CONTENT)

@router.post("/tools/heartbeat/{microservice_id}/{tool_name}", status_code=status.HTTP_200_OK, tags=["Tool Health"])
async def tool_heartbeat(
    microservice_id: str,
    tool_name: str,
    db: Session = Depends(get_db)
):
    """
    Allows a tool to signal it's still alive by updating its last_heartbeat_at timestamp.
    If the tool is not found, it returns 404.
    """
    db_tool = db.query(models.RegisteredTool).filter(
        models.RegisteredTool.microservice_id == microservice_id,
        models.RegisteredTool.tool_name == tool_name
    ).first()

    if db_tool is None:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail=f"Tool '{tool_name}' from microservice '{microservice_id}' not found for heartbeat.")

    db_tool.last_heartbeat_at = func.now() # func.now() is SQLAlchemy's way to use DB's current time
    try:
        db.commit()
        db.refresh(db_tool) # To get the updated timestamp if needed for response, though not strictly necessary for 200 OK.
        return {"message": "Heartbeat received", "tool_name": tool_name, "microservice_id": microservice_id, "last_heartbeat_at": db_tool.last_heartbeat_at}
    except Exception as e:
        db.rollback()
        raise HTTPException(status_code=status.HTTP_500_INTERNAL_SERVER_ERROR, detail=f"Error updating heartbeat: {str(e)}")
