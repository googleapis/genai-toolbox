from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker, declarative_base # Updated import for modern SQLAlchemy
from sqlalchemy.pool import StaticPool # Recommended for SQLite in-memory/file for FastAPI
import os

# Determine the directory of this file to build the path to the SQLite DB
# This ensures the DB is created within the mcp_hub project structure, typically in mcp_hub/db/
DATABASE_DIR = os.path.dirname(os.path.abspath(__file__))
DATABASE_FILE_NAME = "mcp_hub.db"
DATABASE_URL = f"sqlite:///{os.path.join(DATABASE_DIR, DATABASE_FILE_NAME)}"
# For testing, one might use an in-memory SQLite: "sqlite:///:memory:"
# but for persistence, a file-based DB is needed.

# For SQLite, connect_args={"check_same_thread": False} is needed for FastAPI/Uvicorn.
# StaticPool is also recommended for SQLite with FastAPI to avoid issues with
# connections being shared across threads in a way SQLite doesn't like by default.
engine = create_engine(
    DATABASE_URL,
    connect_args={"check_same_thread": False},
    poolclass=StaticPool # Use StaticPool for SQLite with FastAPI
)

SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)

Base = declarative_base()

def init_db():
    # Create all tables in the engine. This is equivalent to "Create Table"
    # statements in raw SQL.
    # This should be called once when the application starts up if tables don't exist.
    # In a production app, you'd likely use Alembic for migrations.
    try:
        Base.metadata.create_all(bind=engine)
        print(f"Database tables created successfully at {os.path.join(DATABASE_DIR, DATABASE_FILE_NAME)}")
    except Exception as e:
        print(f"Error creating database tables: {e}")
        raise

def get_db():
    """ FastAPI dependency to get a DB session. """
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()
