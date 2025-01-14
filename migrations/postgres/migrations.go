// Developer: Joseph Barasa
// Year: 2024
// Developer's Website: https://jbarasa.com
// License: Jbarasa INC
//
// Package migrations provides functionality to manage database migrations
// for a PostgreSQL database using the pgx/v5 library.
//
// The main aim of this package is to facilitate database schema migrations
// in a PostgreSQL database. It allows creating migration files, loading them,
// applying them to the database, rolling them back, and managing migration
// history using a dedicated table.
//
// This package includes functions to:
// - Create new migration files with up and down SQL scripts.
// - Create new many-to-many relation migration files with up and down SQL scripts.
// - Load existing migration files from a specified directory.
// - Parse migration file names to extract version and name information.
// - Apply migrations to the database using transactions.
// - Rollback the last applied migration.
// - Handle fresh migrations by dropping all tables and reapplying migrations.
// - List migration history with their status whether pending or applied.
//
// Each function is designed to handle errors gracefully and provides detailed
// logging and error messages to aid in debugging and operational monitoring.
//
// The package utilizes context.Background() for database operations, ensuring
// that each operation is properly scoped and can be canceled if necessary.
//
// This code is intended to be reusable and adaptable for various PostgreSQL
// database applications, providing a structured approach to managing database
// schema changes over time.
//
// The developer, Joseph Barasa, ensures the robustness and performance of
// database migrations while adhering to best practices for reliability and
// maintainability in database management.
// Package migrations provides functions to manage database migrations for PostgreSQL.
package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jbarasa/jbmdb/migrations/config"
)

// Migration represents a database migration with its version, name, SQL scripts for
// applying and rolling back the migration.
type Migration struct {
	Version int64  // The version of the migration.
	Name    string // The name of the migration.
	UpSQL   string // SQL script for applying the migration.
	DownSQL string // SQL script for rolling back the migration.
}

// Path to the migration files.
var migrationPath string

// SetMigrationPath sets the path for migration files
func SetMigrationPath(path string) {
	migrationPath = path
}

// Color constants for terminal output
const (
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[37m"
	ColorBold   = "\033[1m"
	ColorReset  = "\033[0m"
	ColorYellow = "\033[33m"
)

// extractTableName extracts the table name from the migration name
func extractTableName(name string) string {
	// Remove common prefixes like "create_" or "add_" and suffixes like "_table"
	name = strings.TrimPrefix(name, "create_")
	name = strings.TrimPrefix(name, "add_")
	name = strings.TrimSuffix(name, "_table")

	// Convert to snake_case if it's in CamelCase
	name = camelToSnakeCase(name)

	// return the table name
	return name
}

// camelToSnakeCase converts a string from CamelCase to snake_case
func camelToSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			result.WriteByte('_')
		}
		result.WriteRune(unicode.ToLower(r))
	}
	return result.String()
}

// checkDuplicateTableName checks if a migration with the same table name already exists
func checkDuplicateTableName(newTableName string) error {
	migrations, err := loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	for _, migration := range migrations {
		existingTableName := extractTableName(migration.Name)
		if strings.EqualFold(existingTableName, newTableName) {
			return fmt.Errorf("%stable name '%s' already exists in migration '%s'%s",
				ColorRed, newTableName, migration.Name, ColorReset)
		}
	}
	return nil
}

// CreateMigration creates new migration file with the given name and current timestamp.
func CreateMigration(name string) error {
	// Extract table name from migration name
	tableName := extractTableName(name)

	// Check for duplicate table names
	if err := checkDuplicateTableName(tableName); err != nil {
		return err
	}

	// Generate a timestamp in the format YYYYMMDDHHMMSS.
	timestamp := time.Now().Format("20060102150405")
	// Combine the timestamp and name to create a unique filename.
	filename := fmt.Sprintf("%s_%s.sql", timestamp, name)

	// Write placeholder content to the up and down migration file
	content := fmt.Sprintf(`-- Up Migration
----------------------- Write your up migration here ----------------------------

CREATE TABLE IF NOT EXISTS %s (
    id BIGSERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
	updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL
);


-- Down Migration
----------------------- Write your down migration here ----------------------------

DROP TABLE IF EXISTS %s;`, strings.ToLower(tableName), strings.ToLower(tableName))

	// Create the migration file in the SQL folder within the migration path
	sqlPath := filepath.Join(migrationPath, "sql")
	if err := os.MkdirAll(sqlPath, 0755); err != nil {
		return fmt.Errorf("failed to create SQL directory: %w", err)
	}

	// Write the up and down migration file in the SQL folder
	filePath := filepath.Join(sqlPath, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create migration file: %w", err)
	}

	// Print the paths of the created migration files.
	fmt.Printf("%sCreated migration file: %s%s\n", ColorGreen, filePath, ColorReset)
	return nil
}

