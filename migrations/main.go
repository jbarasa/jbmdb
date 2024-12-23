// Package main provides a command-line tool for managing database migrations
// migrations, applying migrations, rolling them back, listing migrations, and performing fresh migrations.
// It supports both PostgreSQL and ScyllaDB Keyspaces.

// main.go
// ├── loadConfig() -> sets migration paths
// ├── handlePostgres()
// │   ├── make -> creates new migration
// │   ├── migrate -> applies migrations
// │   ├── rollback -> rolls back last migration
// │   ├── fresh -> drops all and remigrates
// │   ├── list -> shows migration status
// │   └── init -> initializes config
// └── handleScylla()
//     ├── make -> creates new migration
//     ├── migrate -> applies migrations
//     ├── rollback -> rolls back last migration
//     ├── fresh -> drops all and reapply migrations
//     ├── list -> shows migration status
//     └── init -> initializes config
//
// Developer: Joseph Barasa
// Year: 2024
// Developer's Website: jbarasa.com
// License: Jbarasa INC

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gocql/gocql"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jbarasa/jbmdb/migrations/postgres"
	"github.com/jbarasa/jbmdb/migrations/scylladb"
	"github.com/jbarasa/jbmdb/migrations/update"
	"github.com/joho/godotenv"
)

const DefaultConfigFile = ".jbmdb.conf"

// Version is set during build time
var Version = "dev"

// Configuration structure
type Config struct {
	PostgresPath string // Path for PostgreSQL migrations
	ScyllaPath   string // Path for ScyllaDB migrations
	SQLFolder    string // Folder name for SQL files
	CQLFolder    string // Folder name for CQL files
}

// Global configuration
var config Config

func main() {
	// Load environment variables
	godotenv.Load()

	// Load or create configuration
	loadConfig()

	if len(os.Args) < 2 {
		showUsage()
		os.Exit(1)
	}
	// Parse command-line flags.

	flag.Parse()

	command := flag.Arg(0)

	// Check for config command first
	if command == "config" {
		handleConfig()
		return
	}

	// Check for update command
	if command == "update" {
		handleUpdate()
		return
	}

	// Check for version command
	if command == "version" {
		fmt.Printf("jbmdb version %s\n", Version)
		return
	}

	// Split command into db type and action
	parts := strings.Split(command, "-")
	if len(parts) != 2 {
		showUsage()
		os.Exit(1)
	}

	dbType := parts[0]
	action := parts[1]

	switch dbType {
	case "postgres":
		handlePostgres(action)
	case "scylla":
		handleScylla(action)
	default:
		fmt.Printf("%sError: Invalid database type. Use 'postgres' or 'scylla'%s\n", postgres.ColorRed, postgres.ColorReset)
		os.Exit(1)
	}
}

func showUsage() {
	fmt.Printf("\n%s%sJBMDB Database Migration Tool%s\n\n", postgres.ColorBold, postgres.ColorBlue, postgres.ColorReset)
	fmt.Println("Usage: jbmdb <command> [args]")
	fmt.Println("\nCommands:")
	fmt.Printf("  %sConfiguration:%s\n", postgres.ColorCyan, postgres.ColorReset)
	fmt.Printf("    config              Configure migration paths and folder names\n")
	fmt.Printf("    update              Check for and install updates\n")
	fmt.Printf("    version             Show version information\n")
	fmt.Printf("\n  %sPostgreSQL:%s\n", postgres.ColorCyan, postgres.ColorReset)
	fmt.Printf("    postgres-migration <name>   Create a new PostgreSQL migration\n")
	fmt.Printf("    postgres-migrate       Run all pending PostgreSQL migrations\n")
	fmt.Printf("    postgres-rollback      Rollback the last PostgreSQL migration\n")
	fmt.Printf("    postgres-fresh         Drop all tables and reapply PostgreSQL migrations\n")
	fmt.Printf("    postgres-list          List all PostgreSQL migrations\n")
	fmt.Printf("    postgres-init          Initialize PostgreSQL configuration\n")
	fmt.Printf("\n  %sScyllaDB:%s\n", postgres.ColorCyan, postgres.ColorReset)
	fmt.Printf("    scylla-migration <name>     Create a new ScyllaDB migration\n")
	fmt.Printf("    scylla-migrate         Run all pending ScyllaDB migrations\n")
	fmt.Printf("    scylla-rollback        Rollback the last ScyllaDB migration\n")
	fmt.Printf("    scylla-fresh           Drop all tables and reapply ScyllaDB migrations\n")
	fmt.Printf("    scylla-list            List all ScyllaDB migrations\n")
	fmt.Printf("    scylla-init            Initialize ScyllaDB configuration\n\n")

	fmt.Printf("Current Configuration:\n")
	fmt.Printf("  PostgreSQL migrations: %s%s/%s%s\n", postgres.ColorCyan, config.PostgresPath, config.SQLFolder, postgres.ColorReset)
	fmt.Printf("  ScyllaDB migrations:   %s%s/%s%s\n\n", postgres.ColorCyan, config.ScyllaPath, config.CQLFolder, postgres.ColorReset)
}

