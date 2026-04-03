package olympian

import (
	"fmt"
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
		{&Column{dataType: "uuid"}, "CHAR(36) CHARACTER SET ascii COLLATE ascii_general_ci"},
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
				columns:    []string{"business_id"},
				refTable:   "businesses",
				refColumns: []string{"id"},
				onDelete:   "cascade",
				onUpdate:   "restrict",
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

func TestInlineForeignKeySQL(t *testing.T) {
	tests := []struct {
		name    string
		dialect Dialect
	}{
		{"PostgreSQL", &PostgresDialect{}},
		{"MySQL", &MySQLDialect{}},
		{"SQLite", &SQLiteDialect{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := &TableBuilder{
				tableName: "card_transactions",
				columns: []*Column{
					{name: "id", dataType: "uuid", primary: true},
					{
						name:      "transaction_id",
						dataType:  "uuid",
						refTable:  "transactions",
						refColumn: "id",
						onDelete:  "cascade",
					},
					{
						name:      "card_id",
						dataType:  "uuid",
						refTable:  "cards",
						refColumn: "id",
						onDelete:  "cascade",
						onUpdate:  "restrict",
					},
				},
			}

			sql := tt.dialect.BuildCreateTable(tb)

			if !strings.Contains(sql, "FOREIGN KEY") {
				t.Error("SQL should contain FOREIGN KEY")
			}

			if !strings.Contains(sql, "REFERENCES transactions(id)") {
				t.Error("SQL should contain REFERENCES transactions(id)")
			}

			if !strings.Contains(sql, "REFERENCES cards(id)") {
				t.Error("SQL should contain REFERENCES cards(id)")
			}

			if !strings.Contains(sql, "ON DELETE CASCADE") {
				t.Error("SQL should contain ON DELETE CASCADE")
			}

			if !strings.Contains(sql, "ON UPDATE RESTRICT") {
				t.Error("SQL should contain ON UPDATE RESTRICT")
			}
		})
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

func TestChangeColumnSQL(t *testing.T) {
	defaultVal := "GBP"

	t.Run("MySQL", func(t *testing.T) {
		dialect := &MySQLDialect{}
		tb := &TableBuilder{
			tableName: "invoices",
			columns: []*Column{
				{name: "currency", dataType: "string", defaultValue: &defaultVal, isChange: true},
			},
		}

		sqls := dialect.BuildModifyTable(tb)

		if len(sqls) != 1 {
			t.Errorf("Expected 1 SQL statement, got %d", len(sqls))
		}

		if !strings.Contains(sqls[0], "MODIFY COLUMN") {
			t.Error("SQL should contain MODIFY COLUMN")
		}

		if !strings.Contains(sqls[0], "VARCHAR(255)") {
			t.Error("SQL should contain VARCHAR(255)")
		}

		if !strings.Contains(sqls[0], "DEFAULT 'GBP'") {
			t.Error("SQL should contain DEFAULT 'GBP'")
		}
	})

	t.Run("PostgreSQL", func(t *testing.T) {
		dialect := &PostgresDialect{}
		tb := &TableBuilder{
			tableName: "invoices",
			columns: []*Column{
				{name: "currency", dataType: "string", defaultValue: &defaultVal, isChange: true},
			},
		}

		sqls := dialect.BuildModifyTable(tb)

		if len(sqls) != 3 {
			t.Errorf("Expected 3 SQL statements (type, nullability, default), got %d", len(sqls))
		}

		if !strings.Contains(sqls[0], "ALTER COLUMN currency TYPE VARCHAR(255)") {
			t.Error("SQL should contain ALTER COLUMN currency TYPE VARCHAR(255)")
		}

		if !strings.Contains(sqls[1], "SET NOT NULL") {
			t.Error("SQL should contain SET NOT NULL")
		}

		if !strings.Contains(sqls[2], "SET DEFAULT 'GBP'") {
			t.Error("SQL should contain SET DEFAULT 'GBP'")
		}
	})

	t.Run("SQLite", func(t *testing.T) {
		dialect := &SQLiteDialect{}
		tb := &TableBuilder{
			tableName: "invoices",
			columns: []*Column{
				{name: "currency", dataType: "string", defaultValue: &defaultVal, isChange: true},
			},
		}

		sqls := dialect.BuildModifyTable(tb)

		if len(sqls) != 1 {
			t.Errorf("Expected 1 SQL statement, got %d", len(sqls))
		}

		if !strings.Contains(sqls[0], "-- ERROR") {
			t.Error("SQLite should return an error comment for MODIFY COLUMN")
		}
	})
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

func TestIndexSQL(t *testing.T) {
	tests := []struct {
		name    string
		dialect Dialect
	}{
		{"PostgreSQL", &PostgresDialect{}},
		{"MySQL", &MySQLDialect{}},
		{"SQLite", &SQLiteDialect{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := &TableBuilder{
				tableName: "users",
				columns: []*Column{
					{name: "id", dataType: "uuid", primary: true},
					{name: "email", dataType: "string", hasIndex: true}, // Auto-generated index name
					{name: "username", dataType: "string", hasIndex: true, indexName: "custom_idx"},
					{name: "name", dataType: "string"},
				},
			}

			sqls := tt.dialect.BuildIndexStatements(tb)

			if len(sqls) != 2 {
				t.Errorf("Expected 2 index statements, got %d", len(sqls))
			}

			// Check auto-generated index
			if !strings.Contains(sqls[0], "idx_users_email") {
				t.Error("Should contain auto-generated index name 'idx_users_email'")
			}

			if !strings.Contains(sqls[0], "CREATE INDEX") {
				t.Error("Should contain CREATE INDEX")
			}

			// Check custom index name
			if !strings.Contains(sqls[1], "custom_idx") {
				t.Error("Should contain custom index name 'custom_idx'")
			}

			if !strings.Contains(sqls[1], "ON users (username)") {
				t.Error("Should contain 'ON users (username)'")
			}
		})
	}
}

func TestModifyTableForeignKeys(t *testing.T) {
	afterCol := "role"

	t.Run("MySQL inline foreign key in modify table", func(t *testing.T) {
		dialect := &MySQLDialect{}
		tb := &TableBuilder{
			tableName: "users",
			columns: []*Column{
				{
					name:        "default_budget_id",
					dataType:    "uuid",
					nullable:    true,
					afterColumn: &afterCol,
					refTable:    "business_budgets",
					refColumn:   "id",
				},
			},
		}

		sqls := dialect.BuildModifyTable(tb)

		if len(sqls) < 2 {
			t.Fatalf("Expected at least 2 SQL statements, got %d", len(sqls))
		}

		if !strings.Contains(sqls[0], "ADD COLUMN") {
			t.Error("First SQL should be ADD COLUMN")
		}
		if !strings.Contains(sqls[0], "AFTER role") {
			t.Error("First SQL should contain AFTER role")
		}
		if !strings.Contains(sqls[1], "ADD CONSTRAINT fk_users_default_budget_id") {
			t.Error("Second SQL should contain ADD CONSTRAINT for foreign key")
		}
		if !strings.Contains(sqls[1], "FOREIGN KEY (default_budget_id) REFERENCES business_budgets(id)") {
			t.Error("Second SQL should contain FOREIGN KEY ... REFERENCES")
		}
	})

	t.Run("PostgreSQL inline foreign key in modify table", func(t *testing.T) {
		dialect := &PostgresDialect{}
		tb := &TableBuilder{
			tableName: "users",
			columns: []*Column{
				{
					name:      "default_budget_id",
					dataType:  "uuid",
					nullable:  true,
					refTable:  "business_budgets",
					refColumn: "id",
					onDelete:  "cascade",
				},
			},
		}

		sqls := dialect.BuildModifyTable(tb)

		if len(sqls) < 2 {
			t.Fatalf("Expected at least 2 SQL statements, got %d", len(sqls))
		}

		if !strings.Contains(sqls[1], "ADD CONSTRAINT fk_users_default_budget_id") {
			t.Error("Should contain ADD CONSTRAINT for foreign key")
		}
		if !strings.Contains(sqls[1], "ON DELETE CASCADE") {
			t.Error("Should contain ON DELETE CASCADE")
		}
	})

	t.Run("MySQL Foreign() style in modify table", func(t *testing.T) {
		dialect := &MySQLDialect{}
		tb := &TableBuilder{
			tableName: "users",
			columns: []*Column{
				{name: "business_id", dataType: "uuid", nullable: false},
			},
			foreignKeys: []*ForeignKey{
				{columns: []string{"business_id"}, refTable: "businesses", refColumns: []string{"id"}, onDelete: "cascade"},
			},
		}

		sqls := dialect.BuildModifyTable(tb)

		found := false
		for _, sql := range sqls {
			if strings.Contains(sql, "ADD CONSTRAINT fk_users_business_id") &&
				strings.Contains(sql, "REFERENCES businesses(id)") &&
				strings.Contains(sql, "ON DELETE CASCADE") {
				found = true
			}
		}
		if !found {
			t.Error("Should contain ALTER TABLE ADD CONSTRAINT for Foreign() style foreign key")
		}
	})
}

func TestChangeColumnWithForeignKey(t *testing.T) {
	t.Run("MySQL Change with FK drops and re-adds constraint", func(t *testing.T) {
		dialect := &MySQLDialect{}
		tb := &TableBuilder{
			tableName: "users",
			columns: []*Column{
				{
					name:      "business_id",
					dataType:  "uuid",
					nullable:  true,
					isChange:  true,
					refTable:  "businesses",
					refColumn: "id",
					onDelete:  "cascade",
				},
			},
		}

		sqls := dialect.BuildModifyTable(tb)

		foundDrop := false
		foundAdd := false
		for _, sql := range sqls {
			if strings.Contains(sql, "DROP FOREIGN KEY") && strings.Contains(sql, "fk_users_business_id") {
				foundDrop = true
			}
			if strings.Contains(sql, "ADD CONSTRAINT fk_users_business_id") &&
				strings.Contains(sql, "REFERENCES businesses(id)") &&
				strings.Contains(sql, "ON DELETE CASCADE") {
				foundAdd = true
			}
		}
		if !foundDrop {
			t.Error("Should drop existing FK constraint before re-adding")
		}
		if !foundAdd {
			t.Error("Should re-add FK constraint after column change")
		}
	})

	t.Run("PostgreSQL Change with FK drops and re-adds constraint", func(t *testing.T) {
		dialect := &PostgresDialect{}
		tb := &TableBuilder{
			tableName: "users",
			columns: []*Column{
				{
					name:      "business_id",
					dataType:  "uuid",
					nullable:  true,
					isChange:  true,
					refTable:  "businesses",
					refColumn: "id",
				},
			},
		}

		sqls := dialect.BuildModifyTable(tb)

		foundDrop := false
		foundAdd := false
		for _, sql := range sqls {
			if strings.Contains(sql, "DROP CONSTRAINT") && strings.Contains(sql, "fk_users_business_id") {
				foundDrop = true
			}
			if strings.Contains(sql, "ADD CONSTRAINT fk_users_business_id") &&
				strings.Contains(sql, "REFERENCES businesses(id)") {
				foundAdd = true
			}
		}
		if !foundDrop {
			t.Error("Should drop existing FK constraint before re-adding")
		}
		if !foundAdd {
			t.Error("Should re-add FK constraint after column change")
		}
	})
}

func TestForeignKeyColumnEscaping(t *testing.T) {
	t.Run("MySQL escapes reserved keywords in FK", func(t *testing.T) {
		dialect := &MySQLDialect{}
		tb := &TableBuilder{
			tableName: "orders",
			columns: []*Column{
				{name: "id", dataType: "uuid", primary: true},
				{
					name:      "user",
					dataType:  "uuid",
					refTable:  "user",
					refColumn: "key",
				},
			},
		}

		sql := dialect.BuildCreateTable(tb)
		if !strings.Contains(sql, "FOREIGN KEY (`user`) REFERENCES `user`(`key`)") {
			t.Errorf("Should escape reserved keywords in FK constraint.\nGot: %s", sql)
		}
	})

	t.Run("PostgreSQL escapes reserved keywords in FK", func(t *testing.T) {
		dialect := &PostgresDialect{}
		tb := &TableBuilder{
			tableName: "orders",
			columns: []*Column{
				{name: "id", dataType: "uuid", primary: true},
				{
					name:      "user",
					dataType:  "uuid",
					refTable:  "user",
					refColumn: "key",
				},
			},
		}

		sql := dialect.BuildCreateTable(tb)
		if !strings.Contains(sql, `FOREIGN KEY ("user") REFERENCES "user"("key")`) {
			t.Errorf("Should escape reserved keywords in FK constraint.\nGot: %s", sql)
		}
	})

	t.Run("SQLite escapes reserved keywords in column definitions", func(t *testing.T) {
		dialect := &SQLiteDialect{}
		tb := &TableBuilder{
			tableName: "events",
			columns: []*Column{
				{name: "id", dataType: "uuid", primary: true},
				{name: "role", dataType: "string"},
				{name: "status", dataType: "string"},
				{name: "order", dataType: "integer"},
			},
		}

		sql := dialect.BuildCreateTable(tb)
		if !strings.Contains(sql, `"role" TEXT`) {
			t.Errorf("SQLite should escape 'role' with double quotes.\nGot: %s", sql)
		}
		if !strings.Contains(sql, `"status" TEXT`) {
			t.Errorf("SQLite should escape 'status' with double quotes.\nGot: %s", sql)
		}
		if !strings.Contains(sql, `"order" INTEGER`) {
			t.Errorf("SQLite should escape 'order' with double quotes.\nGot: %s", sql)
		}
	})

	t.Run("SQLite escapes reserved keywords in ALTER TABLE", func(t *testing.T) {
		dialect := &SQLiteDialect{}
		tb := &TableBuilder{
			tableName: "events",
			columns: []*Column{
				{name: "type", dataType: "string", nullable: true},
			},
		}

		sqls := dialect.BuildModifyTable(tb)
		if len(sqls) == 0 {
			t.Fatal("Expected at least 1 SQL statement")
		}
		if !strings.Contains(sqls[0], `"type"`) {
			t.Errorf("SQLite should escape 'type' in ALTER TABLE.\nGot: %s", sqls[0])
		}
	})

	t.Run("New reserved keywords are escaped", func(t *testing.T) {
		dialect := &MySQLDialect{}
		tb := &TableBuilder{
			tableName: "users",
			columns: []*Column{
				{name: "id", dataType: "uuid", primary: true},
				{name: "role", dataType: "string"},
				{name: "date", dataType: "date"},
				{name: "action", dataType: "string"},
				{name: "comment", dataType: "text"},
				{name: "rank", dataType: "integer"},
				{name: "position", dataType: "integer"},
				{name: "offset", dataType: "integer"},
			},
		}

		sql := dialect.BuildCreateTable(tb)
		for _, keyword := range []string{"role", "date", "action", "comment", "rank", "position", "offset"} {
			escaped := fmt.Sprintf("`%s`", keyword)
			if !strings.Contains(sql, escaped) {
				t.Errorf("MySQL should escape '%s' with backticks.\nGot: %s", keyword, sql)
			}
		}
	})
}

func TestSelfReferencingForeignKey(t *testing.T) {
	t.Run("SQLite self-referencing FK in CreateTable", func(t *testing.T) {
		dialect := &SQLiteDialect{}
		tb := &TableBuilder{
			tableName: "categories",
			columns: []*Column{
				{name: "id", dataType: "uuid", primary: true},
				{name: "name", dataType: "string"},
				{
					name:      "parent_id",
					dataType:  "uuid",
					nullable:  true,
					refTable:  "categories",
					refColumn: "id",
					onDelete:  "set null",
				},
			},
		}

		sql := dialect.BuildCreateTable(tb)
		if !strings.Contains(sql, "FOREIGN KEY (parent_id) REFERENCES categories(id)") {
			t.Errorf("Should support self-referencing FK.\nGot: %s", sql)
		}
		if !strings.Contains(sql, "ON DELETE SET NULL") {
			t.Error("Should contain ON DELETE SET NULL")
		}
	})

	t.Run("MySQL self-referencing FK in CreateTable", func(t *testing.T) {
		dialect := &MySQLDialect{}
		tb := &TableBuilder{
			tableName: "categories",
			columns: []*Column{
				{name: "id", dataType: "uuid", primary: true},
				{name: "name", dataType: "string"},
				{
					name:      "parent_id",
					dataType:  "uuid",
					nullable:  true,
					refTable:  "categories",
					refColumn: "id",
				},
			},
		}

		sql := dialect.BuildCreateTable(tb)
		if !strings.Contains(sql, "CONSTRAINT fk_categories_parent_id FOREIGN KEY (parent_id) REFERENCES categories(id)") {
			t.Errorf("Should support self-referencing FK.\nGot: %s", sql)
		}
	})
}

func TestCompositeForeignKey(t *testing.T) {
	t.Run("MySQL composite FK in CreateTable", func(t *testing.T) {
		dialect := &MySQLDialect{}
		tb := &TableBuilder{
			tableName: "orders",
			columns: []*Column{
				{name: "id", dataType: "uuid", primary: true},
				{name: "user_id", dataType: "uuid"},
				{name: "tenant_id", dataType: "uuid"},
			},
			foreignKeys: []*ForeignKey{
				{
					columns:    []string{"user_id", "tenant_id"},
					refTable:   "users",
					refColumns: []string{"id", "tenant_id"},
					onDelete:   "cascade",
				},
			},
		}

		sql := dialect.BuildCreateTable(tb)
		if !strings.Contains(sql, "FOREIGN KEY (user_id, tenant_id) REFERENCES users(id, tenant_id)") {
			t.Errorf("Should support composite FK.\nGot: %s", sql)
		}
		if !strings.Contains(sql, "ON DELETE CASCADE") {
			t.Error("Should contain ON DELETE CASCADE")
		}
	})

	t.Run("PostgreSQL composite FK in CreateTable", func(t *testing.T) {
		dialect := &PostgresDialect{}
		tb := &TableBuilder{
			tableName: "orders",
			columns: []*Column{
				{name: "id", dataType: "uuid", primary: true},
				{name: "user_id", dataType: "uuid"},
				{name: "tenant_id", dataType: "uuid"},
			},
			foreignKeys: []*ForeignKey{
				{
					columns:    []string{"user_id", "tenant_id"},
					refTable:   "users",
					refColumns: []string{"id", "tenant_id"},
				},
			},
		}

		sql := dialect.BuildCreateTable(tb)
		if !strings.Contains(sql, "FOREIGN KEY (user_id, tenant_id) REFERENCES users(id, tenant_id)") {
			t.Errorf("Should support composite FK.\nGot: %s", sql)
		}
	})

	t.Run("MySQL composite FK in ModifyTable", func(t *testing.T) {
		dialect := &MySQLDialect{}
		tb := &TableBuilder{
			tableName: "orders",
			columns: []*Column{
				{name: "user_id", dataType: "uuid"},
				{name: "tenant_id", dataType: "uuid"},
			},
			foreignKeys: []*ForeignKey{
				{
					columns:    []string{"user_id", "tenant_id"},
					refTable:   "users",
					refColumns: []string{"id", "tenant_id"},
					onDelete:   "cascade",
				},
			},
		}

		sqls := dialect.BuildModifyTable(tb)

		found := false
		for _, sql := range sqls {
			if strings.Contains(sql, "FOREIGN KEY (user_id, tenant_id) REFERENCES users(id, tenant_id)") &&
				strings.Contains(sql, "ON DELETE CASCADE") {
				found = true
			}
		}
		if !found {
			t.Errorf("Should support composite FK in modify.\nGot: %v", sqls)
		}
	})

	t.Run("SQLite composite FK in CreateTable", func(t *testing.T) {
		dialect := &SQLiteDialect{}
		tb := &TableBuilder{
			tableName: "orders",
			columns: []*Column{
				{name: "id", dataType: "uuid", primary: true},
				{name: "user_id", dataType: "uuid"},
				{name: "tenant_id", dataType: "uuid"},
			},
			foreignKeys: []*ForeignKey{
				{
					columns:    []string{"user_id", "tenant_id"},
					refTable:   "users",
					refColumns: []string{"id", "tenant_id"},
				},
			},
		}

		sql := dialect.BuildCreateTable(tb)
		if !strings.Contains(sql, "FOREIGN KEY (user_id, tenant_id) REFERENCES users(id, tenant_id)") {
			t.Errorf("Should support composite FK.\nGot: %s", sql)
		}
	})
}

func TestPreviewCreate(t *testing.T) {
	dialect := &MySQLDialect{}
	tb := &TableBuilder{
		tableName:         "users",
		columns:           make([]*Column, 0),
		dialect:           dialect,
		foreignKeys:       make([]*ForeignKey, 0),
		uniqueConstraints: make([]*UniqueConstraint, 0),
	}

	sqls := tb.PreviewCreate(func() {
		currentBuilder = tb
		col := &Column{name: "id", dataType: "uuid", primary: true}
		tb.columns = append(tb.columns, col)
		col2 := &Column{name: "email", dataType: "string", hasIndex: true}
		tb.columns = append(tb.columns, col2)
	})

	if len(sqls) < 2 {
		t.Fatalf("PreviewCreate should return CREATE TABLE + index statements, got %d", len(sqls))
	}
	if !strings.Contains(sqls[0], "CREATE TABLE") {
		t.Error("First statement should be CREATE TABLE")
	}
	if !strings.Contains(sqls[1], "CREATE INDEX") {
		t.Error("Second statement should be CREATE INDEX")
	}
}

func TestPreviewModify(t *testing.T) {
	dialect := &MySQLDialect{}
	tb := &TableBuilder{
		tableName:         "users",
		columns:           make([]*Column, 0),
		dialect:           dialect,
		foreignKeys:       make([]*ForeignKey, 0),
		uniqueConstraints: make([]*UniqueConstraint, 0),
	}

	sqls := tb.PreviewModify(func() {
		currentBuilder = tb
		col := &Column{name: "age", dataType: "integer", nullable: true, hasIndex: true}
		tb.columns = append(tb.columns, col)
	})

	if len(sqls) < 2 {
		t.Fatalf("PreviewModify should return ALTER + index statements, got %d", len(sqls))
	}
	if !strings.Contains(sqls[0], "ALTER TABLE") {
		t.Error("First statement should be ALTER TABLE")
	}
	if !strings.Contains(sqls[1], "CREATE INDEX") {
		t.Error("Second statement should be CREATE INDEX for indexed column")
	}
}
