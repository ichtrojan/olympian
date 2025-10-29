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
	defer func() { _ = db.Close() }()

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
	defer func() { _ = db.Close() }()

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
	defer func() { _ = rows.Close() }()

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
	defer func() { _ = db.Close() }()

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
	defer func() { _ = db.Close() }()

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
	defer func() { _ = db.Close() }()

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
	defer func() { _ = db.Close() }()

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
	defer func() { _ = db.Close() }()

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

func TestEnumColumn(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	SetDB(db, &SQLiteDialect{})

	err := Table("users").Create(func() {
		Uuid("id").Primary()
		String("name")
		Enum("status", "pending", "active", "inactive")
		Enum("role", "admin", "user", "guest").Default("user")
	})

	if err != nil {
		t.Fatalf("Failed to create table with enum columns: %v", err)
	}

	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='users'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Table was not created: %v", err)
	}

	// Test that enum constraint is enforced
	_, err = db.Exec("INSERT INTO users (id, name, status, role) VALUES ('123', 'John', 'invalid_status', 'user')")
	if err == nil {
		t.Error("Expected enum constraint to reject invalid value")
	}

	// Test that valid values work
	_, err = db.Exec("INSERT INTO users (id, name, status, role) VALUES ('456', 'Jane', 'active', 'admin')")
	if err != nil {
		t.Errorf("Failed to insert valid enum value: %v", err)
	}
}

func TestEnumNullable(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	SetDB(db, &SQLiteDialect{})

	err := Table("products").Create(func() {
		Uuid("id").Primary()
		String("name")
		Enum("status", "draft", "published", "archived").Nullable()
	})

	if err != nil {
		t.Fatalf("Failed to create table with nullable enum: %v", err)
	}

	// Test that NULL is allowed
	_, err = db.Exec("INSERT INTO products (id, name, status) VALUES ('123', 'Product1', NULL)")
	if err != nil {
		t.Errorf("Failed to insert NULL into nullable enum: %v", err)
	}
}

func TestEnumModifyTable(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	SetDB(db, &SQLiteDialect{})

	err := Table("users").Create(func() {
		Uuid("id").Primary()
		String("name")
	})
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	err = Table("users").Modify(func() {
		Enum("status", "active", "inactive").Default("active")
	})
	if err != nil {
		t.Fatalf("Failed to add enum column to existing table: %v", err)
	}

	rows, err := db.Query("PRAGMA table_info(users)")
	if err != nil {
		t.Fatalf("Failed to query table info: %v", err)
	}
	defer func() { _ = rows.Close() }()

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

	if !columns["status"] {
		t.Error("Enum column 'status' was not added to table")
	}
}

func TestPostgresEnumDialect(t *testing.T) {
	dialect := &PostgresDialect{}

	tb := &TableBuilder{
		tableName: "users",
		columns: []*Column{
			{name: "id", dataType: "uuid", primary: true},
			{name: "status", dataType: "enum", enumValues: []string{"active", "inactive", "pending"}},
		},
	}

	sql := dialect.BuildCreateTable(tb)
	if sql == "" {
		t.Error("PostgreSQL dialect failed to build CREATE TABLE with enum")
	}

	if !contains(sql, "VARCHAR(255)") {
		t.Error("PostgreSQL dialect should use VARCHAR(255) for enum")
	}

	if !contains(sql, "CHECK") {
		t.Error("PostgreSQL dialect should add CHECK constraint for enum")
	}

	if !contains(sql, "'active'") || !contains(sql, "'inactive'") || !contains(sql, "'pending'") {
		t.Error("PostgreSQL enum CHECK constraint should include all enum values")
	}
}

func TestMySQLEnumDialect(t *testing.T) {
	dialect := &MySQLDialect{}

	tb := &TableBuilder{
		tableName: "users",
		columns: []*Column{
			{name: "id", dataType: "uuid", primary: true},
			{name: "status", dataType: "enum", enumValues: []string{"active", "inactive"}},
		},
	}

	sql := dialect.BuildCreateTable(tb)
	if sql == "" {
		t.Error("MySQL dialect failed to build CREATE TABLE with enum")
	}

	if !contains(sql, "ENUM(") {
		t.Error("MySQL dialect should use native ENUM type")
	}

	if !contains(sql, "'active'") || !contains(sql, "'inactive'") {
		t.Error("MySQL ENUM should include all enum values")
	}
}

func TestSQLiteEnumDialect(t *testing.T) {
	dialect := &SQLiteDialect{}

	tb := &TableBuilder{
		tableName: "users",
		columns: []*Column{
			{name: "id", dataType: "uuid", primary: true},
			{name: "status", dataType: "enum", enumValues: []string{"active", "inactive"}},
		},
	}

	sql := dialect.BuildCreateTable(tb)
	if sql == "" {
		t.Error("SQLite dialect failed to build CREATE TABLE with enum")
	}

	if !contains(sql, "TEXT") {
		t.Error("SQLite dialect should use TEXT for enum")
	}

	if !contains(sql, "CHECK") {
		t.Error("SQLite dialect should add CHECK constraint for enum")
	}

	if !contains(sql, "'active'") || !contains(sql, "'inactive'") {
		t.Error("SQLite enum CHECK constraint should include all enum values")
	}
}
