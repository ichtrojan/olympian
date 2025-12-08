package olympian

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode"
)

type Seeder struct {
	Name string
	Run  func() error
}

var seederRegistry []Seeder

func RegisterSeeder(s Seeder) {
	seederRegistry = append(seederRegistry, s)
}

func GetSeeders() []Seeder {
	return seederRegistry
}

type SeederRunner struct {
	db      *sql.DB
	dialect Dialect
}

func NewSeederRunner(db *sql.DB, dialect Dialect) *SeederRunner {
	return &SeederRunner{
		db:      db,
		dialect: dialect,
	}
}

func (sr *SeederRunner) Run(seeders []Seeder) error {
	if len(seeders) == 0 {
		fmt.Println("No seeders to run")
		return nil
	}

	for _, seeder := range seeders {
		fmt.Printf("Seeding: %s\n", seeder.Name)
		if err := seeder.Run(); err != nil {
			return fmt.Errorf("failed to run seeder %s: %w", seeder.Name, err)
		}
		fmt.Printf("Seeded:  %s\n", seeder.Name)
	}

	fmt.Println("\nDatabase seeding completed successfully")
	return nil
}

func (sr *SeederRunner) RunSpecific(seeders []Seeder, names ...string) error {
	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}

	var toRun []Seeder
	for _, seeder := range seeders {
		if nameSet[seeder.Name] {
			toRun = append(toRun, seeder)
		}
	}

	if len(toRun) == 0 {
		return fmt.Errorf("no seeders found matching the specified names")
	}

	return sr.Run(toRun)
}

func Seed(tableName string, data ...interface{}) error {
	db, dialect := GetDB()
	if db == nil {
		return fmt.Errorf("database connection not set")
	}

	for _, record := range data {
		columns, values, placeholders := extractColumnsAndValues(record, dialect)
		if len(columns) == 0 {
			continue
		}

		query := buildInsertQuery(tableName, columns, placeholders, dialect)
		if _, err := db.Exec(query, values...); err != nil {
			return fmt.Errorf("failed to seed %s: %w", tableName, err)
		}
	}

	return nil
}

func extractColumnsAndValues(record interface{}, dialect Dialect) ([]string, []interface{}, []string) {
	var columns []string
	var values []interface{}
	var placeholders []string

	v := reflect.ValueOf(record)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			fieldValue := v.Field(i)

			if !field.IsExported() {
				continue
			}

			columnName := field.Tag.Get("db")
			if columnName == "" {
				columnName = field.Tag.Get("json")
				if columnName == "" {
					columnName = toSnakeCase(field.Name)
				}
			}

			if columnName == "-" {
				continue
			}

			if idx := strings.Index(columnName, ","); idx != -1 {
				columnName = columnName[:idx]
			}

			value := convertValue(fieldValue)
			columns = append(columns, escapeSeederColumnName(columnName, dialect))
			values = append(values, value)
			placeholders = append(placeholders, getPlaceholder(len(placeholders)+1, dialect))
		}

	case reflect.Map:
		for _, key := range v.MapKeys() {
			keyStr := fmt.Sprintf("%v", key.Interface())
			columnName := toSnakeCase(keyStr)
			value := convertValue(v.MapIndex(key))

			columns = append(columns, escapeSeederColumnName(columnName, dialect))
			values = append(values, value)
			placeholders = append(placeholders, getPlaceholder(len(placeholders)+1, dialect))
		}
	}

	return columns, values, placeholders
}

func convertValue(v reflect.Value) interface{} {
	if !v.IsValid() {
		return nil
	}

	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	if v.Type() == reflect.TypeOf(time.Time{}) {
		t := v.Interface().(time.Time)
		if t.IsZero() {
			return nil
		}
		return t.Format("2006-01-02 15:04:05")
	}

	if v.Type() == reflect.TypeOf(&time.Time{}) {
		if v.IsNil() {
			return nil
		}
		t := v.Interface().(*time.Time)
		if t.IsZero() {
			return nil
		}
		return t.Format("2006-01-02 15:04:05")
	}

	return v.Interface()
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func escapeSeederColumnName(name string, dialect Dialect) string {
	switch dialect.(type) {
	case *MySQLDialect:
		return fmt.Sprintf("`%s`", name)
	case *PostgresDialect:
		return fmt.Sprintf(`"%s"`, name)
	default:
		return name
	}
}

func getPlaceholder(index int, dialect Dialect) string {
	switch dialect.(type) {
	case *PostgresDialect:
		return fmt.Sprintf("$%d", index)
	default:
		return "?"
	}
}

func buildInsertQuery(tableName string, columns []string, placeholders []string, dialect Dialect) string {
	var tableNameEscaped string
	switch dialect.(type) {
	case *MySQLDialect:
		tableNameEscaped = fmt.Sprintf("`%s`", tableName)
	case *PostgresDialect:
		tableNameEscaped = fmt.Sprintf(`"%s"`, tableName)
	default:
		tableNameEscaped = tableName
	}

	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		tableNameEscaped,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)
}
