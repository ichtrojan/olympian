# Olympian Quick Start Guide

Get up and running with Olympian in minutes!

## Installation

```bash
go get github.com/ichtrojan/olympian
```

## Your First Migration

### 1. Create a simple program

Create a file called `main.go`:

```go
package main

import (
    "database/sql"
    "log"

    _ "github.com/mattn/go-sqlite3"
    "github.com/ichtrojan/olympian"
)

func main() {
    db, err := sql.Open("sqlite3", "./database.db")
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
            Name: "create_users_table",
            Up: func() error {
                return olympian.Table("users").Create(func() {
                    olympian.Uuid("id").Primary()
                    olympian.String("name")
                    olympian.String("email").Unique()
                    olympian.Boolean("active").Default(true)
                    olympian.Timestamps()
                })
            },
            Down: func() error {
                return olympian.Table("users").Drop()
            },
        },
    }

    if err := migrator.Migrate(migrations); err != nil {
        log.Fatal(err)
    }

    log.Println("✓ Migrations completed successfully!")
}
```

### 2. Run it

```bash
go mod init myapp
go mod tidy
go run main.go
```

### 3. See the results

```bash
sqlite3 database.db ".schema users"
```

You should see your table created!

## Next Steps

### Add More Tables

```go
{
    Name: "create_posts_table",
    Up: func() error {
        return olympian.Table("posts").Create(func() {
            olympian.Uuid("id").Primary()
            olympian.Foreign("user_id").
                References("id").
                On("users").
                OnDelete("cascade")
            olympian.String("title")
            olympian.Text("content")
            olympian.Timestamps()
        })
    },
    Down: func() error {
        return olympian.Table("posts").Drop()
    },
}
```

### Modify Existing Tables

```go
{
    Name: "add_bio_to_users",
    Up: func() error {
        return olympian.Table("users").Modify(func() {
            olympian.Text("bio").Nullable()
        })
    },
    Down: func() error {
        return olympian.DropColumnIfExists("users", "bio")
    },
}
```

### Use Different Databases

#### PostgreSQL

```go
import _ "github.com/lib/pq"

db, _ := sql.Open("postgres", "host=localhost user=postgres dbname=mydb sslmode=disable")
migrator := olympian.NewMigrator(db, olympian.Postgres())
```

#### MySQL

```go
import _ "github.com/go-sql-driver/mysql"

db, _ := sql.Open("mysql", "user:password@tcp(localhost:3306)/dbname")
migrator := olympian.NewMigrator(db, olympian.MySQL())
```

## Using the CLI

Install the CLI tool:

```bash
go install github.com/ichtrojan/olympian/cmd/olympian@latest
```

Create a migration:

```bash
olympian migrate create create_users_table
```

Run migrations:

```bash
olympian migrate --driver sqlite3 --dsn ./database.db
```

Rollback:

```bash
olympian migrate rollback
```

Check status:

```bash
olympian migrate status
```

## Common Patterns

### Timestamps

```go
olympian.Timestamps()  // Adds created_at and updated_at
```

### Soft Deletes

```go
olympian.SoftDeletes()  // Adds deleted_at
```

### Indexes

```go
olympian.CreateIndex("users", []string{"email"}, "idx_users_email")
olympian.CreateUniqueIndex("users", []string{"username"}, "idx_users_username")
```

### Foreign Keys

```go
olympian.Foreign("user_id").
    References("id").
    On("users").
    OnDelete("cascade")
```

## Tips

1. Always provide both Up and Down functions
2. Test your migrations before deploying
3. Keep migrations small and focused
4. Use descriptive migration names
5. Check migration status before running
6. Back up your database before migrating in production

## Need Help?

- [Full Documentation](https://github.com/ichtrojan/olympian)
- [Examples](https://github.com/ichtrojan/olympian/tree/main/examples)
- [Issues](https://github.com/ichtrojan/olympian/issues)

Happy migrating! 🚀
