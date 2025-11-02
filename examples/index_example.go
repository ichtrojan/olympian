package main

import (
	"database/sql"
	"log"

	"github.com/ichtrojan/olympian"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "./index_test.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	migrator := olympian.NewMigrator(db, olympian.SQLite())

	if err := migrator.Init(); err != nil {
		log.Fatal(err)
	}

	migrations := []olympian.Migration{
		{
			Name: "create_users_table_with_indexes",
			Up: func() error {
				return olympian.Table("users").Create(func() {
					olympian.Uuid("id").Primary()
					olympian.String("name")
					// Auto-generated index name: idx_users_email
					olympian.String("email").Index()
					// Custom index name
					olympian.String("username").IndexWithName("custom_username_idx")
					// Can combine with other column modifiers
					olympian.String("phone").Nullable().Index()
					olympian.Timestamps()
				})
			},
			Down: func() error {
				return olympian.Table("users").Drop()
			},
		},
		{
			Name: "create_products_table_with_indexes",
			Up: func() error {
				return olympian.Table("products").Create(func() {
					olympian.Uuid("id").Primary()
					olympian.String("name")
					// Index on SKU for fast lookups
					olympian.String("sku").Unique().Index()
					// Index on category for filtering
					olympian.String("category").IndexWithName("idx_product_category")
					olympian.Decimal("price", 10, 2)
					olympian.Integer("stock").Index()
					olympian.Timestamps()
				})
			},
			Down: func() error {
				return olympian.Table("products").Drop()
			},
		},
		{
			Name: "create_orders_with_foreign_keys_and_indexes",
			Up: func() error {
				return olympian.Table("orders").Create(func() {
					olympian.Uuid("id").Primary()
					// Foreign key with index for better join performance
					olympian.Uuid("user_id").References("id").On("users").OnDelete("cascade").Index()
					olympian.Uuid("product_id").References("id").On("products").OnDelete("restrict").Index()
					olympian.String("status").Index()
					olympian.Decimal("total", 10, 2)
					olympian.Timestamps()
				})
			},
			Down: func() error {
				return olympian.Table("orders").Drop()
			},
		},
	}

	if err := migrator.Migrate(migrations); err != nil {
		log.Fatal(err)
	}

	log.Println("Migrations with indexes completed successfully!")
}
