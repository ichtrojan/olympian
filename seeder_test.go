package olympian

import (
	"reflect"
	"testing"
	"time"
)

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Name", "name"},
		{"UserAge", "user_age"},
		{"FirstName", "first_name"},
		{"ID", "i_d"},
		{"UserID", "user_i_d"},
		{"HTTPServer", "h_t_t_p_server"},
		{"name", "name"},
		{"CreatedAt", "created_at"},
	}

	for _, test := range tests {
		result := toSnakeCase(test.input)
		if result != test.expected {
			t.Errorf("toSnakeCase(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestConvertValue(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)

	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{"string", "hello", "hello"},
		{"int", 42, 42},
		{"bool", true, true},
		{"time.Time", now, "2024-01-15 10:30:45"},
		{"nil time.Time", time.Time{}, nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := convertValue(reflect.ValueOf(test.input))
			if result != test.expected {
				t.Errorf("convertValue(%v) = %v, expected %v", test.input, result, test.expected)
			}
		})
	}
}

func TestExtractColumnsAndValuesFromStruct(t *testing.T) {
	type User struct {
		ID        string `db:"id"`
		Name      string `json:"name"`
		Email     string
		UserAge   int
		CreatedAt time.Time
	}

	now := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
	user := User{
		ID:        "123",
		Name:      "John",
		Email:     "john@example.com",
		UserAge:   25,
		CreatedAt: now,
	}

	dialect := &SQLiteDialect{}
	columns, values, placeholders := extractColumnsAndValues(user, dialect)

	if len(columns) != 5 {
		t.Errorf("Expected 5 columns, got %d", len(columns))
	}

	if len(values) != 5 {
		t.Errorf("Expected 5 values, got %d", len(values))
	}

	if len(placeholders) != 5 {
		t.Errorf("Expected 5 placeholders, got %d", len(placeholders))
	}
}

func TestExtractColumnsAndValuesFromMap(t *testing.T) {
	data := map[string]interface{}{
		"ID":      "123",
		"Name":    "John",
		"UserAge": 25,
	}

	dialect := &SQLiteDialect{}
	columns, values, placeholders := extractColumnsAndValues(data, dialect)

	if len(columns) != 3 {
		t.Errorf("Expected 3 columns, got %d", len(columns))
	}

	if len(values) != 3 {
		t.Errorf("Expected 3 values, got %d", len(values))
	}

	if len(placeholders) != 3 {
		t.Errorf("Expected 3 placeholders, got %d", len(placeholders))
	}
}

func TestGetPlaceholder(t *testing.T) {
	tests := []struct {
		dialect  Dialect
		index    int
		expected string
	}{
		{&PostgresDialect{}, 1, "$1"},
		{&PostgresDialect{}, 5, "$5"},
		{&MySQLDialect{}, 1, "?"},
		{&MySQLDialect{}, 5, "?"},
		{&SQLiteDialect{}, 1, "?"},
	}

	for _, test := range tests {
		result := getPlaceholder(test.index, test.dialect)
		if result != test.expected {
			t.Errorf("getPlaceholder(%d, %T) = %q, expected %q", test.index, test.dialect, result, test.expected)
		}
	}
}

func TestEscapeSeederColumnName(t *testing.T) {
	tests := []struct {
		dialect  Dialect
		name     string
		expected string
	}{
		{&MySQLDialect{}, "name", "`name`"},
		{&PostgresDialect{}, "name", `"name"`},
		{&SQLiteDialect{}, "name", "name"},
	}

	for _, test := range tests {
		result := escapeSeederColumnName(test.name, test.dialect)
		if result != test.expected {
			t.Errorf("escapeSeederColumnName(%q, %T) = %q, expected %q", test.name, test.dialect, result, test.expected)
		}
	}
}

func TestBuildInsertQuery(t *testing.T) {
	tests := []struct {
		dialect      Dialect
		tableName    string
		columns      []string
		placeholders []string
		expected     string
	}{
		{
			&MySQLDialect{},
			"users",
			[]string{"`name`", "`email`"},
			[]string{"?", "?"},
			"INSERT INTO `users` (`name`, `email`) VALUES (?, ?)",
		},
		{
			&PostgresDialect{},
			"users",
			[]string{`"name"`, `"email"`},
			[]string{"$1", "$2"},
			`INSERT INTO "users" ("name", "email") VALUES ($1, $2)`,
		},
		{
			&SQLiteDialect{},
			"users",
			[]string{"name", "email"},
			[]string{"?", "?"},
			"INSERT INTO users (name, email) VALUES (?, ?)",
		},
	}

	for _, test := range tests {
		result := buildInsertQuery(test.tableName, test.columns, test.placeholders, test.dialect)
		if result != test.expected {
			t.Errorf("buildInsertQuery() = %q, expected %q", result, test.expected)
		}
	}
}

func TestRegisterSeeder(t *testing.T) {
	seederRegistry = nil

	seeder := Seeder{
		Name: "test_seeder",
		Run: func() error {
			return nil
		},
	}

	RegisterSeeder(seeder)

	seeders := GetSeeders()
	if len(seeders) != 1 {
		t.Errorf("Expected 1 seeder, got %d", len(seeders))
	}

	if seeders[0].Name != "test_seeder" {
		t.Errorf("Expected seeder name 'test_seeder', got '%s'", seeders[0].Name)
	}

	seederRegistry = nil
}

func TestSeedRecordWithSlice(t *testing.T) {
	type User struct {
		ID   string
		Name string
	}

	users := []User{
		{ID: "1", Name: "John"},
		{ID: "2", Name: "Jane"},
		{ID: "3", Name: "Bob"},
	}

	v := reflect.ValueOf(users)
	if v.Kind() != reflect.Slice {
		t.Errorf("Expected slice kind, got %v", v.Kind())
	}

	if v.Len() != 3 {
		t.Errorf("Expected slice length 3, got %d", v.Len())
	}

	dialect := &SQLiteDialect{}
	for i := 0; i < v.Len(); i++ {
		columns, values, _ := extractColumnsAndValues(v.Index(i).Interface(), dialect)
		if len(columns) != 2 {
			t.Errorf("Expected 2 columns for record %d, got %d", i, len(columns))
		}
		if len(values) != 2 {
			t.Errorf("Expected 2 values for record %d, got %d", i, len(values))
		}
	}
}
