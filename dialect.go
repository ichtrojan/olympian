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
	// SQL clauses and DML
	"select": true, "from": true, "where": true, "join": true, "on": true,
	"insert": true, "update": true, "delete": true, "into": true, "create": true,
	"alter": true, "drop": true, "truncate": true, "replace": true,

	// Logical operators
	"and": true, "or": true, "not": true, "like": true, "in": true,
	"between": true, "is": true, "exists": true, "having": true,

	// Sorting and grouping
	"order": true, "group": true, "by": true, "asc": true, "desc": true,
	"limit": true, "offset": true, "distinct": true,

	// Joins
	"inner": true, "outer": true, "left": true, "right": true, "cross": true,
	"natural": true, "using": true,

	// Set operations
	"union": true, "intersect": true, "except": true, "all": true,

	// Conditional
	"case": true, "when": true, "then": true, "else": true,

	// Constraints and keys
	"primary": true, "foreign": true, "key": true, "index": true,
	"references": true, "constraint": true, "unique": true, "check": true,
	"default": true, "null": true,

	// Table/column keywords
	"table": true, "column": true, "add": true, "modify": true,

	// Referential actions
	"cascade": true, "restrict": true, "set": true,

	// Transaction keywords
	"begin": true, "commit": true, "rollback": true, "start": true, "end": true,

	// Window functions and analytics
	"window": true, "over": true, "partition": true, "row": true, "rows": true,
	"range": true, "rank": true, "filter": true, "recursive": true, "with": true,

	// Types and casting
	"cast": true, "interval": true, "some": true, "any": true,

	// Common column names that are reserved
	"type": true, "user": true, "role": true, "status": true, "name": true,
	"value": true, "values": true, "usage": true, "action": true, "comment": true,
	"date": true, "time": true, "timestamp": true, "year": true, "month": true,
	"day": true, "hour": true, "minute": true, "second": true, "position": true,
	"level": true, "mode": true, "system": true, "session": true, "language": true,
	"domain": true, "scope": true, "state": true, "zone": true, "data": true,
	"number": true, "result": true, "size": true, "source": true, "work": true,
	"read": true, "write": true, "input": true, "output": true, "option": true,
	"open": true, "close": true, "release": true, "call": true, "signal": true,

	// Misc reserved
	"collate": true, "escape": true, "grant": true, "revoke": true,
}

func escapeColumnName(name string, dialect Dialect) string {
	if reservedKeywords[strings.ToLower(name)] {
		switch dialect.(type) {
		case *MySQLDialect:
			return fmt.Sprintf("`%s`", name)
		case *PostgresDialect, *SQLiteDialect:
			return fmt.Sprintf(`"%s"`, name)
		}
	}
	return name
}

func escapeTableName(name string, dialect Dialect) string {
	if reservedKeywords[strings.ToLower(name)] {
		switch dialect.(type) {
		case *MySQLDialect:
			return fmt.Sprintf("`%s`", name)
		case *PostgresDialect, *SQLiteDialect:
			return fmt.Sprintf(`"%s"`, name)
		}
	}
	return name
}

// buildFKConstraint generates a foreign key constraint clause for CREATE TABLE statements.
func buildFKConstraint(tableName string, col *Column, dialect Dialect, named bool) string {
	var fkDef string
	if named {
		fkDef = fmt.Sprintf("  CONSTRAINT fk_%s_%s FOREIGN KEY (%s) REFERENCES %s(%s)",
			tableName, col.name, escapeColumnName(col.name, dialect),
			escapeTableName(col.refTable, dialect), escapeColumnName(col.refColumn, dialect))
	} else {
		// SQLite style — no constraint name
		fkDef = fmt.Sprintf("  FOREIGN KEY (%s) REFERENCES %s(%s)",
			col.name, col.refTable, col.refColumn)
	}
	if col.onDelete != "" {
		fkDef += fmt.Sprintf(" ON DELETE %s", strings.ToUpper(col.onDelete))
	}
	if col.onUpdate != "" {
		fkDef += fmt.Sprintf(" ON UPDATE %s", strings.ToUpper(col.onUpdate))
	}
	return fkDef
}

