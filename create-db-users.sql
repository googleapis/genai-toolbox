-- Database User Creation and Permission Setup for Hotel Application
-- This script creates admin_user and support_user with appropriate permissions

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

-- 7. Set up Row Level Security (RLS) if needed for multi-tenancy
-- This is optional but recommended for production environments

-- Enable RLS on tables that need it
ALTER TABLE hotels ENABLE ROW LEVEL SECURITY;
ALTER TABLE support_messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE support_tickets ENABLE ROW LEVEL SECURITY;
ALTER TABLE integration_tokens ENABLE ROW LEVEL SECURITY;

-- Create RLS policies (example - adjust based on your tenant/user structure)
-- For hotels table
CREATE POLICY hotels_select_policy ON hotels FOR SELECT USING (true);
CREATE POLICY hotels_insert_policy ON hotels FOR INSERT WITH CHECK (true);
CREATE POLICY hotels_update_policy ON hotels FOR UPDATE USING (true);
CREATE POLICY hotels_delete_policy ON hotels FOR DELETE USING (true);

-- For support_messages table
CREATE POLICY support_messages_select_policy ON support_messages FOR SELECT USING (true);
CREATE POLICY support_messages_insert_policy ON support_messages FOR INSERT WITH CHECK (true);
CREATE POLICY support_messages_update_policy ON support_messages FOR UPDATE USING (true);
CREATE POLICY support_messages_delete_policy ON support_messages FOR DELETE USING (true);

-- For support_tickets table
CREATE POLICY support_tickets_select_policy ON support_tickets FOR SELECT USING (true);
CREATE POLICY support_tickets_insert_policy ON support_tickets FOR INSERT WITH CHECK (true);
CREATE POLICY support_tickets_update_policy ON support_tickets FOR UPDATE USING (true);
CREATE POLICY support_tickets_delete_policy ON support_tickets FOR DELETE USING (true);

-- For integration_tokens table (admin only)
CREATE POLICY integration_tokens_admin_policy ON integration_tokens FOR ALL USING (current_user = 'admin_user');

-- 8. Verify the setup
-- You can run these queries to verify the permissions:

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

-- 9. Test the setup (run these as different users to verify)
-- Test as support_user:
-- psql -h your-rds-host -U support_user -d toolbox_db
-- SELECT * FROM hotels LIMIT 1;  -- Should work
-- SELECT * FROM integration_tokens LIMIT 1;  -- Should fail

-- Test as admin_user:
-- psql -h your-rds-host -U admin_user -d toolbox_db  
-- SELECT * FROM hotels LIMIT 1;  -- Should work
-- SELECT * FROM integration_tokens LIMIT 1;  -- Should work 