func handleConfig() {
	fmt.Printf("\n%s%sJBMDB Configuration%s\n\n", postgres.ColorBold, postgres.ColorBlue, postgres.ColorReset)

	// PostgreSQL configuration
	fmt.Printf("%sPostgreSQL Migrations Path%s [%s]: ", postgres.ColorCyan, postgres.ColorReset, config.PostgresPath)
	postgresPath := readInput(config.PostgresPath)

	fmt.Printf("%sSQL Files Folder Name%s [%s]: ", postgres.ColorCyan, postgres.ColorReset, config.SQLFolder)
	sqlFolder := readInput(config.SQLFolder)

	// ScyllaDB configuration
	fmt.Printf("%sScyllaDB Migrations Path%s [%s]: ", postgres.ColorCyan, postgres.ColorReset, config.ScyllaPath)
	scyllaPath := readInput(config.ScyllaPath)

	fmt.Printf("%sCQL Files Folder Name%s [%s]: ", postgres.ColorCyan, postgres.ColorReset, config.CQLFolder)
	cqlFolder := readInput(config.CQLFolder)

	// Update configuration
	config = Config{
		PostgresPath: postgresPath,
		ScyllaPath:   scyllaPath,
		SQLFolder:    sqlFolder,
		CQLFolder:    cqlFolder,
	}

	// Save configuration
	saveConfig()

	// Create directories
	createMigrationDirs()

	fmt.Printf("\n%sConfiguration saved successfully!%s\n", postgres.ColorGreen, postgres.ColorReset)
}

func readInput(defaultValue string) string {
	var input string
	fmt.Scanln(&input)
	if input == "" {
		return defaultValue
	}
	return input
}

func loadConfig() {
	// Default configuration
	config = Config{
		PostgresPath: "migrations/postgres/migrations",
		ScyllaPath:   "migrations/scylladb/migrations",
		SQLFolder:    "sql",
		CQLFolder:    "cql",
	}

	// Try to load existing configuration
	data, err := os.ReadFile(DefaultConfigFile)
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "POSTGRES_PATH":
				config.PostgresPath = value
			case "SCYLLA_PATH":
				config.ScyllaPath = value
			case "SQL_FOLDER":
				config.SQLFolder = value
			case "CQL_FOLDER":
				config.CQLFolder = value
			}
		}
	}

	// Set the migration paths in the respective packages
	postgres.SetMigrationPath(config.PostgresPath)
	scylladb.SetMigrationPath(config.ScyllaPath)
}

func saveConfig() {
	content := fmt.Sprintf("POSTGRES_PATH=%s\nSCYLLA_PATH=%s\nSQL_FOLDER=%s\nCQL_FOLDER=%s\n",
		config.PostgresPath,
		config.ScyllaPath,
		config.SQLFolder,
		config.CQLFolder,
	)

	if err := os.WriteFile(DefaultConfigFile, []byte(content), 0644); err != nil {
		fmt.Printf("%sError saving configuration: %v%s\n", postgres.ColorRed, err, postgres.ColorReset)
		os.Exit(1)
	}
}

func createMigrationDirs() {
	// Create PostgreSQL migrations directory
	postgresDir := filepath.Join(config.PostgresPath, config.SQLFolder)
	if err := os.MkdirAll(postgresDir, 0755); err != nil {
		fmt.Printf("%sError creating PostgreSQL migrations directory: %v%s\n", postgres.ColorRed, err, postgres.ColorReset)
		os.Exit(1)
	}

	// Create ScyllaDB migrations directory
	scyllaDir := filepath.Join(config.ScyllaPath, config.CQLFolder)
	if err := os.MkdirAll(scyllaDir, 0755); err != nil {
		fmt.Printf("%sError creating ScyllaDB migrations directory: %v%s\n", postgres.ColorRed, err, postgres.ColorReset)
		os.Exit(1)
	}
}

