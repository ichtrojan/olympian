package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ichtrojan/olympian"
	"github.com/spf13/cobra"
)

var (
	dbDriver      string
	dbDsn         string
	migrationPath string
	useEnv        bool
)

func init() {
	migrateCmd.PersistentFlags().StringVar(&dbDriver, "driver", "", "Database driver (sqlite3, postgres, mysql)")
	migrateCmd.PersistentFlags().StringVar(&dbDsn, "dsn", "", "Database connection string (for SQLite)")
	migrateCmd.PersistentFlags().StringVar(&migrationPath, "path", "./migrations", "Path to migrations directory")
	migrateCmd.PersistentFlags().BoolVar(&useEnv, "env", true, "Use .env file for database configuration (default: true)")

	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateRollbackCmd)
	migrateCmd.AddCommand(migrateStatusCmd)
	migrateCmd.AddCommand(migrateResetCmd)
	migrateCmd.AddCommand(migrateRefreshCmd)
	migrateCmd.AddCommand(migrateFreshCmd)
	migrateCmd.AddCommand(migrateCreateCmd)

	rootCmd.AddCommand(migrateCmd)
}

var migrateCmd = &cobra.Command{
	Use:          "migrate",
	Short:        "Run database migrations",
	RunE:         runMigrate,
	SilenceUsage: true,
}

var migrateUpCmd = &cobra.Command{
	Use:          "up",
	Short:        "Run all pending migrations",
	RunE:         runMigrate,
	SilenceUsage: true,
}

var migrateRollbackCmd = &cobra.Command{
	Use:          "rollback",
	Short:        "Rollback the last batch of migrations",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWithCmdMigrate("rollback")
	},
}

var migrateStatusCmd = &cobra.Command{
	Use:          "status",
	Short:        "Show migration status",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWithCmdMigrate("status")
	},
}

var migrateResetCmd = &cobra.Command{
	Use:          "reset",
	Short:        "Rollback all migrations",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWithCmdMigrate("reset")
	},
}

var migrateRefreshCmd = &cobra.Command{
	Use:          "refresh",
	Short:        "Rollback all migrations and re-run them",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWithCmdMigrate("refresh")
	},
}

var migrateFreshCmd = &cobra.Command{
	Use:          "fresh",
	Short:        "Drop all tables and re-run all migrations",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWithCmdMigrate("fresh")
	},
}

var migrateCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new migration file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return createMigration(args[0])
	},
}

func runMigrate(cmd *cobra.Command, args []string) error {
	return runWithCmdMigrate("up")
}

func runWithCmdMigrate(command string) error {
	// Check if cmd/migrate/main.go exists
	if _, err := os.Stat("cmd/migrate/main.go"); err != nil {
		// Doesn't exist - create it automatically
		fmt.Println("Initializing Olympian (creating cmd/migrate/main.go)...")
		if err := initializeMigrateFile(); err != nil {
			return fmt.Errorf("failed to initialize: %w", err)
		}

		// Also create seeders directory so the import doesn't fail
		if err := os.MkdirAll("seeders", 0755); err != nil {
			return fmt.Errorf("failed to create seeders directory: %w", err)
		}
		seederInit := "package seeders\n"
		if err := os.WriteFile("seeders/init.go", []byte(seederInit), 0644); err != nil {
			return fmt.Errorf("failed to create seeders init: %w", err)
		}

		fmt.Println("Created cmd/migrate/main.go")
		fmt.Println()
	}

	// Map CLI commands to the generated main.go commands
	// "rollback" maps to "down" in the generated template
	if command == "rollback" {
		command = "down"
	}

	runCmd := exec.Command("go", "run", "cmd/migrate/main.go", command)
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr
	runCmd.Env = os.Environ()
	return runCmd.Run()
}

