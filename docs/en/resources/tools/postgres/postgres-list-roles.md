---
title: "postgres-list-roles"
type: docs
weight: 1
description: >
 The "postgres-list-roles" tool lists user-created roles in a Postgres database.
aliases:
- /resources/tools/postgres-list-roles
---

## About

The `postgres-list-roles` tool lists all the user-created roles in the instance, excluding system roles (like `cloudsql%` or `pg_%`). It provides details about each role's attributes and memberships. It's compatible with
any of the following sources:

- [alloydb-postgres](../../sources/alloydb-pg.md)
- [cloud-sql-postgres](../../sources/cloud-sql-pg.md)
- [postgres](../../sources/postgres.md)

`postgres-list-roles` lists detailed information as JSON for each role. The tool
takes the following input parameters:

- `role_name` (optional): A text to filter results by role name. Default: `""`
- `limit` (optional): The maximum number of roles to return. Default: `50`

## Example

```yaml
tools:
  list_indexes:
    kind: postgres-list-roles
    source: postgres-source
    description: |
      Lists all the user-created roles in the instance . It returns the role name,
      Object ID, the maximum number of concurrent connections the role can make,
      along with boolean indicators for: superuser status, privilege inheritance
      from member roles, ability to create roles, ability to create databases,
      ability to log in, replication privilege, and the ability to bypass
      row-level security, the password expiration timestamp, a list of direct
      members belonging to this role, and a list of other roles/groups that this
      role is a member of.
```

The response is a json array with the following elements:

```json
{
 "role_name": "Name of the role",
 "oid": "Object ID of the role",
 "connection_limit": "Maximum concurrent connections allowed (-1 for no limit)",
 "is_superuser": "Boolean, true if the role is a superuser",
 "inherits_privileges": "Boolean, true if the role inherits privileges of roles it is a member of",
 "can_create_roles": "Boolean, true if the role can create other roles",
 "can_create_db": "Boolean, true if the role can create databases",
 "can_login": "Boolean, true if the role can log in",
 "is_replication_role": "Boolean, true if this is a replication role",
 "bypass_rls": "Boolean, true if the role bypasses row-level security policies",
 "valid_until": "Timestamp until the password is valid (null if forever)",
 "direct_members": ["Array of role names that are direct members of this role"],
 "member_of": ["Array of role names that this role is a member of"]
}
```

## Reference

| **field**   | **type** | **required** | **description**                                      |
|-------------|:--------:|:------------:|------------------------------------------------------|
| kind        |  string  |     true     | Must be "postgres-list-roles".                     |
| source      |  string  |     true     | Name of the source the SQL should execute on.        |
| description |  string  |    false     | Description of the tool that is passed to the agent. |
