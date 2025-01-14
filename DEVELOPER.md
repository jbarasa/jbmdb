# JBMDB Developer Documentation

## Table of Contents

1. [Introduction](#introduction)
   - [Purpose](#purpose)
   - [Key Features](#key-features)
   - [Technology Stack](#technology-stack)

2. [Project Structure](#project-structure)
   - [Directory Layout](#directory-layout)
   - [Key Files](#key-files)
   - [Package Organization](#package-organization)

3. [Core Components](#core-components)
   - [Configuration System](#configuration-system)
   - [Command Line Interface](#command-line-interface)
   - [Migration Engine](#migration-engine)
   - [Database Handlers](#database-handlers)

4. [Code Architecture](#code-architecture)
   - [Design Patterns](#design-patterns)
   - [Error Handling](#error-handling)
   - [Configuration Management](#configuration-management)
   - [Database Connections](#database-connections)

5. [Migration System](#migration-system)
   - [Migration File Structure](#migration-file-structure)
   - [Version Control](#version-control)
   - [Rollback Support](#rollback-support)
   - [Transaction Management](#transaction-management)

6. [Database Support](#database-support)
   - [PostgreSQL Integration](#postgresql-integration)
   - [MySQL/MariaDB Integration](#mysqlmariadb-integration)
   - [Cassandra/ScyllaDB Integration](#cassandrascylladb-integration)
   - [Adding New Databases](#adding-new-databases)
   - [Database Management](#database-management)

7. [User Interface](#user-interface)
   - [Command Structure](#command-structure)
   - [Terminal Output](#terminal-output)
   - [Color Coding](#color-coding)
   - [Input Handling](#input-handling)

8. [Development Guide](#development-guide)
   - [Setting Up Development Environment](#setting-up-development-environment)
   - [Building the Project](#building-the-project)
   - [Running Tests](#running-tests)
   - [Making Changes](#making-changes)

9. [Testing Strategy](#testing-strategy)
   - [Unit Tests](#unit-tests)
   - [Integration Tests](#integration-tests)
   - [Test Data Management](#test-data-management)
   - [Mocking Database Connections](#mocking-database-connections)

10. [Common Tasks](#common-tasks)
    - [Adding New Commands](#adding-new-commands)
    - [Modifying Migration Templates](#modifying-migration-templates)
    - [Updating Configuration Options](#updating-configuration-options)
    - [Adding Database Support](#adding-database-support)

11. [Best Practices](#best-practices)
    - [Code Style](#code-style)
    - [Error Handling](#error-handling-1)
    - [Security Considerations](#security-considerations)
    - [Performance Optimization](#performance-optimization)

12. [Troubleshooting](#troubleshooting)
    - [Common Issues](#common-issues)
    - [Debugging Techniques](#debugging-techniques)
    - [Logging System](#logging-system)
    - [Error Messages](#error-messages)

13. [Contributing](#contributing)
    - [Development Workflow](#development-workflow)
    - [Pull Request Process](#pull-request-process)
    - [Code Review Guidelines](#code-review-guidelines)
    - [Documentation Updates](#documentation-updates)

14. [API Reference](#api-reference)
    - [Public Functions](#public-functions)
    - [Configuration Types](#configuration-types)
    - [Database Interfaces](#database-interfaces)
    - [Utility Functions](#utility-functions)

15. [About the Author and Development Philosophy](#about-the-author-and-development-philosophy)
    - [The Developer](#the-developer)
    - [AI-Enhanced Development](#ai-enhanced-development)
    - [Cross-Platform Development](#cross-platform-development)

## Introduction

### Purpose
JBMDB is a versatile database migration tool designed to handle multiple database types through a unified interface. It provides developers with a consistent way to manage database schema changes across different projects and database systems.

### Key Features
- Multi-database support
- Version-controlled migrations
- Rollback capability
- Transaction support
- Configuration management
- Colored terminal output

### Technology Stack
- Go 1.21+
- Database Drivers:
  - pgx (PostgreSQL)
  - go-sql-driver (MySQL)
  - gocql (Cassandra)

## Project Structure

### Directory Layout
```
jbmdb/
├── migrations/           # Core migration functionality
│   ├── config/          # Configuration management
│   │   └── config.go    # Configuration types and functions
│   ├── postgres/        # PostgreSQL-specific code
│   │   └── migrations.go # PostgreSQL migration handler
│   ├── mysql/          # MySQL/MariaDB-specific code
│   │   └── migrations.go # MySQL migration handler
│   └── cql/            # Cassandra/ScyllaDB-specific code
│       └── migrations.go # CQL migration handler
├── build/              # Build artifacts
├── sql/               # SQL migration templates
└── main.go            # CLI entry point
```

### Key Files
- `main.go`: Entry point and command routing
- `config.go`: Configuration management
- `migrations.go`: Database-specific migration handlers

### Package Organization
- `config`: Configuration types and management
- `postgres`, `mysql`, `cql`: Database-specific implementations
- `update`: Auto-update functionality

## Core Components

### Configuration System

#### Configuration Types
```go
type Config interface {
    // Base configuration interface
}

type PostgresConfig struct {
    MigrationPath string
    SQLFolder     string
    Host          string
    Port          string
    User          string
    Password      string
    DBName        string
}

// Similar structures for MySQL and CQL
```

#### Key Functions
```go
func LoadConfig[T Config](configType string) (*T, error)
func SaveConfig[T Config](config T, configType string) error
```

### Command Line Interface

#### Command Structure
Commands follow the pattern: `<database>-<action>`

Examples:
```bash
jbmdb postgres-migration create_users_table
jbmdb mysql-rollback 1
jbmdb cql-list
```

#### Handler Functions
```go
func handlePostgres(action string)
func handleMySQL(action string)
func handleScylla(action string)
```

### Migration Engine

#### Migration File Structure
```sql
-- Up migration
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255)
);

-- Down migration
DROP TABLE users;
```

#### Version Control
- Migrations are timestamped
- Sequential execution
- Rollback support

## Development Guide

### Setting Up Development Environment

1. Install Go 1.21+
2. Clone repository
3. Install dependencies:
```bash
go mod download
```

### Building the Project
```bash
make build
# or for specific platform
make build-linux
make build-darwin
make build-windows
```

### Running Tests
```bash
make test
```

## Testing Strategy

### Unit Tests
- Test individual components
- Mock database connections
- Verify configuration handling

### Integration Tests
- Test complete workflows
- Verify database interactions
- Check migration processes

## Common Tasks

### Adding New Commands

1. Update command handling:
```go
func handleNewCommand(action string) {
    // Implementation
}
```

2. Add to main switch:
```go
switch command {
case "new-command":
    handleNewCommand(action)
}
```

### Modifying Migration Templates

1. Create template in `sql/`:
```sql
-- template.sql
CREATE TABLE {{.TableName}} (
    id {{.IDType}} PRIMARY KEY
);
```

2. Update template processing:
```go
func processTemplate(name string, data TemplateData) error {
    // Implementation
}
```

## Best Practices

### Code Style
- Follow Go standards
- Use meaningful names
- Document public APIs
- Keep functions focused

### Error Handling
- Use descriptive errors
- Wrap database errors
- Provide context
- Use color coding

### Security
- Mask passwords
- Validate input
- Use prepared statements
- Handle sensitive data

## API Reference

### Public Functions

#### Configuration
```go
func LoadConfig[T Config](configType string) (*T, error)
func SaveConfig[T Config](config T, configType string) error
```

#### Migration
```go
func CreateMigration(name string) error
func Migrate(db *sql.DB) error
func RollbackMigrations(db *sql.DB, steps int) error
```

#### Utility
```go
func SetMigrationPath(path string)
func ValidateMigrationName(name string)
func PrintColoredOutput(text string, color string)
```

## Database Support

### PostgreSQL Integration

#### Connection Management
- Uses `pgx/v5` for direct connections
- Connection pooling for migrations
- Super user support for administrative tasks

#### Database Management
```go
// Create database if not exists
func CreateDatabase(pgConfig *config.PostgresConfig) error

// Create user with privileges
func CreateUser(pgConfig *config.PostgresConfig, privileges string) error
```

#### Privilege Levels
- `read`: SELECT privileges
- `write`: SELECT, INSERT, UPDATE, DELETE
- `all`: All privileges on database
- `admin`: All privileges with GRANT OPTION

### MySQL/MariaDB Integration

#### Connection Management
- Uses `go-sql-driver/mysql`
- Direct connections for administrative tasks
- Connection pooling for migrations

#### Database Management
```go
// Create database if not exists
func CreateDatabase(myConfig *config.MySQLConfig) error

// Create user with privileges
func CreateUser(myConfig *config.MySQLConfig, privileges string) error
```

### Cassandra/ScyllaDB Integration

#### Connection Management
- Uses `gocql` for cluster connections
- Session management for operations
- Super user support for administrative tasks

#### Keyspace Management
```go
// Create keyspace with replication strategy
func CreateKeyspace(cqlConfig *config.ScyllaConfig, replicationStrategy string, replicationFactor int) error

// Create user with privileges
func CreateUser(cqlConfig *config.ScyllaConfig, privileges string) error
```

#### Replication Strategies
- **SimpleStrategy**
  - Single datacenter deployments
  - Simple replication factor
  - Example: `RF=3` means 3 copies of data

- **NetworkTopologyStrategy**
  - Multi-datacenter deployments
  - Per-datacenter replication factors
  - Better control over data placement

#### Privilege Levels
- `read`: SELECT privileges
- `write`: SELECT, INSERT, UPDATE, DELETE
- `all`: All privileges on database
- `admin`: All privileges with GRANT OPTION

### Configuration Management

#### Super User Credentials
```go
type PostgresConfig struct {
    // ... existing fields ...
    SuperUser string // Super user for admin operations
    SuperPass string // Super user password
}

type MySQLConfig struct {
    // ... existing fields ...
    SuperUser string // Super user for admin operations
    SuperPass string // Super user password
}
```

### Command Line Interface

#### Database Management Commands
```bash
# PostgreSQL commands
jbmdb postgres-create-db           # Create database
jbmdb postgres-create-user:read    # Create read-only user
jbmdb postgres-create-user:write   # Create read/write user
jbmdb postgres-create-user:all     # Create user with all privileges
jbmdb postgres-create-user:admin   # Create admin user

# MySQL commands
jbmdb mysql-create-db             # Create database
jbmdb mysql-create-user:read      # Create read-only user
jbmdb mysql-create-user:write     # Create read/write user
jbmdb mysql-create-user:all       # Create user with all privileges
jbmdb mysql-create-user:admin     # Create admin user
```

## Contributing

### Development Workflow

1. Fork repository
2. Create feature branch
3. Make changes
4. Add tests
5. Update documentation
6. Submit pull request

### Code Review Guidelines

- Follow Go best practices
- Maintain test coverage
- Update documentation
- Use clear commit messages

## About the Author and Development Philosophy

### The Developer

👋 Hi, I'm Joseph Barasa, a passionate Go developer focused on building high-performance, scalable systems. I specialize in creating developer-friendly tools that make software engineering more efficient and enjoyable.

- 🌐 Portfolio: [jbarasa.com](https://jbarasa.com)
- 📧 Email: jbarasa.ke@gmail.com
- 💼 Open to Work: Looking for exciting Golang projects and opportunities
- 🚀 Specialized in: Distributed systems, microservices, and high-performance applications

### AI-Enhanced Development

JBMDB leverages modern AI tools and practices to enhance development efficiency:

1. **AI-Assisted Code Generation**:
   - Smart template generation for migrations
   - Intelligent error message suggestions
   - Context-aware code completion

2. **Development Workflow**:
   - AI-powered code review suggestions
   - Automated documentation updates
   - Smart test case generation

3. **Best Practices**:
   - AI-assisted code optimization
   - Pattern recognition for common issues
   - Automated style conformance

4. **Documentation**:
   - AI-enhanced documentation generation
   - Smart context gathering
   - Automated example generation

### Cross-Platform Development

JBMDB is designed to work seamlessly across different platforms:

1. **Operating Systems**:
   - Linux (all major distributions)
   - macOS (Intel and Apple Silicon)
   - Windows (native and WSL)

2. **Build System**:
   - Platform-specific optimizations
   - Automated cross-compilation
   - Consistent behavior across systems

3. **Testing**:
   - Cross-platform test suite
   - Environment-specific checks
   - Automated CI/CD pipeline

## Version Control

The tool uses semantic versioning:
- MAJOR.MINOR.PATCH
- Set during build with `-X main.Version`
