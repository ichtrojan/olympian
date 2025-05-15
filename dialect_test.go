package olympian

import (
	"strings"
	"testing"
)

func TestPostgresDialectDataTypes(t *testing.T) {
	dialect := &PostgresDialect{}

	tests := []struct {
		column   *Column
		expected string
	}{
		{&Column{dataType: "uuid"}, "UUID"},
		{&Column{dataType: "string"}, "VARCHAR(255)"},
		{&Column{dataType: "text"}, "TEXT"},
		{&Column{dataType: "integer"}, "INTEGER"},
		{&Column{dataType: "bigint"}, "BIGINT"},
		{&Column{dataType: "boolean"}, "BOOLEAN"},
		{&Column{dataType: "timestamp"}, "TIMESTAMP"},
		{&Column{dataType: "date"}, "DATE"},
		{&Column{dataType: "json"}, "JSONB"},
		{&Column{dataType: "decimal(10,2)"}, "DECIMAL(10,2)"},
		{&Column{dataType: "integer", autoIncrement: true}, "SERIAL"},
		{&Column{dataType: "bigint", autoIncrement: true}, "BIGSERIAL"},
	}

	for _, tt := range tests {
		result := dialect.GetDataType(tt.column)
		if result != tt.expected {
			t.Errorf("Expected %s for %s, got %s", tt.expected, tt.column.dataType, result)
		}
	}
}

func TestMySQLDialectDataTypes(t *testing.T) {
	dialect := &MySQLDialect{}

	tests := []struct {
		column   *Column
		expected string
	}{
		{&Column{dataType: "uuid"}, "CHAR(36)"},
		{&Column{dataType: "string"}, "VARCHAR(255)"},
		{&Column{dataType: "text"}, "TEXT"},
		{&Column{dataType: "integer"}, "INT"},
		{&Column{dataType: "bigint"}, "BIGINT"},
		{&Column{dataType: "boolean"}, "TINYINT(1)"},
		{&Column{dataType: "timestamp"}, "TIMESTAMP"},
		{&Column{dataType: "date"}, "DATE"},
		{&Column{dataType: "json"}, "JSON"},
		{&Column{dataType: "decimal(10,2)"}, "DECIMAL(10,2)"},
	}

	for _, tt := range tests {
		result := dialect.GetDataType(tt.column)
		if result != tt.expected {
			t.Errorf("Expected %s for %s, got %s", tt.expected, tt.column.dataType, result)
		}
	}
}

func TestSQLiteDialectDataTypes(t *testing.T) {
	dialect := &SQLiteDialect{}

	tests := []struct {
		column   *Column
		expected string
	}{
		{&Column{dataType: "uuid"}, "TEXT"},
		{&Column{dataType: "string"}, "TEXT"},
		{&Column{dataType: "text"}, "TEXT"},
		{&Column{dataType: "integer"}, "INTEGER"},
		{&Column{dataType: "bigint"}, "INTEGER"},
		{&Column{dataType: "boolean"}, "INTEGER"},
		{&Column{dataType: "timestamp"}, "TEXT"},
		{&Column{dataType: "date"}, "TEXT"},
		{&Column{dataType: "json"}, "TEXT"},
		{&Column{dataType: "decimal(10,2)"}, "REAL"},
	}

	for _, tt := range tests {
		result := dialect.GetDataType(tt.column)
		if result != tt.expected {
			t.Errorf("Expected %s for %s, got %s", tt.expected, tt.column.dataType, result)
		}
	}
}

func TestPostgresCreateTableSQL(t *testing.T) {
	dialect := &PostgresDialect{}

	tb := &TableBuilder{
		tableName: "users",
		columns: []*Column{
			{name: "id", dataType: "uuid", primary: true, nullable: false},
			{name: "name", dataType: "string", nullable: false},
			{name: "email", dataType: "string", nullable: true, unique: true},
		},
	}

	sql := dialect.BuildCreateTable(tb)

	if !strings.Contains(sql, "CREATE TABLE IF NOT EXISTS users") {
		t.Error("SQL should contain CREATE TABLE IF NOT EXISTS users")
	}

	if !strings.Contains(sql, "id UUID") {
		t.Error("SQL should contain id UUID")
	}

	if !strings.Contains(sql, "PRIMARY KEY") {
		t.Error("SQL should contain PRIMARY KEY")
	}

	if !strings.Contains(sql, "UNIQUE") {
		t.Error("SQL should contain UNIQUE for email")
	}
}

func TestMySQLCreateTableSQL(t *testing.T) {
	dialect := &MySQLDialect{}

	tb := &TableBuilder{
		tableName: "users",
		columns: []*Column{
			{name: "id", dataType: "integer", primary: true, autoIncrement: true},
			{name: "name", dataType: "string"},
		},
	}

	sql := dialect.BuildCreateTable(tb)

	if !strings.Contains(sql, "CREATE TABLE IF NOT EXISTS users") {
		t.Error("SQL should contain CREATE TABLE IF NOT EXISTS users")
	}

	if !strings.Contains(sql, "AUTO_INCREMENT") {
		t.Error("SQL should contain AUTO_INCREMENT")
	}

	if !strings.Contains(sql, "ENGINE=InnoDB") {
		t.Error("SQL should contain ENGINE=InnoDB")
	}
}

