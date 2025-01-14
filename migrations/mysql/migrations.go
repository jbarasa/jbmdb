// Package mysql provides functionality to manage database migrations
// for MySQL and MariaDB databases using the go-sql-driver/mysql library.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jbarasa/jbmdb/migrations/config"
)

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

// CreateDatabase creates a new database if it doesn't exist
func CreateDatabase(myConfig *config.MySQLConfig) error {
	// Connect to MySQL server as super user
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/",
		myConfig.SuperUser, myConfig.SuperPass, myConfig.Host, myConfig.Port)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("error connecting to MySQL: %v", err)
	}
	defer db.Close()

	// Create database if not exists
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", myConfig.DBName))
	if err != nil {
		return fmt.Errorf("error creating database: %v", err)
	}

	fmt.Printf("%sDatabase '%s' created/verified successfully%s\n",
		ColorGreen, myConfig.DBName, ColorReset)

	return nil
}

// CreateUser creates a new user if it doesn't exist and grants privileges
func CreateUser(myConfig *config.MySQLConfig, privileges string) error {
	// Connect to MySQL server as super user
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/",
		myConfig.SuperUser, myConfig.SuperPass, myConfig.Host, myConfig.Port)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("error connecting to MySQL: %v", err)
	}
	defer db.Close()

	// Create user if not exists
	createUserSQL := fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY '%s'",
		myConfig.User, myConfig.Password)
	_, err = db.Exec(createUserSQL)
	if err != nil {
		return fmt.Errorf("error creating user: %v", err)
	}

	fmt.Printf("%sUser '%s' created/verified successfully%s\n",
		ColorGreen, myConfig.User, ColorReset)

	// Grant privileges based on the specified level
	var grantCmd string
	switch privileges {
	case "all":
		grantCmd = fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO '%s'@'%%'",
			myConfig.DBName, myConfig.User)
	case "read":
		grantCmd = fmt.Sprintf("GRANT SELECT ON %s.* TO '%s'@'%%'",
			myConfig.DBName, myConfig.User)
	case "write":
		grantCmd = fmt.Sprintf("GRANT SELECT, INSERT, UPDATE, DELETE ON %s.* TO '%s'@'%%'",
			myConfig.DBName, myConfig.User)
	case "admin":
		grantCmd = fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO '%s'@'%%' WITH GRANT OPTION",
			myConfig.DBName, myConfig.User)
	default:
		return fmt.Errorf("invalid privilege level: %s", privileges)
	}

	_, err = db.Exec(grantCmd)
	if err != nil {
		return fmt.Errorf("error granting privileges: %v", err)
	}

	// Flush privileges to apply changes
	_, err = db.Exec("FLUSH PRIVILEGES")
	if err != nil {
		return fmt.Errorf("error flushing privileges: %v", err)
	}

	fmt.Printf("%sPrivileges '%s' granted to user '%s' on database '%s'%s\n",
		ColorGreen, privileges, myConfig.User, myConfig.DBName, ColorReset)

	return nil
}

// Migration represents a database migration with its version, name, and SQL scripts for
// applying and rolling back the migration.
type Migration struct {
	Version int64  // Version number of the migration
	Name    string // Name of the migration
	UpSQL   string // SQL script for applying the migration
	DownSQL string // SQL script for rolling back the migration
}

// Path to the migration files
var migrationPath string

// SetMigrationPath sets the path for migration files
func SetMigrationPath(path string) {
	migrationPath = path
}

// extractTableName extracts the table name from the migration name
func extractTableName(name string) string {
	name = strings.TrimPrefix(name, "create_")
	name = strings.TrimPrefix(name, "add_")
	name = strings.TrimSuffix(name, "_table")
	return camelToSnakeCase(name)
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
			return fmt.Errorf("%stable name '%s' already exists in migration '%d_%s'%s",
				ColorRed, newTableName, migration.Version, migration.Name, ColorReset)
		}
	}
	return nil
}

// CreateMigration creates new migration file with the given name and current timestamp
func CreateMigration(name string) error {
	// Extract table name from migration name
	tableName := extractTableName(name)

	// Check for duplicate table names
	if err := checkDuplicateTableName(tableName); err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("%s_%s.sql", timestamp, name)

	content := fmt.Sprintf(`-- Migration: %s

-- Up Migration
----------------------- Write your up migration here ----------------------------

CREATE TABLE IF NOT EXISTS %s (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


-- Down Migration
----------------------- Write your down migration here ----------------------------

DROP TABLE IF EXISTS %s;`, name, strings.ToLower(tableName), strings.ToLower(tableName))

	filePath := filepath.Join(migrationPath, "sql", filename)
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write migration file: %w", err)
	}

	fmt.Printf("%s[SUCCESS]%s Created migration %s%s%s\n",
		ColorGreen, ColorReset, ColorCyan, filename, ColorReset)
	return nil
}

