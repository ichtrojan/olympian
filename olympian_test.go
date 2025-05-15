package olympian

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}
	return db
}

func TestTableCreation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	SetDB(db, &SQLiteDialect{})

	err := Table("users").Create(func() {
		Uuid("id").Primary()
		String("name")
		String("email").Unique()
		Integer("age").Nullable()
		Boolean("active").Default(true)
		Timestamps()
	})

	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='users'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Table was not created: %v", err)
	}

	if tableName != "users" {
		t.Errorf("Expected table name 'users', got '%s'", tableName)
	}
}

func TestTableModification(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	SetDB(db, &SQLiteDialect{})

	err := Table("users").Create(func() {
		Uuid("id").Primary()
		String("name")
	})
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	err = Table("users").Modify(func() {
		Integer("age").Nullable()
		String("email")
	})
	if err != nil {
		t.Fatalf("Failed to modify table: %v", err)
	}

	rows, err := db.Query("PRAGMA table_info(users)")
	if err != nil {
		t.Fatalf("Failed to query table info: %v", err)
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		columns[name] = true
	}

	expectedColumns := []string{"id", "name", "age", "email"}
	for _, col := range expectedColumns {
		if !columns[col] {
			t.Errorf("Expected column '%s' not found", col)
		}
	}
}

func TestTableDrop(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	SetDB(db, &SQLiteDialect{})

	err := Table("users").Create(func() {
		Uuid("id").Primary()
		String("name")
	})
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	err = Table("users").Drop()
	if err != nil {
		t.Fatalf("Failed to drop table: %v", err)
	}

	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='users'").Scan(&tableName)
	if err != sql.ErrNoRows {
		t.Errorf("Table was not dropped")
	}
}

func TestForeignKeys(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	SetDB(db, &SQLiteDialect{})

	err := Table("businesses").Create(func() {
		Uuid("id").Primary()
		String("name")
	})
	if err != nil {
		t.Fatalf("Failed to create businesses table: %v", err)
	}

	err = Table("users").Create(func() {
		Uuid("id").Primary()
		String("business_id")
		String("name")
		Foreign("business_id").
			References("id").
			On("businesses").
			OnDelete("cascade")
	})
	if err != nil {
		t.Fatalf("Failed to create users table with foreign key: %v", err)
	}

	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='users'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Table was not created: %v", err)
	}
}

func TestColumnTypes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	SetDB(db, &SQLiteDialect{})

	err := Table("test_types").Create(func() {
		Uuid("uuid_col").Primary()
		String("string_col")
		Text("text_col")
		Integer("int_col")
		BigInteger("bigint_col")
		Boolean("bool_col")
		Decimal("decimal_col", 10, 2)
		Timestamp("timestamp_col")
		Date("date_col")
		Json("json_col")
	})

	if err != nil {
		t.Fatalf("Failed to create table with various column types: %v", err)
	}

	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='test_types'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Table was not created: %v", err)
	}
}

func TestNullableAndDefaults(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	SetDB(db, &SQLiteDialect{})

	err := Table("users").Create(func() {
		Uuid("id").Primary()
		String("name")
		String("email").Nullable()
		Boolean("active").Default(true)
		Integer("status").Default(1)
	})

	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
}

func TestUniqueConstraint(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	SetDB(db, &SQLiteDialect{})

	err := Table("users").Create(func() {
		Uuid("id").Primary()
		String("email").Unique()
	})

	if err != nil {
		t.Fatalf("Failed to create table with unique constraint: %v", err)
	}
}

func TestPostgresDialect(t *testing.T) {
	dialect := &PostgresDialect{}

	tb := &TableBuilder{
		tableName: "users",
		columns: []*Column{
			{name: "id", dataType: "uuid", primary: true},
			{name: "name", dataType: "string"},
			{name: "age", dataType: "integer", nullable: true},
		},
	}

	sql := dialect.BuildCreateTable(tb)
	if sql == "" {
		t.Error("PostgreSQL dialect failed to build CREATE TABLE")
	}

	if !contains(sql, "UUID") {
		t.Error("PostgreSQL dialect should use UUID type")
	}
}

func TestMySQLDialect(t *testing.T) {
	dialect := &MySQLDialect{}

	tb := &TableBuilder{
		tableName: "users",
		columns: []*Column{
			{name: "id", dataType: "uuid", primary: true},
			{name: "name", dataType: "string"},
		},
	}

	sql := dialect.BuildCreateTable(tb)
	if sql == "" {
		t.Error("MySQL dialect failed to build CREATE TABLE")
	}

	if !contains(sql, "CHAR(36)") {
		t.Error("MySQL dialect should use CHAR(36) for UUID")
	}

	if !contains(sql, "ENGINE=InnoDB") {
		t.Error("MySQL dialect should specify InnoDB engine")
	}
}

func TestSQLiteDialect(t *testing.T) {
	dialect := &SQLiteDialect{}

	tb := &TableBuilder{
		tableName: "users",
		columns: []*Column{
			{name: "id", dataType: "uuid", primary: true},
			{name: "name", dataType: "string"},
		},
	}

	sql := dialect.BuildCreateTable(tb)
	if sql == "" {
		t.Error("SQLite dialect failed to build CREATE TABLE")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
