package cql

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/gocql/gocql"
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

// CreateKeyspace creates a new keyspace if it doesn't exist
func CreateKeyspace(cqlConfig *config.ScyllaConfig, replicationStrategy string, replicationFactor int) error {
	// Connect to Cassandra/ScyllaDB cluster
	cluster := gocql.NewCluster(cqlConfig.Hosts...)
	cluster.Port = cqlConfig.Port
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: cqlConfig.SuperUser,
		Password: cqlConfig.SuperPass,
	}
	
	// Set consistency level if specified
	if cqlConfig.Consistency != "" {
		level, err := gocql.ParseConsistencyWrapper(cqlConfig.Consistency)
		if err != nil {
			return fmt.Errorf("invalid consistency level: %v", err)
		}
		cluster.Consistency = level
	} else {
		cluster.Consistency = gocql.Quorum
	}

	session, err := cluster.CreateSession()
	if err != nil {
		return fmt.Errorf("error connecting to cluster: %v", err)
	}
	defer session.Close()

	// Check if keyspace exists
	var count int
	if err := session.Query(
		"SELECT COUNT(*) FROM system_schema.keyspaces WHERE keyspace_name = ?",
		cqlConfig.Keyspace).Scan(&count); err != nil {
		return fmt.Errorf("error checking keyspace existence: %v", err)
	}

	if count == 0 {
		// Create keyspace with specified replication strategy
		var query string
		switch replicationStrategy {
		case "SimpleStrategy":
			query = fmt.Sprintf(
				"CREATE KEYSPACE %s WITH REPLICATION = {'class': 'SimpleStrategy', 'replication_factor': %d}",
				cqlConfig.Keyspace, replicationFactor)
		case "NetworkTopologyStrategy":
			if cqlConfig.Datacenter == "" {
				return fmt.Errorf("datacenter must be specified for NetworkTopologyStrategy")
			}
			query = fmt.Sprintf(
				"CREATE KEYSPACE %s WITH REPLICATION = {'class': 'NetworkTopologyStrategy', '%s': %d}",
				cqlConfig.Keyspace, cqlConfig.Datacenter, replicationFactor)
		default:
			return fmt.Errorf("unsupported replication strategy: %s", replicationStrategy)
		}

		if err := session.Query(query).Exec(); err != nil {
			return fmt.Errorf("error creating keyspace: %v", err)
		}

		fmt.Printf("%sKeyspace '%s' created successfully with %s (RF: %d)%s\n",
			ColorGreen, cqlConfig.Keyspace, replicationStrategy, replicationFactor, ColorReset)
	} else {
		fmt.Printf("%sKeyspace '%s' already exists%s\n",
			ColorBlue, cqlConfig.Keyspace, ColorReset)
	}

	return nil
}

