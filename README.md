# JBMDB (Joseph Barasa Migration Database Tool)

A powerful, cross-platform command-line database migration tool built in Go that supports PostgreSQL, MySQL/MariaDB, and Cassandra/ScyllaDB databases. This tool helps you manage, create, and execute database migrations efficiently with a focus on developer experience and reliability.

![Build Status](https://img.shields.io/github/workflow/status/jbarasa/jbmdb/build)
![Go Version](https://img.shields.io/github/go-mod/go-version/jbarasa/jbmdb)
![License](https://img.shields.io/github/license/jbarasa/jbmdb)

## Features

### Multi-Database Support
- PostgreSQL with full user management
- MySQL/MariaDB with InnoDB and UTF8MB4
- Cassandra/ScyllaDB with advanced clustering

### Migration Management
- Version-controlled migrations
- Forward and rollback support (single, multiple, all)
- Transaction safety for SQL databases
- Migration status tracking with colored output
- Fresh migrations (drop and recreate)

### Database Administration
- Create databases and keyspaces
- Manage users and privileges
- Multiple privilege levels (read, write, all, admin)
- Super user support for administrative tasks

### User-Friendly Interface
- Simple command-line interface
- Colored terminal output
- Clear error messages
- Progress indicators
- Detailed logging
- Auto-update functionality

## Installation

1. Download the appropriate executable for your system from the releases page:
   - Linux: `jbmdb-linux-amd64`
   - Windows: `jbmdb-windows-amd64.exe`
   - macOS: `jbmdb-darwin-amd64`

2. Rename the downloaded file to `jbmdb` (or `jbmdb.exe` on Windows)

3. Move the executable to a directory in your system PATH:
   ```bash
   # Linux/macOS
   sudo mv jbmdb /usr/local/bin/
   sudo chmod +x /usr/local/bin/jbmdb

   # Windows
   # Move jbmdb.exe to C:\Windows\System32\
   ```

## First Time Setup

When you run any command for the first time, the tool will:
1. Ask which databases you want to configure
2. Ask for your database credentials (with secure password input)
3. Create a `.jbmdb.conf` file with your provided credentials
4. Create necessary migration directories
5. Set up initial configuration

You can also configure each database using their respective init commands:
```bash
jbmdb postgres-init  # Configure PostgreSQL
jbmdb mysql-init     # Configure MySQL/MariaDB
jbmdb cql-init       # Configure Cassandra/ScyllaDB
```

## Configuration

The tool uses a `.jbmdb.conf` file in JSON format:

```json
{
  "postgres": {
    "migration_path": "migrations/postgres",
    "sql_folder": "sql",
    "host": "localhost",
    "port": "5432",
    "user": "myapp",
    "password": "secret",
    "dbname": "myapp_db",
    "super_user": "postgres",
    "super_pass": "postgres"
  },
  "mysql": {
    "migration_path": "migrations/mysql",
    "sql_folder": "sql",
    "host": "localhost",
    "port": "3306",
    "user": "myapp",
    "password": "secret",
    "dbname": "myapp_db",
    "super_user": "root",
    "super_pass": "root"
  },
  "cql": {
    "migration_path": "migrations/cql",
    "cql_folder": "cql",
    "hosts": ["localhost"],
    "port": 9042,
    "user": "myapp",
    "password": "secret",
    "keyspace": "myapp_space",
    "super_user": "cassandra",
    "super_pass": "cassandra",
    "datacenter": "dc1",
    "consistency": "quorum"
  }
}
```

## Usage

### Global Commands
```bash
jbmdb config   # Set up migration paths
jbmdb version  # Check version
jbmdb update   # Check for updates
```

### Database Operations

The following commands work for all databases. Replace `<db>` with:
- `postgres` for PostgreSQL
- `mysql` for MySQL/MariaDB
- `cql` for Cassandra/ScyllaDB

```bash
# Migration Commands
jbmdb <db>-migration create_users_table  # Create new migration
jbmdb <db>-migrate                       # Run pending migrations
jbmdb <db>-rollback                      # Rollback last migration
jbmdb <db>-rollback:all                  # Rollback all migrations
jbmdb <db>-rollback:3                    # Rollback last 3 migrations
jbmdb <db>-list                          # List all migrations
jbmdb <db>-fresh                         # Drop and remigrate

# User Management
jbmdb <db>-create-user:read              # Read-only access
jbmdb <db>-create-user:write             # Read/write access
jbmdb <db>-create-user:all               # All privileges
jbmdb <db>-create-user:admin             # Admin privileges

# Database Creation
jbmdb postgres-create-db                 # Create PostgreSQL database
jbmdb mysql-create-db                    # Create MySQL database
jbmdb cql-create-keyspace:SimpleStrategy:3  # Create Cassandra keyspace
```

### Migration File Structure

#### SQL Databases (PostgreSQL, MySQL)
```sql
-- Up Migration
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Down Migration
DROP TABLE users;
```

#### CQL (Cassandra/ScyllaDB)
```sql
-- Up Migration
CREATE TABLE users (
    id uuid PRIMARY KEY,
    name text,
    created_at timestamp,
    updated_at timestamp
) WITH bloom_filter_fp_chance = 0.01
    AND caching = {'keys': 'ALL', 'rows_per_partition': 'ALL'}
    AND compaction = {'class': 'SizeTieredCompactionStrategy'};

-- Down Migration
DROP TABLE users;
```

### Migration Name Rules
1. Must start with `create_`
2. Must end with `_table`
3. Single table names must be plural:
   - `create_users_table`
   - `create_user_comments_table`

### Cassandra/ScyllaDB Specific Features

#### Replication Strategies
- **SimpleStrategy**: For single-datacenter deployments
  ```bash
  jbmdb cql-create-keyspace:SimpleStrategy:3  # RF=3
  ```

- **NetworkTopologyStrategy**: For multi-datacenter deployments
  ```bash
  jbmdb cql-create-keyspace:NetworkTopologyStrategy:2  # RF=2 per DC
  ```

## Version History

### v2.0.0 (2024-01-13)
- Added MySQL/MariaDB support with InnoDB and UTF8MB4
- Enhanced Cassandra/ScyllaDB support:
  - Multi-node cluster configuration
  - Flexible replication strategies
  - Custom consistency levels
  - Datacenter-aware operations
- Improved user management and privileges
- Better error handling and colored output
- Comprehensive test suite
- Updated documentation and examples

### v1.0.0 (2024-01-01)
- Initial release with PostgreSQL and ScyllaDB support

## About the Author

Hi, I'm Joseph Barasa, a passionate Go developer focused on building high-performance, scalable systems.

- Portfolio: [jbarasa.com](https://jbarasa.com)
- Email: jbarasa.ke@gmail.com
- Specialized in: Distributed systems, microservices, and high-performance applications
- Open to exciting Golang projects and opportunities

## License

This project is licensed under the Mozilla Public License Version 2.0 - see the [LICENSE](LICENSE) file for details.

Copyright (C) 2024 Joseph Barasa
