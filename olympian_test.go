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

func TestInlineForeignKeys(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	SetDB(db, &SQLiteDialect{})

	err := Table("transactions").Create(func() {
		Uuid("id").Primary()
		String("amount")
	})
	if err != nil {
		t.Fatalf("Failed to create transactions table: %v", err)
	}

	err = Table("cards").Create(func() {
		Uuid("id").Primary()
		String("number")
	})
	if err != nil {
		t.Fatalf("Failed to create cards table: %v", err)
	}

	err = Table("card_transactions").Create(func() {
		Uuid("id").Primary()
		Uuid("transaction_id").References("id").On("transactions").OnDelete("cascade")
		Uuid("card_id").References("id").On("cards").OnDelete("cascade")
		String("status")
		Timestamps()
	})
	if err != nil {
		t.Fatalf("Failed to create card_transactions table with inline foreign keys: %v", err)
	}

	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='card_transactions'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Table was not created: %v", err)
	}
}

func TestIndexes(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	SetDB(db, &SQLiteDialect{})

	err := Table("users").Create(func() {
		Uuid("id").Primary()
		String("email").Index()
		String("username").IndexWithName("custom_username_idx")
		String("name")
		Timestamps()
	})
	if err != nil {
		t.Fatalf("Failed to create table with indexes: %v", err)
	}

	// Check if auto-generated index exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_users_email'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query for index: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected auto-generated index 'idx_users_email' to exist")
	}

	// Check if custom-named index exists
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='custom_username_idx'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query for custom index: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected custom index 'custom_username_idx' to exist")
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

func TestCompositeUniqueConstraint(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	SetDB(db, &SQLiteDialect{})

	err := Table("invoices").Create(func() {
		Uuid("id").Primary()
		String("business_id")
		Integer("number")
		Unique("business_id", "number")
	})

	if err != nil {
		t.Fatalf("Failed to create table with composite unique constraint: %v", err)
	}

	// Insert first record
	_, err = db.Exec("INSERT INTO invoices (id, business_id, number) VALUES ('1', 'biz1', 1)")
	if err != nil {
		t.Fatalf("Failed to insert first record: %v", err)
	}

	// Insert duplicate should fail
	_, err = db.Exec("INSERT INTO invoices (id, business_id, number) VALUES ('2', 'biz1', 1)")
	if err == nil {
		t.Error("Expected composite unique constraint to reject duplicate")
	}

	// Same number with different business_id should work
	_, err = db.Exec("INSERT INTO invoices (id, business_id, number) VALUES ('3', 'biz2', 1)")
	if err != nil {
		t.Errorf("Same number with different business_id should work: %v", err)
	}
}

func TestCompositeUniqueConstraintWithCustomName(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	SetDB(db, &SQLiteDialect{})

	err := Table("orders").Create(func() {
		Uuid("id").Primary()
		String("store_id")
		String("order_number")
		Unique("store_id", "order_number").Name("uq_store_order")
	})

	if err != nil {
		t.Fatalf("Failed to create table with named composite unique constraint: %v", err)
	}

	// Verify the constraint works by testing it enforces uniqueness
	_, err = db.Exec("INSERT INTO orders (id, store_id, order_number) VALUES ('1', 'store1', 'order1')")
	if err != nil {
		t.Fatalf("Failed to insert first record: %v", err)
	}

	// Insert duplicate should fail
	_, err = db.Exec("INSERT INTO orders (id, store_id, order_number) VALUES ('2', 'store1', 'order1')")
	if err == nil {
		t.Error("Expected composite unique constraint to reject duplicate")
	}

	// Verify constraint name appears in SQL generation
	dialect := &SQLiteDialect{}
	tb := &TableBuilder{
		tableName: "orders",
		columns: []*Column{
			{name: "id", dataType: "uuid", primary: true},
			{name: "store_id", dataType: "string"},
			{name: "order_number", dataType: "string"},
		},
		uniqueConstraints: []*UniqueConstraint{
			{columns: []string{"store_id", "order_number"}, name: "uq_store_order"},
		},
	}
	sql := dialect.BuildCreateTable(tb)
	if !contains(sql, "CONSTRAINT uq_store_order UNIQUE") {
		t.Errorf("Expected SQL to contain custom constraint name.\nGot: %s", sql)
	}
}

func TestCompositeUniqueConstraintDialects(t *testing.T) {
	tests := []struct {
		name    string
		dialect Dialect
		check   string
	}{
		{"PostgreSQL", &PostgresDialect{}, "CONSTRAINT uq_invoices_business_id_number UNIQUE (business_id, number)"},
		{"MySQL", &MySQLDialect{}, "CONSTRAINT uq_invoices_business_id_number UNIQUE (business_id, number)"},
		{"SQLite", &SQLiteDialect{}, "CONSTRAINT uq_invoices_business_id_number UNIQUE (business_id, number)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := &TableBuilder{
				tableName: "invoices",
				columns: []*Column{
					{name: "id", dataType: "uuid", primary: true},
					{name: "business_id", dataType: "string"},
					{name: "number", dataType: "integer"},
				},
				uniqueConstraints: []*UniqueConstraint{
					{columns: []string{"business_id", "number"}},
				},
			}

			sql := tt.dialect.BuildCreateTable(tb)
			if !contains(sql, tt.check) {
				t.Errorf("%s dialect should generate composite unique constraint.\nExpected to contain: %s\nGot: %s", tt.name, tt.check, sql)
			}
		})
	}
}