// CreateUser creates a new user if it doesn't exist and grants privileges
func CreateUser(cqlConfig *config.ScyllaConfig, privileges string) error {
	// Connect to Cassandra/ScyllaDB cluster
	cluster := gocql.NewCluster(cqlConfig.Hosts...)
	cluster.Port = cqlConfig.Port
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: cqlConfig.SuperUser,
		Password: cqlConfig.SuperPass,
	}
	
	// Set consistency level if specified
	if cqlConfig.Consistency != "" {
		level, err := gocql.ParseConsistencyWrapper(cqlConfig.Consistency)
		if err != nil {
			return fmt.Errorf("invalid consistency level: %v", err)
		}
		cluster.Consistency = level
	} else {
		cluster.Consistency = gocql.Quorum
	}

	session, err := cluster.CreateSession()
	if err != nil {
		return fmt.Errorf("error connecting to cluster: %v", err)
	}
	defer session.Close()

	// Check if user exists
	var count int
	if err := session.Query(
		"SELECT COUNT(*) FROM system_auth.roles WHERE role = ?",
		cqlConfig.User).Scan(&count); err != nil {
		return fmt.Errorf("error checking user existence: %v", err)
	}

	if count == 0 {
		// Create user
		if err := session.Query(
			fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s' NOSUPERUSER",
				cqlConfig.User, cqlConfig.Password)).Exec(); err != nil {
			return fmt.Errorf("error creating user: %v", err)
		}

		fmt.Printf("%sUser '%s' created successfully%s\n",
			ColorGreen, cqlConfig.User, ColorReset)
	} else {
		fmt.Printf("%sUser '%s' already exists%s\n",
			ColorBlue, cqlConfig.User, ColorReset)
	}

	// Grant privileges based on the specified level
	var grantCmd string
	switch privileges {
	case "read":
		grantCmd = fmt.Sprintf("GRANT SELECT ON KEYSPACE %s TO %s",
			cqlConfig.Keyspace, cqlConfig.User)
	case "write":
		grantCmd = fmt.Sprintf(
			"GRANT SELECT, MODIFY ON KEYSPACE %s TO %s",
			cqlConfig.Keyspace, cqlConfig.User)
	case "all":
		grantCmd = fmt.Sprintf("GRANT ALL PERMISSIONS ON KEYSPACE %s TO %s",
			cqlConfig.Keyspace, cqlConfig.User)
	case "admin":
		grantCmd = fmt.Sprintf("GRANT ALL PERMISSIONS ON ALL KEYSPACES TO %s",
			cqlConfig.User)
	default:
		return fmt.Errorf("invalid privilege level: %s", privileges)
	}

	if err := session.Query(grantCmd).Exec(); err != nil {
		return fmt.Errorf("error granting privileges: %v", err)
	}

	fmt.Printf("%sPrivileges '%s' granted to user '%s' on keyspace '%s'%s\n",
		ColorGreen, privileges, cqlConfig.User, cqlConfig.Keyspace, ColorReset)

	return nil
}

// Migration represents a database migration with its version, name, and CQL scripts for
// applying and rolling back the migration.
type Migration struct {
	Version int64  // Version number of the migration
	Name    string // Name of the migration
	UpCQL   string // CQL script for applying the migration
	DownCQL string // CQL script for rolling back the migration
}

// Path to the migration files.
var migrationPath string

// SetMigrationPath sets the path for migration files
func SetMigrationPath(path string) {
	migrationPath = path
}

// extractTableName extracts the table name from the migration name.
// This function removes common prefixes and suffixes from the migration name,
// and converts it to snake_case if necessary.
func extractTableName(name string) string {
	// Remove common prefixes like "create_" or "add_"
	name = strings.TrimPrefix(name, "create_")
	name = strings.TrimPrefix(name, "add_")

	// Remove common suffixes like "_table"
	name = strings.TrimSuffix(name, "_table")

	// Convert to snake_case if it's in CamelCase
	name = camelToSnakeCase(name)

	// Return the processed table name
	return name
}

// camelToSnakeCase converts a string from CamelCase to snake_case.
// For example, "CamelCase" becomes "camel_case".
func camelToSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			// Insert an underscore before uppercase letters except the first character
			result.WriteByte('_')
		}
		// Convert each character to lowercase and append to the result
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

// CreateMigration creates new migration file with the given name and current timestamp.
func CreateMigration(name string) error {
	// Extract table name from migration name
	tableName := extractTableName(name)

	// Check for duplicate table names
	if err := checkDuplicateTableName(tableName); err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("%s_%s.cql", timestamp, name)

	content := fmt.Sprintf(`-- Migration: %s

-- Up Migration
----------------------- Write your up migration here ----------------------------

CREATE TABLE IF NOT EXISTS %s (
    id uuid PRIMARY KEY,
    created_at timestamp,
    updated_at timestamp
);


-- Down Migration
----------------------- Write your down migration here ----------------------------

DROP TABLE IF EXISTS %s;`, name, strings.ToLower(tableName), strings.ToLower(tableName))

	// Create the migration file in the CQL folder within the migration path
	cqlPath := filepath.Join(migrationPath, "cql")
	if err := os.MkdirAll(cqlPath, 0755); err != nil {
		return fmt.Errorf("failed to create CQL directory: %w", err)
	}

	// Write the up and down migration file in the CQL folder
	filePath := filepath.Join(cqlPath, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create migration file: %w", err)
	}

	fmt.Printf("%sCreated migration file: %s%s\n", ColorGreen, filePath, ColorReset)
	return nil
}

