package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var seederPath string

func init() {
	seedCmd.PersistentFlags().StringVar(&seederPath, "path", "./seeders", "Path to seeders directory")

	seedCmd.AddCommand(seedRunCmd)
	seedCmd.AddCommand(seedCreateCmd)

	rootCmd.AddCommand(seedCmd)
}

var seedCmd = &cobra.Command{
	Use:          "seed",
	Short:        "Database seeding commands",
	RunE:         runSeed,
	SilenceUsage: true,
}

var seedRunCmd = &cobra.Command{
	Use:          "run [name]",
	Short:        "Run database seeders (all or specific by name)",
	RunE:         runSeed,
	SilenceUsage: true,
}

var seedCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new seeder file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return createSeeder(args[0])
	},
}

func runSeed(cmd *cobra.Command, args []string) error {
	if _, err := os.Stat("cmd/seed/main.go"); err != nil {
		fmt.Println("Initializing Olympian seeder (creating cmd/seed/main.go)...")
		if err := initializeSeederFile(); err != nil {
			return fmt.Errorf("failed to initialize: %w", err)
		}
		fmt.Println("Created cmd/seed/main.go")
		fmt.Println()
	}

	cmdArgs := []string{"run", "cmd/seed/main.go"}
	cmdArgs = append(cmdArgs, args...)

	runCmd := exec.Command("go", cmdArgs...)
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr
	runCmd.Env = os.Environ()
	return runCmd.Run()
}

func initializeSeederFile() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	goModPath := filepath.Join(cwd, "go.mod")
	goModContent, err := os.ReadFile(goModPath)
	if err != nil {
		return fmt.Errorf("failed to read go.mod: %w (make sure you're in a Go project)", err)
	}

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

	seedDir := filepath.Join(cwd, "cmd", "seed")
	if err := os.MkdirAll(seedDir, 0755); err != nil {
		return fmt.Errorf("failed to create cmd/seed directory: %w", err)
	}

	mainGoPath := filepath.Join(seedDir, "main.go")

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

	_ "%s/seeders"
)

func main() {
	_ = godotenv.Load()

	dbDriver := os.Getenv("DB_DRIVER")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")

	if dbDriver == "" {
		log.Fatal("DB_DRIVER environment variable not set")
	}

	var dsn string
	var dialect olympian.Dialect

	switch dbDriver {
	case "mysql":
		dsn = fmt.Sprintf("%%s:%%s@tcp(%%s:%%s)/%%s?parseTime=true", dbUser, dbPass, dbHost, dbPort, dbName)
		dialect = olympian.MySQL()
	case "postgres":
		dsn = fmt.Sprintf("postgres://%%s:%%s@%%s:%%s/%%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, dbName)
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

	olympian.SetDB(db, dialect)

	runner := olympian.NewSeederRunner(db, dialect)
	seeders := olympian.GetSeeders()

	if len(os.Args) > 1 {
		if err := runner.RunSpecific(seeders, os.Args[1:]...); err != nil {
			log.Fatalf("Failed to run seeders: %%v", err)
		}
	} else {
		if err := runner.Run(seeders); err != nil {
			log.Fatalf("Failed to run seeders: %%v", err)
		}
	}
}
`, moduleName)

	return os.WriteFile(mainGoPath, []byte(template), 0644)
}

func createSeeder(name string) error {
	if seederPath == "" {
		seederPath = "./seeders"
	}

	if err := os.MkdirAll(seederPath, 0755); err != nil {
		return fmt.Errorf("failed to create seeders directory: %w", err)
	}

	formattedName := formatSeederName(name)
	filename := fmt.Sprintf("%s_seeder.go", formattedName)
	filePath := filepath.Join(seederPath, filename)

	tableName := name

	template := fmt.Sprintf(`package seeders

import (
	"github.com/ichtrojan/olympian"
)

func init() {
	olympian.RegisterSeeder(olympian.Seeder{
		Name: "%s",
		Run: func() error {
			return olympian.Seed("%s",
				map[string]interface{}{
					"id":   "uuid-here",
					"name": "Example",
				},
			)
		},
	})
}
`, formattedName, tableName)

	if err := os.WriteFile(filePath, []byte(template), 0644); err != nil {
		return fmt.Errorf("failed to create seeder file: %w", err)
	}

	fmt.Printf("Created seeder: %s\n", filePath)
	return nil
}

func formatSeederName(name string) string {
	name = strings.ToLower(name)
	if strings.HasSuffix(name, "_seeder") {
		name = name[:len(name)-7]
	}
	return name
}