func handlePostgres(command string) {
	// Set the migration path from config
	postgres.SetMigrationPath(config.PostgresPath)

	// Construct database connection URL
	dbURL := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DATABASE_NAME"),
	)

	// Create connection pool
	db, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("%sUnable to connect to PostgreSQL: %v%s\n", postgres.ColorRed, err, postgres.ColorReset)
	}
	defer db.Close()

	// Handle commands
	switch command {
	case "migration":
		// Call your existing PostgreSQL migration migration function
		// Ensure a migration name is provided as an argument.
		if flag.NArg() < 2 {
			fmt.Printf("%sError: Migration name is required%s\n", postgres.ColorRed, postgres.ColorReset)
			os.Exit(1)
		}
		name := flag.Arg(1)

		// Validate migration name format
		if !strings.HasPrefix(name, "create_") || !strings.HasSuffix(name, "_table") {
			fmt.Printf("%sError: Migration name must follow format: create_<name>_table\n", postgres.ColorRed)
			fmt.Printf("Example: create_users_table, create_post_comments_table%s\n", postgres.ColorReset)
			os.Exit(1)
		}

		// Check for singular table names
		tableName := strings.TrimPrefix(name, "create_")
		tableName = strings.TrimSuffix(tableName, "_table")
		parts := strings.Split(tableName, "_")

		if len(parts) == 1 && !strings.HasSuffix(parts[0], "s") {
			fmt.Printf("%sError: Single table names should be plural\n", postgres.ColorRed)
			fmt.Printf("Example: 'create_user_table' should be 'create_users_table'%s\n", postgres.ColorReset)
			os.Exit(1)
		}

		// Check for plural in relation tables
		if len(parts) > 1 && !strings.HasSuffix(parts[len(parts)-1], "s") {
			fmt.Printf("%sError: In relation tables, names after the first word should be plural\n", postgres.ColorRed)
			fmt.Printf("Example: 'create_user_comment_table' should be 'create_user_comments_table'%s\n", postgres.ColorReset)
			os.Exit(1)
		}

		// Create a new migration with the specified name.
		if err := postgres.CreateMigration(name); err != nil {
			fmt.Printf("%sFailed to create migration: %v%s\n", postgres.ColorRed, err, postgres.ColorReset)
			os.Exit(1)
		}

	case "migrate":
		// Apply all pending migrations to the database.
		if err := postgres.Migrate(db); err != nil {
			log.Fatalf("%sFailed to run migrations: %v%s", postgres.ColorRed, err, postgres.ColorReset)
		}
		fmt.Printf("%sMigrations completed successfully%s\n", postgres.ColorGreen, postgres.ColorReset)
	case "rollback":
		// Rollback the last applied migration from the database.
		if err := postgres.RollbackLast(db); err != nil {
			log.Fatalf("%sFailed to rollback migration: %v%s", postgres.ColorRed, err, postgres.ColorReset)
		}
		fmt.Printf("%sRollback completed successfully%s\n", postgres.ColorGreen, postgres.ColorReset)
	case "fresh":
		fmt.Printf("%s[WARNING]%s This will drop all tables and reapply all migrations.\n", postgres.ColorRed, postgres.ColorReset)
		fmt.Printf("Are you sure you want to continue? (y/N): ")

		var response string
		fmt.Scanln(&response)

		if strings.ToLower(response) != "y" {
			fmt.Printf("%sOperation cancelled%s\n", postgres.ColorYellow, postgres.ColorReset)
			os.Exit(0)
		}

		if err := postgres.MigrateFresh(db); err != nil {
			fmt.Printf("%sFailed to run fresh migrations: %v%s\n", postgres.ColorRed, err, postgres.ColorReset)
			os.Exit(1)
		}
		fmt.Printf("%sFresh migration completed successfully%s\n", postgres.ColorGreen, postgres.ColorReset)
	case "list":
		// List all migrations and their status.
		if err := postgres.ListMigrations(db); err != nil {
			fmt.Printf("%sFailed to list migrations: %v%s\n", postgres.ColorRed, err, postgres.ColorReset)
			os.Exit(1)
		}
	case "init":
		initPostgresConfig()

	default:
		fmt.Printf("%sError: Unknown command: %s%s\n", postgres.ColorRed, command, postgres.ColorReset)
		os.Exit(1)
	}
}