func initializeMigrateFile() error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Read go.mod to get module name
	goModPath := filepath.Join(cwd, "go.mod")
	goModContent, err := os.ReadFile(goModPath)
	if err != nil {
		return fmt.Errorf("failed to read go.mod: %w (make sure you're in a Go project)", err)
	}

	// Extract module name from go.mod
	var moduleName string
	lines := string(goModContent)
	for i := 0; i < len(lines); i++ {
		if i+7 < len(lines) && lines[i:i+7] == "module " {
			start := i + 7
			end := start
			for end < len(lines) && lines[end] != '\n' && lines[end] != '\r' {
				end++
			}
			moduleName = lines[start:end]
			break
		}
	}

	if moduleName == "" {
		return fmt.Errorf("could not find module name in go.mod")
	}

	// Create cmd/migrate directory
	migrateDir := filepath.Join(cwd, "cmd", "migrate")
	if err := os.MkdirAll(migrateDir, 0755); err != nil {
		return fmt.Errorf("failed to create cmd/migrate directory: %w", err)
	}

	// Create main.go
	mainGoPath := filepath.Join(migrateDir, "main.go")

	template := fmt.Sprintf(`package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/ichtrojan/olympian"
	"github.com/joho/godotenv"

	_ "%s/migrations"
	_ "%s/seeders"
)

func main() {
	_ = godotenv.Load()

	dbDriver := getEnv("DB_DRIVER", "")
	if dbDriver == "" {
		fmt.Println("DB_DRIVER environment variable not set")
		os.Exit(1)
	}

	var dsn string
	var dialect olympian.Dialect

	switch dbDriver {
	case "mysql":
		dbHost := getEnv("DB_HOST", "127.0.0.1")
		dbPort := getEnv("DB_PORT", "3306")
		dbUser := getEnv("DB_USER", "root")
		dbPass := getEnv("DB_PASS", "")
		dbName := getEnv("DB_NAME", "")
		dsn = fmt.Sprintf("%%s:%%s@tcp(%%s:%%s)/%%s?parseTime=true", dbUser, dbPass, dbHost, dbPort, dbName)
		dialect = olympian.MySQL()
	case "postgres":
		dbHost := getEnv("DB_HOST", "localhost")
		dbPort := getEnv("DB_PORT", "5432")
		dbUser := getEnv("DB_USER", "postgres")
		dbPass := getEnv("DB_PASS", "")
		dbName := getEnv("DB_NAME", "")
		sslMode := getEnv("DB_SSLMODE", "disable")
		if dbPass == "" {
			dsn = fmt.Sprintf("postgres://%%s@%%s:%%s/%%s?sslmode=%%s", dbUser, dbHost, dbPort, dbName, sslMode)
		} else {
			dsn = fmt.Sprintf("postgres://%%s:%%s@%%s:%%s/%%s?sslmode=%%s", dbUser, dbPass, dbHost, dbPort, dbName, sslMode)
		}
		dialect = olympian.Postgres()
	case "sqlite3":
		dsn = getEnv("DB_DSN", "./database.db")
		dialect = olympian.SQLite()
	default:
		fmt.Printf("Unsupported database driver: %%s\n", dbDriver)
		os.Exit(1)
	}

	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		fmt.Printf("Failed to connect to database: %%v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		fmt.Printf("Failed to ping database: %%v\n", err)
		os.Exit(1)
	}

	migrator := olympian.NewMigrator(db, dialect)
	if err := migrator.Init(); err != nil {
		fmt.Printf("Failed to initialize migrator: %%v\n", err)
		os.Exit(1)
	}

	olympian.SetDB(db, dialect)
	migrations := olympian.GetMigrations()

	if len(os.Args) < 2 {
		fmt.Println("Usage: migrate [command]")
		fmt.Println("Commands:")
		fmt.Println("  up        Run all pending migrations")
		fmt.Println("  down      Rollback the last batch of migrations")
		fmt.Println("  status    Show migration status")
		fmt.Println("  reset     Rollback all migrations")
		fmt.Println("  refresh   Rollback all and re-run migrations")
		fmt.Println("  fresh     Drop all tables and re-run migrations")
		fmt.Println("  seed      Run all seeders")
		os.Exit(1)
	}

	seederRunner := olympian.NewSeederRunner(db, dialect)

	switch os.Args[1] {
	case "up":
		fmt.Println("Running migrations...")
		if err := migrator.Migrate(migrations); err != nil {
			fmt.Printf("Migration failed: %%v\n", err)
			os.Exit(1)
		}
		fmt.Println("Migrations completed")
	case "down":
		fmt.Println("Rolling back last batch...")
		if err := migrator.Rollback(migrations, 1); err != nil {
			fmt.Printf("Rollback failed: %%v\n", err)
			os.Exit(1)
		}
		fmt.Println("Rollback completed")
	case "status":
		if err := migrator.Status(migrations); err != nil {
			fmt.Printf("Status check failed: %%v\n", err)
			os.Exit(1)
		}
	case "reset":
		fmt.Println("Resetting all migrations...")
		if err := migrator.Reset(migrations); err != nil {
			fmt.Printf("Reset failed: %%v\n", err)
			os.Exit(1)
		}
		fmt.Println("Reset completed")
	case "refresh":
		fmt.Println("Refreshing migrations...")
		if err := migrator.Refresh(migrations); err != nil {
			fmt.Printf("Refresh failed: %%v\n", err)
			os.Exit(1)
		}
		fmt.Println("Refresh completed")
	case "fresh":
		fmt.Println("Dropping all tables and re-running migrations...")
		if err := migrator.Fresh(migrations); err != nil {
			fmt.Printf("Fresh migration failed: %%v\n", err)
			os.Exit(1)
		}
		fmt.Println("Fresh migration completed")
	case "seed":
		fmt.Println("Running seeders...")
		seeders := olympian.GetSeeders()
		if len(os.Args) > 2 {
			if err := seederRunner.RunSpecific(seeders, os.Args[2:]...); err != nil {
				fmt.Printf("Seeding failed: %%v\n", err)
				os.Exit(1)
			}
		} else {
			if err := seederRunner.Run(seeders); err != nil {
				fmt.Printf("Seeding failed: %%v\n", err)
				os.Exit(1)
			}
		}
		fmt.Println("Seeding completed")
	default:
		fmt.Printf("Unknown command: %%s\n", os.Args[1])
		os.Exit(1)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
`, moduleName, moduleName)

	return os.WriteFile(mainGoPath, []byte(template), 0644)
}