func TestIncrements(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	SetDB(db, &SQLiteDialect{})

	err := Table("products").Create(func() {
		Increments("id")
		String("name")
	})

	if err != nil {
		t.Fatalf("Failed to create table with Increments: %v", err)
	}

	// Insert without specifying id
	result, err := db.Exec("INSERT INTO products (name) VALUES ('Product 1')")
	if err != nil {
		t.Fatalf("Failed to insert record: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get last insert id: %v", err)
	}
	if id != 1 {
		t.Errorf("Expected auto-generated id to be 1, got %d", id)
	}

	// Insert another record
	result, err = db.Exec("INSERT INTO products (name) VALUES ('Product 2')")
	if err != nil {
		t.Fatalf("Failed to insert second record: %v", err)
	}

	id, err = result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get last insert id: %v", err)
	}
	if id != 2 {
		t.Errorf("Expected auto-generated id to be 2, got %d", id)
	}
}

func TestBigIncrements(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	SetDB(db, &SQLiteDialect{})

	err := Table("events").Create(func() {
		BigIncrements("id")
		String("name")
		Timestamps()
	})

	if err != nil {
		t.Fatalf("Failed to create table with BigIncrements: %v", err)
	}

	// Insert without specifying id
	result, err := db.Exec("INSERT INTO events (name) VALUES ('Event 1')")
	if err != nil {
		t.Fatalf("Failed to insert record: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get last insert id: %v", err)
	}
	if id != 1 {
		t.Errorf("Expected auto-generated id to be 1, got %d", id)
	}
}

func TestIncrementsDialects(t *testing.T) {
	tests := []struct {
		name    string
		dialect Dialect
		check   string
	}{
		{"PostgreSQL", &PostgresDialect{}, "SERIAL"},
		{"MySQL", &MySQLDialect{}, "AUTO_INCREMENT"},
		{"SQLite", &SQLiteDialect{}, "AUTOINCREMENT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := &TableBuilder{
				tableName: "products",
				columns: []*Column{
					{name: "id", dataType: "integer", primary: true, autoIncrement: true},
					{name: "name", dataType: "string"},
				},
			}

			sql := tt.dialect.BuildCreateTable(tb)
			if !contains(sql, tt.check) {
				t.Errorf("%s dialect should generate %s.\nGot: %s", tt.name, tt.check, sql)
			}
		})
	}
}

func TestUlidTableCreation(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	SetDB(db, &SQLiteDialect{})

	err := Table("events").Create(func() {
		Ulid("id").Primary()
		String("name")
		Timestamps()
	})

	if err != nil {
		t.Fatalf("Failed to create table with ULID primary key: %v", err)
	}

	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='events'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Table was not created: %v", err)
	}

	// Insert and retrieve a record using a ULID-format id
	_, err = db.Exec("INSERT INTO events (id, name) VALUES ('01ARZ3NDEKTSV4RRFFQ69G5FAV', 'Launch')")
	if err != nil {
		t.Fatalf("Failed to insert record with ULID id: %v", err)
	}

	var name string
	err = db.QueryRow("SELECT name FROM events WHERE id = '01ARZ3NDEKTSV4RRFFQ69G5FAV'").Scan(&name)
	if err != nil {
		t.Fatalf("Failed to retrieve record by ULID id: %v", err)
	}
	if name != "Launch" {
		t.Errorf("Expected name 'Launch', got '%s'", name)
	}
}

func TestUlidDialects(t *testing.T) {
	tests := []struct {
		name    string
		dialect Dialect
		check   string
	}{
		{"PostgreSQL", &PostgresDialect{}, "CHAR(26)"},
		{"MySQL", &MySQLDialect{}, "CHAR(26) CHARACTER SET ascii COLLATE ascii_general_ci"},
		{"SQLite", &SQLiteDialect{}, "TEXT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := &TableBuilder{
				tableName: "events",
				columns: []*Column{
					{name: "id", dataType: "ulid", primary: true},
					{name: "name", dataType: "string"},
				},
			}

			sql := tt.dialect.BuildCreateTable(tb)
			if !contains(sql, tt.check) {
				t.Errorf("%s dialect should use %s for ULID.\nGot: %s", tt.name, tt.check, sql)
			}
		})
	}
}

func TestBigIncrementsDialects(t *testing.T) {
	tests := []struct {
		name    string
		dialect Dialect
		check   string
	}{
		{"PostgreSQL", &PostgresDialect{}, "BIGSERIAL"},
		{"MySQL", &MySQLDialect{}, "BIGINT"},
		{"SQLite", &SQLiteDialect{}, "INTEGER"}, // SQLite uses INTEGER for all integers
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := &TableBuilder{
				tableName: "events",
				columns: []*Column{
					{name: "id", dataType: "bigint", primary: true, autoIncrement: true},
					{name: "name", dataType: "string"},
				},
			}

			sql := tt.dialect.BuildCreateTable(tb)
			if !contains(sql, tt.check) {
				t.Errorf("%s dialect should generate %s.\nGot: %s", tt.name, tt.check, sql)
			}
		})
	}
}
