# Olympian

A powerful, Laravel-inspired database migration system for Go. Olympian provides an elegant fluent API for managing database schema changes across PostgreSQL, MySQL, and SQLite.

## Features

- **Fluent API**: Chain methods elegantly to define your schema
- **Database Agnostic**: Works with PostgreSQL, MySQL, and SQLite using standard `database/sql`
- **Migration Tracking**: Automatic tracking of executed migrations with batch support
- **Rollback Support**: Roll back migrations individually or in batches
- **Foreign Keys**: Full support for foreign key constraints with cascading actions
- **Rich Column Types**: UUID, String, Text, Integer, Boolean, Decimal, Timestamp, Date, JSON, and more
- **Schema Modifications**: Add columns to existing tables with position control
- **CLI Tool**: Command-line interface for running migrations
- **Type-Safe**: Leverage Go's type system for compile-time safety

## Installation

```bash
go get github.com/ichtrojan/olympian
```

## Quick Start

### Basic Usage

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
                    olympian.Boolean("verified").Default(false)
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
}
```

## Column Types

Olympian supports a rich set of column types:

```go
olympian.Uuid("id")                    // UUID column
olympian.String("name")                 // VARCHAR(255)
olympian.Text("description")            // TEXT
olympian.Integer("count")               // INTEGER
olympian.BigInteger("big_count")        // BIGINT
olympian.Boolean("active")              // BOOLEAN
olympian.Decimal("price", 10, 2)        // DECIMAL(10,2)
olympian.Timestamp("created_at")        // TIMESTAMP
olympian.Date("birth_date")             // DATE
olympian.Json("metadata")               // JSON/JSONB
```

## Column Modifiers

Chain modifiers to customize column behavior:

```go
olympian.String("email").Nullable()                    // Allow NULL values
olympian.Uuid("id").Primary()                          // Primary key
olympian.String("username").Unique()                   // Unique constraint
olympian.Boolean("active").Default(true)               // Default value
olympian.Integer("age").After("name")                  // Column position (MySQL)
olympian.Integer("id").AutoIncrement()                 // Auto increment
```

## Foreign Keys

Define relationships between tables:

```go
olympian.Foreign("user_id").
    References("id").
    On("users").
    OnDelete("cascade").
    OnUpdate("restrict")
