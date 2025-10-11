# Oracle Tools Documentation

Complete guide for using Oracle Database with MCP Toolbox for Databases.

---

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Source Configuration](#source-configuration)
4. [Available Tools](#available-tools)
5. [Tool: oracle-execute-sql](#tool-oracle-execute-sql)
6. [Tool: oracle-sql](#tool-oracle-sql)
7. [Complete Examples](#complete-examples)
8. [Best Practices](#best-practices)
9. [Troubleshooting](#troubleshooting)

---

## Overview

The Oracle source enables AI applications to interact with Oracle Database through MCP Toolbox. It supports:

- ✅ Oracle Database (11g, 12c, 18c, 19c, 21c, 23ai)
- ✅ Oracle Autonomous Database (with Wallet)
- ✅ Oracle Cloud Infrastructure (OCI)
- ✅ On-premise Oracle installations
- ✅ Oracle Express Edition (XE)

### Quick Start

```yaml
sources:
  my-oracle:
    kind: oracle
    host: localhost
    port: 1521
    service_name: FREEPDB1
    user: ${ORACLE_USER}
    password: ${ORACLE_PASSWORD}

tools:
  execute-sql:
    kind: oracle-execute-sql
    source: my-oracle
    description: Execute SQL queries
```

---

## Prerequisites

Before using Oracle tools with MCP Toolbox, you must install the Oracle Instant Client and SQL*Plus.

### Required Software

| Software | Minimum Version | Purpose |
|----------|----------------|---------|
| Oracle Instant Client | 23.9 or higher | Database connectivity |
| SQL*Plus | 23.9 or higher | Connection testing |
| MCP Toolbox | Latest | MCP server |

### Installation Instructions

#### Linux (x86_64)

**Step 1: Download Oracle Instant Client**

Visit [Oracle Instant Client Downloads](https://www.oracle.com/database/technologies/instant-client/downloads.html)

Download these packages:
- `instantclient-basic-linux.x64-23.9.0.0.0.zip`
- `instantclient-sqlplus-linux.x64-23.9.0.0.0.zip`

**Step 2: Install**

```bash
# Create installation directory
sudo mkdir -p /opt/oracle

# Extract files
cd /opt/oracle
sudo unzip instantclient-basic-linux.x64-23.9.0.0.0.zip
sudo unzip instantclient-sqlplus-linux.x64-23.9.0.0.0.zip

# Create symbolic links
cd /opt/oracle/instantclient_23_9
sudo ln -s libclntsh.so.23.1 libclntsh.so
sudo ln -s libocci.so.23.1 libocci.so
```

**Step 3: Configure Environment**

Add to `~/.bashrc` or `~/.bash_profile`:

```bash
# Oracle Instant Client
export ORACLE_HOME=/opt/oracle/instantclient_23_9
export LD_LIBRARY_PATH=$ORACLE_HOME:$LD_LIBRARY_PATH
export PATH=$ORACLE_HOME:$PATH
```

Apply changes:
```bash
source ~/.bashrc
```

**Step 4: Install libaio (if not present)**

```bash
# RHEL/CentOS/Oracle Linux
sudo yum install libaio

# Ubuntu/Debian
sudo apt-get install libaio1

# Verify installation
ldconfig -p | grep libaio
```

#### macOS (ARM64 - Apple Silicon)

**Step 1: Download**

Visit [Oracle Instant Client Downloads](https://www.oracle.com/database/technologies/instant-client/downloads.html)

Download:
- `instantclient-basic-macos.arm64-23.9.0.0.0.zip`
- `instantclient-sqlplus-macos.arm64-23.9.0.0.0.zip`

**Step 2: Install**

```bash
# Create directory
sudo mkdir -p /opt/oracle

# Extract
cd /opt/oracle
sudo unzip ~/Downloads/instantclient-basic-macos.arm64-23.9.0.0.0.zip
sudo unzip ~/Downloads/instantclient-sqlplus-macos.arm64-23.9.0.0.0.zip

# Remove quarantine attribute
sudo xattr -r -d com.apple.quarantine /opt/oracle/instantclient_23_9
```

**Step 3: Configure Environment**

Add to `~/.zshrc`:

```bash
# Oracle Instant Client
export ORACLE_HOME=/opt/oracle/instantclient_23_9
export DYLD_LIBRARY_PATH=$ORACLE_HOME:$DYLD_LIBRARY_PATH
export PATH=$ORACLE_HOME:$PATH
```

Apply:
```bash
source ~/.zshrc
```

#### macOS (x86_64 - Intel)

Similar to ARM64, but download x86_64 packages:
- `instantclient-basic-macos.x64-23.9.0.0.0.zip`
- `instantclient-sqlplus-macos.x64-23.9.0.0.0.zip`

#### Windows

**Step 1: Download**

Download from Oracle:
- `instantclient-basic-windows.x64-23.9.0.0.0.zip`
- `instantclient-sqlplus-windows.x64-23.9.0.0.0.zip`

**Step 2: Install**

```powershell
# Extract to C:\oracle\instantclient_23_9
# (Right-click > Extract All)

# Or use PowerShell
Expand-Archive -Path instantclient-basic-windows.x64-23.9.0.0.0.zip -DestinationPath C:\oracle\
Expand-Archive -Path instantclient-sqlplus-windows.x64-23.9.0.0.0.zip -DestinationPath C:\oracle\
```

**Step 3: Add to PATH**

1. Open System Properties > Environment Variables
2. Edit `Path` variable
3. Add: `C:\oracle\instantclient_23_9`
4. Click OK

**Step 4: Install Visual C++ Redistributable**

Download and install: [Microsoft Visual C++ Redistributable](https://aka.ms/vs/17/release/vc_redist.x64.exe)

### Verify Installation

#### Test Instant Client

```bash
# Linux/macOS
ls -la $ORACLE_HOME/libclntsh.so*

# Windows
dir C:\oracle\instantclient_23_9\oci.dll
```

#### Test SQL*Plus

```bash
sqlplus -v
```

Expected output:
```
SQL*Plus: Release 23.0.0.0.0 - Production
Version 23.9.0.0.0
```

#### Test Database Connection

**Standard Connection:**
```bash
sqlplus username/password@hostname:port/service_name
```

Example:
```bash
sqlplus admin/MyPassword@localhost:1521/FREEPDB1
```

**Autonomous Database with Wallet:**
```bash
# Set TNS_ADMIN to wallet directory
export TNS_ADMIN=/path/to/wallet

# Connect using TNS alias
sqlplus admin/MyPassword@myadb_medium
```

**Expected Output (Success):**
```
SQL*Plus: Release 23.0.0.0.0 - Production

Connected to:
Oracle Database 23ai Enterprise Edition Release 23.0.0.0.0 - Production

SQL>
```

**Test Query:**
```sql
SQL> SELECT * FROM dual;

D
-
X

SQL> SELECT USER, TO_CHAR(SYSDATE, 'YYYY-MM-DD HH24:MI:SS') FROM DUAL;

USER                           TO_CHAR(SYSDATE,'Y
------------------------------ -------------------
ADMIN                          2025-10-10 21:30:45

SQL> EXIT
```

### Configure MCP Toolbox for Oracle

**Linux/macOS:**

Ensure the Go application can find Oracle libraries:

```bash
# In your MCP Toolbox startup script or .bashrc
export ORACLE_HOME=/opt/oracle/instantclient_23_9
export LD_LIBRARY_PATH=$ORACLE_HOME:$LD_LIBRARY_PATH

# For macOS
export DYLD_LIBRARY_PATH=$ORACLE_HOME:$DYLD_LIBRARY_PATH
```

If using Go code, initialize Oracle client:

```go
import "github.com/godror/godror"

func init() {
    // Initialize Oracle client library
    oracledb.init_oracle_client(lib_dir="/opt/oracle/instantclient_23_9")
}
```

**Windows:**

No additional configuration needed if `instantclient_23_9` is in PATH.

### Troubleshooting Installation

#### Error: libclntsh.so: cannot open shared object file

**Solution:**
```bash
# Linux
export LD_LIBRARY_PATH=/opt/oracle/instantclient_23_9:$LD_LIBRARY_PATH
sudo ldconfig

# Verify
ldd /opt/oracle/instantclient_23_9/libclntsh.so
```

#### Error: libaio.so.1: cannot open shared object file

**Solution:**
```bash
# RHEL/CentOS
sudo yum install libaio

# Ubuntu/Debian
sudo apt-get install libaio1
```

#### Error: ORA-12154 after wallet extraction

**Solution:**
```bash
# Ensure TNS_ADMIN is set
export TNS_ADMIN=/path/to/wallet

# Verify files exist
ls -la $TNS_ADMIN/
# Should show: tnsnames.ora, sqlnet.ora, cwallet.sso, ewallet.p12

# Test connection
sqlplus admin/password@tns_alias
```

#### macOS: Library not loaded

**Solution:**
```bash
# Remove quarantine
sudo xattr -r -d com.apple.quarantine /opt/oracle/instantclient_23_9

# Set library path
export DYLD_LIBRARY_PATH=/opt/oracle/instantclient_23_9:$DYLD_LIBRARY_PATH
```

### Version Compatibility

| Oracle DB Version | Instant Client Version | Supported |
|-------------------|----------------------|-----------|
| 11g | 23.9+ | ✅ Yes |
| 12c | 23.9+ | ✅ Yes |
| 18c | 23.9+ | ✅ Yes |
| 19c | 23.9+ | ✅ Yes |
| 21c | 23.9+ | ✅ Yes |
| 23ai | 23.9+ | ✅ Yes (Recommended) |

**Note:** Oracle Instant Client 23.9+ can connect to Oracle Database 11g and higher.

### Additional Resources

- [Oracle Instant Client Downloads](https://www.oracle.com/database/technologies/instant-client/downloads.html)
- [Oracle Instant Client Installation Guide](https://www.oracle.com/database/technologies/instant-client/linux-x86-64-downloads.html)
- [Oracle Documentation](https://docs.oracle.com/en/database/)

---

## Source Configuration

### Method 1: Host/Port Connection

For standard Oracle Database:

```yaml
sources:
  oracle-standard:
    kind: oracle
    host: oracle.example.com
    port: 1521
    service_name: PRODDB
    user: ${ORACLE_USER}
    password: ${ORACLE_PASSWORD}
```

### Method 2: Autonomous Database with Wallet

For Oracle Autonomous Database:

```yaml
sources:
  oracle-adb:
    kind: oracle
    tns_alias: myadb_medium        # From tnsnames.ora
    tns_admin: /path/to/wallet     # Wallet directory
    user: ADMIN
    password: ${ORACLE_PASSWORD}
```

**Wallet Setup:**
1. Download wallet from Oracle Cloud Console
2. Unzip to directory: `/home/user/wallet`
3. Set `tns_admin` to wallet path
4. Use TNS alias from `tnsnames.ora` (e.g., `myadb_low`, `myadb_medium`, `myadb_high`)

### Method 3: DSN String

For complex connection strings:

```yaml
sources:
  oracle-dsn:
    kind: oracle
    dsn: |
      (DESCRIPTION=
        (ADDRESS=(PROTOCOL=TCP)(HOST=oracle.example.com)(PORT=1521))
        (CONNECT_DATA=(SERVICE_NAME=PRODDB)))
    user: ${ORACLE_USER}
    password: ${ORACLE_PASSWORD}
```

### Configuration Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `kind` | string | Yes | Must be `oracle` |
| `host` | string | No* | Database hostname |
| `port` | integer | No* | Port (default: 1521) |
| `service_name` | string | No* | Oracle service name |
| `sid` | string | No* | Oracle SID (alternative) |
| `tns_alias` | string | No* | TNS alias from tnsnames.ora |
| `tns_admin` | string | No* | Path to wallet directory |
| `dsn` | string | No* | Complete DSN string |
| `user` | string | Yes | Database username |
| `password` | string | Yes | Database password |

*Provide either: `host`/`port`/`service_name` OR `tns_alias`/`tns_admin` OR `dsn`

---

## Available Tools

### 1. oracle-execute-sql

Execute arbitrary SQL statements dynamically.

**Use Cases:**
- Administrative tasks
- Database setup/migration
- Ad-hoc queries
- Development and testing

**Security:** ⚠️ Allows any SQL - use with caution

### 2. oracle-sql

Execute pre-defined parameterized queries.

**Use Cases:**
- Production applications
- User-facing features
- Secure data access
- Reusable queries

**Security:** ✅ SQL injection protected

---

## Tool: oracle-execute-sql

Execute dynamic SQL statements against Oracle Database.

### Configuration

```yaml
tools:
  oracle-execute-sql:
    kind: oracle-execute-sql
    source: my-oracle-source
    description: Execute arbitrary SQL queries on Oracle database
```

### Usage

**Input:**
```json
{
  "sql": "SELECT employee_id, first_name, last_name FROM employees WHERE department_id = 10"
}
```

**Output:**
```json
[
  {
    "EMPLOYEE_ID": "100",
    "FIRST_NAME": "Steven",
    "LAST_NAME": "King"
  },
  {
    "EMPLOYEE_ID": "101",
    "FIRST_NAME": "Neena",
    "LAST_NAME": "Kochhar"
  }
]
```

### Examples

#### Example 1: SELECT Query

```json
{
  "sql": "SELECT * FROM departments WHERE location_id = 1700 ORDER BY department_name"
}
```

Returns:
```json
[
  {"DEPARTMENT_ID": "10", "DEPARTMENT_NAME": "Administration", "MANAGER_ID": "200", "LOCATION_ID": "1700"},
  {"DEPARTMENT_ID": "20", "DEPARTMENT_NAME": "Marketing", "MANAGER_ID": "201", "LOCATION_ID": "1700"}
]
```

#### Example 2: INSERT Statement

```json
{
  "sql": "INSERT INTO employees (employee_id, first_name, last_name, email, hire_date) VALUES (999, 'John', 'Doe', 'jdoe@example.com', SYSDATE)"
}
```

Returns:
```json
[]
```

#### Example 3: Query with JOIN

```json
{
  "sql": "SELECT e.first_name, e.last_name, d.department_name FROM employees e JOIN departments d ON e.department_id = d.department_id WHERE e.salary > 10000"
}
```

#### Example 4: Aggregate Query

```json
{
  "sql": "SELECT department_id, COUNT(*) as emp_count, AVG(salary) as avg_salary FROM employees GROUP BY department_id ORDER BY department_id"
}
```

#### Example 5: Complex Query with CASE

```json
{
  "sql": "SELECT employee_id, first_name, salary, CASE WHEN salary < 5000 THEN 'Low' WHEN salary BETWEEN 5000 AND 10000 THEN 'Medium' ELSE 'High' END as salary_grade FROM employees ORDER BY salary DESC"
}
```

#### Example 6: Date Range Query

```json
{
  "sql": "SELECT employee_id, first_name, hire_date FROM employees WHERE hire_date BETWEEN TO_DATE('2020-01-01', 'YYYY-MM-DD') AND TO_DATE('2020-12-31', 'YYYY-MM-DD')"
}
```

#### Example 7: Oracle 23ai Vector Search

```json
{
  "sql": "SELECT id, text, VECTOR_DISTANCE(embedding, VECTOR_EMBEDDING(model USING 'search query' AS data), COSINE) as similarity FROM documents ORDER BY similarity FETCH FIRST 5 ROWS ONLY"
}
```

### Multi-Row Results

✅ **Correctly handles multiple rows:**

```json
{
  "sql": "SELECT * FROM employees WHERE ROWNUM <= 5"
}
```

Returns array with 5 elements:
```json
[
  {"EMPLOYEE_ID": "100", ...},
  {"EMPLOYEE_ID": "101", ...},
  {"EMPLOYEE_ID": "102", ...},
  {"EMPLOYEE_ID": "103", ...},
  {"EMPLOYEE_ID": "104", ...}
]
```

### Important Notes

- Column names returned in **UPPERCASE**
- Always returns **array**, even for single row: `[{...}]`
- Empty results return empty array: `[]`
- INSERT/UPDATE/DELETE without RETURNING return `[]`

---

## Tool: oracle-sql

Execute pre-defined parameterized SQL queries.

### Configuration

```yaml
tools:
  get-employee:
    kind: oracle-sql
    source: my-oracle-source
    description: Get employee by ID
    parameters:
      - name: employee_id
        type: integer
        description: The employee ID to look up
    statement: |
      SELECT employee_id, first_name, last_name, email, salary
      FROM employees
      WHERE employee_id = :1
```

### Parameter Types

| Type | Oracle Type | Example | Use For |
|------|------------|---------|---------|
| `string` | VARCHAR2 | `'Hello'` | Text, emails, names |
| `integer` | NUMBER | `42` | IDs, counts |
| `number` | NUMBER | `3.14` | Decimals, salaries |
| `boolean` | NUMBER(1) | `1`/`0` | True/false flags |

### Parameter Binding

Oracle uses **positional parameters**: `:1`, `:2`, `:3`, etc.

```yaml
statement: |
  SELECT * FROM employees
  WHERE department_id = :1 AND salary >= :2
parameters:
  - name: dept_id      # Maps to :1
    type: integer
  - name: min_salary   # Maps to :2
    type: number
```

### Examples

#### Example 1: Simple Lookup

```yaml
tools:
  get-department:
    kind: oracle-sql
    source: my-oracle
    description: Get department by ID
    parameters:
      - name: dept_id
        type: integer
    statement: |
      SELECT department_id, department_name, manager_id, location_id
      FROM departments
      WHERE department_id = :1
```

**Usage:**
```json
{"dept_id": 10}
```

**Returns:**
```json
[
  {
    "DEPARTMENT_ID": "10",
    "DEPARTMENT_NAME": "Administration",
    "MANAGER_ID": "200",
    "LOCATION_ID": "1700"
  }
]
```

#### Example 2: Search with LIKE

```yaml
tools:
  search-employees:
    kind: oracle-sql
    source: my-oracle
    description: Search employees by name
    parameters:
      - name: name_pattern
        type: string
    statement: |
      SELECT employee_id, first_name, last_name, email
      FROM employees
      WHERE UPPER(first_name) LIKE UPPER('%' || :1 || '%')
         OR UPPER(last_name) LIKE UPPER('%' || :1 || '%')
      ORDER BY last_name, first_name
```

**Usage:**
```json
{"name_pattern": "john"}
```

#### Example 3: Range Query

```yaml
tools:
  employees-by-salary-range:
    kind: oracle-sql
    source: my-oracle
    description: Find employees within salary range
    parameters:
      - name: min_salary
        type: number
      - name: max_salary
        type: number
    statement: |
      SELECT employee_id, first_name, last_name, salary
      FROM employees
      WHERE salary BETWEEN :1 AND :2
      ORDER BY salary DESC
```

**Usage:**
```json
{
  "min_salary": 50000,
  "max_salary": 100000
}
```

#### Example 4: Complex Multi-Parameter

```yaml
tools:
  search-employees-advanced:
    kind: oracle-sql
    source: my-oracle
    parameters:
      - name: department_name
        type: string
      - name: min_salary
        type: number
      - name: job_title_pattern
        type: string
    statement: |
      SELECT e.employee_id, e.first_name, e.last_name, e.salary, j.job_title
      FROM employees e
      JOIN jobs j ON e.job_id = j.job_id
      JOIN departments d ON e.department_id = d.department_id
      WHERE UPPER(d.department_name) = UPPER(:1)
        AND e.salary >= :2
        AND UPPER(j.job_title) LIKE UPPER('%' || :3 || '%')
      ORDER BY e.salary DESC
```

**Usage:**
```json
{
  "department_name": "IT",
  "min_salary": 60000,
  "job_title_pattern": "developer"
}
```

#### Example 5: Aggregate Query

```yaml
tools:
  department-statistics:
    kind: oracle-sql
    source: my-oracle
    parameters:
      - name: dept_id
        type: integer
    statement: |
      SELECT 
        department_id,
        COUNT(*) as employee_count,
        ROUND(AVG(salary), 2) as avg_salary,
        MIN(salary) as min_salary,
        MAX(salary) as max_salary
      FROM employees
      WHERE department_id = :1
      GROUP BY department_id
```

#### Example 6: Date Range

```yaml
tools:
  employees-hired-between:
    kind: oracle-sql
    source: my-oracle
    parameters:
      - name: start_date
        type: string
        description: Start date (YYYY-MM-DD)
      - name: end_date
        type: string
    statement: |
      SELECT employee_id, first_name, last_name, hire_date
      FROM employees
      WHERE hire_date BETWEEN TO_DATE(:1, 'YYYY-MM-DD') 
                          AND TO_DATE(:2, 'YYYY-MM-DD')
      ORDER BY hire_date
```

**Usage:**
```json
{
  "start_date": "2020-01-01",
  "end_date": "2020-12-31"
}
```

#### Example 7: Hierarchical Query

```yaml
tools:
  get-employee-hierarchy:
    kind: oracle-sql
    source: my-oracle
    parameters:
      - name: manager_id
        type: integer
    statement: |
      SELECT 
        LEVEL as hierarchy_level,
        employee_id,
        first_name || ' ' || last_name as full_name,
        manager_id,
        LPAD(' ', (LEVEL-1)*2) || first_name as indented_name
      FROM employees
      START WITH manager_id = :1
      CONNECT BY PRIOR employee_id = manager_id
      ORDER SIBLINGS BY last_name
```

#### Example 8: Window Functions

```yaml
tools:
  salary-rankings:
    kind: oracle-sql
    source: my-oracle
    parameters:
      - name: dept_id
        type: integer
    statement: |
      SELECT 
        employee_id,
        first_name,
        last_name,
        salary,
        RANK() OVER (ORDER BY salary DESC) as salary_rank,
        PERCENT_RANK() OVER (ORDER BY salary DESC) * 100 as percentile
      FROM employees
      WHERE department_id = :1
      ORDER BY salary DESC
```

#### Example 9: Oracle AI Vector Search

```yaml
tools:
  vector-similarity-search:
    kind: oracle-sql
    source: my-oracle
    description: Search similar documents using AI Vector Search
    parameters:
      - name: search_query
        type: string
    statement: |
      WITH query_vector AS (
        SELECT VECTOR_EMBEDDING(ALL_MINILM_L6V2MODEL USING :1 AS data) as embedding
      )
      SELECT 
        id, 
        text,
        VECTOR_DISTANCE(doc.embedding, query_vector.embedding, COSINE) as similarity
      FROM documents doc, query_vector
      ORDER BY VECTOR_DISTANCE(doc.embedding, query_vector.embedding, COSINE)
      FETCH FIRST 10 ROWS ONLY
```

**Usage:**
```json
{"search_query": "oracle database performance tuning"}
```

---

## Complete Examples

### Example 1: Employee Management System

```yaml
sources:
  hr-database:
    kind: oracle
    host: hr-prod.example.com
    port: 1521
    service_name: HRPROD
    user: ${ORACLE_USER}
    password: ${ORACLE_PASSWORD}

tools:
  # Dynamic SQL for admin
  hr-execute-sql:
    kind: oracle-execute-sql
    source: hr-database
    description: Execute administrative SQL queries
  
  # Get employee details
  get-employee:
    kind: oracle-sql
    source: hr-database
    description: Get detailed employee information
    parameters:
      - name: emp_id
        type: integer
    statement: |
      SELECT 
        e.employee_id,
        e.first_name || ' ' || e.last_name as full_name,
        e.email,
        e.phone_number,
        e.hire_date,
        e.salary,
        d.department_name,
        j.job_title,
        m.first_name || ' ' || m.last_name as manager_name
      FROM employees e
      LEFT JOIN departments d ON e.department_id = d.department_id
      LEFT JOIN jobs j ON e.job_id = j.job_id
      LEFT JOIN employees m ON e.manager_id = m.employee_id
      WHERE e.employee_id = :1
  
  # Search employees
  search-employees:
    kind: oracle-sql
    source: hr-database
    description: Search employees by name or email
    parameters:
      - name: search_term
        type: string
    statement: |
      SELECT employee_id, first_name, last_name, email, department_id
      FROM employees
      WHERE UPPER(first_name || ' ' || last_name) LIKE UPPER('%' || :1 || '%')
         OR UPPER(email) LIKE UPPER('%' || :1 || '%')
      ORDER BY last_name, first_name
      FETCH FIRST 50 ROWS ONLY
  
  # Department summary
  department-summary:
    kind: oracle-sql
    source: hr-database
    description: Get department statistics
    parameters:
      - name: dept_name
        type: string
    statement: |
      SELECT 
        d.department_name,
        COUNT(e.employee_id) as employee_count,
        ROUND(AVG(e.salary), 2) as avg_salary,
        MIN(e.salary) as min_salary,
        MAX(e.salary) as max_salary,
        SUM(e.salary) as total_payroll
      FROM departments d
      LEFT JOIN employees e ON d.department_id = e.department_id
      WHERE UPPER(d.department_name) = UPPER(:1)
      GROUP BY d.department_name
```

### Example 2: E-Commerce Order System

```yaml
sources:
  ecommerce-db:
    kind: oracle
    tns_alias: ecommerce_high
    tns_admin: /app/oracle_wallet
    user: ${ORACLE_USER}
    password: ${ORACLE_PASSWORD}

tools:
  # Get order details
  get-order:
    kind: oracle-sql
    source: ecommerce-db
    parameters:
      - name: order_id
        type: integer
    statement: |
      SELECT 
        o.order_id,
        o.order_date,
        o.status,
        c.customer_name,
        c.email,
        o.total_amount,
        COUNT(oi.item_id) as item_count
      FROM orders o
      JOIN customers c ON o.customer_id = c.customer_id
      LEFT JOIN order_items oi ON o.order_id = oi.order_id
      WHERE o.order_id = :1
      GROUP BY o.order_id, o.order_date, o.status, c.customer_name, c.email, o.total_amount
  
  # Search orders by customer
  search-orders-by-customer:
    kind: oracle-sql
    source: ecommerce-db
    parameters:
      - name: customer_email
        type: string
    statement: |
      SELECT 
        o.order_id,
        o.order_date,
        o.status,
        o.total_amount
      FROM orders o
      JOIN customers c ON o.customer_id = c.customer_id
      WHERE UPPER(c.email) = UPPER(:1)
      ORDER BY o.order_date DESC
  
  # Sales by date range
  sales-report:
    kind: oracle-sql
    source: ecommerce-db
    parameters:
      - name: start_date
        type: string
      - name: end_date
        type: string
    statement: |
      SELECT 
        TO_CHAR(order_date, 'YYYY-MM-DD') as order_day,
        COUNT(*) as order_count,
        SUM(total_amount) as total_sales,
        ROUND(AVG(total_amount), 2) as avg_order_value
      FROM orders
      WHERE order_date BETWEEN TO_DATE(:1, 'YYYY-MM-DD') 
                           AND TO_DATE(:2, 'YYYY-MM-DD')
      GROUP BY TO_CHAR(order_date, 'YYYY-MM-DD')
      ORDER BY order_day
```

### Example 3: Document Search with AI Vectors

```yaml
sources:
  knowledge-base:
    kind: oracle
    host: localhost
    port: 1521
    service_name: KB23AI
    user: ${ORACLE_USER}
    password: ${ORACLE_PASSWORD}

tools:
  # Execute administrative SQL
  kb-execute-sql:
    kind: oracle-execute-sql
    source: knowledge-base
    description: Execute SQL for knowledge base management
  
  # Vector similarity search
  search-similar-documents:
    kind: oracle-sql
    source: knowledge-base
    description: Find similar documents using AI vector search
    parameters:
      - name: search_query
        type: string
        description: The search query text
    statement: |
      WITH query_vector AS (
        SELECT VECTOR_EMBEDDING(ALL_MINILM_L6V2MODEL USING :1 AS data) as embedding
      )
      SELECT 
        id,
        title,
        text,
        VECTOR_DISTANCE(d.embedding, qv.embedding, COSINE) as similarity_score
      FROM documents d, query_vector qv
      ORDER BY VECTOR_DISTANCE(d.embedding, qv.embedding, COSINE)
      FETCH APPROX FIRST 10 ROWS ONLY
  
  # Search by category
  search-by-category:
    kind: oracle-sql
    source: knowledge-base
    parameters:
      - name: category
        type: string
      - name: limit_results
        type: integer
    statement: |
      SELECT id, title, category, created_date
      FROM documents
      WHERE UPPER(category) = UPPER(:1)
      ORDER BY created_date DESC
      FETCH FIRST :2 ROWS ONLY
```

---

## Best Practices

### 1. Use oracle-sql for Production

✅ **Recommended:**
```yaml
tools:
  get-user:
    kind: oracle-sql
    statement: SELECT * FROM users WHERE user_id = :1
    parameters:
      - name: user_id
        type: integer
```

❌ **Avoid:**
```yaml
tools:
  get-user:
    kind: oracle-execute-sql  # Allows any SQL
```

### 2. Always Limit Results

```yaml
statement: |
  SELECT * FROM large_table
  WHERE category = :1
  FETCH FIRST 100 ROWS ONLY  -- Limit results
```

### 3. Use Environment Variables

```yaml
sources:
  my-oracle:
    user: ${ORACLE_USER}      # ✅ Good
    password: ${ORACLE_PASSWORD}  # ✅ Good
    # password: "hardcoded123"    # ❌ Never hardcode
```

### 4. Provide Clear Descriptions

```yaml
tools:
  search-products:
    description: Search products by name, SKU, or category with optional price range filtering
    parameters:
      - name: search_term
        type: string
        description: Product name, SKU, or category to search for
      - name: min_price
        type: number
        description: Minimum price in dollars (optional, use 0 for no minimum)
```

### 5. Handle NULL Values

```yaml
statement: |
  SELECT 
    employee_id,
    NVL(commission_pct, 0) as commission_pct,  -- Convert NULL to 0
    NVL2(manager_id, 'Has Manager', 'CEO') as manager_status
  FROM employees
  WHERE employee_id = :1
```

### 6. Create Appropriate Indexes

```sql
-- Index frequently queried columns
CREATE INDEX idx_employees_dept ON employees(department_id);
CREATE INDEX idx_employees_name ON employees(UPPER(last_name), UPPER(first_name));
CREATE INDEX idx_orders_customer ON orders(customer_id, order_date);
```

### 7. Use Least Privilege

```sql
-- Create read-only user
CREATE USER app_readonly IDENTIFIED BY SecurePass123!;
GRANT CONNECT TO app_readonly;
GRANT SELECT ON employees TO app_readonly;
GRANT SELECT ON departments TO app_readonly;

-- Do NOT grant
-- GRANT DELETE, UPDATE, DROP TO app_readonly;  -- ❌ Too much access
```

---

## Troubleshooting

### Error: ORA-12154: TNS:could not resolve

**Cause:** TNS alias not found or tns_admin incorrect

**Solution:**
```bash
# Verify wallet path
ls -la /path/to/wallet
# Should contain: tnsnames.ora, sqlnet.ora, cwallet.sso

# Check TNS alias
cat /path/to/wallet/tnsnames.ora
```

### Error: ORA-01017: invalid username/password

**Solution:**
```bash
# Test connection
sqlplus username/password@connection_string

# For Autonomous DB, verify admin password in Cloud Console
```

### Error: ORA-00942: table or view does not exist

**Solution:**
```sql
-- Check accessible tables
SELECT table_name FROM user_tables;

-- Grant access if needed
GRANT SELECT ON schema.table_name TO username;
```

### Error: Connection timeout

**Solution:**
```bash
# Test network connectivity
telnet hostname 1521

# Check firewall rules
# For Autonomous DB, ensure wallet is properly configured
```

### Multi-Row Not Working

**Issue:** Only getting first row

**Solution:** Ensure client code iterates through all content items:
```python
# Correct
results = []
for item in content:
    results.append(json.loads(item["text"]))

# Wrong
result = json.loads(content[0]["text"])  # Only first row
```

---

## Testing

Run the comprehensive test suite:

```bash
cd tests/oracle

# Run all tests
go test -v

# Run specific test
go test -v -run TestOracleIntegration/03_SelectMultipleRows
go test -v -run TestOracleSQLParameterized/06_GetUsersInDept_MultiRow
```

Expected output:
```
--- PASS: TestOracleIntegration (0.57s)
    --- PASS: TestOracleIntegration/03_SelectMultipleRows
        ✅ Multi-row retrieval successful: 5 rows
--- PASS: TestOracleSQLParameterized (0.72s)
    --- PASS: TestOracleSQLParameterized/06_GetUsersInDept_MultiRow
        ✅ Get users in dept returned 4 rows
```

---

## Related Documentation

- [MCP Toolbox Documentation](https://googleapis.github.io/genai-toolbox/)
- [Oracle SQL Reference](https://docs.oracle.com/en/database/oracle/oracle-database/23/sqlrf/)
- [Oracle 23ai Vector Search](https://docs.oracle.com/en/database/oracle/oracle-database/23/vecse/)
- [Integration Tests](https://github.com/googleapis/genai-toolbox/tree/main/tests/oracle)

---

## Support

- [GitHub Issues](https://github.com/googleapis/genai-toolbox/issues)
- [Oracle Support](https://support.oracle.com)
- [Community Discussions](https://github.com/googleapis/genai-toolbox/discussions)

---

**Last Updated:** 2025-10-10  
**Version:** 1.0  
**Status:** Production Ready ✅
