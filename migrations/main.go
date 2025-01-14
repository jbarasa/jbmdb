// Package main provides a command-line tool for managing database migrations
// migrations, applying migrations, rolling them back, listing migrations, and performing fresh migrations.
// It supports PostgreSQL, MySQL/MariaDB, and Cassandra/ScyllaDB databases.

package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/gocql/gocql"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jbarasa/jbmdb/migrations/config"
	"github.com/jbarasa/jbmdb/migrations/cql"
	"github.com/jbarasa/jbmdb/migrations/mysql"
	"github.com/jbarasa/jbmdb/migrations/postgres"
	"github.com/jbarasa/jbmdb/migrations/update"
)

const (
	// Color codes
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"

	// Text styles
	textBold      = "\033[1m"
	textUnderline = "\033[4m"
)

// Version is set during build time
var Version = "dev"

func main() {
	// Load environment variables
	// godotenv.Load()

	if len(os.Args) < 2 {
		showUsage()
		os.Exit(1)
	}

	// Parse command-line flags
	flag.Parse()
	command := flag.Arg(0)

	// Handle special commands first
	switch command {
	case "config":
		initConfig()
		return
	case "update":
		handleUpdate()
		return
	case "version":
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
	case "cql", "cassandra":
		handleScylla(action)
	case "mysql":
		handleMySQL(action)
	default:
		fmt.Printf("%sError: Invalid database type. Use 'postgres', 'mysql', or 'cql'%s\n",
			postgres.ColorRed, postgres.ColorReset)
		os.Exit(1)
	}
}

func handlePostgres(action string) {
	pgConfig, err := config.LoadConfig[config.PostgresConfig]("postgres")
	if err != nil {
		log.Fatalf("%sError loading PostgreSQL config: %v%s\n",
			postgres.ColorRed, err, postgres.ColorReset)
	}

	// Set migration path
	postgres.SetMigrationPath(pgConfig.MigrationPath)

	// Handle different actions
	switch {
	case action == "init":
		initPostgresConfig()
		return
	case action == "create-db":
		if err := postgres.CreateDatabase(pgConfig); err != nil {
			log.Fatalf("%s%v%s\n", postgres.ColorRed, err, postgres.ColorReset)
		}
		return
	case strings.HasPrefix(action, "create-user"):
		parts := strings.Split(action, ":")
		if len(parts) != 2 {
			log.Fatalf("%sUsage: postgres-create-user:[read|write|all|admin]%s\n",
				postgres.ColorRed, postgres.ColorReset)
		}
		if err := postgres.CreateUser(pgConfig, parts[1]); err != nil {
			log.Fatalf("%s%v%s\n", postgres.ColorRed, err, postgres.ColorReset)
		}
		return
	case strings.HasPrefix(action, "rollback"):
		handlePostgresRollback(action, pgConfig)
		return
	}

	// Connect to database
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		pgConfig.User, pgConfig.Password, pgConfig.Host, pgConfig.Port, pgConfig.DBName)

	db, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("%sUnable to connect to PostgreSQL: %v%s\n",
			postgres.ColorRed, err, postgres.ColorReset)
	}
	defer db.Close()

	// Handle other actions
	switch action {
	case "migration":
		if flag.NArg() < 2 {
			fmt.Printf("%sError: Migration name is required%s\n",
				postgres.ColorRed, postgres.ColorReset)
			os.Exit(1)
		}
		name := flag.Arg(1)
		validateMigrationName(name)
		if err := postgres.CreateMigration(name); err != nil {
			log.Fatalf("%sFailed to create migration: %v%s\n",
				postgres.ColorRed, err, postgres.ColorReset)
		}

	case "migrate":
		if err := postgres.Migrate(db); err != nil {
			log.Fatalf("%sFailed to run migrations: %v%s\n",
				postgres.ColorRed, err, postgres.ColorReset)
		}
		fmt.Printf("%sMigrations completed successfully%s\n",
			postgres.ColorGreen, postgres.ColorReset)

	case "fresh":
		confirmFreshMigration()
		if err := postgres.MigrateFresh(db); err != nil {
			log.Fatalf("%sFailed to run fresh migrations: %v%s\n",
				postgres.ColorRed, err, postgres.ColorReset)
		}
		fmt.Printf("%sFresh migration completed successfully%s\n",
			postgres.ColorGreen, postgres.ColorReset)

	case "list":
		if err := postgres.ListMigrations(db); err != nil {
			log.Fatalf("%sFailed to list migrations: %v%s\n",
				postgres.ColorRed, err, postgres.ColorReset)
		}

	default:
		fmt.Printf("%sError: Unknown command: %s%s\n",
			postgres.ColorRed, action, postgres.ColorReset)
		os.Exit(1)
	}
}