// buildFKConstraintFromFK generates a foreign key constraint clause from a ForeignKey struct.
func buildFKConstraintFromFK(tableName string, fk *ForeignKey, dialect Dialect, named bool) string {
	cols := strings.Join(fk.columns, ", ")
	refCols := strings.Join(fk.refColumns, ", ")

	var fkDef string
	if named {
		fkDef = fmt.Sprintf("  CONSTRAINT fk_%s_%s FOREIGN KEY (%s) REFERENCES %s(%s)",
			tableName, fk.columns[0], cols,
			escapeTableName(fk.refTable, dialect), refCols)
	} else {
		fkDef = fmt.Sprintf("  FOREIGN KEY (%s) REFERENCES %s(%s)",
			cols, fk.refTable, refCols)
	}
	if fk.onDelete != "" {
		fkDef += fmt.Sprintf(" ON DELETE %s", strings.ToUpper(fk.onDelete))
	}
	if fk.onUpdate != "" {
		fkDef += fmt.Sprintf(" ON UPDATE %s", strings.ToUpper(fk.onUpdate))
	}
	return fkDef
}

// buildAlterAddFK generates an ALTER TABLE ADD CONSTRAINT statement for a foreign key.
func buildAlterAddFK(tableName string, col *Column, dialect Dialect) string {
	fkSQL := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT fk_%s_%s FOREIGN KEY (%s) REFERENCES %s(%s)",
		escapeTableName(tableName, dialect), tableName, col.name,
		escapeColumnName(col.name, dialect),
		escapeTableName(col.refTable, dialect), escapeColumnName(col.refColumn, dialect))
	if col.onDelete != "" {
		fkSQL += fmt.Sprintf(" ON DELETE %s", strings.ToUpper(col.onDelete))
	}
	if col.onUpdate != "" {
		fkSQL += fmt.Sprintf(" ON UPDATE %s", strings.ToUpper(col.onUpdate))
	}
	return fkSQL
}

