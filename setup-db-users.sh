#!/bin/bash

# Database User Setup Script for Hotel Application
# This script helps you create admin_user and support_user with appropriate permissions

# Configuration - Update these values for your RDS instance
RDS_HOST="anetac-hotel-rds-instance-1.c32e2y2ouk20.us-east-2.rds.amazonaws.com"
RDS_PORT="5432"
DATABASE="toolbox_db"
SUPERUSER="postgres"
SUPERUSER_PASSWORD="my-password"  # Update this to your actual postgres password

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Database User Setup for Hotel Application ===${NC}"
echo ""

# Check if psql is available
if ! command -v psql &> /dev/null; then
    echo -e "${RED}Error: psql command not found. Please install PostgreSQL client tools.${NC}"
    exit 1
fi

# Function to test database connection
test_connection() {
    echo -e "${YELLOW}Testing database connection...${NC}"
    PGPASSWORD="$SUPERUSER_PASSWORD" psql -h "$RDS_HOST" -p "$RDS_PORT" -U "$SUPERUSER" -d "$DATABASE" -c "SELECT version();" > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Database connection successful${NC}"
        return 0
    else
        echo -e "${RED}✗ Database connection failed${NC}"
        return 1
    fi
}

# Function to check if users already exist
check_existing_users() {
    echo -e "${YELLOW}Checking for existing users...${NC}"
    PGPASSWORD="$SUPERUSER_PASSWORD" psql -h "$RDS_HOST" -p "$RDS_PORT" -U "$SUPERUSER" -d "$DATABASE" -t -c "SELECT rolname FROM pg_roles WHERE rolname IN ('admin_user', 'support_user');" | grep -E "(admin_user|support_user)" > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        echo -e "${YELLOW}⚠ Users already exist. Do you want to recreate them? (y/N)${NC}"
        read -r response
        if [[ "$response" =~ ^[Yy]$ ]]; then
            echo -e "${YELLOW}Dropping existing users...${NC}"
            PGPASSWORD="$SUPERUSER_PASSWORD" psql -h "$RDS_HOST" -p "$RDS_PORT" -U "$SUPERUSER" -d "$DATABASE" -c "DROP USER IF EXISTS admin_user; DROP USER IF EXISTS support_user;"
        else
            echo -e "${YELLOW}Skipping user creation.${NC}"
            return 1
        fi
    fi
    return 0
}

# Function to create users
create_users() {
    echo -e "${YELLOW}Creating database users...${NC}"
    
    # Use the simplified script (without RLS)
    PGPASSWORD="$SUPERUSER_PASSWORD" psql -h "$RDS_HOST" -p "$RDS_PORT" -U "$SUPERUSER" -d "$DATABASE" -f create-db-users-simple.sql
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Users created successfully${NC}"
        return 0
    else
        echo -e "${RED}✗ Failed to create users${NC}"
        return 1
    fi
}

# Function to verify permissions
verify_permissions() {
    echo -e "${YELLOW}Verifying permissions...${NC}"
    PGPASSWORD="$SUPERUSER_PASSWORD" psql -h "$RDS_HOST" -p "$RDS_PORT" -U "$SUPERUSER" -d "$DATABASE" -f verify-permissions.sql
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Permissions verification completed${NC}"
        return 0
    else
        echo -e "${RED}✗ Permissions verification failed${NC}"
        return 1
    fi
}

# Function to test user access
test_user_access() {
    echo -e "${YELLOW}Testing user access...${NC}"
    
    # Test support_user access
    echo -e "${BLUE}Testing support_user access:${NC}"
    PGPASSWORD="support_password" psql -h "$RDS_HOST" -p "$RDS_PORT" -U "support_user" -d "$DATABASE" -c "SELECT 'support_user can access hotels' as test, COUNT(*) as count FROM hotels;" 2>/dev/null
    PGPASSWORD="support_password" psql -h "$RDS_HOST" -p "$RDS_PORT" -U "support_user" -d "$DATABASE" -c "SELECT 'support_user cannot access integration_tokens' as test, COUNT(*) as count FROM integration_tokens;" 2>/dev/null || echo -e "${GREEN}✓ support_user correctly denied access to integration_tokens${NC}"
    
    # Test admin_user access
    echo -e "${BLUE}Testing admin_user access:${NC}"
    PGPASSWORD="admin_password" psql -h "$RDS_HOST" -p "$RDS_PORT" -U "admin_user" -d "$DATABASE" -c "SELECT 'admin_user can access hotels' as test, COUNT(*) as count FROM hotels;" 2>/dev/null
    PGPASSWORD="admin_password" psql -h "$RDS_HOST" -p "$RDS_PORT" -U "admin_user" -d "$DATABASE" -c "SELECT 'admin_user can access integration_tokens' as test, COUNT(*) as count FROM integration_tokens;" 2>/dev/null
    
    echo -e "${GREEN}✓ User access testing completed${NC}"
}

# Main execution
main() {
    echo -e "${BLUE}Configuration:${NC}"
    echo "  Host: $RDS_HOST"
    echo "  Port: $RDS_PORT"
    echo "  Database: $DATABASE"
    echo "  Superuser: $SUPERUSER"
    echo ""
    
    # Test connection
    if ! test_connection; then
        echo -e "${RED}Please check your database connection parameters and try again.${NC}"
        exit 1
    fi
    
    # Check existing users
    if ! check_existing_users; then
        echo -e "${YELLOW}Skipping user creation.${NC}"
    else
        # Create users
        if ! create_users; then
            echo -e "${RED}Failed to create users. Please check the error messages above.${NC}"
            exit 1
        fi
    fi
    
    # Verify permissions
    verify_permissions
    
    # Test user access
    test_user_access
    
    echo ""
    echo -e "${GREEN}=== Setup Complete ===${NC}"
    echo ""
    echo -e "${BLUE}Connection details for your genai-toolbox configuration:${NC}"
    echo "  admin-db:"
    echo "    host: $RDS_HOST"
    echo "    port: $RDS_PORT"
    echo "    database: $DATABASE"
    echo "    user: admin_user"
    echo "    password: admin_password"
    echo ""
    echo "  support-db:"
    echo "    host: $RDS_HOST"
    echo "    port: $RDS_PORT"
    echo "    database: $DATABASE"
    echo "    user: support_user"
    echo "    password: support_password"
    echo ""
    echo -e "${YELLOW}Note: Update the passwords in production for security!${NC}"
}

# Run main function
main 