package olympian

import (
	"fmt"
	"strings"
)

type Dialect interface {
	BuildCreateTable(tb *TableBuilder) string
	BuildModifyTable(tb *TableBuilder) []string
	BuildDropTable(tableName string) string
	BuildDropColumn(tableName, columnName string) string
	GetDataType(column *Column) string
	BuildIndexStatements(tb *TableBuilder) []string
	Placeholder(n int) string
}

type PostgresDialect struct{}
type MySQLDialect struct{}
type SQLiteDialect struct{}

var reservedKeywords = map[string]bool{
	"limit": true, "order": true, "group": true, "key": true, "index": true,
	"type": true, "desc": true, "asc": true, "primary": true, "foreign": true,
	"references": true, "constraint": true, "table": true, "column": true,
	"select": true, "from": true, "where": true, "join": true, "on": true,
	"and": true, "or": true, "not": true, "like": true, "in": true,
	"between": true, "is": true, "null": true, "default": true, "unique": true,
	"check": true, "cascade": true, "restrict": true, "set": true, "user": true,
	"end": true, "start": true, "begin": true, "commit": true, "rollback": true,
	"interval": true, "status": true, "name": true, "value": true, "values": true,
}

func escapeColumnName(name string, dialect Dialect) string {
	if reservedKeywords[strings.ToLower(name)] {
		switch dialect.(type) {
		case *MySQLDialect:
			return fmt.Sprintf("`%s`", name)
		case *PostgresDialect:
			return fmt.Sprintf(`"%s"`, name)
		}
	}
	return name
}

func (d *PostgresDialect) GetDataType(col *Column) string {
	switch col.dataType {
	case "uuid":
		return "UUID"
	case "string":
		return "VARCHAR(255)"
	case "text":
		return "TEXT"
	case "tinyint":
		return "SMALLINT"
	case "integer":
		if col.autoIncrement {
			return "SERIAL"
		}
		return "INTEGER"
	case "bigint":
		if col.autoIncrement {
			return "BIGSERIAL"
		}
		return "BIGINT"
	case "boolean":
		return "BOOLEAN"
	case "timestamp":
		return "TIMESTAMP"
	case "date":
		return "DATE"
	case "json":
		return "JSONB"
	case "enum":
		return "VARCHAR(255)"
	default:
		if strings.HasPrefix(col.dataType, "decimal") {
			return "DECIMAL" + strings.TrimPrefix(col.dataType, "decimal")
		}
		return col.dataType
	}
}