// parseInt converts a string to an integer.
func parseInt(s string) int64 {
	var result int64
	fmt.Sscanf(s, "%d", &result)
	return result
}

// loadMigrations loads all migration files from the migration directory and returns a slice of Migration structs.
func loadMigrations() ([]Migration, error) {
	// Get the SQL directory path
	sqlPath := filepath.Join(migrationPath, "sql")

	// Read the migration directory.
	files, err := os.ReadDir(sqlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration directory: %w", err)
	}

	var migrations []Migration // Slice to hold the loaded migrations.
	for _, file := range files {
		// Process only .sql files.
		if filepath.Ext(file.Name()) == ".sql" {
			// Split the filename by underscores.
			parts := strings.Split(file.Name(), "_")
			if len(parts) < 2 {
				continue // Skip files that do not have at least a version and name part.
			}

			// Get the version from the first part of the filename.
			version := parts[0]
			// Get the name from the remaining parts of the filename.
			name := strings.TrimSuffix(strings.Join(parts[1:], "_"), filepath.Ext(file.Name()))

			// Read the content of the migration file.
			content, err := os.ReadFile(filepath.Join(sqlPath, file.Name()))
			if err != nil {
				return nil, fmt.Errorf("failed to read migration file %s: %w", file.Name(), err)
			}

			upDown := strings.Split(string(content), "-- Down Migration")
			if len(upDown) != 2 {
				return nil, fmt.Errorf("invalid migration format in file %s", file.Name())
			}

			up := strings.TrimSpace(strings.TrimPrefix(upDown[0], "-- Up Migration"))
			down := strings.TrimSpace(upDown[1])

			// Create a new Migration struct.
			migrations = append(migrations, Migration{
				Version: parseInt(version),
				Name:    name,
				UpSQL:   up,
				DownSQL: down,
			})
		}
	}

	// Sort the migrations by version.
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// Migrate applies all pending migrations to the database.
func Migrate(db *pgxpool.Pool) error {
	// Create the migrations table if it doesn't exist.
	if err := createMigrationsTable(db); err != nil {
		return err
	}

	// Load all migrations from the migration directory.
	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	// Apply each migration in sequence.
	for _, migration := range migrations {
		if err := applyMigration(db, migration); err != nil {
			return err
		}
	}

	return nil
}

// RollbackLast rolls back the most recently applied migration.
func RollbackLast(db *pgxpool.Pool) error {
	// Get the version of the latest applied migration.
	latestMigration, err := getLatestMigration(db)
	if err != nil {
		return err
	}

	// If there are no migrations to roll back, print a message and return.
	if latestMigration == 0 {
		fmt.Println("No migrations to rollback")
		return nil
	}

	// Load all migrations from the migration directory.
	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	// Find the migration to roll back.
	var migrationToRollback Migration
	for _, m := range migrations {
		if m.Version == latestMigration {
			migrationToRollback = m
			break
		}
	}

	// If the migration to roll back is not found, return an error.
	if migrationToRollback.Version == 0 {
		return fmt.Errorf("migration %d not found", latestMigration)
	}

	// Roll back the migration.
	if err := rollbackMigration(db, migrationToRollback); err != nil {
		return err
	}

	// Print a message indicating the migration has been rolled back.
	fmt.Printf("Rolled back migration: %d_%s\n", migrationToRollback.Version, migrationToRollback.Name)
	return nil
}

// RollbackSteps rolls back a specified number of migrations
func RollbackSteps(db *pgxpool.Pool, steps int) error {
	// Get all applied migrations
	appliedMigrations, err := getAppliedMigrations(db)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	if len(appliedMigrations) == 0 {
		fmt.Printf("%sNo migrations to rollback%s\n", ColorYellow, ColorReset)
		return nil
	}

	// Sort migrations by version in descending order
	sort.Slice(appliedMigrations, func(i, j int) bool {
		return appliedMigrations[i].Version > appliedMigrations[j].Version
	})

	// Limit steps to available migrations
	if steps > len(appliedMigrations) {
		steps = len(appliedMigrations)
		fmt.Printf("%sNote: Only %d migrations available to rollback%s\n",
			ColorYellow, steps, ColorReset)
	}

	// Rollback each migration
	for i := 0; i < steps; i++ {
		migration := appliedMigrations[i]
		fmt.Printf("%s[ROLLBACK]%s Rolling back migration %s%d_%s%s... ",
			ColorBlue, ColorReset, ColorCyan, migration.Version, migration.Name, ColorReset)

		if err := rollbackMigration(db, migration); err != nil {
			fmt.Printf("%sFAILED%s\n", ColorRed, ColorReset)
			return fmt.Errorf("failed to rollback migration %d_%s: %w",
				migration.Version, migration.Name, err)
		}

		fmt.Printf("%sDONE%s\n", ColorGreen, ColorReset)
	}

	return nil
}

// MigrateFresh drops all tables and applies all migrations from scratch.
func MigrateFresh(db *pgxpool.Pool) error {
	// Drop all tables in the database.
	if err := dropAllTables(db); err != nil {
		return err
	}

	fmt.Printf("%s[FRESH]%s All tables dropped successfully\n", ColorGreen, ColorReset)
	fmt.Printf("%s[FRESH]%s Reapplying all migrations...\n", ColorBlue, ColorReset)

	// Apply all migrations.
	return Migrate(db)
}

// createMigrationsTable creates the migrations table if it doesn't exist.
func createMigrationsTable(db *pgxpool.Pool) error {
	_, err := db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS migrations (
			id SERIAL PRIMARY KEY,
			version BIGINT NOT NULL,
			name TEXT NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

// applyMigration applies a single migration to the database.
func applyMigration(db *pgxpool.Pool, migration Migration) error {
	// Check if the migration has already been applied.
	applied, err := isMigrationApplied(db, migration.Version)
	if err != nil {
		return err
	}

	// If the migration has been applied, print a message and return.
	if applied {
		fmt.Printf("%s[SKIPPED]%s Migration %s%d_%s%s already applied\n",
			ColorYellow,
			ColorReset,
			ColorCyan,
			migration.Version,
			migration.Name,
			ColorReset,
		)
		return nil
	}

	// Start a new transaction.
	tx, err := db.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("%sfailed to start transaction: %w%s", ColorRed, err, ColorReset)
	}
	defer tx.Rollback(context.Background())

	fmt.Printf("%s[MIGRATING]%s %s%d_%s%s... ",
		ColorYellow,
		ColorReset,
		ColorCyan,
		migration.Version,
		migration.Name,
		ColorReset,
	)

	// Convert SQL to lowercase before executing
	lowercaseSQL := strings.ToLower(migration.UpSQL)

	// Execute the up migration SQL script.
	if _, err := tx.Exec(context.Background(), lowercaseSQL); err != nil {
		fmt.Printf("%sFAILED%s\n", ColorRed, ColorReset)
		return fmt.Errorf("failed to apply migration %d_%s: %w", migration.Version, migration.Name, err)
	}

	// Insert a record of the applied migration into the migrations table.
	if _, err := tx.Exec(context.Background(), `
		INSERT INTO migrations (version, name) VALUES ($1, $2)
	`, migration.Version, migration.Name); err != nil {
		fmt.Printf("%sFAILED%s\n", ColorRed, ColorReset)
		return fmt.Errorf("failed to record migration %d_%s: %w", migration.Version, migration.Name, err)
	}

	// Commit the transaction.
	if err := tx.Commit(context.Background()); err != nil {
		fmt.Printf("%sFAILED%s\n", ColorRed, ColorReset)
		return fmt.Errorf("failed to commit migration %d_%s: %w", migration.Version, migration.Name, err)
	}

	fmt.Printf("%sDONE%s\n", ColorGreen, ColorReset)
	return nil
}

// rollbackMigration rolls back a single migration within a transaction
func rollbackMigration(db *pgxpool.Pool, migration Migration) error {
	tx, err := db.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	// Execute down migration
	statements := strings.Split(migration.DownSQL, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if _, err := tx.Exec(context.Background(), stmt); err != nil {
			return fmt.Errorf("failed to execute down migration: %w", err)
		}
	}

	// Remove migration record
	if _, err := tx.Exec(context.Background(), `
		DELETE FROM migrations WHERE version = $1
	`, migration.Version); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// getAppliedMigrations returns all applied migrations from the database
func getAppliedMigrations(db *pgxpool.Pool) ([]Migration, error) {
	rows, err := db.Query(context.Background(), `
		SELECT version, name FROM migrations 
		ORDER BY version DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	var migrations []Migration
	for rows.Next() {
		var m Migration
		if err := rows.Scan(&m.Version, &m.Name); err != nil {
			return nil, fmt.Errorf("failed to scan migration: %w", err)
		}

		// Load migration file content
		filename := fmt.Sprintf("%d_%s.sql", m.Version, m.Name)
		filePath := filepath.Join(migrationPath, "sql", filename)

		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		// Split content into up and down migrations
		parts := strings.Split(string(content), "-- Down Migration")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid migration format in file %s", filename)
		}

		m.DownSQL = strings.TrimSpace(parts[1])
		migrations = append(migrations, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating migrations: %w", err)
	}

	return migrations, nil
}

// isMigrationApplied checks if a migration with a given version has already been applied.
func isMigrationApplied(db *pgxpool.Pool, version int64) (bool, error) {
	var count int
	// Query the migrations table to check if the migration has been applied.
	err := db.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM migrations WHERE version = $1
	`, version).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("failed to check if migration is applied: %w", err)
	}

	return count > 0, nil
}

// getLatestMigration gets the version of the latest applied migration.
func getLatestMigration(db *pgxpool.Pool) (int64, error) {
	var version int64
	// Query the migrations table to get the latest migration version.
	err := db.QueryRow(context.Background(), `
		SELECT COALESCE(MAX(version), 0) FROM migrations
	`).Scan(&version)

	if err != nil {
		return 0, fmt.Errorf("failed to get latest migration: %w", err)
	}

	return version, nil
}

// ListMigrations retrieves and lists all migrations along with their status (applied or pending).
func ListMigrations(db *pgxpool.Pool) error {
	// Load all migrations from files
	migrations, err := loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Get all applied migrations from the database
	rows, err := db.Query(context.Background(), "SELECT version, applied_at FROM migrations ORDER BY version")
	if err != nil {
		return fmt.Errorf("failed to query migrations table: %w", err)
	}
	defer rows.Close()

	// Create a map of applied migrations
	appliedMigrations := make(map[int64]time.Time)
	for rows.Next() {
		var version int64
		var appliedAt time.Time
		if err := rows.Scan(&version, &appliedAt); err != nil {
			return fmt.Errorf("failed to scan migration row: %w", err)
		}
		appliedMigrations[version] = appliedAt
	}

	// Print header
	fmt.Printf("\n%sMigration Status%s\n", ColorBold, ColorReset)
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-20s %-30s %-15s %s\n", "Version", "Name", "Status", "Applied At")
	fmt.Println(strings.Repeat("-", 80))

	// Print each migration with its status
	for _, m := range migrations {
		appliedAt, isApplied := appliedMigrations[m.Version]
		status := fmt.Sprintf("%sPending%s", ColorYellow, ColorReset)
		appliedAtStr := "Not Applied"
		if isApplied {
			status = fmt.Sprintf("%sApplied%s", ColorGreen, ColorReset)
			appliedAtStr = appliedAt.Format("2006-01-02 15:04:05")
		}
		fmt.Printf("%-20d %-30s %-15s %s\n", m.Version, m.Name, status, appliedAtStr)
	}
	fmt.Println(strings.Repeat("-", 80))

	return nil
}

// dropAllTables drops all user-created tables in the database, excluding system tables and extensions.
func dropAllTables(db *pgxpool.Pool) error {
	// Execute a PostgreSQL anonymous code block to drop all user-created tables in the current schema
	_, err := db.Exec(context.Background(), `
		DO $$ 
		DECLARE
			r RECORD;
		BEGIN
			-- Disable triggers temporarily
			SET session_replication_role = 'replica';
			
			-- Drop all user-created tables, excluding system tables and extensions
			FOR r IN (
				SELECT tablename 
				FROM pg_tables 
				WHERE schemaname = current_schema()
					AND tablename != 'spatial_ref_sys'  -- Exclude PostGIS system table
					AND tablename NOT LIKE 'pg_%'       -- Exclude postgres system tables
					AND tablename != 'geography_columns'
					AND tablename != 'geometry_columns'
			) LOOP
				EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
			END LOOP;
			
			-- Re-enable triggers
			SET session_replication_role = 'origin';
		END $$;
	`)
	return err
}

// CreateDatabase creates a new database if it doesn't exist
func CreateDatabase(pgConfig *config.PostgresConfig) error {
	// Connect to postgres database to create new database
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=disable",
		pgConfig.SuperUser, pgConfig.SuperPass, pgConfig.Host, pgConfig.Port)

	// Use pgx.Connect instead of pgxpool for admin operations
	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		return fmt.Errorf("unable to connect to PostgreSQL: %v", err)
	}
	defer conn.Close(context.Background())

	// Check if database exists
	var exists bool
	err = conn.QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)",
		pgConfig.DBName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("error checking database existence: %v", err)
	}

	if !exists {
		_, err = conn.Exec(context.Background(),
			fmt.Sprintf("CREATE DATABASE %s", pgConfig.DBName))
		if err != nil {
			return fmt.Errorf("error creating database: %v", err)
		}
		fmt.Printf("%sDatabase '%s' created successfully%s\n",
			ColorGreen, pgConfig.DBName, ColorReset)
	} else {
		fmt.Printf("%sDatabase '%s' already exists%s\n",
			ColorBlue, pgConfig.DBName, ColorReset)
	}

	return nil
}