```

## Helper Functions

### Timestamps

Automatically add `created_at` and `updated_at` columns:

```go
olympian.Timestamps()  // Adds created_at and updated_at
```

### Soft Deletes

Add soft delete support:

```go
olympian.SoftDeletes()  // Adds deleted_at column
```

## Table Operations

### Creating Tables

```go
olympian.Table("products").Create(func() {
    olympian.Uuid("id").Primary()
    olympian.String("name")
    olympian.Decimal("price", 10, 2)
    olympian.Integer("stock").Default(0)
    olympian.Timestamps()
})
```

### Modifying Tables

Add columns to existing tables:

```go
olympian.Table("users").Modify(func() {
    olympian.Integer("age").After("name").Nullable()
    olympian.String("phone").Nullable()
})
```

### Dropping Tables

```go
olympian.Table("users").Drop()
```

### Dropping Columns

```go
olympian.DropColumnIfExists("users", "phone")
```

## Advanced Operations

### Renaming Columns

```go
olympian.RenameColumn("users", "username", "user_name")
```

### Renaming Tables

```go
olympian.RenameTable("old_users", "users")
```

### Creating Indexes

```go
olympian.CreateIndex("users", []string{"email"}, "idx_users_email")
olympian.CreateUniqueIndex("users", []string{"username"}, "idx_users_username")
```

### Dropping Indexes

```go
olympian.DropIndex("idx_users_email")
```

## Complete Migration Example

```go
migrations := []olympian.Migration{
    {
        Name: "create_businesses_table",
        Up: func() error {
            return olympian.Table("businesses").Create(func() {
                olympian.Uuid("id").Primary()
                olympian.String("name")
                olympian.String("industry").Nullable()
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
                olympian.String("name")
                olympian.String("email").Unique()
                olympian.Boolean("verified").Default(false)
                olympian.Timestamps()
                olympian.SoftDeletes()
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
```

## Database Dialects

Olympian supports three database systems:

### PostgreSQL

```go
import _ "github.com/lib/pq"

db, _ := sql.Open("postgres", "host=localhost user=postgres dbname=mydb sslmode=disable")
migrator := olympian.NewMigrator(db, olympian.Postgres())
```

### MySQL

```go
import _ "github.com/go-sql-driver/mysql"

db, _ := sql.Open("mysql", "user:password@tcp(localhost:3306)/dbname")
migrator := olympian.NewMigrator(db, olympian.MySQL())
```

### SQLite

```go
import _ "github.com/mattn/go-sqlite3"

db, _ := sql.Open("sqlite3", "./database.db")
migrator := olympian.NewMigrator(db, olympian.SQLite())
```

## Migration Commands

### Running Migrations

```go
migrator.Migrate(migrations)  // Run all pending migrations
```

### Rolling Back

```go
migrator.Rollback(migrations, 1)  // Rollback last batch
migrator.Rollback(migrations, 2)  // Rollback last 2 batches
```

### Migration Status

```go
migrator.Status(migrations)  // Show status of all migrations
```

### Reset All Migrations

```go
migrator.Reset(migrations)  // Rollback all migrations
```

### Fresh Migration

```go
migrator.Fresh(migrations)  // Drop all tables and re-run migrations
```

## CLI Tool

### Installation

```bash
go install github.com/ichtrojan/olympian/cmd/olympian@latest
```

### Configuration

Olympian supports two ways to configure database connections:

#### 1. Using .env File (Recommended for MySQL/PostgreSQL)

Create a `.env` file in your project root:

```env
DB_DRIVER=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=bandit
DB_USER=root
DB_PASS=
```

For PostgreSQL:
```env
DB_DRIVER=postgres
DB_HOST=127.0.0.1
DB_PORT=5432
DB_NAME=mydb
DB_USER=postgres
DB_PASS=secret
```

Then run migrations without any flags:
```bash
olympian migrate
```

#### 2. Using Command Line Flags (For SQLite)

For SQLite databases, use the `--driver` and `--dsn` flags:

```bash
olympian migrate --driver sqlite3 --dsn ./database.db
```

### Commands

```bash
# Run migrations (uses .env by default)
olympian migrate

# Run migrations with SQLite
olympian migrate --driver sqlite3 --dsn ./database.db

# Rollback last batch
olympian migrate rollback

# Show migration status
olympian migrate status

# Reset all migrations
olympian migrate reset

# Fresh migration (drop all tables)
olympian migrate fresh

# Create new migration file (creates in ./migrations by default)
olympian migrate create create_users_table

# Create migration in custom path
olympian migrate create create_posts_table --path ./database/migrations
```

### CLI Flags

- `--driver`: Database driver (`sqlite3`, `postgres`, `mysql`)
- `--dsn`: Database connection string (for SQLite)
- `--path`: Path to migrations directory (default: `./migrations`)
- `--env`: Use .env file for configuration (default: `true`)

### Environment Variables

- `DB_DRIVER`: Database driver (`mysql`, `postgres`, `sqlite3`)
- `DB_HOST`: Database host
- `DB_PORT`: Database port
- `DB_NAME`: Database name
- `DB_USER`: Database user
- `DB_PASS`: Database password

## Migration File Structure

When using the CLI to create migrations:

```bash
olympian migrate create create_products_table
```

This creates a file like `migrations/1634567890_create_products_table.go`:

```go
package migrations

import "github.com/ichtrojan/olympian"

func init() {
    olympian.RegisterMigration(olympian.Migration{
        Name: "1634567890_create_products_table",
        Up: func() error {
            return olympian.Table("products").Create(func() {
                olympian.Uuid("id").Primary()
                olympian.Timestamps()
            })
        },
        Down: func() error {
            return olympian.Table("products").Drop()
        },
    })
}
```

## How It Works

### Migration Tracking

Olympian automatically creates an `olympian_migrations` table to track:
- Migration name
- Batch number (for grouped rollbacks)
- Execution timestamp

### Batching

Migrations are grouped into batches. Each time you run `Migrate()`, pending migrations are executed as a new batch. This allows you to rollback related migrations together.

### Transactions

Each migration runs in a transaction. If a migration fails, it's automatically rolled back and the error is returned.

### Dialect System

Olympian uses a dialect system to generate database-specific SQL:
- PostgreSQL: Uses `UUID`, `JSONB`, `SERIAL` types
- MySQL: Uses `CHAR(36)` for UUIDs, `JSON`, `AUTO_INCREMENT`
- SQLite: Uses `TEXT` for most types, `AUTOINCREMENT`

## Best Practices

1. **Always provide a Down function**: This ensures migrations can be rolled back
2. **Use descriptive migration names**: Include timestamp and clear description
3. **One logical change per migration**: Don't mix unrelated schema changes
4. **Test migrations**: Test both up and down migrations before deploying
5. **Use foreign keys carefully**: Ensure referenced tables exist before creating foreign keys
6. **Handle nullable columns**: Be explicit about nullable vs required fields
7. **Use transactions**: Olympian handles this automatically, but be aware

## Error Handling

```go
if err := migrator.Migrate(migrations); err != nil {
    log.Printf("Migration failed: %v", err)
    // Migration is automatically rolled back
}
```

## Testing

Run the test suite:

```bash
go test -v ./...
```

Example test:

```go
func TestMigration(t *testing.T) {
    db, _ := sql.Open("sqlite3", ":memory:")
    defer db.Close()

    olympian.SetDB(db, olympian.SQLite())

    err := olympian.Table("users").Create(func() {
        olympian.Uuid("id").Primary()
        olympian.String("name")
    })

    if err != nil {
        t.Fatalf("Failed to create table: %v", err)
    }
}
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see LICENSE file for details

## Credits

Inspired by Laravel's elegant migration system, adapted for the Go ecosystem.

## Documentation

For more detailed documentation, visit our [documentation site](https://ichtrojan.github.io/olympian).

## Support

- GitHub Issues: [Report bugs or request features](https://github.com/ichtrojan/olympian/issues)
- Discussions: [Ask questions and share ideas](https://github.com/ichtrojan/olympian/discussions)

## Roadmap

- [ ] Migration squashing
- [ ] Schema dumping
- [ ] Seed data support
- [ ] Migration dependencies
- [ ] Dry-run mode
- [ ] SQL Server support
- [ ] Migration templates
