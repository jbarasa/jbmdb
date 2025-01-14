// Package config provides configuration management for database migrations
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	configFile = ".jbmdb.conf"
)

// Config represents the base configuration structure
type Config struct {
	MigrationPath string `json:"migration_path"`
	SQLFolder     string `json:"sql_folder,omitempty"`
	CQLFolder     string `json:"cql_folder,omitempty"`
}

// PostgresConfig represents PostgreSQL specific configuration
type PostgresConfig struct {
	MigrationPath string `json:"migration_path"`
	SQLFolder     string `json:"sql_folder"`
	Host          string `json:"host"`
	Port          string `json:"port"`
	User          string `json:"user"`
	Password      string `json:"password"`
	DBName        string `json:"dbname"`
	SuperUser     string `json:"super_user"`
	SuperPass     string `json:"super_pass"`
}

// MySQLConfig represents MySQL/MariaDB specific configuration
type MySQLConfig struct {
	MigrationPath string `json:"migration_path"`
	SQLFolder     string `json:"sql_folder"`
	Host          string `json:"host"`
	Port          string `json:"port"`
	User          string `json:"user"`
	Password      string `json:"password"`
	DBName        string `json:"dbname"`
	SuperUser     string `json:"super_user"`
	SuperPass     string `json:"super_pass"`
}

// ScyllaConfig represents CQL database (Cassandra/ScyllaDB) specific configuration
type ScyllaConfig struct {
	MigrationPath string   `json:"migration_path"`
	CQLFolder     string   `json:"cql_folder"`
	Hosts         []string `json:"hosts"`
	Port          int      `json:"port"`         // Using int as gocql expects port as integer
	Keyspace      string   `json:"keyspace"`
	User          string   `json:"user"`
	Password      string   `json:"password"`
	SuperUser     string   `json:"super_user"`
	SuperPass     string   `json:"super_pass"`
	Datacenter    string   `json:"datacenter"`   // For NetworkTopologyStrategy
	Consistency   string   `json:"consistency"`  // For custom consistency levels
}

// JBMDBConfig represents the complete configuration
type JBMDBConfig struct {
	Postgres *PostgresConfig `json:"postgres,omitempty"`
	Scylla   *ScyllaConfig   `json:"scylla,omitempty"`
	MySQL    *MySQLConfig    `json:"mysql,omitempty"`
}

var currentConfig *JBMDBConfig

// LoadConfig loads configuration from file
func LoadConfig[T Config | PostgresConfig | ScyllaConfig | MySQLConfig](configType string) (*T, error) {
	if err := loadConfigFile(); err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	var config T
	switch configType {
	case "postgres":
		if currentConfig.Postgres == nil {
			// Return default config if not configured
			return createDefaultConfig[T](configType)
		}
		if pg, ok := any(&config).(*PostgresConfig); ok {
			*pg = *currentConfig.Postgres
		}
	case "cql":
		if currentConfig.Scylla == nil {
			// Return default config if not configured
			return createDefaultConfig[T](configType)
		}
		if sc, ok := any(&config).(*ScyllaConfig); ok {
			*sc = *currentConfig.Scylla
		}
	case "mysql":
		if currentConfig.MySQL == nil {
			// Return default config if not configured
			return createDefaultConfig[T](configType)
		}
		if my, ok := any(&config).(*MySQLConfig); ok {
			*my = *currentConfig.MySQL
		}
	default:
		return nil, fmt.Errorf("invalid config type: %s", configType)
	}

	return &config, nil
}