// CreateUser creates a new user if it doesn't exist and grants privileges
func CreateUser(pgConfig *config.PostgresConfig, privileges string) error {
	// Connect as super user
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=disable",
		pgConfig.SuperUser, pgConfig.SuperPass, pgConfig.Host, pgConfig.Port)

	// Use pgx.Connect for admin operations
	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		return fmt.Errorf("unable to connect to PostgreSQL: %v", err)
	}
	defer conn.Close(context.Background())

	// Check if user exists
	var exists bool
	err = conn.QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname = $1)",
		pgConfig.User).Scan(&exists)
	if err != nil {
		return fmt.Errorf("error checking user existence: %v", err)
	}

	if !exists {
		// Create user
		_, err = conn.Exec(context.Background(),
			fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'",
				pgConfig.User, pgConfig.Password))
		if err != nil {
			return fmt.Errorf("error creating user: %v", err)
		}
		fmt.Printf("%sUser '%s' created successfully%s\n",
			ColorGreen, pgConfig.User, ColorReset)
	} else {
		fmt.Printf("%sUser '%s' already exists%s\n",
			ColorBlue, pgConfig.User, ColorReset)
	}

	// Grant privileges based on the specified level
	var grantCmd string
	switch privileges {
	case "all":
		grantCmd = fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s",
			pgConfig.DBName, pgConfig.User)
	case "read":
		grantCmd = fmt.Sprintf("GRANT CONNECT, SELECT ON DATABASE %s TO %s",
			pgConfig.DBName, pgConfig.User)
	case "write":
		grantCmd = fmt.Sprintf("GRANT CONNECT, SELECT, INSERT, UPDATE, DELETE ON DATABASE %s TO %s",
			pgConfig.DBName, pgConfig.User)
	case "admin":
		grantCmd = fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s WITH GRANT OPTION",
			pgConfig.DBName, pgConfig.User)
	default:
		return fmt.Errorf("invalid privilege level: %s", privileges)
	}

	_, err = conn.Exec(context.Background(), grantCmd)
	if err != nil {
		return fmt.Errorf("error granting privileges: %v", err)
	}

	fmt.Printf("%sPrivileges '%s' granted to user '%s' on database '%s'%s\n",
		ColorGreen, privileges, pgConfig.User, pgConfig.DBName, ColorReset)

	return nil
}