func (d *PostgresDialect) BuildCreateTable(tb *TableBuilder) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (", tb.tableName))

	var columnDefs []string
	for _, col := range tb.columns {
		def := fmt.Sprintf("  %s %s", escapeColumnName(col.name, d), d.GetDataType(col))

		if col.primary {
			def += " PRIMARY KEY"
		}
		if !col.nullable {
			def += " NOT NULL"
		}
		if col.unique && !col.primary {
			def += " UNIQUE"
		}
		if col.defaultValue != nil {
			if col.dataType == "boolean" || col.dataType == "integer" || col.dataType == "bigint" {
				def += fmt.Sprintf(" DEFAULT %s", *col.defaultValue)
			} else {
				def += fmt.Sprintf(" DEFAULT '%s'", *col.defaultValue)
			}
		}
		columnDefs = append(columnDefs, def)

		// Add CHECK constraint for enum types
		if col.dataType == "enum" && len(col.enumValues) > 0 {
			var enumVals []string
			for _, v := range col.enumValues {
				enumVals = append(enumVals, fmt.Sprintf("'%s'", v))
			}
			checkConstraint := fmt.Sprintf("  CONSTRAINT chk_%s_%s CHECK (%s IN (%s))",
				tb.tableName, col.name, col.name, strings.Join(enumVals, ", "))
			columnDefs = append(columnDefs, checkConstraint)
		}
	}

	// Add inline foreign keys defined on columns
	for _, col := range tb.columns {
		if col.refTable != "" && col.refColumn != "" {
			fkDef := fmt.Sprintf("  CONSTRAINT fk_%s_%s FOREIGN KEY (%s) REFERENCES %s(%s)",
				tb.tableName, col.name, col.name, col.refTable, col.refColumn)

			if col.onDelete != "" {
				fkDef += fmt.Sprintf(" ON DELETE %s", strings.ToUpper(col.onDelete))
			}
			if col.onUpdate != "" {
				fkDef += fmt.Sprintf(" ON UPDATE %s", strings.ToUpper(col.onUpdate))
			}
			columnDefs = append(columnDefs, fkDef)
		}
	}

	// Add foreign keys defined using Foreign()
	for _, fk := range tb.foreignKeys {
		fkDef := fmt.Sprintf("  CONSTRAINT fk_%s_%s FOREIGN KEY (%s) REFERENCES %s(%s)",
			tb.tableName, fk.column, fk.column, fk.refTable, fk.refColumn)

		if fk.onDelete != "" {
			fkDef += fmt.Sprintf(" ON DELETE %s", strings.ToUpper(fk.onDelete))
		}
		if fk.onUpdate != "" {
			fkDef += fmt.Sprintf(" ON UPDATE %s", strings.ToUpper(fk.onUpdate))
		}
		columnDefs = append(columnDefs, fkDef)
	}

	// Add composite unique constraints
	for _, uc := range tb.uniqueConstraints {
		constraintName := uc.name
		if constraintName == "" {
			constraintName = fmt.Sprintf("uq_%s_%s", tb.tableName, strings.Join(uc.columns, "_"))
		}
		ucDef := fmt.Sprintf("  CONSTRAINT %s UNIQUE (%s)", constraintName, strings.Join(uc.columns, ", "))
		columnDefs = append(columnDefs, ucDef)
	}

	parts = append(parts, strings.Join(columnDefs, ",\n"))
	parts = append(parts, ");")

	return strings.Join(parts, "\n")
}

func (d *PostgresDialect) BuildModifyTable(tb *TableBuilder) []string {
	var sqls []string
	for _, col := range tb.columns {
		query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
			tb.tableName, escapeColumnName(col.name, d), d.GetDataType(col))

		if !col.nullable {
			query += " NOT NULL"
		}
		if col.defaultValue != nil {
			if col.dataType == "boolean" || col.dataType == "integer" || col.dataType == "bigint" {
				query += fmt.Sprintf(" DEFAULT %s", *col.defaultValue)
			} else {
				query += fmt.Sprintf(" DEFAULT '%s'", *col.defaultValue)
			}
		}
		sqls = append(sqls, query)

		// Add CHECK constraint for enum types
		if col.dataType == "enum" && len(col.enumValues) > 0 {
			var enumVals []string
			for _, v := range col.enumValues {
				enumVals = append(enumVals, fmt.Sprintf("'%s'", v))
			}
			checkConstraint := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT chk_%s_%s CHECK (%s IN (%s))",
				tb.tableName, tb.tableName, col.name, col.name, strings.Join(enumVals, ", "))
			sqls = append(sqls, checkConstraint)
		}
	}
	return sqls
}

func (d *PostgresDialect) BuildDropTable(tableName string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tableName)
}

func (d *PostgresDialect) BuildDropColumn(tableName, columnName string) string {
	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", tableName, columnName)
}

func (d *PostgresDialect) BuildIndexStatements(tb *TableBuilder) []string {
	var sqls []string
	for _, col := range tb.columns {
		if col.hasIndex {
			indexName := col.indexName
			if indexName == "" {
				// Auto-generate index name: idx_tablename_columnname
				indexName = fmt.Sprintf("idx_%s_%s", tb.tableName, col.name)
			}
			sql := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)", indexName, tb.tableName, col.name)
			sqls = append(sqls, sql)
		}
	}
	return sqls
}