func handleScylla(command string) {
	// Set the migration path from config
	scylladb.SetMigrationPath(config.ScyllaPath)

	// Get ScyllaDB hosts
	hosts := strings.Split(os.Getenv("SCYLLA_HOSTS"), ",")

	// Create ScyllaDB session
	cluster := gocql.NewCluster(hosts...)
	cluster.Keyspace = os.Getenv("SCYLLA_KEYSPACE")
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: os.Getenv("SCYLLA_USER"),
		Password: os.Getenv("SCYLLA_PASSWORD"),
	}
	cluster.Consistency = gocql.LocalOne

	session, err := cluster.CreateSession()
	if err != nil {
		log.Fatalf("%sUnable to connect to ScyllaDB: %v%s\n", postgres.ColorRed, err, postgres.ColorReset)
	}
	defer session.Close()

	// Handle commands
	switch command {
	case "migration":
		// Call your existing ScyllaDB migration migration function
		// Ensure a migration name is provided as an argument.
		if flag.NArg() < 2 {
			fmt.Printf("%sError: Migration name is required%s\n", scylladb.ColorRed, scylladb.ColorReset)
			os.Exit(1)
		}
		name := flag.Arg(1)
		// Validate migration name format
		if !strings.HasPrefix(name, "create_") || !strings.HasSuffix(name, "_table") {
			fmt.Printf("%sError: Migration name must follow format: create_<name>_table\n", scylladb.ColorRed)
			fmt.Printf("Example: create_users_table, create_post_comments_table%s\n", scylladb.ColorReset)
			os.Exit(1)
		}

		// Check for singular table names
		tableName := strings.TrimPrefix(name, "create_")
		tableName = strings.TrimSuffix(tableName, "_table")
		parts := strings.Split(tableName, "_")

		if len(parts) == 1 && !strings.HasSuffix(parts[0], "s") {
			fmt.Printf("%sError: Single table names should be plural\n", scylladb.ColorRed)
			fmt.Printf("Example: 'create_user_table' should be 'create_users_table'%s\n", scylladb.ColorReset)
			os.Exit(1)
		}

		// Check for plural in relation tables
		if len(parts) > 1 && !strings.HasSuffix(parts[len(parts)-1], "s") {
			fmt.Printf("%sError: In relation tables, names after the first word should be plural\n", scylladb.ColorRed)
			fmt.Printf("Example: 'create_user_comment_table' should be 'create_user_comments_table'%s\n", scylladb.ColorReset)
			os.Exit(1)
		}

		// Create a new migration with the specified name.
		if err := scylladb.CreateMigration(name); err != nil {
			fmt.Printf("%sError: Failed to create migration: %v%s\n", scylladb.ColorRed, err, scylladb.ColorReset)
			os.Exit(1)
		}
	case "migrate":
		if err := scylladb.Migrate(session); err != nil {
			fmt.Printf("%sError: Failed to run migrations: %v%s\n", scylladb.ColorRed, err, scylladb.ColorReset)
			os.Exit(1)
		}
		fmt.Printf("%sMigrations completed successfully%s\n", scylladb.ColorGreen, scylladb.ColorReset)
	case "rollback":
		if err := scylladb.RollbackLast(session); err != nil {
			fmt.Printf("%sError: Failed to rollback migration: %v%s\n", scylladb.ColorRed, err, scylladb.ColorReset)
			os.Exit(1)
		}
		fmt.Printf("%sRollback completed successfully%s\n", scylladb.ColorGreen, scylladb.ColorReset)
	case "fresh":
		fmt.Printf("%s[WARNING]%s This will drop all tables and reapply all migrations.\n", scylladb.ColorRed, scylladb.ColorReset)
		fmt.Printf("Are you sure you want to continue? (y/N): ")

		var response string
		fmt.Scanln(&response)

		if strings.ToLower(response) != "y" {
			fmt.Printf("%sOperation cancelled%s\n", scylladb.ColorYellow, scylladb.ColorReset)
			os.Exit(0)
		}

		if err := scylladb.MigrateFresh(session); err != nil {
			fmt.Printf("%sError: Failed to run fresh migrations: %v%s\n", scylladb.ColorRed, err, scylladb.ColorReset)
			os.Exit(1)
		}
		fmt.Printf("%sFresh migration completed successfully%s\n", scylladb.ColorGreen, scylladb.ColorReset)
	case "list":
		if err := scylladb.ListMigrations(session); err != nil {
			fmt.Printf("%sError: Failed to list migrations: %v%s\n", scylladb.ColorRed, err, scylladb.ColorReset)
			os.Exit(1)
		}
	case "init":
		initScyllaConfig()

	default:
		fmt.Printf("%sError: Unknown command: %s%s\n", postgres.ColorRed, command, postgres.ColorReset)
		os.Exit(1)
	}
}