// buildAlterAddFKFromFK generates an ALTER TABLE ADD CONSTRAINT from a ForeignKey struct.
func buildAlterAddFKFromFK(tableName string, fk *ForeignKey, dialect Dialect) string {
	cols := strings.Join(fk.columns, ", ")
	refCols := strings.Join(fk.refColumns, ", ")

	fkSQL := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT fk_%s_%s FOREIGN KEY (%s) REFERENCES %s(%s)",
		escapeTableName(tableName, dialect), tableName, fk.columns[0], cols,
		escapeTableName(fk.refTable, dialect), refCols)
	if fk.onDelete != "" {
		fkSQL += fmt.Sprintf(" ON DELETE %s", strings.ToUpper(fk.onDelete))
	}
	if fk.onUpdate != "" {
		fkSQL += fmt.Sprintf(" ON UPDATE %s", strings.ToUpper(fk.onUpdate))
	}
	return fkSQL
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
			columnDefs = append(columnDefs, buildFKConstraint(tb.tableName, col, d, true))
		}
	}

	// Add foreign keys defined using Foreign()
	for _, fk := range tb.foreignKeys {
		columnDefs = append(columnDefs, buildFKConstraintFromFK(tb.tableName, fk, d, true))
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
		if col.isChange {
			// PostgreSQL requires separate statements for type, nullability, and default
			colName := escapeColumnName(col.name, d)

			// Change the data type
			sqls = append(sqls, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s",
				tb.tableName, colName, d.GetDataType(col)))

			// Change nullability
			if col.nullable {
				sqls = append(sqls, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL",
					tb.tableName, colName))
			} else {
				sqls = append(sqls, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL",
					tb.tableName, colName))
			}

			// Change default
			if col.defaultValue != nil {
				var defaultVal string
				if col.dataType == "boolean" || col.dataType == "integer" || col.dataType == "bigint" {
					defaultVal = *col.defaultValue
				} else {
					defaultVal = fmt.Sprintf("'%s'", *col.defaultValue)
				}
				sqls = append(sqls, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s",
					tb.tableName, colName, defaultVal))
			}

			// Handle foreign key on changed column: drop old FK and re-add
			if col.refTable != "" && col.refColumn != "" {
				sqls = append(sqls, fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS fk_%s_%s",
					tb.tableName, tb.tableName, col.name))
				sqls = append(sqls, buildAlterAddFK(tb.tableName, col, d))
			}
		} else {
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

			// Add foreign key constraint for inline references
			if col.refTable != "" && col.refColumn != "" {
				sqls = append(sqls, buildAlterAddFK(tb.tableName, col, d))
			}
		}
	}

	// Add foreign keys defined using Foreign()
	for _, fk := range tb.foreignKeys {
		sqls = append(sqls, buildAlterAddFKFromFK(tb.tableName, fk, d))
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
			columnDefs = append(columnDefs, buildFKConstraint(tb.tableName, col, d, true))
		}
	}

	// Add foreign keys defined using Foreign()
	for _, fk := range tb.foreignKeys {
		columnDefs = append(columnDefs, buildFKConstraintFromFK(tb.tableName, fk, d, true))
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
		var query string
		if col.isChange {
			query = fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s %s",
				tb.tableName, escapeColumnName(col.name, d), d.GetDataType(col))
		} else {
			query = fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
				tb.tableName, escapeColumnName(col.name, d), d.GetDataType(col))
		}

		if !col.nullable {
			query += " NOT NULL"
		} else if col.isChange {
			query += " NULL"
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

		// Add foreign key constraint for inline references
		if col.refTable != "" && col.refColumn != "" {
			if col.isChange {
				// Drop existing FK before re-adding
				sqls = append(sqls, fmt.Sprintf("ALTER TABLE %s DROP FOREIGN KEY IF EXISTS fk_%s_%s",
					tb.tableName, tb.tableName, col.name))
			}
			sqls = append(sqls, buildAlterAddFK(tb.tableName, col, d))
		}
	}

	// Add foreign keys defined using Foreign()
	for _, fk := range tb.foreignKeys {
		sqls = append(sqls, buildAlterAddFKFromFK(tb.tableName, fk, d))
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
		def := fmt.Sprintf("  %s %s", escapeColumnName(col.name, d), d.GetDataType(col))

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
			def += fmt.Sprintf(" CHECK (%s IN (%s))", escapeColumnName(col.name, d), strings.Join(enumVals, ", "))
		}

		columnDefs = append(columnDefs, def)
	}

	// Add inline foreign keys defined on columns
	for _, col := range tb.columns {
		if col.refTable != "" && col.refColumn != "" {
			columnDefs = append(columnDefs, buildFKConstraint(tb.tableName, col, d, false))
		}
	}

	// Add foreign keys defined using Foreign()
	for _, fk := range tb.foreignKeys {
		columnDefs = append(columnDefs, buildFKConstraintFromFK(tb.tableName, fk, d, false))
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
		if col.isChange {
			// SQLite does not support modifying columns directly
			// This requires table recreation which is not currently supported
			sqls = append(sqls, fmt.Sprintf("-- ERROR: SQLite does not support MODIFY COLUMN for '%s'. Manual table recreation required.", col.name))
			continue
		}

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

		// Add CHECK constraint for enum types inline
		if col.dataType == "enum" && len(col.enumValues) > 0 {
			var enumVals []string
			for _, v := range col.enumValues {
				enumVals = append(enumVals, fmt.Sprintf("'%s'", v))
			}
			query += fmt.Sprintf(" CHECK (%s IN (%s))", escapeColumnName(col.name, d), strings.Join(enumVals, ", "))
		}

		sqls = append(sqls, query)

		// SQLite does not support adding foreign key constraints via ALTER TABLE
		if col.refTable != "" && col.refColumn != "" {
			sqls = append(sqls, fmt.Sprintf("-- WARNING: SQLite does not support adding foreign keys via ALTER TABLE for column '%s'. Foreign key must be defined at table creation.", col.name))
		}
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