func (d *PostgresDialect) Placeholder(n int) string {
	return fmt.Sprintf("$%d", n)
}

func (d *MySQLDialect) GetDataType(col *Column) string {
	switch col.dataType {
	case "uuid":
		return "CHAR(36) CHARACTER SET ascii COLLATE ascii_general_ci"
	case "string":
		return "VARCHAR(255)"
	case "text":
		return "TEXT"
	case "tinyint":
		return "TINYINT"
	case "integer":
		return "INT"
	case "bigint":
		return "BIGINT"
	case "boolean":
		return "TINYINT(1)"
	case "timestamp":
		return "TIMESTAMP"
	case "date":
		return "DATE"
	case "json":
		return "JSON"
	case "enum":
		if len(col.enumValues) > 0 {
			var enumVals []string
			for _, v := range col.enumValues {
				enumVals = append(enumVals, fmt.Sprintf("'%s'", v))
			}
			return fmt.Sprintf("ENUM(%s)", strings.Join(enumVals, ", "))
		}
		return "VARCHAR(255)"
	default:
		if strings.HasPrefix(col.dataType, "decimal") {
			return "DECIMAL" + strings.TrimPrefix(col.dataType, "decimal")
		}
		return col.dataType
	}
}

func (d *MySQLDialect) BuildCreateTable(tb *TableBuilder) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (", tb.tableName))

	var columnDefs []string
	for _, col := range tb.columns {
		def := fmt.Sprintf("  %s %s", escapeColumnName(col.name, d), d.GetDataType(col))

		if col.autoIncrement {
			def += " AUTO_INCREMENT"
		}
		if col.primary {
			def += " PRIMARY KEY"
		}
		if !col.nullable {
			def += " NOT NULL"
		}
		if col.unique && !col.primary {
			def += " UNIQUE"
		}
		if col.defaultValue != nil {
			if col.dataType == "boolean" || col.dataType == "integer" || col.dataType == "bigint" {
				def += fmt.Sprintf(" DEFAULT %s", *col.defaultValue)
			} else {
				def += fmt.Sprintf(" DEFAULT '%s'", *col.defaultValue)
			}
		}
		columnDefs = append(columnDefs, def)
	}

	// Add inline foreign keys defined on columns
	for _, col := range tb.columns {
		if col.refTable != "" && col.refColumn != "" {
			fkDef := fmt.Sprintf("  CONSTRAINT fk_%s_%s FOREIGN KEY (%s) REFERENCES %s(%s)",
				tb.tableName, col.name, col.name, col.refTable, col.refColumn)

			if col.onDelete != "" {
				fkDef += fmt.Sprintf(" ON DELETE %s", strings.ToUpper(col.onDelete))
			}
			if col.onUpdate != "" {
				fkDef += fmt.Sprintf(" ON UPDATE %s", strings.ToUpper(col.onUpdate))
			}
			columnDefs = append(columnDefs, fkDef)
		}
	}

	// Add foreign keys defined using Foreign()
	for _, fk := range tb.foreignKeys {
		fkDef := fmt.Sprintf("  CONSTRAINT fk_%s_%s FOREIGN KEY (%s) REFERENCES %s(%s)",
			tb.tableName, fk.column, fk.column, fk.refTable, fk.refColumn)

		if fk.onDelete != "" {
			fkDef += fmt.Sprintf(" ON DELETE %s", strings.ToUpper(fk.onDelete))
		}
		if fk.onUpdate != "" {
			fkDef += fmt.Sprintf(" ON UPDATE %s", strings.ToUpper(fk.onUpdate))
		}
		columnDefs = append(columnDefs, fkDef)
	}

	// Add composite unique constraints
	for _, uc := range tb.uniqueConstraints {
		constraintName := uc.name
		if constraintName == "" {
			constraintName = fmt.Sprintf("uq_%s_%s", tb.tableName, strings.Join(uc.columns, "_"))
		}
		ucDef := fmt.Sprintf("  CONSTRAINT %s UNIQUE (%s)", constraintName, strings.Join(uc.columns, ", "))
		columnDefs = append(columnDefs, ucDef)
	}

	parts = append(parts, strings.Join(columnDefs, ",\n"))
	parts = append(parts, ") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;")

	return strings.Join(parts, "\n")
}