func initPostgresConfig() {

	config := []struct {
		key, description, defaultValue string
	}{
		{"POSTGRES_HOST", "PostgreSQL host", "localhost"},
		{"POSTGRES_PORT", "PostgreSQL port", "5432"},
		{"POSTGRES_USER", "PostgreSQL username", "postgres"},
		{"POSTGRES_PASSWORD", "PostgreSQL password", ""},
		{"POSTGRES_DATABASE_NAME", "PostgreSQL database name", ""},
		{"POSTGRES_MIGRATIONS_PATH", "PostgreSQL Migrations Path", "database/migrations"},
	}

	fmt.Printf("\n%sPostgreSQL Configuration%s\n\n", postgres.ColorBold, postgres.ColorReset)
	envContent := ""
	var migrationPath string

	for _, c := range config {
		var value string
		fmt.Printf("%s%s%s [%s]: ", postgres.ColorCyan, c.description, postgres.ColorReset, c.defaultValue)
		fmt.Scanln(&value)

		if value == "" {
			value = c.defaultValue
		}

		if c.key == "POSTGRES_MIGRATIONS_PATH" {
			migrationPath = value
		} else {
			envContent += fmt.Sprintf("%s=%s\n", c.key, value)
		}
	}

	// Save the migration path to .jbmdb.conf
	// Read existing config first to preserve ScyllaDB settings if they exist
	existingConfig := make(map[string]string)
	if data, err := os.ReadFile(DefaultConfigFile); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				existingConfig[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	// Update PostgreSQL settings while preserving ScyllaDB settings
	existingConfig["POSTGRES_MIGRATIONS_PATH"] = migrationPath
	existingConfig["SQL_FOLDER"] = "sql"

	// Create migrations directory and SQL folder inside it
	if err := os.MkdirAll(migrationPath, 0755); err != nil {
		fmt.Printf("%sError creating migrations directory: %v%s\n", postgres.ColorRed, err, postgres.ColorReset)
		os.Exit(1)
	}

	// Create SQL folder inside the migrations directory
	sqlPath := filepath.Join(migrationPath, "sql")
	if err := os.MkdirAll(sqlPath, 0755); err != nil {
		fmt.Printf("%sError creating SQL directory: %v%s\n", postgres.ColorRed, err, postgres.ColorReset)
		os.Exit(1)
	}

	// Write back the complete configuration
	var content string
	for k, v := range existingConfig {
		content += fmt.Sprintf("%s=%s\n", k, v)
	}

	if err := os.WriteFile(DefaultConfigFile, []byte(content), 0644); err != nil {
		fmt.Printf("%sError saving migration path configuration: %v%s\n", postgres.ColorRed, err, postgres.ColorReset)
		os.Exit(1)
	}

	writeEnvFile(envContent, true)

	// Set the migration path in the postgres package
	postgres.SetMigrationPath(migrationPath)
}

func initScyllaConfig() {
	config := []struct {
		key, description, defaultValue string
	}{
		{"SCYLLA_HOSTS", "ScyllaDB hosts (comma-separated)", "localhost"},
		{"SCYLLA_KEYSPACE", "ScyllaDB keyspace", ""},
		{"SCYLLA_USER", "ScyllaDB username", ""},
		{"SCYLLA_PASSWORD", "ScyllaDB password", ""},
		{"SCYLLA_MIGRATIONS_PATH", "ScyllaDB Migrations Path", "database/migrations"},
	}

	fmt.Printf("\n%sScyllaDB Configuration%s\n\n", scylladb.ColorBold, scylladb.ColorReset)
	envContent := ""
	var migrationPath string

	for _, c := range config {
		var value string
		fmt.Printf("%s%s%s [%s]: ", scylladb.ColorCyan, c.description, scylladb.ColorReset, c.defaultValue)
		fmt.Scanln(&value)

		if value == "" {
			value = c.defaultValue
		}

		if c.key == "SCYLLA_MIGRATIONS_PATH" {
			migrationPath = value
		} else {
			envContent += fmt.Sprintf("%s=%s\n", c.key, value)
		}
	}

	// Read existing config first to preserve PostgreSQL settings if they exist
	existingConfig := make(map[string]string)
	if data, err := os.ReadFile(DefaultConfigFile); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				existingConfig[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	// Update with new ScyllaDB settings
	existingConfig["SCYLLA_MIGRATIONS_PATH"] = migrationPath
	existingConfig["CQL_FOLDER"] = "cql"

	// Create migrations directory and CQL folder inside it
	if err := os.MkdirAll(migrationPath, 0755); err != nil {
		fmt.Printf("%sError creating migrations directory: %v%s\n", scylladb.ColorRed, err, scylladb.ColorReset)
		os.Exit(1)
	}

	// Create CQL folder inside the migrations directory
	cqlPath := filepath.Join(migrationPath, "cql")
	if err := os.MkdirAll(cqlPath, 0755); err != nil {
		fmt.Printf("%sError creating CQL directory: %v%s\n", scylladb.ColorRed, err, scylladb.ColorReset)
		os.Exit(1)
	}

	// Write back the complete configuration
	var content string
	for k, v := range existingConfig {
		content += fmt.Sprintf("%s=%s\n", k, v)
	}

	if err := os.WriteFile(DefaultConfigFile, []byte(content), 0644); err != nil {
		fmt.Printf("%sError saving migration path configuration: %v%s\n", scylladb.ColorRed, err, scylladb.ColorReset)
		os.Exit(1)
	}

	writeEnvFile(envContent, false)

	// Set the migration path in the scylladb package
	scylladb.SetMigrationPath(migrationPath)
}

func handleUpdate() {
	release, err := update.CheckForUpdates(Version)
	if err != nil {
		fmt.Printf("%sError checking for updates: %v%s\n", scylladb.ColorRed, err, scylladb.ColorReset)
		os.Exit(1)
	}
	if release == nil {
		fmt.Printf("%sYou are already running the latest version%s\n", scylladb.ColorGreen, scylladb.ColorReset)
		return
	}

	fmt.Printf("%sNew version %s available!%s\n", scylladb.ColorCyan, release.TagName, scylladb.ColorReset)
	update.PrintUpdateChangelog(release)

	fmt.Print("Do you want to update now? (y/N): ")
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "y" {
		fmt.Printf("%sUpdate cancelled%s\n", postgres.ColorYellow, postgres.ColorReset)
		return
	}

	fmt.Printf("Downloading and installing update...\n")
	if err := update.DownloadUpdate(release); err != nil {
		fmt.Printf("%sError installing update: %v%s\n", postgres.ColorRed, err, postgres.ColorReset)
		os.Exit(1)
	}
	fmt.Printf("%sUpdate successful! Please restart jbmdb to use the new version if it doesn't start automatically`%s\n", postgres.ColorGreen, postgres.ColorReset)
}

func writeEnvFile(content string, isPostgres bool) {
	// Read existing .env file if it exists
	existingContent := ""
	if data, err := os.ReadFile(".env"); err == nil {
		existingContent = string(data)
	}

	// Combine existing content with new content
	if isPostgres {
		// Remove existing PostgreSQL config
		lines := strings.Split(existingContent, "\n")
		filtered := []string{}
		for _, line := range lines {
			if !strings.HasPrefix(line, "POSTGRES_") {
				filtered = append(filtered, line)
			}
		}
		existingContent = strings.Join(filtered, "\n")
	} else {
		// Remove existing ScyllaDB config
		lines := strings.Split(existingContent, "\n")
		filtered := []string{}
		for _, line := range lines {
			if !strings.HasPrefix(line, "SCYLLA_") {
				filtered = append(filtered, line)
			}
		}
		existingContent = strings.Join(filtered, "\n")
	}

	// Write combined content to .env file
	finalContent := strings.TrimSpace(existingContent) + "\n\n" + strings.TrimSpace(content) + "\n"
	if err := os.WriteFile(".env", []byte(finalContent), 0644); err != nil {
		log.Fatalf("%sError writing .env file: %v%s\n", postgres.ColorRed, err, postgres.ColorReset)
	}

	fmt.Printf("\n%sConfiguration saved to .env file%s\n", postgres.ColorGreen, postgres.ColorReset)
}