func TestSQLiteCreateTableSQL(t *testing.T) {
	dialect := &SQLiteDialect{}

	tb := &TableBuilder{
		tableName: "users",
		columns: []*Column{
			{name: "id", dataType: "integer", primary: true, autoIncrement: true},
			{name: "name", dataType: "string"},
		},
	}

	sql := dialect.BuildCreateTable(tb)

	if !strings.Contains(sql, "CREATE TABLE IF NOT EXISTS users") {
		t.Error("SQL should contain CREATE TABLE IF NOT EXISTS users")
	}

	if !strings.Contains(sql, "AUTOINCREMENT") {
		t.Error("SQL should contain AUTOINCREMENT")
	}
}

func TestForeignKeySQL(t *testing.T) {
	dialect := &PostgresDialect{}

	tb := &TableBuilder{
		tableName: "users",
		columns: []*Column{
			{name: "id", dataType: "uuid", primary: true},
		},
		foreignKeys: []*ForeignKey{
			{
				column:    "business_id",
				refTable:  "businesses",
				refColumn: "id",
				onDelete:  "cascade",
				onUpdate:  "restrict",
			},
		},
	}

	sql := dialect.BuildCreateTable(tb)

	if !strings.Contains(sql, "FOREIGN KEY") {
		t.Error("SQL should contain FOREIGN KEY")
	}

	if !strings.Contains(sql, "REFERENCES businesses(id)") {
		t.Error("SQL should contain REFERENCES businesses(id)")
	}

	if !strings.Contains(sql, "ON DELETE CASCADE") {
		t.Error("SQL should contain ON DELETE CASCADE")
	}

	if !strings.Contains(sql, "ON UPDATE RESTRICT") {
		t.Error("SQL should contain ON UPDATE RESTRICT")
	}
}

func TestDefaultValuesSQL(t *testing.T) {
	dialect := &PostgresDialect{}

	trueVal := "true"
	oneVal := "1"

	tb := &TableBuilder{
		tableName: "users",
		columns: []*Column{
			{name: "id", dataType: "uuid", primary: true},
			{name: "active", dataType: "boolean", defaultValue: &trueVal},
			{name: "status", dataType: "integer", defaultValue: &oneVal},
		},
	}

	sql := dialect.BuildCreateTable(tb)

	if !strings.Contains(sql, "DEFAULT true") {
		t.Error("SQL should contain DEFAULT true for boolean")
	}

	if !strings.Contains(sql, "DEFAULT 1") {
		t.Error("SQL should contain DEFAULT 1 for integer")
	}
}

func TestModifyTableSQL(t *testing.T) {
	dialect := &PostgresDialect{}

	tb := &TableBuilder{
		tableName: "users",
		columns: []*Column{
			{name: "age", dataType: "integer", nullable: true},
		},
	}

	sqls := dialect.BuildModifyTable(tb)

	if len(sqls) == 0 {
		t.Error("Should return at least one SQL statement")
	}

	if !strings.Contains(sqls[0], "ALTER TABLE users ADD COLUMN age") {
		t.Error("SQL should contain ALTER TABLE users ADD COLUMN age")
	}
}

func TestMySQLAfterColumn(t *testing.T) {
	dialect := &MySQLDialect{}

	afterCol := "name"
	tb := &TableBuilder{
		tableName: "users",
		columns: []*Column{
			{name: "age", dataType: "integer", nullable: true, afterColumn: &afterCol},
		},
	}

	sqls := dialect.BuildModifyTable(tb)

	if !strings.Contains(sqls[0], "AFTER name") {
		t.Error("SQL should contain AFTER name")
	}
}

func TestDropTableSQL(t *testing.T) {
	dialects := []Dialect{
		&PostgresDialect{},
		&MySQLDialect{},
		&SQLiteDialect{},
	}

	for _, dialect := range dialects {
		sql := dialect.BuildDropTable("users")
		if !strings.Contains(sql, "DROP TABLE IF EXISTS users") {
			t.Errorf("Dialect %T should generate DROP TABLE IF EXISTS users", dialect)
		}
	}
}

func TestDropColumnSQL(t *testing.T) {
	dialects := []Dialect{
		&PostgresDialect{},
		&MySQLDialect{},
		&SQLiteDialect{},
	}

	for _, dialect := range dialects {
		sql := dialect.BuildDropColumn("users", "age")
		if !strings.Contains(sql, "ALTER TABLE users DROP COLUMN age") {
			t.Errorf("Dialect %T should generate ALTER TABLE users DROP COLUMN age", dialect)
		}
	}
}