// SaveConfig saves configuration to file and creates necessary directories
func SaveConfig[T Config | PostgresConfig | ScyllaConfig | MySQLConfig](config T, configType string) error {
	if err := loadConfigFile(); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load existing config: %w", err)
	}

	if currentConfig == nil {
		currentConfig = &JBMDBConfig{}
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Create migration directories based on config type
	var migrationPath, subFolder string
	switch configType {
	case "postgres":
		if pg, ok := any(config).(*PostgresConfig); ok {
			currentConfig.Postgres = pg
			migrationPath = pg.MigrationPath
			subFolder = pg.SQLFolder
		} else if pg, ok := any(config).(PostgresConfig); ok {
			currentConfig.Postgres = &pg
			migrationPath = pg.MigrationPath
			subFolder = pg.SQLFolder
		}
	case "cql":
		if sc, ok := any(config).(*ScyllaConfig); ok {
			currentConfig.Scylla = sc
			migrationPath = sc.MigrationPath
			subFolder = sc.CQLFolder
		} else if sc, ok := any(config).(ScyllaConfig); ok {
			currentConfig.Scylla = &sc
			migrationPath = sc.MigrationPath
			subFolder = sc.CQLFolder
		}
	case "mysql":
		if my, ok := any(config).(*MySQLConfig); ok {
			currentConfig.MySQL = my
			migrationPath = my.MigrationPath
			subFolder = my.SQLFolder
		} else if my, ok := any(config).(MySQLConfig); ok {
			currentConfig.MySQL = &my
			migrationPath = my.MigrationPath
			subFolder = my.SQLFolder
		}
	default:
		return fmt.Errorf("invalid config type: %s", configType)
	}

	// Create migration directory and its subdirectory using absolute paths
	if migrationPath != "" {
		// Convert to absolute path if relative
		if !filepath.IsAbs(migrationPath) {
			migrationPath = filepath.Join(cwd, migrationPath)
		}

		// Create migration directory if it doesn't exist
		if _, err := os.Stat(migrationPath); os.IsNotExist(err) {
			if err := os.MkdirAll(migrationPath, 0755); err != nil {
				return fmt.Errorf("failed to create migration directory: %w", err)
			}
			fmt.Printf("Created migration directory: %s\n", migrationPath)
		}

		if subFolder != "" {
			subFolderPath := filepath.Join(migrationPath, subFolder)
			// Create subfolder if it doesn't exist
			if _, err := os.Stat(subFolderPath); os.IsNotExist(err) {
				if err := os.MkdirAll(subFolderPath, 0755); err != nil {
					return fmt.Errorf("failed to create %s subdirectory: %w", subFolder, err)
				}
				fmt.Printf("Created %s subdirectory: %s\n", subFolder, subFolderPath)
			}
		}
	}

	data, err := json.MarshalIndent(currentConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// loadConfigFile loads the configuration file into currentConfig
func loadConfigFile() error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			currentConfig = &JBMDBConfig{}
			return nil
		}
		return err
	}

	currentConfig = &JBMDBConfig{}
	if err := json.Unmarshal(data, currentConfig); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

// createDefaultConfig creates a default configuration
func createDefaultConfig[T Config | PostgresConfig | ScyllaConfig | MySQLConfig](configType string) (*T, error) {
	var config T

	switch configType {
	case "postgres":
		if pg, ok := any(&config).(*PostgresConfig); ok {
			*pg = PostgresConfig{
				MigrationPath: "migrations/postgres",
				SQLFolder:     "sql",
				Host:          "localhost",
				Port:          "5432",
				User:          "postgres",
				Password:      "",
				DBName:        "postgres",
				SuperUser:     "postgres",
				SuperPass:     "",
			}
		}
	case "cql":
		if sc, ok := any(&config).(*ScyllaConfig); ok {
			*sc = ScyllaConfig{
				MigrationPath: "migrations/cql",
				CQLFolder:     "cql",
				Port:          9042,
				Hosts:         []string{"localhost"},
				Keyspace:      "system",
				User:          "",
				Password:      "",
				SuperUser:     "",
				SuperPass:     "",
				Datacenter:    "",
				Consistency:   "",
			}
		}
	case "mysql":
		if my, ok := any(&config).(*MySQLConfig); ok {
			*my = MySQLConfig{
				MigrationPath: "migrations/mysql",
				SQLFolder:     "sql",
				Host:          "localhost",
				Port:          "3306",
				User:          "root",
				Password:      "",
				DBName:        "mysql",
				SuperUser:     "root",
				SuperPass:     "",
			}
		}
	}

	return &config, nil
}

// SaveFullConfig saves a complete configuration
func SaveFullConfig(config *JBMDBConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	currentConfig = config
	return nil
}
