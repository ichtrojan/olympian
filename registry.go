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
	if err := dropColumnForeignKeys(tableName, columnName); err != nil {
		return err
	}
	db, dialect := GetDB()
	_, err := db.Exec(dialect.BuildDropColumn(tableName, columnName))
	return err
}

func dropColumnForeignKeys(tableName, columnName string) error {
	db, dialect := GetDB()

	var (
		query string
		args  []interface{}
	)
	switch dialect.(type) {
	case *MySQLDialect:
		query = `SELECT constraint_name FROM information_schema.key_column_usage
                 WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?
                 AND referenced_table_name IS NOT NULL`
		args = []interface{}{tableName, columnName}
	case *PostgresDialect:
		query = `SELECT con.conname
                 FROM pg_constraint con
                 JOIN pg_class rel ON rel.oid = con.conrelid
                 JOIN pg_attribute att ON att.attrelid = con.conrelid AND att.attnum = ANY(con.conkey)
                 WHERE con.contype = 'f' AND rel.relname = $1 AND att.attname = $2`
		args = []interface{}{tableName, columnName}
	default:
		return nil
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return fmt.Errorf("dropColumnForeignKeys: lookup failed: %w", err)
	}
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			_ = rows.Close()
			return err
		}
		names = append(names, n)
	}
	if err := rows.Close(); err != nil {
		return err
	}

	for _, n := range names {
		if err := DropForeignKey(tableName, n); err != nil {
			return fmt.Errorf("dropColumnForeignKeys: drop %q: %w", n, err)
		}
	}
	return nil
}

func RenameColumn(tableName, oldName, newName string) error {
	db, dialect := GetDB()

	query := fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s",
		escapeTableName(tableName, dialect),
		escapeColumnName(oldName, dialect),
		escapeColumnName(newName, dialect))

	_, err := db.Exec(query)
	return err
}

func RenameTable(oldName, newName string) error {
	db, dialect := GetDB()
	query := fmt.Sprintf("ALTER TABLE %s RENAME TO %s",
		escapeTableName(oldName, dialect),
		escapeTableName(newName, dialect))
	_, err := db.Exec(query)
	return err
}

func CreateIndex(tableName string, columns []string, indexName string) error {
	db, dialect := GetDB()
	query := fmt.Sprintf("CREATE INDEX %s ON %s (%s)",
		indexName, escapeTableName(tableName, dialect), escapeColumnList(columns, dialect))
	_, err := db.Exec(query)
	return err
}

func CreateUniqueIndex(tableName string, columns []string, indexName string) error {
	db, dialect := GetDB()
	query := fmt.Sprintf("CREATE UNIQUE INDEX %s ON %s (%s)",
		indexName, escapeTableName(tableName, dialect), escapeColumnList(columns, dialect))
	_, err := db.Exec(query)
	return err
}

func DropIndex(indexName string) error {
	db, dialect := GetDB()

	switch dialect.(type) {
	case *MySQLDialect:
		var tableName string
		err := db.QueryRow(
			"SELECT table_name FROM information_schema.statistics WHERE table_schema = DATABASE() AND index_name = ? LIMIT 1",
			indexName,
		).Scan(&tableName)
		if err != nil {
			return fmt.Errorf("DropIndex: could not locate index %q: %w", indexName, err)
		}
		_, err = db.Exec(fmt.Sprintf("DROP INDEX %s ON %s",
			indexName, escapeTableName(tableName, dialect)))
		return err
	default:
		_, err := db.Exec(fmt.Sprintf("DROP INDEX IF EXISTS %s", indexName))
		return err
	}
}

// HasTable checks if a table exists in the database.
func HasTable(tableName string) (bool, error) {
	db, dialect := GetDB()
	var query string
	switch dialect.(type) {
	case *PostgresDialect:
		query = "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1)"
	case *MySQLDialect:
		query = "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = ?)"
	case *SQLiteDialect:
		query = "SELECT EXISTS (SELECT 1 FROM sqlite_master WHERE type='table' AND name = ?)"
	}
	var exists bool
	err := db.QueryRow(query, tableName).Scan(&exists)
	return exists, err
}

// HasColumn checks if a column exists in a table.
func HasColumn(tableName, columnName string) (bool, error) {
	db, dialect := GetDB()
	var query string
	switch dialect.(type) {
	case *PostgresDialect:
		query = "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2)"
	case *MySQLDialect:
		query = "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = ? AND column_name = ?)"
	case *SQLiteDialect:
		// SQLite doesn't support parameterized PRAGMA, use information from table_info
		rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
		if err != nil {
			return false, err
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var cid int
			var name, dataType string
			var notNull, pk int
			var dfltValue interface{}
			if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
				return false, err
			}
			if name == columnName {
				return true, nil
			}
		}
		return false, rows.Err()
	}
	var exists bool
	err := db.QueryRow(query, tableName, columnName).Scan(&exists)
	return exists, err
}

func DropForeignKey(tableName, constraintName string) error {
	db, dialect := GetDB()

	var query string
	switch dialect.(type) {
	case *MySQLDialect:
		query = fmt.Sprintf("ALTER TABLE %s DROP FOREIGN KEY %s",
			escapeTableName(tableName, dialect), constraintName)
	default:
		query = fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s",
			escapeTableName(tableName, dialect), constraintName)
	}

	_, err := db.Exec(query)
	return err
}