func handlePostgresRollback(action string, pgConfig *config.PostgresConfig) {
	// Parse rollback steps
	parts := strings.Split(action, ":")
	steps := 1 // Default to 1 step

	if len(parts) > 1 {
		if parts[1] == "all" {
			steps = -1 // Special case for rolling back all migrations
		} else {
			var err error
			steps, err = strconv.Atoi(parts[1])
			if err != nil || steps < 1 {
				log.Fatalf("%sInvalid rollback steps: %s%s\n",
					postgres.ColorRed, parts[1], postgres.ColorReset)
			}
		}
	}

	// Connect to database
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		pgConfig.User, pgConfig.Password, pgConfig.Host, pgConfig.Port, pgConfig.DBName)

	db, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("%sUnable to connect to PostgreSQL: %v%s\n",
			postgres.ColorRed, err, postgres.ColorReset)
	}
	defer db.Close()

	// Handle rollback
	if err := postgres.RollbackSteps(db, steps); err != nil {
		log.Fatalf("%sFailed to rollback migrations: %v%s\n",
			postgres.ColorRed, err, postgres.ColorReset)
	}

	if steps == -1 {
		fmt.Printf("%sRolled back all migrations successfully%s\n",
			postgres.ColorGreen, postgres.ColorReset)
	} else {
		fmt.Printf("%sRolled back %d migration(s) successfully%s\n",
			postgres.ColorGreen, steps, postgres.ColorReset)
	}
}

func handleScylla(action string) {
	scyllaConfig, err := config.LoadConfig[config.ScyllaConfig]("cql")
	if err != nil {
		log.Fatalf("%sError loading CQL database config: %v%s\n",
			postgres.ColorRed, err, postgres.ColorReset)
	}

	switch {
	case action == "init":
		initScyllaConfig()
		return
	case strings.HasPrefix(action, "create-keyspace"):
		parts := strings.Split(action, ":")
		if len(parts) != 3 {
			log.Fatalf("%sUsage: cql-create-keyspace:[SimpleStrategy|NetworkTopologyStrategy]:[replication_factor]%s\n",
				cql.ColorRed, cql.ColorReset)
		}
		strategy := parts[1]
		factor, err := strconv.Atoi(parts[2])
		if err != nil {
			log.Fatalf("%sInvalid replication factor: %v%s\n",
				cql.ColorRed, err, cql.ColorReset)
		}
		if err := cql.CreateKeyspace(scyllaConfig, strategy, factor); err != nil {
			log.Fatalf("%s%v%s\n", cql.ColorRed, err, cql.ColorReset)
		}
		return
	case strings.HasPrefix(action, "create-user"):
		parts := strings.Split(action, ":")
		if len(parts) != 2 {
			log.Fatalf("%sUsage: cql-create-user:[read|write|all|admin]%s\n",
				cql.ColorRed, cql.ColorReset)
		}
		if err := cql.CreateUser(scyllaConfig, parts[1]); err != nil {
			log.Fatalf("%s%v%s\n", cql.ColorRed, err, cql.ColorReset)
		}
		return
	case strings.HasPrefix(action, "rollback"):
		handleScyllaRollback(action, scyllaConfig)
		return
	}

	// Create CQL session
	cluster := gocql.NewCluster(scyllaConfig.Hosts...)
	cluster.Keyspace = scyllaConfig.Keyspace
	cluster.Consistency = gocql.Quorum
	cluster.ProtoVersion = 4
	if scyllaConfig.User != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: scyllaConfig.User,
			Password: scyllaConfig.Password,
		}
	}

	session, err := cluster.CreateSession()
	if err != nil {
		log.Fatalf("%sUnable to connect to CQL database: %v%s\n",
			postgres.ColorRed, err, postgres.ColorReset)
	}
	defer session.Close()

	// Handle commands
	switch action {
	case "migration":
		if flag.NArg() < 2 {
			fmt.Printf("%sError: Migration name is required%s\n",
				postgres.ColorRed, postgres.ColorReset)
			os.Exit(1)
		}
		name := flag.Arg(1)
		validateMigrationName(name)
		if err := cql.CreateMigration(name); err != nil {
			log.Fatalf("%sFailed to create migration: %v%s\n",
				postgres.ColorRed, err, postgres.ColorReset)
		}

	case "migrate":
		if err := cql.Migrate(session); err != nil {
			log.Fatalf("%sFailed to run migrations: %v%s\n",
				postgres.ColorRed, err, postgres.ColorReset)
		}
		fmt.Printf("%sMigrations completed successfully%s\n",
			postgres.ColorGreen, postgres.ColorReset)

	case "fresh":
		confirmFreshMigration()
		if err := cql.MigrateFresh(session); err != nil {
			log.Fatalf("%sFailed to run fresh migrations: %v%s\n",
				postgres.ColorRed, err, postgres.ColorReset)
		}
		fmt.Printf("%sFresh migration completed successfully%s\n",
			postgres.ColorGreen, postgres.ColorReset)

	case "list":
		if err := cql.ListMigrations(session); err != nil {
			log.Fatalf("%sFailed to list migrations: %v%s\n",
				postgres.ColorRed, err, postgres.ColorReset)
		}

	default:
		fmt.Printf("%sError: Unknown command: %s%s\n",
			postgres.ColorRed, action, postgres.ColorReset)
		os.Exit(1)
	}
}

