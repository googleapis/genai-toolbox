-- Verification Script for Database User Permissions
-- Run this script to verify that admin_user and support_user have correct permissions

-- 1. Check if users exist
SELECT rolname, rolsuper, rolinherit, rolcreaterole, rolcreatedb, rolcanlogin
FROM pg_roles 
WHERE rolname IN ('admin_user', 'support_user', 'postgres');

-- 2. Check table permissions for each user
-- For admin_user
SELECT 
    schemaname,
    tablename,
    grantee,
    privilege_type,
    is_grantable
FROM information_schema.table_privileges 
WHERE grantee = 'admin_user' 
AND tablename IN ('hotels', 'support_messages', 'support_tickets', 'integration_tokens')
ORDER BY tablename, privilege_type;

-- For support_user
SELECT 
    schemaname,
    tablename,
    grantee,
    privilege_type,
    is_grantable
FROM information_schema.table_privileges 
WHERE grantee = 'support_user' 
AND tablename IN ('hotels', 'support_messages', 'support_tickets', 'integration_tokens')
ORDER BY tablename, privilege_type;

-- 3. Check if tables exist
SELECT 
    schemaname,
    tablename,
    tableowner
FROM pg_tables 
WHERE tablename IN ('hotels', 'support_messages', 'support_tickets', 'integration_tokens')
ORDER BY tablename;

-- 4. Check sequence permissions (for auto-incrementing columns)
SELECT 
    sequence_schema,
    sequence_name,
    grantee,
    privilege_type
FROM information_schema.usage_privileges 
WHERE grantee IN ('admin_user', 'support_user')
AND object_type = 'SEQUENCE';

-- 5. Test queries (these should work when run as the respective users)
-- Run these as admin_user:
-- SELECT COUNT(*) FROM hotels;
-- SELECT COUNT(*) FROM support_messages;
-- SELECT COUNT(*) FROM support_tickets;
-- SELECT COUNT(*) FROM integration_tokens;

-- Run these as support_user:
-- SELECT COUNT(*) FROM hotels;
-- SELECT COUNT(*) FROM support_messages;
-- SELECT COUNT(*) FROM support_tickets;
-- SELECT COUNT(*) FROM integration_tokens;  -- This should fail with permission denied

-- 6. Check database connection permissions
SELECT 
    datname,
    usename,
    application_name,
    client_addr,
    state
FROM pg_stat_activity 
WHERE usename IN ('admin_user', 'support_user', 'postgres')
AND state = 'active'; 