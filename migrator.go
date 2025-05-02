package olympian

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
)

type Migrator struct {
	db      *sql.DB
	dialect Dialect
}

func NewMigrator(db *sql.DB, dialect Dialect) *Migrator {
	return &Migrator{
		db:      db,
		dialect: dialect,
	}
}

func (m *Migrator) Init() error {
	SetDB(m.db, m.dialect)

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS olympian_migrations (
		id INTEGER PRIMARY KEY,
		migration VARCHAR(255) NOT NULL,
		batch INTEGER NOT NULL,
		executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	if _, ok := m.dialect.(*PostgresDialect); ok {
		createTableSQL = `
		CREATE TABLE IF NOT EXISTS olympian_migrations (
			id SERIAL PRIMARY KEY,
			migration VARCHAR(255) NOT NULL,
			batch INTEGER NOT NULL,
			executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`
	} else if _, ok := m.dialect.(*MySQLDialect); ok {
		createTableSQL = `
		CREATE TABLE IF NOT EXISTS olympian_migrations (
			id INT AUTO_INCREMENT PRIMARY KEY,
			migration VARCHAR(255) NOT NULL,
			batch INT NOT NULL,
			executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`
	}

	_, err := m.db.Exec(createTableSQL)
	return err
}

func (m *Migrator) GetLastBatch() (int, error) {
	var batch sql.NullInt64
	err := m.db.QueryRow("SELECT MAX(batch) FROM olympian_migrations").Scan(&batch)
	if err != nil {
		return 0, err
	}
	if !batch.Valid {
		return 0, nil
	}
	return int(batch.Int64), nil
}

func (m *Migrator) GetExecutedMigrations() (map[string]bool, error) {
	rows, err := m.db.Query("SELECT migration FROM olympian_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	executed := make(map[string]bool)
	for rows.Next() {
		var migration string
		if err := rows.Scan(&migration); err != nil {
			return nil, err
		}
		executed[migration] = true
	}
	return executed, rows.Err()
}

func (m *Migrator) RecordMigration(name string, batch int) error {
	_, err := m.db.Exec(
		"INSERT INTO olympian_migrations (migration, batch, executed_at) VALUES (?, ?, ?)",
		name, batch, time.Now(),
	)
	return err
}

func (m *Migrator) RemoveMigration(name string) error {
	_, err := m.db.Exec("DELETE FROM olympian_migrations WHERE migration = ?", name)
	return err
}

func (m *Migrator) GetMigrationsFromBatch(batch int) ([]string, error) {
	rows, err := m.db.Query(
		"SELECT migration FROM olympian_migrations WHERE batch = ? ORDER BY id DESC",
		batch,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migrations []string
	for rows.Next() {
		var migration string
		if err := rows.Scan(&migration); err != nil {
			return nil, err
		}
		migrations = append(migrations, migration)
	}
	return migrations, rows.Err()
}

func (m *Migrator) Migrate(migrations []Migration) error {
	SetDB(m.db, m.dialect)

	executed, err := m.GetExecutedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get executed migrations: %w", err)
	}

	batch, err := m.GetLastBatch()
	if err != nil {
		return fmt.Errorf("failed to get last batch: %w", err)
	}
	batch++

	var pending []Migration
	for _, migration := range migrations {
		if !executed[migration.Name] {
			pending = append(pending, migration)
		}
	}

	if len(pending) == 0 {
		fmt.Println("Nothing to migrate")
		return nil
	}

	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Name < pending[j].Name
	})

	for _, migration := range pending {
		fmt.Printf("Migrating: %s\n", migration.Name)

		if err := migration.Up(); err != nil {
			return fmt.Errorf("migration %s failed: %w", migration.Name, err)
		}

		if err := m.RecordMigration(migration.Name, batch); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", migration.Name, err)
		}

		fmt.Printf("Migrated:  %s\n", migration.Name)
	}

	return nil
}

func (m *Migrator) Rollback(migrations []Migration, steps int) error {
	SetDB(m.db, m.dialect)

	if steps <= 0 {
		steps = 1
	}

	migrationMap := make(map[string]Migration)
	for _, migration := range migrations {
		migrationMap[migration.Name] = migration
	}

	lastBatch, err := m.GetLastBatch()
	if err != nil {
		return fmt.Errorf("failed to get last batch: %w", err)
	}

	if lastBatch == 0 {
		fmt.Println("Nothing to rollback")
		return nil
	}

	for i := 0; i < steps; i++ {
		batch := lastBatch - i
		if batch <= 0 {
			break
		}

		toRollback, err := m.GetMigrationsFromBatch(batch)
		if err != nil {
			return fmt.Errorf("failed to get migrations from batch %d: %w", batch, err)
		}

		if len(toRollback) == 0 {
			continue
		}

		for _, name := range toRollback {
			migration, ok := migrationMap[name]
			if !ok {
				return fmt.Errorf("migration file not found: %s", name)
			}

			fmt.Printf("Rolling back: %s\n", name)

			if err := migration.Down(); err != nil {
				return fmt.Errorf("rollback %s failed: %w", name, err)
			}

			if err := m.RemoveMigration(name); err != nil {
				return fmt.Errorf("failed to remove migration record %s: %w", name, err)
			}

			fmt.Printf("Rolled back: %s\n", name)
		}
	}

	return nil
}

func (m *Migrator) Status(migrations []Migration) error {
	SetDB(m.db, m.dialect)

	executed, err := m.GetExecutedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get executed migrations: %w", err)
	}

	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("| %-8s | %-45s |\n", "Status", "Migration")
	fmt.Println(strings.Repeat("-", 60))

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Name < migrations[j].Name
	})

	for _, migration := range migrations {
		status := "Pending"
		if executed[migration.Name] {
			status = "Ran"
		}
		fmt.Printf("| %-8s | %-45s |\n", status, migration.Name)
	}

	fmt.Println(strings.Repeat("-", 60))
	return nil
}

func (m *Migrator) Reset(migrations []Migration) error {
	SetDB(m.db, m.dialect)

	lastBatch, err := m.GetLastBatch()
	if err != nil {
		return err
	}

	if lastBatch == 0 {
		fmt.Println("Nothing to reset")
		return nil
	}

	return m.Rollback(migrations, lastBatch)
}

func (m *Migrator) Fresh(migrations []Migration) error {
	SetDB(m.db, m.dialect)

	rows, err := m.db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		if _, ok := m.dialect.(*PostgresDialect); ok {
			rows, err = m.db.Query("SELECT tablename FROM pg_tables WHERE schemaname='public'")
		} else if _, ok := m.dialect.(*MySQLDialect); ok {
			rows, err = m.db.Query("SHOW TABLES")
		}
		if err != nil {
			return fmt.Errorf("failed to get tables: %w", err)
		}
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return err
		}
		if table != "olympian_migrations" {
			tables = append(tables, table)
		}
	}

	for _, table := range tables {
		if _, err := m.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table)); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	if _, err := m.db.Exec("DELETE FROM olympian_migrations"); err != nil {
		return fmt.Errorf("failed to clear migrations table: %w", err)
	}

	return m.Migrate(migrations)
}