func (d *MySQLDialect) BuildModifyTable(tb *TableBuilder) []string {
	var sqls []string
	for _, col := range tb.columns {
		query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
			tb.tableName, escapeColumnName(col.name, d), d.GetDataType(col))

		if !col.nullable {
			query += " NOT NULL"
		}
		if col.defaultValue != nil {
			if col.dataType == "boolean" || col.dataType == "integer" || col.dataType == "bigint" {
				query += fmt.Sprintf(" DEFAULT %s", *col.defaultValue)
			} else {
				query += fmt.Sprintf(" DEFAULT '%s'", *col.defaultValue)
			}
		}
		if col.afterColumn != nil {
			query += fmt.Sprintf(" AFTER %s", *col.afterColumn)
		}
		sqls = append(sqls, query)
	}
	return sqls
}

func (d *MySQLDialect) BuildDropTable(tableName string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
}

func (d *MySQLDialect) BuildDropColumn(tableName, columnName string) string {
	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", tableName, columnName)
}

func (d *MySQLDialect) BuildIndexStatements(tb *TableBuilder) []string {
	var sqls []string
	for _, col := range tb.columns {
		if col.hasIndex {
			indexName := col.indexName
			if indexName == "" {
				// Auto-generate index name: idx_tablename_columnname
				indexName = fmt.Sprintf("idx_%s_%s", tb.tableName, col.name)
			}
			sql := fmt.Sprintf("CREATE INDEX %s ON %s (%s)", indexName, tb.tableName, col.name)
			sqls = append(sqls, sql)
		}
	}
	return sqls
}

func (d *MySQLDialect) Placeholder(_ int) string {
	return "?"
}

func (d *SQLiteDialect) GetDataType(col *Column) string {
	switch col.dataType {
	case "uuid", "string":
		return "TEXT"
	case "text":
		return "TEXT"
	case "tinyint":
		return "INTEGER"
	case "integer":
		return "INTEGER"
	case "bigint":
		return "INTEGER"
	case "boolean":
		return "INTEGER"
	case "timestamp", "date":
		return "TEXT"
	case "json":
		return "TEXT"
	case "enum":
		return "TEXT"
	default:
		if strings.HasPrefix(col.dataType, "decimal") {
			return "REAL"
		}
		return "TEXT"
	}
}

