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
}

type PostgresDialect struct{}
type MySQLDialect struct{}
type SQLiteDialect struct{}

var mysqlReservedKeywords = map[string]bool{
	"limit": true, "order": true, "group": true, "key": true, "index": true,
	"type": true, "desc": true, "asc": true, "primary": true, "foreign": true,
	"references": true, "constraint": true, "table": true, "column": true,
	"select": true, "from": true, "where": true, "join": true, "on": true,
	"and": true, "or": true, "not": true, "like": true, "in": true,
	"between": true, "is": true, "null": true, "default": true, "unique": true,
	"check": true, "cascade": true, "restrict": true, "set": true,
}

func escapeColumnName(name string, dialect Dialect) string {
	if _, isMySQLDialect := dialect.(*MySQLDialect); isMySQLDialect {
		if mysqlReservedKeywords[strings.ToLower(name)] {
			return fmt.Sprintf("`%s`", name)
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
		def := fmt.Sprintf("  %s %s", col.name, d.GetDataType(col))

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

	parts = append(parts, strings.Join(columnDefs, ",\n"))
	parts = append(parts, ");")

	return strings.Join(parts, "\n")
}

func (d *PostgresDialect) BuildModifyTable(tb *TableBuilder) []string {
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
		sqls = append(sqls, query)
	}
	return sqls
}

func (d *PostgresDialect) BuildDropTable(tableName string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
}

func (d *PostgresDialect) BuildDropColumn(tableName, columnName string) string {
	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", tableName, columnName)
}

func (d *MySQLDialect) GetDataType(col *Column) string {
	switch col.dataType {
	case "uuid":
		return "CHAR(36)"
	case "string":
		return "VARCHAR(255)"
	case "text":
		return "TEXT"
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

func (d *SQLiteDialect) GetDataType(col *Column) string {
	switch col.dataType {
	case "uuid", "string":
		return "TEXT"
	case "text":
		return "TEXT"
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
		columnDefs = append(columnDefs, def)
	}

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