func handleScyllaRollback(action string, scyllaConfig *config.ScyllaConfig) {
	// Parse rollback steps
	parts := strings.Split(action, ":")
	steps := 1 // Default to 1 step

	if len(parts) > 1 {
		if parts[1] == "all" {
			steps = -1 // Special case for rolling back all migrations
		} else {
			var err error
			steps, err = strconv.Atoi(parts[1])
			if err != nil || steps < 1 {
				log.Fatalf("%sInvalid rollback steps: %s%s\n",
					postgres.ColorRed, parts[1], postgres.ColorReset)
			}
		}
	}

	// Create CQL session
	cluster := gocql.NewCluster(scyllaConfig.Hosts...)
	cluster.Keyspace = scyllaConfig.Keyspace
	cluster.Consistency = gocql.Quorum
	cluster.ProtoVersion = 4
	if scyllaConfig.User != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: scyllaConfig.User,
			Password: scyllaConfig.Password,
		}
	}

	session, err := cluster.CreateSession()
	if err != nil {
		log.Fatalf("%sUnable to connect to CQL database: %v%s\n",
			postgres.ColorRed, err, postgres.ColorReset)
	}
	defer session.Close()

	// Handle rollback
	if err := cql.RollbackSteps(session, steps); err != nil {
		log.Fatalf("%sFailed to rollback migrations: %v%s\n",
			postgres.ColorRed, err, postgres.ColorReset)
	}

	if steps == -1 {
		fmt.Printf("%sRolled back all migrations successfully%s\n",
			postgres.ColorGreen, postgres.ColorReset)
	} else {
		fmt.Printf("%sRolled back %d migration(s) successfully%s\n",
			postgres.ColorGreen, steps, postgres.ColorReset)
	}
}