// loadMigrations loads all migration files from the migration directory.
// It reads the directory, parses each migration file, and returns a slice of Migration structs.
func loadMigrations() ([]Migration, error) {
	// Get the CQL directory path
	cqlPath := filepath.Join(migrationPath, "cql")

	// Read the migration directory
	files, err := os.ReadDir(cqlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration directory: %w", err)
	}

	var migrations []Migration
	for _, file := range files {
		// Process only .cql files
		if filepath.Ext(file.Name()) == ".cql" {
			// Split the filename by underscores
			parts := strings.Split(file.Name(), "_")
			if len(parts) < 2 {
				continue // Skip files that don't have at least a version and name part
			}

			// Parse version and name from filename
			version := parseInt(parts[0])
			name := strings.TrimSuffix(strings.Join(parts[1:], "_"), filepath.Ext(file.Name()))

			// Read the content of the migration file
			content, err := os.ReadFile(filepath.Join(cqlPath, file.Name()))
			if err != nil {
				return nil, fmt.Errorf("failed to read migration file %s: %w", file.Name(), err)
			}

			// Split content into up and down migrations
			upDown := strings.Split(string(content), "-- Down Migration")
			if len(upDown) != 2 {
				return nil, fmt.Errorf("invalid migration format in file %s", file.Name())
			}

			// Extract UpCQL and DownCQL scripts from the content
			up := strings.TrimSpace(strings.TrimPrefix(upDown[0], "-- Up Migration"))
			down := strings.TrimSpace(upDown[1])

			// Append the parsed migration to the slice
			migrations = append(migrations, Migration{
				Version: version,
				Name:    name,
				UpCQL:   up,
				DownCQL: down,
			})
		}
	}

	// Sort migrations by version number in ascending order
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// Migrate applies all pending migrations to the database.
// It first creates the migrations table if it does not exist,
// then applies each migration in order.
func Migrate(session *gocql.Session) error {
	// Create the migrations table if it doesn't exist
	if err := createMigrationsTable(session); err != nil {
		return err
	}

	// Load all migrations from the migration directory
	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	// Apply each migration to the database
	for _, migration := range migrations {
		if err := applyMigration(session, migration); err != nil {
			return err
		}
	}

	return nil
}

// RollbackLast rolls back the most recently applied migration.
// It retrieves the latest migration version and applies the rollback operation.
func RollbackLast(session *gocql.Session) error {
	// Get the version of the most recently applied migration
	latestMigration, err := getLatestMigration(session)
	if err != nil {
		return err
	}

	// Check if there are no migrations to rollback
	if latestMigration == 0 {
		fmt.Println("No migrations to rollback")
		return nil
	}

	// Load all migrations from the migration directory
	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	var migrationToRollback Migration
	// Find the migration to rollback based on the latest migration version
	for _, m := range migrations {
		if m.Version == latestMigration {
			migrationToRollback = m
			break
		}
	}

	// Check if the migration to rollback is found
	if migrationToRollback.Version == 0 {
		return fmt.Errorf("migration %d not found", latestMigration)
	}

	// Apply the rollback operation
	if err := rollbackMigration(session, migrationToRollback); err != nil {
		return err
	}

	// Print confirmation of the rollback operation
	fmt.Printf("Rolled back migration: %d_%s\n", migrationToRollback.Version, migrationToRollback.Name)
	return nil
}

// RollbackSteps rolls back a specified number of migrations
func RollbackSteps(session *gocql.Session, steps int) error {
	// Get all applied migrations
	appliedMigrations, err := getAppliedMigrations(session)
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

		if err := rollbackMigration(session, migration); err != nil {
			fmt.Printf("%sFAILED%s\n", ColorRed, ColorReset)
			return fmt.Errorf("failed to rollback migration %d_%s: %w",
				migration.Version, migration.Name, err)
		}

		fmt.Printf("%sDONE%s\n", ColorGreen, ColorReset)
	}

	return nil
}