// loadMigrations loads all migration files from the migration directory
func loadMigrations() ([]Migration, error) {
	var migrations []Migration

	sqlDir := filepath.Join(migrationPath, "sql")
	files, err := os.ReadDir(sqlDir)
	if err != nil {
		if os.IsNotExist(err) {
			return migrations, nil
		}
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		version := parseInt(file.Name()[:14])
		name := strings.TrimSuffix(file.Name()[15:], ".sql")

		content, err := os.ReadFile(filepath.Join(sqlDir, file.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", file.Name(), err)
		}

		parts := strings.Split(string(content), "-- Down Migration")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid migration file format %s", file.Name())
		}

		upSQL := strings.Split(parts[0], "-- Up Migration")[1]
		downSQL := parts[1]

		migrations = append(migrations, Migration{
			Version: version,
			Name:    name,
			UpSQL:   strings.TrimSpace(upSQL),
			DownSQL: strings.TrimSpace(downSQL),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// Migrate applies all pending migrations to the database
func Migrate(db *sql.DB) error {
	if err := createMigrationsTable(db); err != nil {
		return err
	}

	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		applied, err := isMigrationApplied(db, migration.Version)
		if err != nil {
			return err
		}

		if !applied {
			fmt.Printf("%s[MIGRATE]%s Applying migration %s%d_%s%s... ",
				ColorBlue, ColorReset, ColorCyan, migration.Version, migration.Name, ColorReset)

			if err := applyMigration(db, migration); err != nil {
				fmt.Printf("%sFAILED%s\n", ColorRed, ColorReset)
				return fmt.Errorf("failed to apply migration %d_%s: %w",
					migration.Version, migration.Name, err)
			}

			fmt.Printf("%sOK%s\n", ColorGreen, ColorReset)
		}
	}

	return nil
}

// RollbackLast rolls back the most recently applied migration
func RollbackLast(db *sql.DB) error {
	latestVersion, err := getLatestMigration(db)
	if err != nil {
		return err
	}

	if latestVersion == 0 {
		fmt.Printf("%sNo migrations to rollback%s\n", ColorYellow, ColorReset)
		return nil
	}

	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		if migration.Version == latestVersion {
			fmt.Printf("%s[ROLLBACK]%s Rolling back migration %s%d_%s%s... ",
				ColorBlue, ColorReset, ColorCyan, migration.Version, migration.Name, ColorReset)

			if err := rollbackMigration(db, migration); err != nil {
				fmt.Printf("%sFAILED%s\n", ColorRed, ColorReset)
				return fmt.Errorf("failed to rollback migration %d_%s: %w",
					migration.Version, migration.Name, err)
			}

			fmt.Printf("%sOK%s\n", ColorGreen, ColorReset)
			return nil
		}
	}

	return fmt.Errorf("migration version %d not found", latestVersion)
}

// RollbackSteps rolls back a specified number of migrations
func RollbackSteps(db *sql.DB, steps int) error {
	appliedMigrations, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	if len(appliedMigrations) == 0 {
		fmt.Printf("%sNo migrations to rollback%s\n", ColorYellow, ColorReset)
		return nil
	}

	// Limit steps to available migrations
	if steps > len(appliedMigrations) {
		steps = len(appliedMigrations)
		fmt.Printf("%sNote: Only %d migrations available to rollback%s\n",
			ColorYellow, steps, ColorReset)
	}

	// Rollback migrations in reverse order
	for i := 0; i < steps; i++ {
		migration := appliedMigrations[i]
		fmt.Printf("%s[ROLLBACK]%s Rolling back migration %s%d_%s%s... ",
			ColorBlue, ColorReset, ColorCyan, migration.Version, migration.Name, ColorReset)

		if err := rollbackMigration(db, migration); err != nil {
			fmt.Printf("%sFAILED%s\n", ColorRed, ColorReset)
			return fmt.Errorf("failed to rollback migration %d_%s: %w",
				migration.Version, migration.Name, err)
		}

		fmt.Printf("%sOK%s\n", ColorGreen, ColorReset)
	}

	return nil
}

// MigrateFresh drops all tables and reapplies all migrations
func MigrateFresh(db *sql.DB) error {
	if err := dropAllTables(db); err != nil {
		return err
	}

	return Migrate(db)
}

// ListMigrations retrieves and lists all migrations along with their status
func ListMigrations(db *sql.DB) error {
	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	if len(migrations) == 0 {
		fmt.Printf("%sNo migrations found%s\n", ColorYellow, ColorReset)
		return nil
	}

	fmt.Println("\nMigration Status:")
	fmt.Println("------------------")

	for _, migration := range migrations {
		applied, err := isMigrationApplied(db, migration.Version)
		if err != nil {
			return err
		}

		status := fmt.Sprintf("%s[PENDING]%s", ColorYellow, ColorReset)
		if applied {
			status = fmt.Sprintf("%s[APPLIED]%s", ColorGreen, ColorReset)
		}

		fmt.Printf("%s %s%d_%s%s\n",
			status, ColorCyan, migration.Version, migration.Name, ColorReset)
	}

	fmt.Println()
	return nil
}

// createMigrationsTable creates the migrations table if it doesn't exist
func createMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			version BIGINT UNSIGNED PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`)
	return err
}

// applyMigration applies a single migration to the database
func applyMigration(db *sql.DB, migration Migration) error {
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Split the up migration into individual statements
	statements := strings.Split(migration.UpSQL, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}

	// Record the migration
	if _, err := tx.Exec(
		"INSERT INTO migrations (version, name) VALUES (?, ?)",
		migration.Version, migration.Name,
	); err != nil {
		return err
	}

	return tx.Commit()
}

// rollbackMigration rolls back a single migration
func rollbackMigration(db *sql.DB, migration Migration) error {
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Split the down migration into individual statements
	statements := strings.Split(migration.DownSQL, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}

	// Remove the migration record
	if _, err := tx.Exec(
		"DELETE FROM migrations WHERE version = ?",
		migration.Version,
	); err != nil {
		return err
	}

	return tx.Commit()
}

// getAppliedMigrations returns all applied migrations from the database
func getAppliedMigrations(db *sql.DB) ([]Migration, error) {
	var migrations []Migration

	rows, err := db.Query("SELECT version, name FROM migrations ORDER BY version DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var version int64
		var name string
		if err := rows.Scan(&version, &name); err != nil {
			return nil, err
		}

		// Load migration file content
		filename := fmt.Sprintf("%d_%s.sql", version, name)
		filePath := filepath.Join(migrationPath, "sql", filename)

		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		parts := strings.Split(string(content), "-- Down Migration")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid migration file format %s", filename)
		}

		migrations = append(migrations, Migration{
			Version: version,
			Name:    name,
			DownSQL: strings.TrimSpace(parts[1]),
		})
	}

	return migrations, rows.Err()
}

// isMigrationApplied checks if a migration has already been applied
func isMigrationApplied(db *sql.DB, version int64) (bool, error) {
	var exists bool
	err := db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM migrations WHERE version = ?)",
		version,
	).Scan(&exists)
	return exists, err
}

// getLatestMigration gets the version of the latest applied migration
func getLatestMigration(db *sql.DB) (int64, error) {
	var version int64
	err := db.QueryRow(
		"SELECT COALESCE(MAX(version), 0) FROM migrations",
	).Scan(&version)
	return version, err
}

// parseInt converts a string to an integer
func parseInt(s string) int64 {
	var version int64
	fmt.Sscanf(s, "%d", &version)
	return version
}

// dropAllTables drops all user-created tables in the database
func dropAllTables(db *sql.DB) error {
	fmt.Printf("%s[WARNING]%s Dropping all tables... ", ColorYellow, ColorReset)

	// Disable foreign key checks temporarily
	if _, err := db.Exec("SET FOREIGN_KEY_CHECKS = 0"); err != nil {
		return err
	}
	defer db.Exec("SET FOREIGN_KEY_CHECKS = 1")

	// Get all table names
	rows, err := db.Query(`
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = DATABASE()
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Drop each table
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return err
		}

		if _, err := db.Exec("DROP TABLE IF EXISTS " + tableName); err != nil {
			return err
		}
	}

	if err := rows.Err(); err != nil {
		return err
	}

	fmt.Printf("%sOK%s\n", ColorGreen, ColorReset)
	return nil
}
