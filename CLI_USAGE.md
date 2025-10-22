# Olympian CLI Usage Guide

Complete guide for using the Olympian CLI tool.

## Installation

```bash
go get github.com/ichtrojan/olympian/cmd/olympian@latest
```

## Quick Start (No Setup Required!)

Olympian automatically initializes on first use. Just create your .env and start migrating:

**Step 1:** Create a `.env` file in your project root:

```env
DB_DRIVER=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=bandit
DB_USER=root
DB_PASS=
```

**Step 2:** Create and run migrations:
```bash
# Create a migration (smart naming - automatically formats)
olympian migrate create users

# Run migrations (auto-initializes on first use)
olympian migrate
```

That's it! Olympian automatically creates `cmd/migrate/main.go` on first use.

## Database Configuration

### Option 1: Using .env File (MySQL/PostgreSQL)

This is the recommended approach for MySQL and PostgreSQL databases.

**For MySQL:**
```env
DB_DRIVER=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=bandit
DB_USER=root
DB_PASS=
```

**For PostgreSQL:**
```env
DB_DRIVER=postgres
DB_HOST=127.0.0.1
DB_PORT=5432
DB_NAME=mydb
DB_USER=postgres
DB_PASS=secret
```

**Run migrations:**
```bash
olympian migrate
```

No flags needed!

### Option 2: Command Line Flags (SQLite)

For SQLite databases, use flags directly:

```bash
olympian migrate --driver sqlite3 --dsn ./database.db
```

### Option 3: Override .env with Flags

You can override .env settings with flags:

```bash
olympian migrate --driver mysql --dsn "user:pass@tcp(localhost:3306)/dbname"
```

Disable .env file loading:
```bash
olympian migrate --env=false --driver sqlite3 --dsn ./db.sqlite
```

## Creating Migrations

### Smart Migration Naming

Olympian automatically formats migration names to follow best practices. Just provide the table name:

```bash
# All of these create the same properly formatted migration
olympian migrate create users                # → create_users_table
olympian migrate create create_users_table   # → create_users_table (no duplication)
olympian migrate create create_users         # → create_users_table
olympian migrate create users_table          # → create_users_table
```

Result: `./migrations/1234567890_create_users_table.go`

### Custom Migration Path

```bash
olympian migrate create posts --path ./database/migrations
```

### Migration File Structure

The generated migration file looks like this:

```go
package migrations

import (
	"github.com/ichtrojan/olympian"
)

func init() {
	olympian.RegisterMigration(olympian.Migration{
		Name: "1234567890_create_users_table",
		Up: func() error {
			return olympian.Table("users").Create(func() {
				olympian.Uuid("id").Primary()
				olympian.Timestamps()
			})
		},
		Down: func() error {
			return olympian.Table("users").Drop()
		},
	})
}
```

**Customize it** by adding your columns:

```go
Up: func() error {
	return olympian.Table("users").Create(func() {
		olympian.Uuid("id").Primary()
		olympian.String("name")
		olympian.String("email").Unique()
		olympian.Boolean("active").Default(true)
		olympian.Timestamps()
	})
},
```

## Running Migrations

### Run All Pending Migrations

```bash
olympian migrate
```

Or explicitly:
```bash
olympian migrate up
```

**With .env:**
```bash
# Uses DB_DRIVER, DB_HOST, etc. from .env
olympian migrate
```

**With SQLite:**
```bash
olympian migrate --driver sqlite3 --dsn ./database.db
```

### Check Migration Status

See which migrations have run:

```bash
olympian migrate status
```

Output:
```
------------------------------------------------------------
| Status   | Migration                                     |
------------------------------------------------------------
| Ran      | create_businesses_table                       |
| Ran      | create_users_table                            |
| Pending  | create_posts_table                            |
------------------------------------------------------------
```

## Rolling Back Migrations

### Rollback Last Batch

```bash
olympian migrate rollback
```

This rolls back the last batch of migrations that were run together.

### Reset All Migrations

Rollback everything:

```bash
olympian migrate reset
```

### Fresh Migration

Drop all tables and re-run all migrations:

```bash
olympian migrate fresh
```

**Warning:** This will delete all your data!

## Complete Examples

### Example 1: MySQL Project Setup

**Step 1:** Create `.env`:
```env
DB_DRIVER=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=bandit
DB_USER=root
DB_PASS=
```

