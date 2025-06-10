# Migrations Directory

This directory is for migration files when using the Olympian CLI tool.

## Creating a new migration

```bash
olympian migrate create create_users_table --path ./examples/migrations
```

## Running migrations

```bash
olympian migrate --driver sqlite3 --dsn ./database.db --path ./examples/migrations
```

## Rollback

```bash
olympian migrate rollback --driver sqlite3 --dsn ./database.db --path ./examples/migrations
```
