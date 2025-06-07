from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker
from sqlalchemy.pool import StaticPool # Good for SQLite in tests

from mcp_hub.db.database import Base # Use the same Base as the main app
import os

# Use an in-memory SQLite database for testing
# Or a file-based one that gets cleaned up
TEST_DATABASE_URL = "sqlite:///:memory:"
# TEST_DATABASE_URL = "sqlite:///./test_mcp_hub.db"


engine = create_engine(
    TEST_DATABASE_URL,
    connect_args={"check_same_thread": False}, # Required for SQLite
    poolclass=StaticPool # Ensures single connection for the test session
)

TestingSessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)

def override_get_db():
    """
    FastAPI dependency override to use the test database session.
    """
    try:
        db = TestingSessionLocal()
        yield db
    finally:
        db.close()

def create_test_tables():
    # Creates tables in the test database engine
    Base.metadata.create_all(bind=engine)
    print(f"Test database tables created (using {TEST_DATABASE_URL}).")

def drop_test_tables():
    # Drops all tables from the test database engine
    Base.metadata.drop_all(bind=engine)
    print(f"Test database tables dropped (from {TEST_DATABASE_URL}).")
    # if "test_mcp_hub.db" in TEST_DATABASE_URL and os.path.exists("./test_mcp_hub.db"):
    #     os.remove("./test_mcp_hub.db")
