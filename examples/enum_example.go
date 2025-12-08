//go:build ignore

package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"github.com/ichtrojan/olympian"
)

func main() {
	// Create in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Set up Olympian with SQLite dialect
	olympian.SetDB(db, olympian.SQLite())

	// Create a table with enum columns
	err = olympian.Table("orders").Create(func() {
		olympian.Uuid("id").Primary()
		olympian.String("customer_name")
		olympian.Enum("status", "pending", "processing", "shipped", "delivered", "cancelled").Default("pending")
		olympian.Enum("priority", "low", "medium", "high").Default("medium")
		olympian.Enum("payment_method", "credit_card", "paypal", "bank_transfer").Nullable()
		olympian.Decimal("total", 10, 2)
		olympian.Timestamps()
	})

	if err != nil {
		log.Fatalf("Failed to create orders table: %v", err)
	}

	log.Println("✓ Orders table created successfully with enum columns")

	// Insert valid data
	_, err = db.Exec(`
		INSERT INTO orders (id, customer_name, status, priority, payment_method, total)
		VALUES ('550e8400-e29b-41d4-a716-446655440000', 'John Doe', 'processing', 'high', 'credit_card', 99.99)
	`)
	if err != nil {
		log.Fatalf("Failed to insert order: %v", err)
	}

	log.Println("✓ Valid order inserted successfully")

	// Try to insert invalid enum value (this should fail)
	_, err = db.Exec(`
		INSERT INTO orders (id, customer_name, status, priority, total)
		VALUES ('550e8400-e29b-41d4-a716-446655440001', 'Jane Smith', 'invalid_status', 'medium', 49.99)
	`)
	if err != nil {
		log.Printf("✓ Invalid enum value correctly rejected: %v", err)
	} else {
		log.Println("✗ Invalid enum value was not rejected (unexpected)")
	}

	// Query the data
	var customerName, status, priority string
	var total float64
	err = db.QueryRow("SELECT customer_name, status, priority, total FROM orders WHERE customer_name = 'John Doe'").
		Scan(&customerName, &status, &priority, &total)

	if err != nil {
		log.Fatalf("Failed to query order: %v", err)
	}

	log.Printf("✓ Retrieved order: %s - Status: %s, Priority: %s, Total: $%.2f",
		customerName, status, priority, total)

	// Add an enum column to existing table
	err = olympian.Table("orders").Modify(func() {
		olympian.Enum("shipping_method", "standard", "express", "overnight").Default("standard")
	})

	if err != nil {
		log.Fatalf("Failed to add enum column: %v", err)
	}

	log.Println("✓ Enum column added to existing table successfully")
}