**Step 2:** Create migration:
```bash
olympian migrate create create_users_table
```

**Step 3:** Edit `./migrations/TIMESTAMP_create_users_table.go`:
```go
Up: func() error {
	return olympian.Table("users").Create(func() {
		olympian.Uuid("id").Primary()
		olympian.String("username").Unique()
		olympian.String("email").Unique()
		olympian.String("password")
		olympian.Boolean("verified").Default(false)
		olympian.Timestamps()
	})
},
```

**Step 4:** Run migration:
```bash
olympian migrate
```

**Step 5:** Check status:
```bash
olympian migrate status
```

### Example 2: PostgreSQL Project

**`.env`:**
```env
DB_DRIVER=postgres
DB_HOST=localhost
DB_PORT=5432
DB_NAME=myapp
DB_USER=postgres
DB_PASS=secret
```

**Create and run:**
```bash
olympian migrate create create_products_table
# Edit the migration file
olympian migrate
```

### Example 3: SQLite Project

No `.env` needed:

```bash
olympian migrate create create_users_table --driver sqlite3 --dsn ./app.db
# Edit the migration file
olympian migrate --driver sqlite3 --dsn ./app.db
```

### Example 4: Adding a Foreign Key

**Create migration:**
```bash
olympian migrate create create_posts_table
```

**Edit migration:**
```go
Up: func() error {
	return olympian.Table("posts").Create(func() {
		olympian.Uuid("id").Primary()
		olympian.String("user_id")
		olympian.String("title")
		olympian.Text("content")
		olympian.Foreign("user_id").
			References("id").
			On("users").
			OnDelete("cascade")
		olympian.Timestamps()
	})
},
```

### Example 5: Modifying Existing Table

**Create migration:**
```bash
olympian migrate create add_bio_to_users
```

**Edit migration:**
```go
Up: func() error {
	return olympian.Table("users").Modify(func() {
		olympian.Text("bio").Nullable()
		olympian.String("avatar_url").Nullable()
	})
},
Down: func() error {
	olympian.DropColumnIfExists("users", "bio")
	olympian.DropColumnIfExists("users", "avatar_url")
	return nil
},
```

## Common Workflows

### Development Workflow

```bash
# 1. Create migration
olympian migrate create create_feature_table

# 2. Edit migration file

# 3. Run migration
olympian migrate

# 4. Made a mistake? Rollback
olympian migrate rollback

# 5. Fix the migration file

# 6. Run again
olympian migrate
```

### Production Workflow

```bash
# 1. Check status
olympian migrate status

# 2. Review pending migrations

# 3. Backup database

# 4. Run migrations
olympian migrate

# 5. Verify
olympian migrate status
```

## Troubleshooting

### "No .env file found"

This is just a warning. If you're using flags, it's fine:
```bash
olympian migrate --driver sqlite3 --dsn ./db.sqlite
```

### "database driver not specified"

Either create a `.env` file or use flags:
```bash
olympian migrate --driver mysql --dsn "user:pass@tcp(localhost:3306)/db"
```

### Migration Files Not Loading

Make sure your migration files are in the correct directory (default: `./migrations`).

### Connection Refused

Check your `.env` settings:
- Is `DB_HOST` correct?
- Is `DB_PORT` correct?
- Is the database server running?

## Advanced Usage

### Custom Migration Path

Always use a custom path:
```bash
olympian migrate --path ./database/migrations
olympian migrate create my_migration --path ./database/migrations
```

### Multiple Environments

Use different `.env` files:

```bash
# Development
cp .env.development .env
olympian migrate

# Production
cp .env.production .env
olympian migrate
```

Or use different flags entirely.

## Tips

1. **Always test rollbacks** - Make sure your `Down` function works
2. **One change per migration** - Keep migrations focused
3. **Use descriptive names** - `create_users_table` not `migration1`
4. **Version control** - Commit migration files to git
5. **Never edit ran migrations** - Create a new migration instead
6. **Backup before fresh** - `migrate fresh` deletes everything

## Need Help?

- [Full Documentation](https://github.com/ichtrojan/olympian)
- [Examples](https://github.com/ichtrojan/olympian/tree/main/examples)
- [Issues](https://github.com/ichtrojan/olympian/issues)
