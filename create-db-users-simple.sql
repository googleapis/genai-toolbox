-- Simplified Database User Creation and Permission Setup
-- This script creates admin_user and support_user with appropriate permissions
-- WITHOUT Row Level Security (RLS) for easier setup

-- Connect as superuser (postgres) to create users and grant permissions

-- 1. Create the users
CREATE USER admin_user WITH PASSWORD 'admin_password';
CREATE USER support_user WITH PASSWORD 'support_password';

-- 2. Grant basic connection permissions
GRANT CONNECT ON DATABASE toolbox_db TO admin_user;
GRANT CONNECT ON DATABASE toolbox_db TO support_user;

-- 3. Grant usage on schema (assuming public schema)
GRANT USAGE ON SCHEMA public TO admin_user;
GRANT USAGE ON SCHEMA public TO support_user;

-- 4. Grant permissions for support_user (limited access)
-- support_user can access: hotels, support_messages, support_tickets
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE hotels TO support_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE support_messages TO support_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE support_tickets TO support_user;

-- Grant sequence permissions for auto-incrementing columns (if any)
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO support_user;

-- 5. Grant permissions for admin_user (full access)
-- admin_user can access all tables including integration_tokens
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE hotels TO admin_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE support_messages TO admin_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE support_tickets TO admin_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE integration_tokens TO admin_user;

-- Grant sequence permissions for admin_user
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO admin_user;

-- 6. Grant additional admin privileges (optional - for system maintenance)
-- These allow admin_user to perform system-level operations
GRANT CREATE ON SCHEMA public TO admin_user;
GRANT CREATE ON DATABASE toolbox_db TO admin_user;

-- 7. Verify the setup
-- Check user permissions on tables
SELECT 
    schemaname,
    tablename,
    tableowner,
    hasinsert,
    hasselect,
    hasupdate,
    hasdelete
FROM pg_tables 
WHERE tablename IN ('hotels', 'support_messages', 'support_tickets', 'integration_tokens');

-- Check user roles
SELECT rolname, rolsuper, rolinherit, rolcreaterole, rolcreatedb, rolcanlogin
FROM pg_roles 
WHERE rolname IN ('admin_user', 'support_user'); 