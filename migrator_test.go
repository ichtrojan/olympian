package olympian

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestMigratorInit(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	migrator := NewMigrator(db, &SQLiteDialect{})
	err := migrator.Init()
	if err != nil {
		t.Fatalf("Failed to initialize migrator: %v", err)
	}

	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='olympian_migrations'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Migrations table was not created: %v", err)
	}

	if tableName != "olympian_migrations" {
		t.Errorf("Expected table name 'olympian_migrations', got '%s'", tableName)
	}
}

func TestMigratorMigrate(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	migrator := NewMigrator(db, &SQLiteDialect{})
	if err := migrator.Init(); err != nil {
		t.Fatalf("Failed to initialize migrator: %v", err)
	}

	migrations := []Migration{
		{
			Name: "create_users_table",
			Up: func() error {
				return Table("users").Create(func() {
					Uuid("id").Primary()
					String("name")
				})
			},
			Down: func() error {
				return Table("users").Drop()
			},
		},
	}

	err := migrator.Migrate(migrations)
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='users'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Migrated table was not created: %v", err)
	}

	var migrationName string
	err = db.QueryRow("SELECT migration FROM olympian_migrations WHERE migration = 'create_users_table'").Scan(&migrationName)
	if err != nil {
		t.Fatalf("Migration was not recorded: %v", err)
	}
}

func TestMigratorRollback(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	migrator := NewMigrator(db, &SQLiteDialect{})
	if err := migrator.Init(); err != nil {
		t.Fatalf("Failed to initialize migrator: %v", err)
	}

	migrations := []Migration{
		{
			Name: "create_users_table",
			Up: func() error {
				return Table("users").Create(func() {
					Uuid("id").Primary()
					String("name")
				})
			},
			Down: func() error {
				return Table("users").Drop()
			},
		},
	}

	if err := migrator.Migrate(migrations); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	if err := migrator.Rollback(migrations, 1); err != nil {
		t.Fatalf("Failed to rollback migrations: %v", err)
	}

	var tableName string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='users'").Scan(&tableName)
	if err != sql.ErrNoRows {
		t.Error("Table should have been dropped after rollback")
	}

	var migrationName string
	err = db.QueryRow("SELECT migration FROM olympian_migrations WHERE migration = 'create_users_table'").Scan(&migrationName)
	if err != sql.ErrNoRows {
		t.Error("Migration record should have been removed after rollback")
	}
}

func TestMigratorBatches(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	migrator := NewMigrator(db, &SQLiteDialect{})
	if err := migrator.Init(); err != nil {
		t.Fatalf("Failed to initialize migrator: %v", err)
	}

	migrations1 := []Migration{
		{
			Name: "create_users_table",
			Up: func() error {
				return Table("users").Create(func() {
					Uuid("id").Primary()
				})
			},
			Down: func() error {
				return Table("users").Drop()
			},
		},
	}

	if err := migrator.Migrate(migrations1); err != nil {
		t.Fatalf("Failed to run first batch: %v", err)
	}

	migrations2 := []Migration{
		{
			Name: "create_users_table",
			Up:   func() error { return nil },
			Down: func() error { return nil },
		},
		{
			Name: "create_products_table",
			Up: func() error {
				return Table("products").Create(func() {
					Uuid("id").Primary()
				})
			},
			Down: func() error {
				return Table("products").Drop()
			},
		},
	}

	if err := migrator.Migrate(migrations2); err != nil {
		t.Fatalf("Failed to run second batch: %v", err)
	}

	batch, err := migrator.GetLastBatch()
	if err != nil {
		t.Fatalf("Failed to get last batch: %v", err)
	}

	if batch != 2 {
		t.Errorf("Expected batch 2, got %d", batch)
	}
}

func TestMigratorStatus(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	migrator := NewMigrator(db, &SQLiteDialect{})
	if err := migrator.Init(); err != nil {
		t.Fatalf("Failed to initialize migrator: %v", err)
	}

	migrations := []Migration{
		{
			Name: "create_users_table",
			Up: func() error {
				return Table("users").Create(func() {
					Uuid("id").Primary()
				})
			},
			Down: func() error {
				return Table("users").Drop()
			},
		},
		{
			Name: "create_products_table",
			Up: func() error {
				return Table("products").Create(func() {
					Uuid("id").Primary()
				})
			},
			Down: func() error {
				return Table("products").Drop()
			},
		},
	}

	if err := migrator.Migrate([]Migration{migrations[0]}); err != nil {
		t.Fatalf("Failed to run migration: %v", err)
	}

	err := migrator.Status(migrations)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}
}

func TestMigratorReset(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	migrator := NewMigrator(db, &SQLiteDialect{})
	if err := migrator.Init(); err != nil {
		t.Fatalf("Failed to initialize migrator: %v", err)
	}

	migrations := []Migration{
		{
			Name: "create_users_table",
			Up: func() error {
				return Table("users").Create(func() {
					Uuid("id").Primary()
				})
			},
			Down: func() error {
				return Table("users").Drop()
			},
		},
		{
			Name: "create_products_table",
			Up: func() error {
				return Table("products").Create(func() {
					Uuid("id").Primary()
				})
			},
			Down: func() error {
				return Table("products").Drop()
			},
		},
	}

	if err := migrator.Migrate(migrations); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	if err := migrator.Reset(migrations); err != nil {
		t.Fatalf("Failed to reset migrations: %v", err)
	}

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM olympian_migrations").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count migrations: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 migrations after reset, got %d", count)
	}
}

func TestGetExecutedMigrations(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	migrator := NewMigrator(db, &SQLiteDialect{})
	if err := migrator.Init(); err != nil {
		t.Fatalf("Failed to initialize migrator: %v", err)
	}

	if err := migrator.RecordMigration("test_migration", 1); err != nil {
		t.Fatalf("Failed to record migration: %v", err)
	}

	executed, err := migrator.GetExecutedMigrations()
	if err != nil {
		t.Fatalf("Failed to get executed migrations: %v", err)
	}

	if !executed["test_migration"] {
		t.Error("Expected test_migration to be in executed migrations")
	}
}

func TestRecordAndRemoveMigration(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	migrator := NewMigrator(db, &SQLiteDialect{})
	if err := migrator.Init(); err != nil {
		t.Fatalf("Failed to initialize migrator: %v", err)
	}

	if err := migrator.RecordMigration("test_migration", 1); err != nil {
		t.Fatalf("Failed to record migration: %v", err)
	}

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM olympian_migrations WHERE migration = 'test_migration'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count migrations: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 migration record, got %d", count)
	}

	if err := migrator.RemoveMigration("test_migration"); err != nil {
		t.Fatalf("Failed to remove migration: %v", err)
	}

	err = db.QueryRow("SELECT COUNT(*) FROM olympian_migrations WHERE migration = 'test_migration'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count migrations: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 migration records after removal, got %d", count)
	}
}
