# JBMDB (Joseph Barasa Migration Database Tool)

A powerful command-line database migration tool that supports both PostgreSQL and ScyllaDB. This tool helps you manage, create, and execute database migrations efficiently.

## Features

- Support for both PostgreSQL and ScyllaDB databases
- Simple command-line interface
- Create and manage migration files
- Apply and rollback migrations
- List migration status
- Fresh migrations (drop and recreate)
- Configuration management
- Interactive setup wizard
- Auto-update functionality

## Installation

1. Download the appropriate executable for your system from the releases page:
   - Linux: `jbmdb-linux`
   - Windows: `jbmdb-windows.exe`
   - macOS: `jbmdb-darwin`

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

The tool will create a `.env` file in your project directory on first run. You can also create it manually:

### PostgreSQL Configuration
```env
POSTGRES_HOST=your_host
POSTGRES_PORT=your_port
POSTGRES_USER=your_user
POSTGRES_PASSWORD=your_password
POSTGRES_DB=your_database
```

### ScyllaDB Configuration
```env
SCYLLA_HOSTS=your_hosts
SCYLLA_KEYSPACE=your_keyspace
SCYLLA_USERNAME=your_username
SCYLLA_PASSWORD=your_password
```

## Usage

### Configuration
```bash
# Set up migration paths and folder names
jbmdb config

# Check version information
jbmdb version
```

### Update
```bash
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
Note: Table names must be plural (e.g., users, posts, comments)

3. Run all pending migrations:
```bash
jbmdb postgres-migrate
```

4. Rollback the last migration:
```bash
jbmdb postgres-rollback
```

5. List all migrations:
```bash
jbmdb postgres-list
```

6. Fresh migrations (drops everything and remigrates):
```bash
jbmdb postgres-fresh
```

### ScyllaDB Commands

1. Initialize ScyllaDB configuration:
```bash
jbmdb scylla-init
```

2. Create a new migration:
```bash
jbmdb scylla-migration create_users_table
```
Note: Table names must be plural (e.g., users, posts, comments)

3. Run all pending migrations:
```bash
jbmdb scylla-migrate
```

4. Rollback the last migration:
```bash
jbmdb scylla-rollback
```

5. List all migrations:
```bash
jbmdb scylla-list
```

6. Fresh migrations:
```bash
jbmdb scylla-fresh
```

### Migration File Structure

#### PostgreSQL Migrations
```sql
-- up migration
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255)
);

-- down migration
DROP TABLE users;
```

#### ScyllaDB Migrations
```sql
-- up migration
CREATE TABLE users (
    id uuid PRIMARY KEY,
    name text
);

-- down migration
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
1. Ask for your database credentials
2. Create a `.env` file with your provided credentials
3. Create necessary migration directories
4. Set up initial configuration

## Testing

To test if the tool is working correctly:
```bash
make test
```

This will run through the basic commands to ensure everything is functioning properly.

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

Copyright (C) 2024 Joseph Barasa <jbarasa.com>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

## Author

Joseph Barasa
- Website: [jbarasa.com](https://jbarasa.com)
- Year: 2024
