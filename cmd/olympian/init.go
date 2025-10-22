package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Olympian in your project",
	Long: `Creates a cmd/migrate/main.go file in your project that imports your migrations.
This allows you to run migrations using 'go run cmd/migrate/main.go'.`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
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
		if lines[i:] >= "module " && i+7 < len(lines) {
			start := i + 7
			end := start
			for end < len(lines) && lines[end] != '\n' {
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

	// Check if file already exists
	if _, err := os.Stat(mainGoPath); err == nil {
		return fmt.Errorf("cmd/migrate/main.go already exists")
	}

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

	if err := os.WriteFile(mainGoPath, []byte(template), 0644); err != nil {
		return fmt.Errorf("failed to write main.go: %w", err)
	}

	fmt.Println("✓ Created cmd/migrate/main.go")
	fmt.Println("\nYou can now run migrations using:")
	fmt.Println("  go run cmd/migrate/main.go          # Run pending migrations")
	fmt.Println("  go run cmd/migrate/main.go status   # Check migration status")
	fmt.Println("  go run cmd/migrate/main.go rollback # Rollback last batch")
	fmt.Println("  go run cmd/migrate/main.go reset    # Reset all migrations")
	fmt.Println("  go run cmd/migrate/main.go fresh    # Drop all tables and re-run")

	return nil
}