func handleMySQL(action string) {
	myConfig, err := config.LoadConfig[config.MySQLConfig]("mysql")
	if err != nil {
		log.Fatalf("%sError loading MySQL config: %v%s\n",
			mysql.ColorRed, err, mysql.ColorReset)
	}

	switch {
	case action == "init":
		initMySQLConfig()
		return
	case action == "create-db":
		if err := mysql.CreateDatabase(myConfig); err != nil {
			log.Fatalf("%s%v%s\n", mysql.ColorRed, err, mysql.ColorReset)
		}
		return
	case strings.HasPrefix(action, "create-user"):
		parts := strings.Split(action, ":")
		if len(parts) != 2 {
			log.Fatalf("%sUsage: mysql-create-user:[read|write|all|admin]%s\n",
				mysql.ColorRed, mysql.ColorReset)
		}
		if err := mysql.CreateUser(myConfig, parts[1]); err != nil {
			log.Fatalf("%s%v%s\n", mysql.ColorRed, err, mysql.ColorReset)
		}
		return
	case strings.HasPrefix(action, "rollback"):
		handleMySQLRollback(action, myConfig)
		return
	}

	// Connect to database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?multiStatements=true&parseTime=true",
		myConfig.User, myConfig.Password, myConfig.Host, myConfig.Port, myConfig.DBName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("%sError connecting to MySQL: %v%s\n",
			mysql.ColorRed, err, mysql.ColorReset)
	}
	defer db.Close()

	// Handle different actions
	switch action {
	case "migrate":
		err = mysql.Migrate(db)
	case "fresh":
		err = mysql.MigrateFresh(db)
	case "list":
		err = mysql.ListMigrations(db)
	case "create":
		name := flag.Arg(1)
		if name == "" {
			log.Fatalf("%sError: Migration name is required%s\n",
				mysql.ColorRed, mysql.ColorReset)
		}
		err = mysql.CreateMigration(name)
	default:
		showUsage()
		os.Exit(1)
	}

	if err != nil {
		log.Fatalf("%sError: %v%s\n", mysql.ColorRed, err, mysql.ColorReset)
	}
}

func handleMySQLRollback(action string, myConfig *config.MySQLConfig) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?multiStatements=true&parseTime=true",
		myConfig.User, myConfig.Password, myConfig.Host, myConfig.Port, myConfig.DBName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("%sError connecting to MySQL: %v%s\n",
			mysql.ColorRed, err, mysql.ColorReset)
	}
	defer db.Close()

	if action == "rollback" {
		err = mysql.RollbackLast(db)
	} else {
		steps, err := strconv.Atoi(action[9:])
		if err != nil {
			log.Fatalf("%sError: Invalid rollback steps%s\n",
				mysql.ColorRed, mysql.ColorReset)
		}
		err = mysql.RollbackSteps(db, steps)
	}

	if err != nil {
		log.Fatalf("%sError: %v%s\n", mysql.ColorRed, err, mysql.ColorReset)
	}
}

func initMySQLConfig() {
	myConfig := getMySQLConfig()
	if err := config.SaveConfig(myConfig, "mysql"); err != nil {
		log.Fatalf("%sError saving MySQL config: %v%s\n",
			mysql.ColorRed, err, mysql.ColorReset)
	}
	fmt.Printf("%s[SUCCESS]%s MySQL configuration saved\n",
		mysql.ColorGreen, mysql.ColorReset)
}

func getMySQLConfig() config.MySQLConfig {
	defaultConfig := config.MySQLConfig{
		MigrationPath: "migrations/mysql",
		SQLFolder:     "sql",
		Host:          "localhost",
		Port:          "3306",
		User:          "root",
		Password:      "",
		DBName:        "mysql",
	}

	existingConfig, err := config.LoadConfig[config.MySQLConfig]("mysql")
	if err == nil && existingConfig != nil {
		defaultConfig = *existingConfig
	}

	printQuestion(fmt.Sprintf("Host [%s]: ", defaultConfig.Host))
	host := readInput(defaultConfig.Host)

	printQuestion(fmt.Sprintf("Port [%s]: ", defaultConfig.Port))
	port := readInput(defaultConfig.Port)

	printQuestion(fmt.Sprintf("Database [%s]: ", defaultConfig.DBName))
	dbname := readInput(defaultConfig.DBName)

	printQuestion(fmt.Sprintf("User [%s]: ", defaultConfig.User))
	user := readInput(defaultConfig.User)

	printQuestion(fmt.Sprintf("Password [%s]: ", maskPassword(defaultConfig.Password)))
	password := readInput(defaultConfig.Password)

	printQuestion(fmt.Sprintf("Migration Path [%s]: ", defaultConfig.MigrationPath))
	migrationPath := readInput(defaultConfig.MigrationPath)

	config := defaultConfig
	config.MigrationPath = migrationPath
	config.Host = host
	config.Port = port
	config.User = user
	config.Password = password
	config.DBName = dbname

	return config
}

