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

func escapeIdentifier(name string, dialect Dialect) string {
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

// escapeColumnName escapes a column name if it is a reserved keyword.
func escapeColumnName(name string, dialect Dialect) string {
	return escapeIdentifier(name, dialect)
}

// escapeTableName escapes a table name if it is a reserved keyword.
func escapeTableName(name string, dialect Dialect) string {
	return escapeIdentifier(name, dialect)
}

// escapeColumnList escapes each column name in a list and joins them with ", ".
func escapeColumnList(columns []string, dialect Dialect) string {
	escaped := make([]string, len(columns))
	for i, col := range columns {
		escaped[i] = escapeColumnName(col, dialect)
	}
	return strings.Join(escaped, ", ")
}

// buildCompositeIndexColumns renders a composite index's column list, appending
// " DESC" to any column marked descending.
func buildCompositeIndexColumns(ci *CompositeIndex, dialect Dialect) string {
	parts := make([]string, len(ci.columns))
	for i, col := range ci.columns {
		s := escapeColumnName(col, dialect)
		if ci.desc[col] {
			s += " DESC"
		}
		parts[i] = s
	}
	return strings.Join(parts, ", ")
}

// buildFKConstraint generates a foreign key constraint clause for CREATE TABLE statements.
func buildFKConstraint(tableName string, col *Column, dialect Dialect, named bool) string {
	var fkDef string
	if named {
		fkDef = fmt.Sprintf("  CONSTRAINT fk_%s_%s FOREIGN KEY (%s) REFERENCES %s(%s)",
			tableName, col.name, escapeColumnName(col.name, dialect),
			escapeTableName(col.refTable, dialect), escapeColumnName(col.refColumn, dialect))
	} else {
		fkDef = fmt.Sprintf("  FOREIGN KEY (%s) REFERENCES %s(%s)",
			escapeColumnName(col.name, dialect), escapeTableName(col.refTable, dialect), escapeColumnName(col.refColumn, dialect))
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
	cols := escapeColumnList(fk.columns, dialect)
	refCols := escapeColumnList(fk.refColumns, dialect)

	var fkDef string
	if named {
		fkDef = fmt.Sprintf("  CONSTRAINT fk_%s_%s FOREIGN KEY (%s) REFERENCES %s(%s)",
			tableName, fk.columns[0], cols,
			escapeTableName(fk.refTable, dialect), refCols)
	} else {
		fkDef = fmt.Sprintf("  FOREIGN KEY (%s) REFERENCES %s(%s)",
			cols, escapeTableName(fk.refTable, dialect), refCols)
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
	cols := escapeColumnList(fk.columns, dialect)
	refCols := escapeColumnList(fk.refColumns, dialect)

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
	case "ulid":
		return "CHAR(26)"
	case "string":
		length := 255
		if col.length > 0 {
			length = col.length
		}
		return fmt.Sprintf("VARCHAR(%d)", length)
	case "char":
		length := 1
		if col.length > 0 {
			length = col.length
		}
		return fmt.Sprintf("CHAR(%d)", length)
	case "text", "mediumtext", "longtext":
		return "TEXT"
	case "tinyint":
		return "SMALLINT"
	case "smallint":
		return "SMALLINT"
	case "mediumint":
		return "INTEGER"
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
	case "float":
		return "REAL"
	case "double":
		return "DOUBLE PRECISION"
	case "boolean":
		return "BOOLEAN"
	case "timestamp":
		return "TIMESTAMP"
	case "date":
		return "DATE"
	case "time":
		return "TIME"
	case "year":
		return "INTEGER"
	case "json":
		return "JSONB"
	case "binary":
		return "BYTEA"
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
	parts = append(parts, fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (", escapeTableName(tb.tableName, d)))

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
				tb.tableName, col.name, escapeColumnName(col.name, d), strings.Join(enumVals, ", "))
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
		ucDef := fmt.Sprintf("  CONSTRAINT %s UNIQUE (%s)", constraintName, escapeColumnList(uc.columns, d))
		columnDefs = append(columnDefs, ucDef)
	}

	parts = append(parts, strings.Join(columnDefs, ",\n"))
	parts = append(parts, ");")

	return strings.Join(parts, "\n")
}

func (d *PostgresDialect) BuildModifyTable(tb *TableBuilder) []string {
	var sqls []string
	escapedTable := escapeTableName(tb.tableName, d)
	for _, col := range tb.columns {
		if col.isChange {
			// PostgreSQL requires separate statements for type, nullability, and default
			colName := escapeColumnName(col.name, d)

			// Change the data type
			sqls = append(sqls, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s",
				escapedTable, colName, d.GetDataType(col)))

			// Change nullability
			if col.nullable {
				sqls = append(sqls, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL",
					escapedTable, colName))
			} else {
				sqls = append(sqls, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL",
					escapedTable, colName))
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
					escapedTable, colName, defaultVal))
			}

			// Handle foreign key on changed column: drop old FK and re-add
			if col.refTable != "" && col.refColumn != "" {
				sqls = append(sqls, fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS fk_%s_%s",
					escapedTable, tb.tableName, col.name))
				sqls = append(sqls, buildAlterAddFK(tb.tableName, col, d))
			}
		} else {
			query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
				escapedTable, escapeColumnName(col.name, d), d.GetDataType(col))

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
					escapedTable, tb.tableName, col.name, escapeColumnName(col.name, d), strings.Join(enumVals, ", "))
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
	return fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", escapeTableName(tableName, d))
}