func createMigration(name string) error {
	if migrationPath == "" {
		migrationPath = "./migrations"
	}

	if err := os.MkdirAll(migrationPath, 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	// Format the migration name
	formattedName := formatMigrationName(name)

	// Extract table name from migration name (remove create_ prefix and _table suffix)
	tableName := extractTableName(formattedName)

	timestamp := fmt.Sprintf("%d", olympian.GetTimestamp())
	filename := fmt.Sprintf("%s_%s.go", timestamp, formattedName)
	filePath := filepath.Join(migrationPath, filename)

	template := fmt.Sprintf(`package migrations

import (
	"github.com/ichtrojan/olympian"
)

func init() {
	olympian.RegisterMigration(olympian.Migration{
		Name: "%s_%s",
		Up: func() error {
			return olympian.Table("%s").Create(func() {
				olympian.Uuid("id").Primary()
				olympian.Timestamps()
			})
		},
		Down: func() error {
			return olympian.Table("%s").Drop()
		},
	})
}
`, timestamp, formattedName, tableName, tableName)

	if err := os.WriteFile(filePath, []byte(template), 0644); err != nil {
		return fmt.Errorf("failed to create migration file: %w", err)
	}

	fmt.Printf("Created migration: %s\n", filePath)
	return nil
}

func formatMigrationName(name string) string {
	// Check if it already has the create_ prefix
	hasCreatePrefix := len(name) >= 7 && name[:7] == "create_"

	// Check if it already has _table suffix
	hasTableSuffix := len(name) >= 6 && name[len(name)-6:] == "_table"

	// Add create_ prefix if missing
	if !hasCreatePrefix {
		name = "create_" + name
	}

	// Add _table suffix if missing
	if !hasTableSuffix {
		name = name + "_table"
	}

	return name
}

func extractTableName(migrationName string) string {
	tableName := migrationName

	// Remove "create_" prefix if present
	if len(tableName) >= 7 && tableName[:7] == "create_" {
		tableName = tableName[7:]
	}

	// Remove "_table" suffix if present
	if len(tableName) >= 6 && tableName[len(tableName)-6:] == "_table" {
		tableName = tableName[:len(tableName)-6]
	}

	return tableName
}