func getPostgresConfig() config.PostgresConfig {
	defaultConfig := config.PostgresConfig{
		MigrationPath: "migrations/postgres",
		SQLFolder:     "sql",
		Host:          "localhost",
		Port:          "5432",
		User:          "postgres",
		Password:      "",
		DBName:        "postgres",
	}

	existingConfig, err := config.LoadConfig[config.PostgresConfig]("postgres")
	if err == nil && existingConfig != nil {
		defaultConfig = *existingConfig
	}

	printQuestion(fmt.Sprintf("Host [%s]: ", defaultConfig.Host))
	host := readInput(defaultConfig.Host)

	printQuestion(fmt.Sprintf("Port [%s]: ", defaultConfig.Port))
	port := readInput(defaultConfig.Port)

	printQuestion(fmt.Sprintf("Database [%s]: ", defaultConfig.DBName))
	dbname := readInput(defaultConfig.DBName)

	printQuestion(fmt.Sprintf("User [%s]: ", defaultConfig.User))
	user := readInput(defaultConfig.User)

	printQuestion(fmt.Sprintf("Password [%s]: ", maskPassword(defaultConfig.Password)))
	password := readInput(defaultConfig.Password)

	printQuestion(fmt.Sprintf("Migration Path [%s]: ", defaultConfig.MigrationPath))
	migrationPath := readInput(defaultConfig.MigrationPath)

	config := defaultConfig
	config.MigrationPath = migrationPath
	config.Host = host
	config.Port = port
	config.User = user
	config.Password = password
	config.DBName = dbname

	return config
}

func getScyllaConfig() config.ScyllaConfig {
	defaultConfig := config.ScyllaConfig{
		MigrationPath: "migrations/cql",
		CQLFolder:     "cql",
		Hosts:         []string{"localhost"},
		User:          "",
		Password:      "",
		Keyspace:      "system",
	}

	existingConfig, err := config.LoadConfig[config.ScyllaConfig]("cql")
	if err == nil && existingConfig != nil {
		defaultConfig = *existingConfig
	}

	printQuestion(fmt.Sprintf("Hosts (comma-separated) [%s]: ", strings.Join(defaultConfig.Hosts, ",")))
	hostsStr := readInput(strings.Join(defaultConfig.Hosts, ","))
	hosts := strings.Split(hostsStr, ",")

	printQuestion(fmt.Sprintf("Keyspace [%s]: ", defaultConfig.Keyspace))
	keyspace := readInput(defaultConfig.Keyspace)

	printQuestion(fmt.Sprintf("User [%s]: ", defaultString(defaultConfig.User, "<none>")))
	user := readInput(defaultConfig.User)

	printQuestion(fmt.Sprintf("Password [%s]: ", maskPassword(defaultConfig.Password)))
	password := readInput(defaultConfig.Password)

	printQuestion(fmt.Sprintf("Migration Path [%s]: ", defaultConfig.MigrationPath))
	migrationPath := readInput(defaultConfig.MigrationPath)

	config := defaultConfig
	config.MigrationPath = migrationPath
	config.Hosts = hosts
	config.User = user
	config.Password = password
	config.Keyspace = keyspace

	return config
}

// Helper function to mask password in display
func maskPassword(password string) string {
	if password == "" {
		return ""
	}
	return "********"
}