func (d *SQLiteDialect) BuildCreateTable(tb *TableBuilder) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (", tb.tableName))

	var columnDefs []string
	for _, col := range tb.columns {
		def := fmt.Sprintf("  %s %s", col.name, d.GetDataType(col))

		if col.primary {
			def += " PRIMARY KEY"
			if col.autoIncrement {
				def += " AUTOINCREMENT"
			}
		}
		if !col.nullable {
			def += " NOT NULL"
		}
		if col.unique && !col.primary {
			def += " UNIQUE"
		}
		if col.defaultValue != nil {
			if col.dataType == "boolean" || col.dataType == "integer" || col.dataType == "bigint" {
				def += fmt.Sprintf(" DEFAULT %s", *col.defaultValue)
			} else {
				def += fmt.Sprintf(" DEFAULT '%s'", *col.defaultValue)
			}
		}

		// Add CHECK constraint for enum types inline
		if col.dataType == "enum" && len(col.enumValues) > 0 {
			var enumVals []string
			for _, v := range col.enumValues {
				enumVals = append(enumVals, fmt.Sprintf("'%s'", v))
			}
			def += fmt.Sprintf(" CHECK (%s IN (%s))", col.name, strings.Join(enumVals, ", "))
		}

		columnDefs = append(columnDefs, def)
	}

	// Add inline foreign keys defined on columns
	for _, col := range tb.columns {
		if col.refTable != "" && col.refColumn != "" {
			fkDef := fmt.Sprintf("  FOREIGN KEY (%s) REFERENCES %s(%s)",
				col.name, col.refTable, col.refColumn)

			if col.onDelete != "" {
				fkDef += fmt.Sprintf(" ON DELETE %s", strings.ToUpper(col.onDelete))
			}
			if col.onUpdate != "" {
				fkDef += fmt.Sprintf(" ON UPDATE %s", strings.ToUpper(col.onUpdate))
			}
			columnDefs = append(columnDefs, fkDef)
		}
	}

	// Add foreign keys defined using Foreign()
	for _, fk := range tb.foreignKeys {
		fkDef := fmt.Sprintf("  FOREIGN KEY (%s) REFERENCES %s(%s)",
			fk.column, fk.refTable, fk.refColumn)

		if fk.onDelete != "" {
			fkDef += fmt.Sprintf(" ON DELETE %s", strings.ToUpper(fk.onDelete))
		}
		if fk.onUpdate != "" {
			fkDef += fmt.Sprintf(" ON UPDATE %s", strings.ToUpper(fk.onUpdate))
		}
		columnDefs = append(columnDefs, fkDef)
	}

	// Add composite unique constraints
	for _, uc := range tb.uniqueConstraints {
		constraintName := uc.name
		if constraintName == "" {
			constraintName = fmt.Sprintf("uq_%s_%s", tb.tableName, strings.Join(uc.columns, "_"))
		}
		ucDef := fmt.Sprintf("  CONSTRAINT %s UNIQUE (%s)", constraintName, strings.Join(uc.columns, ", "))
		columnDefs = append(columnDefs, ucDef)
	}

	parts = append(parts, strings.Join(columnDefs, ",\n"))
	parts = append(parts, ");")

	return strings.Join(parts, "\n")
}

func (d *SQLiteDialect) BuildModifyTable(tb *TableBuilder) []string {
	var sqls []string
	for _, col := range tb.columns {
		query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
			tb.tableName, col.name, d.GetDataType(col))

		if !col.nullable {
			query += " NOT NULL"
		}
		if col.defaultValue != nil {
			if col.dataType == "boolean" || col.dataType == "integer" || col.dataType == "bigint" {
				query += fmt.Sprintf(" DEFAULT %s", *col.defaultValue)
			} else {
				query += fmt.Sprintf(" DEFAULT '%s'", *col.defaultValue)
			}
		}

		// Add CHECK constraint for enum types inline
		if col.dataType == "enum" && len(col.enumValues) > 0 {
			var enumVals []string
			for _, v := range col.enumValues {
				enumVals = append(enumVals, fmt.Sprintf("'%s'", v))
			}
			query += fmt.Sprintf(" CHECK (%s IN (%s))", col.name, strings.Join(enumVals, ", "))
		}

		sqls = append(sqls, query)
	}
	return sqls
}

func (d *SQLiteDialect) BuildDropTable(tableName string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
}

func (d *SQLiteDialect) BuildDropColumn(tableName, columnName string) string {
	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", tableName, columnName)
}

func (d *SQLiteDialect) BuildIndexStatements(tb *TableBuilder) []string {
	var sqls []string
	for _, col := range tb.columns {
		if col.hasIndex {
			indexName := col.indexName
			if indexName == "" {
				// Auto-generate index name: idx_tablename_columnname
				indexName = fmt.Sprintf("idx_%s_%s", tb.tableName, col.name)
			}
			sql := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)", indexName, tb.tableName, col.name)
			sqls = append(sqls, sql)
		}
	}
	return sqls
}

func (d *SQLiteDialect) Placeholder(_ int) string {
	return "?"
}
