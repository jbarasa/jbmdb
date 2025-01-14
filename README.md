# JBMDB (Joseph Barasa Migration Database Tool)

A powerful, cross-platform command-line database migration tool built in Go that supports PostgreSQL, MySQL/MariaDB, and Cassandra/ScyllaDB databases. This tool helps you manage, create, and execute database migrations efficiently with a focus on developer experience and reliability.

![Build Status](https://img.shields.io/github/workflow/status/jbarasa/jbmdb/build)
![Go Version](https://img.shields.io/github/go-mod/go-version/jbarasa/jbmdb)
![License](https://img.shields.io/github/license/jbarasa/jbmdb)

## Features

- Support for multiple databases:
  - PostgreSQL
  - MySQL/MariaDB
  - Cassandra/ScyllaDB (CQL)
- Simple command-line interface
- Create and manage migration files
- Apply and rollback migrations (single, multiple, or all)
- List migration status with colored output
- Fresh migrations (drop and recreate)
- Configuration management with wizard
- Auto-update functionality
- Table name validation and conventions
- Transaction support for SQL databases
- Colored terminal output for better readability
- **Multi-Database Support**
  - PostgreSQL
  - MySQL/MariaDB
  - Cassandra/ScyllaDB
- **Migration Management**
  - Version-controlled migrations
  - Forward and rollback support
  - Transaction safety
  - Migration status tracking
- **Database Administration**
  - Create databases
  - Manage users and privileges
  - Multiple privilege levels (read, write, all, admin)
  - Super user support for administrative tasks
- **User-Friendly Interface**
  - Colored terminal output
  - Clear error messages
  - Progress indicators
  - Detailed logging

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

## Database Configuration

The tool will create a `.jbmdb.conf` file in your project directory on first run. You can also configure each database using their respective init commands.

### Configuration File Structure
The configuration is stored in JSON format in `.jbmdb.conf`:

```json
{
  "postgres": {
    "migration_path": "migrations/postgres",  // default: migrations/postgres
    "sql_folder": "sql",                     // SQL files location
    "host": "localhost",                     // default: localhost
    "port": "5432",                          // default: 5432
    "user": "postgres",                      // default: postgres
    "password": "your_password",             // optional
    "dbname": "your_database"                // default: postgres
  },
  "mysql": {
    "migration_path": "migrations/mysql",     // default: migrations/mysql
    "sql_folder": "sql",                     // SQL files location
    "host": "localhost",                     // default: localhost
    "port": "3306",                          // default: 3306
    "user": "root",                          // default: root
    "password": "your_password",             // optional
    "dbname": "your_database"                // default: mysql
  },
  "cql": {
    "migration_path": "migrations/cql",       // default: migrations/cql
    "cql_folder": "cql",                     // CQL files location
    "hosts": ["localhost"],                  // default: ["localhost"]
    "keyspace": "your_keyspace",             // default: system
    "user": "your_username",                 // optional
    "password": "your_password"              // optional
  }
}
```

You don't need to edit this file manually. Use the init commands to configure each database:
```bash
jbmdb postgres-init  # Configure PostgreSQL
jbmdb mysql-init     # Configure MySQL/MariaDB
jbmdb cql-init       # Configure Cassandra/ScyllaDB
```

## Usage

### Global Commands
```bash
# Set up migration paths and folder names
jbmdb config

# Check version information
jbmdb version

# Check for and install updates
jbmdb update
```

### PostgreSQL Commands

1. Initialize PostgreSQL configuration:
```bash
jbmdb postgres-init
```

2. Create a new migration:
```bash
jbmdb postgres-migration create_users_table
```

3. Run all pending migrations:
```bash
jbmdb postgres-migrate
```

4. Rollback migrations:
```bash
jbmdb postgres-rollback        # Rollback last migration
jbmdb postgres-rollback:all    # Rollback all migrations
jbmdb postgres-rollback:3      # Rollback last 3 migrations
```

5. List all migrations:
```bash
jbmdb postgres-list
```

6. Fresh migrations (drops everything and remigrates):
```bash
jbmdb postgres-fresh
```

### MySQL/MariaDB Commands

1. Initialize MySQL configuration:
```bash
jbmdb mysql-init
```

2. Create a new migration:
```bash
jbmdb mysql-migration create_users_table
```

3. Run all pending migrations:
```bash
jbmdb mysql-migrate
```

4. Rollback migrations:
```bash
jbmdb mysql-rollback        # Rollback last migration
jbmdb mysql-rollback:all    # Rollback all migrations
jbmdb mysql-rollback:3      # Rollback last 3 migrations
```

5. List all migrations:
```bash
jbmdb mysql-list
```

6. Fresh migrations:
```bash
jbmdb mysql-fresh
```

### CQL Commands (Cassandra/ScyllaDB)

1. Initialize CQL configuration:
```bash
jbmdb cql-init
```

2. Create a new migration:
```bash
jbmdb cql-migration create_users_table
```

3. Run all pending migrations:
```bash
jbmdb cql-migrate
```

4. Rollback migrations:
```bash
jbmdb cql-rollback        # Rollback last migration
jbmdb cql-rollback:all    # Rollback all migrations
jbmdb cql-rollback:3      # Rollback last 3 migrations
```

5. List all migrations:
```bash
jbmdb cql-list
```

6. Fresh migrations:
```bash
jbmdb cql-fresh
```

### Migration File Structure

#### PostgreSQL Migrations
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

#### MySQL/MariaDB Migrations
```sql
-- Up Migration
CREATE TABLE users (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Down Migration
DROP TABLE users;
```

#### CQL Migrations (Cassandra/ScyllaDB)
```sql
-- Up Migration
CREATE TABLE users (
    id uuid PRIMARY KEY,
    name text,
    created_at timestamp,
    updated_at timestamp
) WITH bloom_filter_fp_chance = 0.01
    AND caching = {'keys': 'ALL', 'rows_per_partition': 'ALL'}
    AND comment = ''
    AND compaction = {'class': 'SizeTieredCompactionStrategy'}
    AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'}
    AND crc_check_chance = 1.0;

-- Down Migration
DROP TABLE users;
```

### Migration Name Rules

1. Must start with `create_`
2. Must end with `_table`
3. Single table names must be plural:
   - ✅ `create_users_table`
   - ❌ `create_user_table`
4. In relation tables, names after the first word should be plural:
   - ✅ `create_user_comments_table`
   - ❌ `create_user_comment_table`

## First Time Setup

When you run any command for the first time, the tool will:
1. Ask which databases you want to configure (PostgreSQL, MySQL/MariaDB, Cassandra/ScyllaDB, or all)
2. Ask for your database credentials (with secure password input)
3. Create a `.jbmdb.conf` file with your provided credentials
4. Create necessary migration directories
5. Set up initial configuration

## Make Commands

You can also use make commands for easier access:
```bash
make postgres-init              # Initialize PostgreSQL configuration
make postgres-migration name=create_users_table  # Create a new migration
make postgres-migrate          # Run pending migrations
make postgres-rollback         # Rollback last migration
make postgres-fresh           # Fresh migration
make postgres-list           # List migrations
```
(Replace postgres with mysql or cql for other databases)

## Database Management

### Creating Databases and Keyspaces

```bash
# PostgreSQL
jbmdb postgres-create-db

# MySQL
jbmdb mysql-create-db

# Cassandra/ScyllaDB
jbmdb cql-create-keyspace:SimpleStrategy:3           # Single datacenter with RF=3
jbmdb cql-create-keyspace:NetworkTopologyStrategy:2  # Multi-datacenter with RF=2
```

### Managing Users

```bash
# PostgreSQL users
jbmdb postgres-create-user:read    # Read-only access
jbmdb postgres-create-user:write   # Read/write access
jbmdb postgres-create-user:all     # All privileges
jbmdb postgres-create-user:admin   # Admin privileges

# MySQL users
jbmdb mysql-create-user:read       # Read-only access
jbmdb mysql-create-user:write      # Read/write access
jbmdb mysql-create-user:all        # All privileges
jbmdb mysql-create-user:admin      # Admin privileges

# Cassandra/ScyllaDB users
jbmdb cql-create-user:read        # SELECT on keyspace
jbmdb cql-create-user:write       # SELECT, MODIFY on keyspace
jbmdb cql-create-user:all         # ALL PERMISSIONS on keyspace
jbmdb cql-create-user:admin       # ALL PERMISSIONS on ALL KEYSPACES
```

## Configuration

JBMDB uses a `.jbmdb.conf` file for configuration. Example configuration:

```json
{
  "postgres": {
    "host": "localhost",
    "port": "5432",
    "user": "myapp",
    "password": "secret",
    "dbname": "myapp_db",
    "super_user": "postgres",
    "super_pass": "postgres"
  },
  "mysql": {
    "host": "localhost",
    "port": "3306",
    "user": "myapp",
    "password": "secret",
    "dbname": "myapp_db",
    "super_user": "root",
    "super_pass": "root"
  },
  "cql": {
    "host": "localhost",
    "port": 9042,
    "user": "myapp",
    "password": "secret",
    "keyspace": "myapp_space",
    "super_user": "cassandra",
    "super_pass": "cassandra",
    "datacenter": "datacenter1"
  }
}
```

### Replication Strategies (Cassandra/ScyllaDB)

- **SimpleStrategy**: Use for single-datacenter deployments
  ```bash
  jbmdb cql-create-keyspace:SimpleStrategy:3  # RF=3
  ```

- **NetworkTopologyStrategy**: Use for multi-datacenter deployments
  ```bash
  jbmdb cql-create-keyspace:NetworkTopologyStrategy:2  # RF=2 per datacenter
  ```

The replication factor (RF) determines how many copies of each piece of data should be kept. Higher values provide better availability and fault tolerance but require more storage.

## Version History

### v2.0.0 (2024-01-13)
- Added MySQL/MariaDB support with InnoDB and UTF8MB4
- Unified CQL support for both Cassandra and ScyllaDB
- Improved table name validation with better error messages
- Added transaction support for SQL databases
- Unified configuration system with `.jbmdb.conf`
- Better error handling and colored output
- Improved build system with platform-specific binaries
- Added rollback options (single, multiple, all)
- Enhanced Cassandra/ScyllaDB support:
  - Multi-node cluster configuration
  - Flexible replication strategies
  - Custom consistency levels
  - Datacenter-aware operations

### v1.0.0 (2024-01-01)
- Initial release with PostgreSQL and ScyllaDB support

## License

This project is licensed under the Mozilla Public License Version 2.0 - see the [LICENSE](LICENSE) file for details.

Copyright (C) 2024 Joseph Barasa <jbarasa.com>

## About the Author

Hi, I'm Joseph Barasa, a passionate Go developer focused on building high-performance, scalable systems.

- Portfolio: [jbarasa.com](https://jbarasa.com)
- Email: jbarasa.ke@gmail.com
- Open to Work: Looking for exciting Golang projects and opportunities
- AI Expertise: Experienced in leveraging AI tools to enhance developer productivity
- Specialized in: Distributed systems, microservices, and high-performance applications

### Work With Me

I'm currently available for Go/Golang positions and projects. I believe in practical problem-solving and writing clean, efficient code. I have extensive experience in:
- Building robust, production-ready systems
- Implementing efficient database solutions
- Creating developer-friendly tools
- Leveraging AI for enhanced productivity
- Writing clean, maintainable code

Let's connect! Reach out via:
- Email: jbarasa.ke@gmail.com
- Website: [jbarasa.com](https://jbarasa.com)