// Helper function to show empty string as specified default
func defaultString(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func readInput(defaultValue string) string {
	var value string
	fmt.Scanln(&value)
	if value == "" {
		return defaultValue
	}
	return strings.TrimSpace(value)
}

func initPostgresConfig() {
	pgConfig := getPostgresConfig()
	if err := config.SaveConfig(pgConfig, "postgres"); err != nil {
		fmt.Printf("%sError saving PostgreSQL configuration: %v%s\n", postgres.ColorRed, err, postgres.ColorReset)
		os.Exit(1)
	}
	fmt.Printf("\n%sConfiguration saved successfully%s\n", postgres.ColorGreen, postgres.ColorReset)
}

func initScyllaConfig() {
	scConfig := getScyllaConfig()
	if err := config.SaveConfig(scConfig, "cql"); err != nil {
		fmt.Printf("%sError saving ScyllaDB configuration: %v%s\n", postgres.ColorRed, err, postgres.ColorReset)
		os.Exit(1)
	}
	fmt.Printf("\n%sConfiguration saved successfully%s\n", postgres.ColorGreen, postgres.ColorReset)
}

func handleUpdate() {
	release, err := update.CheckForUpdates(Version)
	if err != nil {
		fmt.Printf("%sError checking for updates: %v%s\n", cql.ColorRed, err, cql.ColorReset)
		os.Exit(1)
	}
	if release == nil {
		fmt.Printf("%sYou are already running the latest version%s\n", cql.ColorGreen, cql.ColorReset)
		return
	}

	fmt.Printf("%sNew version %s available!%s\n", cql.ColorCyan, release.TagName, cql.ColorReset)
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

func validateMigrationName(name string) {
	if !strings.HasPrefix(name, "create_") || !strings.HasSuffix(name, "_table") {
		fmt.Printf("%sError: Migration name must follow format: create_<name>_table\n", postgres.ColorRed)
		fmt.Printf("Example: create_users_table, create_post_comments_table%s\n", postgres.ColorReset)
		os.Exit(1)
	}

	tableName := strings.TrimPrefix(name, "create_")
	tableName = strings.TrimSuffix(tableName, "_table")
	parts := strings.Split(tableName, "_")

	if len(parts) == 1 && !strings.HasSuffix(parts[0], "s") {
		fmt.Printf("%sError: Single table names should be plural\n", postgres.ColorRed)
		fmt.Printf("Example: 'create_user_table' should be 'create_users_table'%s\n", postgres.ColorReset)
		os.Exit(1)
	}

	if len(parts) > 1 && !strings.HasSuffix(parts[len(parts)-1], "s") {
		fmt.Printf("%sError: In relation tables, names after the first word should be plural\n", postgres.ColorRed)
		fmt.Printf("Example: 'create_user_comment_table' should be 'create_user_comments_table'%s\n", postgres.ColorReset)
		os.Exit(1)
	}
}

func confirmFreshMigration() {
	fmt.Printf("%s[WARNING]%s This will drop all tables and reapply all migrations.\n", postgres.ColorRed, postgres.ColorReset)
	fmt.Printf("Are you sure you want to continue? (y/N): ")

	var response string
	fmt.Scanln(&response)

	if strings.ToLower(response) != "y" {
		fmt.Printf("%sOperation cancelled%s\n", postgres.ColorYellow, postgres.ColorReset)
		os.Exit(0)
	}
}

func showUsage() {
	fmt.Printf(`
JBMDB Database Migration Tool

Usage: jbmdb <command>

Commands:
    config                Initialize configuration
    update                Update jbmdb to latest version
    version               Show version information

PostgreSQL Commands:
    postgres-migration <n>   Create a new PostgreSQL migration
    postgres-migrate       Run all pending PostgreSQL migrations
    postgres-rollback      Rollback the last PostgreSQL migration
    postgres-rollback:all  Rollback all PostgreSQL migrations
    postgres-rollback:<n>  Rollback n PostgreSQL migrations
    postgres-fresh         Drop all tables and reapply PostgreSQL migrations
    postgres-list          List all PostgreSQL migrations
    postgres-init          Initialize PostgreSQL configuration
    postgres-create-db     Create database if not exists
    postgres-create-user:[read|write|all|admin]  Create user with specified privileges

MySQL Commands:
    mysql-migration <n>     Create a new MySQL migration
    mysql-migrate         Run all pending MySQL migrations
    mysql-rollback        Rollback the last MySQL migration
    mysql-rollback:all    Rollback all MySQL migrations
    mysql-rollback:<n>    Rollback n MySQL migrations
    mysql-fresh           Drop all tables and reapply MySQL migrations
    mysql-list            List all MySQL migrations
    mysql-init            Initialize MySQL configuration
    mysql-create-db       Create database if not exists
    mysql-create-user:[read|write|all|admin]    Create user with specified privileges

CQL Commands (Cassandra/ScyllaDB):
    cql-migration <n>     Create a new CQL migration
    cql-migrate         Run all pending CQL migrations
    cql-rollback        Rollback the last CQL migration
    cql-rollback:all    Rollback all CQL migrations
    cql-rollback:<n>    Rollback n CQL migrations
    cql-fresh           Drop all tables and reapply CQL migrations
    cql-list            List all CQL migrations
    cql-init            Initialize CQL configuration
    cql-create-keyspace:[strategy]:[rf]  Create keyspace with replication
    cql-create-user:[read|write|all|admin]  Create user with specified privileges

Current Configuration:
  PostgreSQL migrations: migrations/postgres
  MySQL migrations:      migrations/mysql
  CQL migrations:        migrations/cql

Privilege Levels:
  read:   SELECT privileges only
  write:  SELECT, MODIFY privileges (SELECT, INSERT, UPDATE, DELETE for SQL)
  all:    All privileges on database/keyspace
  admin:  All privileges with GRANT OPTION

Replication Strategies (Cassandra/ScyllaDB):
  SimpleStrategy:           Single datacenter deployment
  NetworkTopologyStrategy: Multi-datacenter deployment
  RF: Replication Factor (number of copies)
`)
}

func initConfig() error {
	printHeader("Database Configuration")

	printQuestion("\nWhich databases would you like to configure?\n")
	printOption(1, "PostgreSQL only")
	printOption(2, "MySQL/MariaDB only")
	printOption(3, "Cassandra/ScyllaDB only")
	printOption(4, "All databases")
	printQuestion("Choose (1-4): ")

	var choice int
	_, err := fmt.Scanf("%d", &choice)
	if err != nil {
		return fmt.Errorf("invalid input: %v", err)
	}

	switch choice {
	case 1:
		printSubHeader("PostgreSQL Configuration")
		pgConfig := getPostgresConfig()
		if err := config.SaveConfig(pgConfig, "postgres"); err != nil {
			return fmt.Errorf("failed to save PostgreSQL config: %v", err)
		}
	case 2:
		printSubHeader("MySQL/MariaDB Configuration")
		mysqlConfig := getMySQLConfig()
		if err := config.SaveConfig(mysqlConfig, "mysql"); err != nil {
			return fmt.Errorf("failed to save MySQL config: %v", err)
		}
	case 3:
		printSubHeader("Cassandra/ScyllaDB Configuration")
		cqlConfig := getScyllaConfig()
		if err := config.SaveConfig(cqlConfig, "cql"); err != nil {
			return fmt.Errorf("failed to save CQL config: %v", err)
		}
	case 4:
		// Configure all databases
		printSubHeader("PostgreSQL Configuration")
		pgConfig := getPostgresConfig()
		if err := config.SaveConfig(pgConfig, "postgres"); err != nil {
			return fmt.Errorf("failed to save PostgreSQL config: %v", err)
		}

		fmt.Println() // Add a blank line between configurations
		printSubHeader("MySQL/MariaDB Configuration")
		mysqlConfig := getMySQLConfig()
		if err := config.SaveConfig(mysqlConfig, "mysql"); err != nil {
			return fmt.Errorf("failed to save MySQL config: %v", err)
		}

		fmt.Println() // Add a blank line between configurations
		printSubHeader("Cassandra/ScyllaDB Configuration")
		cqlConfig := getScyllaConfig()
		if err := config.SaveConfig(cqlConfig, "cql"); err != nil {
			return fmt.Errorf("failed to save CQL config: %v", err)
		}
	default:
		return fmt.Errorf("%sinvalid choice: %d. Please choose between 1-4%s", colorRed, choice, colorReset)
	}

	return nil
}

func printHeader(text string) {
	fmt.Printf("\n%s%s%s%s\n", colorBlue, textBold, text, colorReset)
	fmt.Println(strings.Repeat("=", len(text)))
}

func printSubHeader(text string) {
	fmt.Printf("\n%s%s%s%s\n", colorPurple, textBold, text, colorReset)
	fmt.Println(strings.Repeat("-", len(text)))
}

func printQuestion(text string) {
	fmt.Printf("%s%s%s", colorCyan, text, colorReset)
}

func printOption(num int, text string) {
	fmt.Printf("%s%d%s. %s\n", colorGreen, num, colorReset, text)
}
