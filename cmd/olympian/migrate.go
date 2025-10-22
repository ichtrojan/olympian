package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ichtrojan/olympian"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
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
		return runWithGeneratedRunner("rollback")
	},
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWithGeneratedRunner("status")
	},
}

var migrateResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Rollback all migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWithGeneratedRunner("reset")
	},
}

var migrateFreshCmd = &cobra.Command{
	Use:   "fresh",
	Short: "Drop all tables and re-run all migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWithGeneratedRunner("fresh")
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
	return runWithGeneratedRunner("migrate")
}

func runWithGeneratedRunner(command string) error {
	// Read go.mod to get module name
	goModContent, err := os.ReadFile("go.mod")
	if err != nil {
		return fmt.Errorf("failed to read go.mod: %w (make sure you're in a Go project root)", err)
	}

	// Extract module name
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

	// Create temporary directory
	tmpDir := filepath.Join(os.TempDir(), "olympian-runner")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate temporary main.go
	mainContent := fmt.Sprintf(`package main

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

	switch "%s" {
	case "migrate":
		if err := migrator.Migrate(migrations); err != nil {
			log.Fatalf("Failed to run migrations: %%v", err)
		}
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
	}
}
`, moduleName, command)

	tmpMainPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(tmpMainPath, []byte(mainContent), 0644); err != nil {
		return fmt.Errorf("failed to write temporary main.go: %w", err)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Run go run with the temporary file
	runCmd := exec.Command("go", "run", tmpMainPath)
	runCmd.Dir = cwd
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr
	runCmd.Env = os.Environ()

	return runCmd.Run()
}

func connectDB() (*sql.DB, olympian.Dialect, error) {
	var driver, dsn string

	if useEnv && dbDriver == "" {
		if err := godotenv.Load(); err != nil {
			fmt.Println("No .env file found, using command line flags")
		}

		dbHost := os.Getenv("DB_HOST")
		dbPort := os.Getenv("DB_PORT")
		dbName := os.Getenv("DB_NAME")
		dbUser := os.Getenv("DB_USER")
		dbPass := os.Getenv("DB_PASS")
		dbDriverEnv := os.Getenv("DB_DRIVER")

		if dbDriverEnv != "" {
			driver = dbDriverEnv
		} else if dbHost != "" {
			driver = "mysql"
		}

		switch driver {
		case "postgres":
			dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
				dbHost, dbPort, dbUser, dbPass, dbName)
		case "mysql":
			dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
				dbUser, dbPass, dbHost, dbPort, dbName)
		}
	}

	if dbDriver != "" {
		driver = dbDriver
	}

	if dbDsn != "" {
		dsn = dbDsn
	}

	if driver == "" {
		return nil, nil, fmt.Errorf("database driver not specified. Use --driver flag or set DB_DRIVER in .env")
	}

	if dsn == "" {
		return nil, nil, fmt.Errorf("database connection string not specified. Use --dsn flag or configure .env file")
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	var dialect olympian.Dialect
	switch driver {
	case "postgres":
		dialect = &olympian.PostgresDialect{}
	case "mysql":
		dialect = &olympian.MySQLDialect{}
	case "sqlite3":
		dialect = &olympian.SQLiteDialect{}
	default:
		return nil, nil, fmt.Errorf("unsupported database driver: %s", driver)
	}

	return db, dialect, nil
}

func loadMigrations() ([]olympian.Migration, error) {
	// This function is not used when running migrations via the CLI
	// Instead, we use the generated runner approach
	return olympian.GetMigrations(), nil
}

func createMigration(name string) error {
	if migrationPath == "" {
		migrationPath = "./migrations"
	}

	if err := os.MkdirAll(migrationPath, 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	timestamp := fmt.Sprintf("%d", olympian.GetTimestamp())
	filename := fmt.Sprintf("%s_%s.go", timestamp, name)
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
`, timestamp, name, name, name)

	if err := os.WriteFile(filePath, []byte(template), 0644); err != nil {
		return fmt.Errorf("failed to create migration file: %w", err)
	}

	fmt.Printf("Created migration: %s\n", filePath)
	return nil
}
