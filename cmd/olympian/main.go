package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

var rootCmd = &cobra.Command{
	Use:   "olympian",
	Short: "Olympian - A powerful database migration tool for Go",
	Long: `Olympian is a Laravel-inspired database migration system for Go.
It provides an elegant, fluent API for managing database schema changes
across multiple database systems (PostgreSQL, MySQL, SQLite).`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
