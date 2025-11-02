package main

import (
	"database/sql"
	"log"

	"github.com/ichtrojan/olympian"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "./inline_fk_test.db")
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
			Name: "create_transactions_table",
			Up: func() error {
				return olympian.Table("transactions").Create(func() {
					olympian.Uuid("id").Primary()
					olympian.String("amount")
					olympian.Timestamps()
				})
			},
			Down: func() error {
				return olympian.Table("transactions").Drop()
			},
		},
		{
			Name: "create_cards_table",
			Up: func() error {
				return olympian.Table("cards").Create(func() {
					olympian.Uuid("id").Primary()
					olympian.String("number")
					olympian.Timestamps()
				})
			},
			Down: func() error {
				return olympian.Table("cards").Drop()
			},
		},
		{
			Name: "create_card_transactions_table",
			Up: func() error {
				return olympian.Table("card_transactions").Create(func() {
					olympian.Uuid("id").Primary()
					// Inline foreign key syntax - more concise!
					olympian.Uuid("transaction_id").References("id").On("transactions").OnDelete("cascade")
					olympian.Uuid("card_id").References("id").On("cards").OnDelete("cascade").OnUpdate("restrict")
					olympian.String("status")
					olympian.String("merchant")
					olympian.String("category")
					olympian.Timestamps()
				})
			},
			Down: func() error {
				return olympian.Table("card_transactions").Drop()
			},
		},
	}

	if err := migrator.Migrate(migrations); err != nil {
		log.Fatal(err)
	}

	log.Println("Migrations with inline foreign keys completed successfully!")
}
