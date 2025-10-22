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
	migrateCmd.AddCommand(migrateFreshCmd)
	migrateCmd.AddCommand(migrateCreateCmd)

	rootCmd.AddCommand(migrateCmd)
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	RunE:  runMigrate,
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Run all pending migrations",
	RunE:  runMigrate,
}

var migrateRollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback the last batch of migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWithCmdMigrate("rollback")
	},
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWithCmdMigrate("status")
	},
}

var migrateResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Rollback all migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWithCmdMigrate("reset")
	},
}

var migrateFreshCmd = &cobra.Command{
	Use:   "fresh",
	Short: "Drop all tables and re-run all migrations",
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
	return runWithCmdMigrate("migrate")
}

func runWithCmdMigrate(command string) error {
	// Check if cmd/migrate/main.go exists
	if _, err := os.Stat("cmd/migrate/main.go"); err != nil {
		// Doesn't exist - create it automatically
		fmt.Println("Initializing Olympian (creating cmd/migrate/main.go)...")
		if err := initializeMigrateFile(); err != nil {
			return fmt.Errorf("failed to initialize: %w", err)
		}
		fmt.Println("✓ Created cmd/migrate/main.go")
		fmt.Println()
	}

	// Use the existing cmd/migrate/main.go
	var runCmd *exec.Cmd
	if command == "migrate" {
		// No argument needed for migrate - it's the default
		runCmd = exec.Command("go", "run", "cmd/migrate/main.go")
	} else {
		runCmd = exec.Command("go", "run", "cmd/migrate/main.go", command)
	}
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
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/ichtrojan/olympian"
	"github.com/joho/godotenv"

	_ "%s/migrations"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	dbDriver := os.Getenv("DB_DRIVER")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")

	if dbDriver == "" {
		log.Fatal("DB_DRIVER not set in .env")
	}

	var dsn string
	var dialect olympian.Dialect

	switch dbDriver {
	case "mysql":
		dsn = fmt.Sprintf("%%s:%%s@tcp(%%s:%%s)/%%s?parseTime=true", dbUser, dbPass, dbHost, dbPort, dbName)
		dialect = olympian.MySQL()
	case "postgres":
		dsn = fmt.Sprintf("host=%%s port=%%s user=%%s password=%%s dbname=%%s sslmode=disable", dbHost, dbPort, dbUser, dbPass, dbName)
		dialect = olympian.Postgres()
	case "sqlite3":
		dsn = os.Getenv("DB_DSN")
		if dsn == "" {
			dsn = "./database.db"
		}
		dialect = olympian.SQLite()
	default:
		log.Fatalf("Unsupported database driver: %%s", dbDriver)
	}

	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %%v", err)
	}
	defer db.Close()

	migrator := olympian.NewMigrator(db, dialect)
	if err := migrator.Init(); err != nil {
		log.Fatalf("Failed to initialize migrator: %%v", err)
	}

	migrations := olympian.GetMigrations()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "status":
			if err := migrator.Status(migrations); err != nil {
				log.Fatalf("Failed to get status: %%v", err)
			}
		case "rollback":
			if err := migrator.Rollback(migrations, 1); err != nil {
				log.Fatalf("Failed to rollback: %%v", err)
			}
			fmt.Println("Rollback completed successfully")
		case "reset":
			if err := migrator.Reset(migrations); err != nil {
				log.Fatalf("Failed to reset: %%v", err)
			}
			fmt.Println("Reset completed successfully")
		case "fresh":
			if err := migrator.Fresh(migrations); err != nil {
				log.Fatalf("Failed to fresh: %%v", err)
			}
			fmt.Println("Fresh migration completed successfully")
		default:
			fmt.Printf("Unknown command: %%s\n", os.Args[1])
			fmt.Println("Available commands: migrate (default), status, rollback, reset, fresh")
			os.Exit(1)
		}
	} else {
		if err := migrator.Migrate(migrations); err != nil {
			log.Fatalf("Failed to run migrations: %%v", err)
		}
		fmt.Println("Migrations completed successfully")
	}
}
`, moduleName)

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