func (d *PostgresDialect) BuildDropColumn(tableName, columnName string) string {
	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", escapeTableName(tableName, d), escapeColumnName(columnName, d))
}

func (d *PostgresDialect) BuildIndexStatements(tb *TableBuilder) []string {
	var sqls []string
	for _, col := range tb.columns {
		if col.hasIndex {
			indexName := col.indexName
			if indexName == "" {
				indexName = fmt.Sprintf("idx_%s_%s", tb.tableName, col.name)
			}
			sql := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)",
				indexName, escapeTableName(tb.tableName, d), escapeColumnName(col.name, d))
			sqls = append(sqls, sql)
		}
	}
	for _, ci := range tb.compositeIndexes {
		indexName := ci.name
		if indexName == "" {
			indexName = fmt.Sprintf("idx_%s_%s", tb.tableName, strings.Join(ci.columns, "_"))
		}
		sql := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)",
			indexName, escapeTableName(tb.tableName, d), buildCompositeIndexColumns(ci, d))
		sqls = append(sqls, sql)
	}
	return sqls
}

func (d *PostgresDialect) Placeholder(n int) string {
	return fmt.Sprintf("$%d", n)
}

func (d *MySQLDialect) GetDataType(col *Column) string {
	var baseType string
	switch col.dataType {
	case "uuid":
		return "CHAR(36) CHARACTER SET ascii COLLATE ascii_general_ci"
	case "ulid":
		return "CHAR(26) CHARACTER SET ascii COLLATE ascii_general_ci"
	case "string":
		length := 255
		if col.length > 0 {
			length = col.length
		}
		baseType = fmt.Sprintf("VARCHAR(%d)", length)
	case "char":
		length := 1
		if col.length > 0 {
			length = col.length
		}
		baseType = fmt.Sprintf("CHAR(%d)", length)
	case "text":
		baseType = "TEXT"
	case "mediumtext":
		baseType = "MEDIUMTEXT"
	case "longtext":
		baseType = "LONGTEXT"
	case "tinyint":
		baseType = "TINYINT"
	case "smallint":
		baseType = "SMALLINT"
	case "mediumint":
		baseType = "MEDIUMINT"
	case "integer":
		baseType = "INT"
	case "bigint":
		baseType = "BIGINT"
	case "float":
		baseType = "FLOAT"
	case "double":
		baseType = "DOUBLE"
	case "boolean":
		return "TINYINT(1)"
	case "timestamp":
		return "TIMESTAMP"
	case "date":
		return "DATE"
	case "time":
		return "TIME"
	case "year":
		return "YEAR"
	case "json":
		return "JSON"
	case "binary":
		return "BLOB"
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
			baseType = "DECIMAL" + strings.TrimPrefix(col.dataType, "decimal")
		} else {
			baseType = col.dataType
		}
	}
	if col.unsigned {
		baseType += " UNSIGNED"
	}
	return baseType
}

func (d *MySQLDialect) BuildCreateTable(tb *TableBuilder) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (", escapeTableName(tb.tableName, d)))

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
		if col.comment != "" {
			def += fmt.Sprintf(" COMMENT '%s'", col.comment)
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
		ucDef := fmt.Sprintf("  CONSTRAINT %s UNIQUE (%s)", constraintName, escapeColumnList(uc.columns, d))
		columnDefs = append(columnDefs, ucDef)
	}

	parts = append(parts, strings.Join(columnDefs, ",\n"))
	parts = append(parts, ") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;")

	return strings.Join(parts, "\n")
}

