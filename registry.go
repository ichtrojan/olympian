package olympian

import (
	"fmt"
	"time"
)

var registry []Migration

func RegisterMigration(m Migration) {
	registry = append(registry, m)
}

func GetMigrations() []Migration {
	return registry
}

func GetTimestamp() int64 {
	return time.Now().Unix()
}

func Postgres() Dialect {
	return &PostgresDialect{}
}

func MySQL() Dialect {
	return &MySQLDialect{}
}

func SQLite() Dialect {
	return &SQLiteDialect{}
}

func DropColumnIfExists(tableName, columnName string) error {
	db, dialect := GetDB()
	query := dialect.BuildDropColumn(tableName, columnName)
	_, err := db.Exec(query)
	return err
}

func RenameColumn(tableName, oldName, newName string) error {
	db, dialect := GetDB()

	var query string
	switch dialect.(type) {
	case *PostgresDialect:
		query = fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", tableName, oldName, newName)
	case *MySQLDialect:
		query = fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", tableName, oldName, newName)
	case *SQLiteDialect:
		query = fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", tableName, oldName, newName)
	}

	_, err := db.Exec(query)
	return err
}

func RenameTable(oldName, newName string) error {
	db, _ := GetDB()
	query := fmt.Sprintf("ALTER TABLE %s RENAME TO %s", oldName, newName)
	_, err := db.Exec(query)
	return err
}

func CreateIndex(tableName string, columns []string, indexName string) error {
	db, _ := GetDB()
	query := fmt.Sprintf("CREATE INDEX %s ON %s (%s)",
		indexName, tableName, joinColumns(columns))
	_, err := db.Exec(query)
	return err
}

func CreateUniqueIndex(tableName string, columns []string, indexName string) error {
	db, _ := GetDB()
	query := fmt.Sprintf("CREATE UNIQUE INDEX %s ON %s (%s)",
		indexName, tableName, joinColumns(columns))
	_, err := db.Exec(query)
	return err
}

func DropIndex(indexName string) error {
	db, dialect := GetDB()

	var query string
	switch dialect.(type) {
	case *MySQLDialect:
		query = fmt.Sprintf("DROP INDEX %s", indexName)
	default:
		query = fmt.Sprintf("DROP INDEX IF EXISTS %s", indexName)
	}

	_, err := db.Exec(query)
	return err
}

func joinColumns(columns []string) string {
	result := ""
	for i, col := range columns {
		if i > 0 {
			result += ", "
		}
		result += col
	}
	return result
}
