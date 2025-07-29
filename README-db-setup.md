# Database User Setup for Hotel Application

This directory contains SQL scripts and tools to set up database users with appropriate permissions for the hotel application.

## Overview

The setup creates two database users with different permission levels:

- **`admin_user`**: Full access to all tables including `integration_tokens`
- **`support_user`**: Limited access to customer-facing tables only

## Tables and Access Levels

| Table | admin_user | support_user | Description |
|-------|------------|--------------|-------------|
| `hotels` | ✅ Full access | ✅ Full access | Hotel information |
| `support_messages` | ✅ Full access | ✅ Full access | Support chat messages |
| `support_tickets` | ✅ Full access | ✅ Full access | Support ticket system |
| `integration_tokens` | ✅ Full access | ❌ No access | Sensitive integration data |

## Files

- `create-db-users.sql` - Complete setup with Row Level Security (RLS)
- `create-db-users-simple.sql` - Simplified setup without RLS (recommended for initial setup)
- `verify-permissions.sql` - Verification queries to check permissions
- `setup-db-users.sh` - Automated setup script
- `README-db-setup.md` - This file

## Quick Setup

### Option 1: Automated Setup (Recommended)

1. **Update the configuration** in `setup-db-users.sh`:
   ```bash
   RDS_HOST="your-rds-host.amazonaws.com"
   SUPERUSER_PASSWORD="your-postgres-password"
   ```

2. **Run the setup script**:
   ```bash
   ./setup-db-users.sh
   ```

### Option 2: Manual Setup

1. **Connect to your RDS instance** as the postgres superuser:
   ```bash
   psql -h your-rds-host -p 5432 -U postgres -d toolbox_db
   ```

2. **Run the SQL script**:
   ```sql
   \i create-db-users-simple.sql
   ```

3. **Verify the setup**:
   ```sql
   \i verify-permissions.sql
   ```

## Manual SQL Commands

If you prefer to run commands manually:

```sql
-- Create users
CREATE USER admin_user WITH PASSWORD 'admin_password';
CREATE USER support_user WITH PASSWORD 'support_password';

-- Grant basic permissions
GRANT CONNECT ON DATABASE toolbox_db TO admin_user, support_user;
GRANT USAGE ON SCHEMA public TO admin_user, support_user;

-- Grant table permissions for support_user (limited access)
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE hotels TO support_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE support_messages TO support_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE support_tickets TO support_user;

-- Grant table permissions for admin_user (full access)
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE hotels TO admin_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE support_messages TO admin_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE support_tickets TO admin_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE integration_tokens TO admin_user;

-- Grant sequence permissions
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO admin_user, support_user;
```

## Testing the Setup

### Test support_user access:
```bash
PGPASSWORD="support_password" psql -h your-rds-host -U support_user -d toolbox_db
```

```sql
-- These should work:
SELECT COUNT(*) FROM hotels;
SELECT COUNT(*) FROM support_messages;
SELECT COUNT(*) FROM support_tickets;

-- This should fail:
SELECT COUNT(*) FROM integration_tokens;
```

### Test admin_user access:
```bash
PGPASSWORD="admin_password" psql -h your-rds-host -U admin_user -d toolbox_db
```

```sql
-- All of these should work:
SELECT COUNT(*) FROM hotels;
SELECT COUNT(*) FROM support_messages;
SELECT COUNT(*) FROM support_tickets;
SELECT COUNT(*) FROM integration_tokens;
```

## Integration with genai-toolbox

After setting up the users, update your `tools-hotel.yaml` configuration:

```yaml
sources:
  admin-db:
    kind: postgres
    host: your-rds-host.amazonaws.com
    port: 5432
    database: toolbox_db
    user: admin_user
    password: admin_password
  
  support-db:
    kind: postgres
    host: your-rds-host.amazonaws.com
    port: 5432
    database: toolbox_db
    user: support_user
    password: support_password
```

## Security Considerations

1. **Change default passwords** in production
2. **Use environment variables** for passwords in your configuration
3. **Consider enabling RLS** for multi-tenant scenarios
4. **Regularly audit permissions** using the verification script
5. **Use SSL connections** for production databases

## Troubleshooting

### Common Issues

1. **Connection failed**: Check RDS security groups and network access
2. **Permission denied**: Ensure you're connected as postgres superuser
3. **User already exists**: The script will prompt to recreate users
4. **Tables don't exist**: Create the tables before running user setup

### Verification Commands

```sql
-- Check if users exist
SELECT rolname FROM pg_roles WHERE rolname IN ('admin_user', 'support_user');

-- Check table permissions
SELECT grantee, tablename, privilege_type 
FROM information_schema.table_privileges 
WHERE grantee IN ('admin_user', 'support_user')
AND tablename IN ('hotels', 'support_messages', 'support_tickets', 'integration_tokens');
```

## Row Level Security (RLS)

For production environments with multi-tenancy, consider enabling RLS:

1. Use `create-db-users.sql` instead of the simple version
2. Customize RLS policies based on your tenant structure
3. Test thoroughly before deploying to production

## Support

If you encounter issues:

1. Check the verification script output
2. Review PostgreSQL logs for detailed error messages
3. Ensure all tables exist before running user setup
4. Verify network connectivity to your RDS instance 