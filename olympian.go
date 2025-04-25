package olympian

import (
	"database/sql"
	"fmt"
	"sync"
)

var (
	globalDB     *sql.DB
	globalDialect Dialect
	mu           sync.RWMutex
)

func SetDB(db *sql.DB, dialect Dialect) {
	mu.Lock()
	defer mu.Unlock()
	globalDB = db
	globalDialect = dialect
}

func GetDB() (*sql.DB, Dialect) {
	mu.RLock()
	defer mu.RUnlock()
	return globalDB, globalDialect
}

type Migration struct {
	Name string
	Up   func() error
	Down func() error
}

type TableBuilder struct {
	tableName  string
	columns    []*Column
	operation  string
	dialect    Dialect
	db         *sql.DB
	foreignKeys []*ForeignKey
}

type Column struct {
	name         string
	dataType     string
	nullable     bool
	primary      bool
	unique       bool
	defaultValue *string
	afterColumn  *string
	autoIncrement bool
}

type ForeignKey struct {
	column       string
	refTable     string
	refColumn    string
	onDelete     string
	onUpdate     string
}

func Table(name string) *TableBuilder {
	db, dialect := GetDB()
	return &TableBuilder{
		tableName: name,
		columns:   make([]*Column, 0),
		dialect:   dialect,
		db:        db,
		foreignKeys: make([]*ForeignKey, 0),
	}
}

func (tb *TableBuilder) Create(fn func()) error {
	tb.operation = "create"
	currentBuilder = tb
	fn()
	currentBuilder = nil

	query := tb.dialect.BuildCreateTable(tb)
	_, err := tb.db.Exec(query)
	return err
}

func (tb *TableBuilder) Modify(fn func()) error {
	tb.operation = "modify"
	currentBuilder = tb
	fn()
	currentBuilder = nil

	sqls := tb.dialect.BuildModifyTable(tb)
	for _, query := range sqls {
		if _, err := tb.db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func (tb *TableBuilder) Drop() error {
	query := tb.dialect.BuildDropTable(tb.tableName)
	_, err := tb.db.Exec(query)
	return err
}

func (tb *TableBuilder) DropColumn(columnName string) error {
	query := tb.dialect.BuildDropColumn(tb.tableName, columnName)
	_, err := tb.db.Exec(query)
	return err
}

var currentBuilder *TableBuilder

type ColumnBuilder struct {
	column *Column
}

func Uuid(name string) *ColumnBuilder {
	col := &Column{
		name:     name,
		dataType: "uuid",
		nullable: false,
	}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

func String(name string) *ColumnBuilder {
	col := &Column{
		name:     name,
		dataType: "string",
		nullable: false,
	}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

func Text(name string) *ColumnBuilder {
	col := &Column{
		name:     name,
		dataType: "text",
		nullable: false,
	}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

func Integer(name string) *ColumnBuilder {
	col := &Column{
		name:     name,
		dataType: "integer",
		nullable: false,
	}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

func BigInteger(name string) *ColumnBuilder {
	col := &Column{
		name:     name,
		dataType: "bigint",
		nullable: false,
	}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

func Boolean(name string) *ColumnBuilder {
	col := &Column{
		name:     name,
		dataType: "boolean",
		nullable: false,
	}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

func Decimal(name string, precision, scale int) *ColumnBuilder {
	col := &Column{
		name:     name,
		dataType: fmt.Sprintf("decimal(%d,%d)", precision, scale),
		nullable: false,
	}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

func Timestamp(name string) *ColumnBuilder {
	col := &Column{
		name:     name,
		dataType: "timestamp",
		nullable: false,
	}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

func Date(name string) *ColumnBuilder {
	col := &Column{
		name:     name,
		dataType: "date",
		nullable: false,
	}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

func Json(name string) *ColumnBuilder {
	col := &Column{
		name:     name,
		dataType: "json",
		nullable: false,
	}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

func Timestamps() {
	Timestamp("created_at").Nullable()
	Timestamp("updated_at").Nullable()
}

func SoftDeletes() {
	Timestamp("deleted_at").Nullable()
}

func (cb *ColumnBuilder) Nullable() *ColumnBuilder {
	cb.column.nullable = true
	return cb
}

func (cb *ColumnBuilder) Primary() *ColumnBuilder {
	cb.column.primary = true
	return cb
}

func (cb *ColumnBuilder) Unique() *ColumnBuilder {
	cb.column.unique = true
	return cb
}

func (cb *ColumnBuilder) Default(value interface{}) *ColumnBuilder {
	val := fmt.Sprintf("%v", value)
	cb.column.defaultValue = &val
	return cb
}

func (cb *ColumnBuilder) After(columnName string) *ColumnBuilder {
	cb.column.afterColumn = &columnName
	return cb
}

func (cb *ColumnBuilder) AutoIncrement() *ColumnBuilder {
	cb.column.autoIncrement = true
	return cb
}

type ForeignKeyBuilder struct {
	fk *ForeignKey
}

func Foreign(columnName string) *ForeignKeyBuilder {
	fk := &ForeignKey{
		column: columnName,
	}
	if currentBuilder != nil {
		currentBuilder.foreignKeys = append(currentBuilder.foreignKeys, fk)
	}
	return &ForeignKeyBuilder{fk: fk}
}

func (fkb *ForeignKeyBuilder) References(column string) *ForeignKeyBuilder {
	fkb.fk.refColumn = column
	return fkb
}

func (fkb *ForeignKeyBuilder) On(table string) *ForeignKeyBuilder {
	fkb.fk.refTable = table
	return fkb
}

func (fkb *ForeignKeyBuilder) OnDelete(action string) *ForeignKeyBuilder {
	fkb.fk.onDelete = action
	return fkb
}

func (fkb *ForeignKeyBuilder) OnUpdate(action string) *ForeignKeyBuilder {
	fkb.fk.onUpdate = action
	return fkb
}
