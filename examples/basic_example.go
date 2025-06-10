package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"github.com/ichtrojan/olympian"
)

func main() {
	db, err := sql.Open("sqlite3", "./test.db")
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
			Name: "create_businesses_table",
			Up: func() error {
				return olympian.Table("businesses").Create(func() {
					olympian.Uuid("id").Primary()
					olympian.String("name")
					olympian.Timestamps()
				})
			},
			Down: func() error {
				return olympian.Table("businesses").Drop()
			},
		},
		{
			Name: "create_users_table",
			Up: func() error {
				return olympian.Table("users").Create(func() {
					olympian.Uuid("id").Primary()
					olympian.Foreign("business_id").
						References("id").
						On("businesses").
						OnDelete("cascade")
					olympian.String("name").Nullable()
					olympian.Boolean("verified").Default(false)
					olympian.Timestamps()
				})
			},
			Down: func() error {
				return olympian.Table("users").Drop()
			},
		},
		{
			Name: "add_age_to_users",
			Up: func() error {
				return olympian.Table("users").Modify(func() {
					olympian.Integer("age").After("name").Nullable()
				})
			},
			Down: func() error {
				return olympian.DropColumnIfExists("users", "age")
			},
		},
	}

	if err := migrator.Migrate(migrations); err != nil {
		log.Fatal(err)
	}

	log.Println("Migrations completed successfully!")
}