func (d *MySQLDialect) BuildModifyTable(tb *TableBuilder) []string {
	var sqls []string
	escapedTable := escapeTableName(tb.tableName, d)
	for _, col := range tb.columns {
		var query string
		if col.isChange {
			query = fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s %s",
				escapedTable, escapeColumnName(col.name, d), d.GetDataType(col))
		} else {
			query = fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
				escapedTable, escapeColumnName(col.name, d), d.GetDataType(col))
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
			query += fmt.Sprintf(" AFTER %s", escapeColumnName(*col.afterColumn, d))
		}
		if col.firstColumn {
			query += " FIRST"
		}
		if col.comment != "" {
			query += fmt.Sprintf(" COMMENT '%s'", col.comment)
		}
		sqls = append(sqls, query)

		// Add foreign key constraint for inline references
		if col.refTable != "" && col.refColumn != "" {
			if col.isChange {
				// Drop existing FK before re-adding
				sqls = append(sqls, fmt.Sprintf("ALTER TABLE %s DROP FOREIGN KEY IF EXISTS fk_%s_%s",
					escapedTable, tb.tableName, col.name))
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
	return fmt.Sprintf("DROP TABLE IF EXISTS %s", escapeTableName(tableName, d))
}

func (d *MySQLDialect) BuildDropColumn(tableName, columnName string) string {
	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", escapeTableName(tableName, d), escapeColumnName(columnName, d))
}

func (d *MySQLDialect) BuildIndexStatements(tb *TableBuilder) []string {
	var sqls []string
	for _, col := range tb.columns {
		if col.hasIndex {
			indexName := col.indexName
			if indexName == "" {
				indexName = fmt.Sprintf("idx_%s_%s", tb.tableName, col.name)
			}
			sql := fmt.Sprintf("CREATE INDEX %s ON %s (%s)",
				indexName, escapeTableName(tb.tableName, d), escapeColumnName(col.name, d))
			sqls = append(sqls, sql)
		}
	}
	for _, ci := range tb.compositeIndexes {
		indexName := ci.name
		if indexName == "" {
			indexName = fmt.Sprintf("idx_%s_%s", tb.tableName, strings.Join(ci.columns, "_"))
		}
		sql := fmt.Sprintf("CREATE INDEX %s ON %s (%s)",
			indexName, escapeTableName(tb.tableName, d), buildCompositeIndexColumns(ci, d))
		sqls = append(sqls, sql)
	}
	return sqls
}

func (d *MySQLDialect) Placeholder(_ int) string {
	return "?"
}

func (d *SQLiteDialect) GetDataType(col *Column) string {
	switch col.dataType {
	case "uuid", "ulid", "string", "char", "text", "mediumtext", "longtext":
		return "TEXT"
	case "tinyint", "smallint", "mediumint", "integer", "bigint":
		return "INTEGER"
	case "float", "double":
		return "REAL"
	case "boolean":
		return "INTEGER"
	case "timestamp", "date", "time", "year":
		return "TEXT"
	case "json":
		return "TEXT"
	case "binary":
		return "BLOB"
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
	parts = append(parts, fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (", escapeTableName(tb.tableName, d)))

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
		ucDef := fmt.Sprintf("  CONSTRAINT %s UNIQUE (%s)", constraintName, escapeColumnList(uc.columns, d))
		columnDefs = append(columnDefs, ucDef)
	}

	parts = append(parts, strings.Join(columnDefs, ",\n"))
	parts = append(parts, ");")

	return strings.Join(parts, "\n")
}

func (d *SQLiteDialect) BuildModifyTable(tb *TableBuilder) []string {
	var sqls []string
	escapedTable := escapeTableName(tb.tableName, d)
	for _, col := range tb.columns {
		if col.isChange {
			// SQLite does not support modifying columns directly
			// This requires table recreation which is not currently supported
			sqls = append(sqls, fmt.Sprintf("-- ERROR: SQLite does not support MODIFY COLUMN for '%s'. Manual table recreation required.", col.name))
			continue
		}

		query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
			escapedTable, escapeColumnName(col.name, d), d.GetDataType(col))

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
	return fmt.Sprintf("DROP TABLE IF EXISTS %s", escapeTableName(tableName, d))
}

func (d *SQLiteDialect) BuildDropColumn(tableName, columnName string) string {
	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", escapeTableName(tableName, d), escapeColumnName(columnName, d))
}

func (d *SQLiteDialect) BuildIndexStatements(tb *TableBuilder) []string {
	var sqls []string
	for _, col := range tb.columns {
		if col.hasIndex {
			indexName := col.indexName
			if indexName == "" {
				indexName = fmt.Sprintf("idx_%s_%s", tb.tableName, col.name)
			}
			sql := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)",
				indexName, escapeTableName(tb.tableName, d), escapeColumnName(col.name, d))
			sqls = append(sqls, sql)
		}
	}
	for _, ci := range tb.compositeIndexes {
		indexName := ci.name
		if indexName == "" {
			indexName = fmt.Sprintf("idx_%s_%s", tb.tableName, strings.Join(ci.columns, "_"))
		}
		sql := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)",
			indexName, escapeTableName(tb.tableName, d), buildCompositeIndexColumns(ci, d))
		sqls = append(sqls, sql)
	}
	return sqls
}

func (d *SQLiteDialect) Placeholder(_ int) string {
	return "?"
}