// getAppliedMigrations returns all applied migrations from the database
func getAppliedMigrations(session *gocql.Session) ([]Migration, error) {
	var migrations []Migration

	iter := session.Query(`SELECT version, name FROM migrations`).Iter()
	var version int64
	var name string

	for iter.Scan(&version, &name) {
		// Load migration file content
		filename := fmt.Sprintf("%d_%s.cql", version, name)
		filePath := filepath.Join(migrationPath, "cql", filename)

		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		// Split content into up and down migrations
		parts := strings.Split(string(content), "-- Down Migration")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid migration format in file %s", filename)
		}

		migrations = append(migrations, Migration{
			Version: version,
			Name:    name,
			DownCQL: strings.TrimSpace(parts[1]),
		})
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("error iterating migrations: %w", err)
	}

	return migrations, nil
}

// createMigrationsTable creates the migrations table if it doesn't exist.
// This table keeps track of the applied migrations.
func createMigrationsTable(session *gocql.Session) error {
	return session.Query(`
		CREATE TABLE IF NOT EXISTS migrations (
			version bigint PRIMARY KEY,
			name text,
			applied_at timestamp
		)
	`).Exec()
}

// applyMigration applies a single migration to the database.
// It executes the UpCQL script and records the migration in the migrations table.
func applyMigration(session *gocql.Session, migration Migration) error {
	applied, err := isMigrationApplied(session, migration.Version)
	if err != nil {
		return err
	}

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

	fmt.Printf("%s[MIGRATING]%s %s%d_%s%s... ",
		ColorBlue,
		ColorReset,
		ColorCyan,
		migration.Version,
		migration.Name,
		ColorReset,
	)

	statements := strings.Split(migration.UpCQL, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if err := session.Query(stmt).Exec(); err != nil {
			fmt.Printf("%sFAILED%s\n", ColorRed, ColorReset)
			return fmt.Errorf("failed to apply migration %d_%s: %w", migration.Version, migration.Name, err)
		}
	}

	if err := session.Query(`
		INSERT INTO migrations (version, name, applied_at) VALUES (?, ?, ?)
	`, migration.Version, migration.Name, time.Now()).Exec(); err != nil {
		fmt.Printf("%sFAILED%s\n", ColorRed, ColorReset)
		return fmt.Errorf("failed to record migration %d_%s: %w", migration.Version, migration.Name, err)
	}

	fmt.Printf("%sDONE%s\n", ColorGreen, ColorReset)

	return nil
}

// rollbackMigration rolls back a single migration
func rollbackMigration(session *gocql.Session, migration Migration) error {
	// Split the down migration into individual statements
	statements := strings.Split(migration.DownCQL, ";")

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		// Execute each statement
		if err := session.Query(stmt).Exec(); err != nil {
			return fmt.Errorf("failed to execute down migration: %w", err)
		}
	}

	// Remove migration record
	if err := session.Query(`
		DELETE FROM migrations WHERE version = ?
	`, migration.Version).Exec(); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	return nil
}

// isMigrationApplied checks if a migration with a given version has already been applied.
// It queries the migrations table to check if the version exists.
func isMigrationApplied(session *gocql.Session, version int64) (bool, error) {
	var count int
	if err := session.Query(`SELECT COUNT(*) FROM migrations WHERE version = ?`, version).Scan(&count); err != nil {
		return false, fmt.Errorf("failed to check if migration is applied: %w", err)
	}
	return count > 0, nil
}

