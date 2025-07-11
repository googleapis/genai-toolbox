# Snowflake Examples

This directory contains examples for using the genai-toolbox with Snowflake databases.

## Files

- `snowflake-config.yaml` - Example configuration file with Snowflake tools
- `snowflake-env.sh` - Environment setup script template

## Quick Start

### 1. Set up your environment

Copy the environment template and update it with your Snowflake credentials:

```bash
cp examples/snowflake-env.sh my-snowflake-env.sh
```

Edit `my-snowflake-env.sh` with your actual Snowflake connection details:

```bash
export SNOWFLAKE_ACCOUNT="your-account-identifier"
export SNOWFLAKE_USER="your-username"
export SNOWFLAKE_PASSWORD="your-password"
export SNOWFLAKE_DATABASE="your-database"
export SNOWFLAKE_SCHEMA="your-schema"
export SNOWFLAKE_WAREHOUSE="COMPUTE_WH"
export SNOWFLAKE_ROLE="ACCOUNTADMIN"
```

### 2. Load the environment

```bash
source my-snowflake-env.sh
```

### 3. Run the toolbox

You have two options:

#### Option A: Use the prebuilt configuration

```bash
./toolbox --prebuilt snowflake
```

#### Option B: Use the custom configuration

```bash
./toolbox --tools-file examples/snowflake-config.yaml
```

## Available Tools

### execute_sql
Execute arbitrary SQL statements on Snowflake.

**Usage:**
```bash
curl -X POST http://localhost:5000/tools/execute_sql \
  -H "Content-Type: application/json" \
  -d '{"sql": "SELECT CURRENT_TIMESTAMP()"}'
```

### get_customer_orders
Get orders for a specific customer.

**Usage:**
```bash
curl -X POST http://localhost:5000/tools/get_customer_orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id": "CUST123"}'
```

### daily_sales_report
Generate daily sales report for a specific date.

**Usage:**
```bash
curl -X POST http://localhost:5000/tools/daily_sales_report \
  -H "Content-Type: application/json" \
  -d '{"report_date": "2024-01-15"}'
```

## Environment Variables

### Required
- `SNOWFLAKE_ACCOUNT` - Your Snowflake account identifier
- `SNOWFLAKE_USER` - Your Snowflake username
- `SNOWFLAKE_PASSWORD` - Your Snowflake password
- `SNOWFLAKE_DATABASE` - Database name
- `SNOWFLAKE_SCHEMA` - Schema name

### Optional
- `SNOWFLAKE_WAREHOUSE` - Warehouse name (default: COMPUTE_WH)
- `SNOWFLAKE_ROLE` - Role name (default: ACCOUNTADMIN)

## Security Notes

- Never commit your actual credentials to version control
- Use environment variables for all sensitive information
- Consider using Snowflake's key-pair authentication for production environments
- Ensure your Snowflake account has appropriate network access policies

## Troubleshooting

1. **Connection Issues**: Verify your account identifier and credentials
2. **Permission Issues**: Ensure your user has the necessary privileges
3. **Network Issues**: Check if your IP is allowed to access Snowflake
4. **Warehouse Issues**: Verify the warehouse exists and is accessible

For more information, see the [Snowflake documentation](https://docs.snowflake.com/).