// getLatestMigration gets the version of the latest applied migration.
// It queries the migrations table for the highest version number.
func getLatestMigration(session *gocql.Session) (int64, error) {
	var version int64
	if err := session.Query(`SELECT version FROM migrations ORDER BY version DESC LIMIT 1`).Scan(&version); err != nil {
		if err == gocql.ErrNotFound {
			// No migrations have been applied yet
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get latest migration: %w", err)
	}
	return version, nil
}

// ListMigrations retrieves and lists all migrations along with their status.
func ListMigrations(session *gocql.Session) error {
	// Load all migrations from files
	migrations, err := loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Get all applied migrations from the database
	appliedMigrations := make(map[int64]time.Time)
	iter := session.Query("SELECT version, applied_at FROM migrations").Iter()
	var version int64
	var appliedAt time.Time
	for iter.Scan(&version, &appliedAt) {
		appliedMigrations[version] = appliedAt
	}
	if err := iter.Close(); err != nil {
		return fmt.Errorf("failed to query migrations table: %w", err)
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

// parseInt converts a string to an integer.
// It uses Sscanf to parse the integer value from the string.
func parseInt(s string) int64 {
	var result int64
	fmt.Sscanf(s, "%d", &result)
	return result
}

// MigrateFresh drops all tables and reapplies all migrations
func MigrateFresh(session *gocql.Session) error {
	fmt.Printf("%s[FRESH]%s Dropping all tables...\n", ColorYellow, ColorReset)

	// Drop all user-created tables
	if err := dropAllTables(session); err != nil {
		return fmt.Errorf("failed to drop tables: %w", err)
	}

	fmt.Printf("%s[FRESH]%s All tables dropped successfully\n", ColorGreen, ColorReset)
	fmt.Printf("%s[FRESH]%s Reapplying all migrations...\n", ColorBlue, ColorYellow)

	// Reapply all migrations
	if err := Migrate(session); err != nil {
		return fmt.Errorf("failed to reapply migrations: %w", err)
	}

	return nil
}

// dropAllTables drops all user-created tables in the keyspace
func dropAllTables(session *gocql.Session) error {
	// Get the current keyspace name
	keyspace := session.Query(`SELECT keyspace_name FROM system_schema.tables WHERE table_name = 'migrations'`).Keyspace()

	// Query to get only user-created tables in the keyspace
	query := `SELECT table_name 
			 FROM system_schema.tables 
			 WHERE keyspace_name = ?`

	iter := session.Query(query, keyspace).Iter()
	var tableName string
	var tables []string

	// System keyspaces to ignore
	systemKeyspaces := map[string]bool{
		"system":             true,
		"system_schema":      true,
		"system_auth":        true,
		"system_distributed": true,
		"system_traces":      true,
	}

	// Collect all user-created table names
	for iter.Scan(&tableName) {
		// Skip system tables and migrations table
		if !systemKeyspaces[tableName] && !strings.HasPrefix(tableName, "system_") &&
			!strings.HasPrefix(tableName, "scylla_") && tableName != "migrations" {
			tables = append(tables, tableName)
		}
	}

	if err := iter.Close(); err != nil {
		return fmt.Errorf("failed to get tables: %w", err)
	}

	// Drop each user-created table
	for _, table := range tables {
		fmt.Printf("%s[DROP]%s Dropping table %s%s%s...",
			ColorYellow,
			ColorReset,
			ColorCyan,
			table,
			ColorReset,
		)

		if err := session.Query(fmt.Sprintf("DROP TABLE IF EXISTS %s", table)).Exec(); err != nil {
			fmt.Printf(" %sFAILED%s\n", ColorRed, ColorReset)
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
		fmt.Printf(" %sDONE%s\n", ColorGreen, ColorReset)
	}

	// Finally, drop the migrations table
	fmt.Printf("%s[DROP]%s Dropping migrations table...", ColorYellow, ColorReset)
	if err := session.Query("DROP TABLE IF EXISTS migrations").Exec(); err != nil {
		fmt.Printf(" %sFAILED%s\n", ColorRed, ColorReset)
		return fmt.Errorf("failed to drop migrations table: %w", err)
	}
	fmt.Printf(" %sDONE%s\n", ColorGreen, ColorReset)

	return nil
}
