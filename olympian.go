package olympian

import (
	"database/sql"
	"fmt"
	"sync"
)

var (
	globalDB      *sql.DB
	globalDialect Dialect
	mu            sync.RWMutex
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
	tableName         string
	columns           []*Column
	operation         string
	dialect           Dialect
	db                *sql.DB
	foreignKeys       []*ForeignKey
	uniqueConstraints []*UniqueConstraint
	compositeIndexes  []*CompositeIndex
}

type Column struct {
	name          string
	dataType      string
	nullable      bool
	primary       bool
	unique        bool
	defaultValue  *string
	afterColumn   *string
	firstColumn   bool
	autoIncrement bool
	unsigned      bool
	length        int
	comment       string
	enumValues    []string
	// Foreign key fields
	refTable  string
	refColumn string
	onDelete  string
	onUpdate  string
	// Index fields
	hasIndex  bool
	indexName string
	// Change existing column
	isChange bool
}

// CompositeIndex defines a multi-column index
type CompositeIndex struct {
	columns []string
	name    string
	desc    map[string]bool
}

type ForeignKey struct {
	columns    []string
	refTable   string
	refColumns []string
	onDelete   string
	onUpdate   string
}

type UniqueConstraint struct {
	columns []string
	name    string
}

func Table(name string) *TableBuilder {
	db, dialect := GetDB()
	return &TableBuilder{
		tableName:         name,
		columns:           make([]*Column, 0),
		dialect:           dialect,
		db:                db,
		foreignKeys:       make([]*ForeignKey, 0),
		uniqueConstraints: make([]*UniqueConstraint, 0),
		compositeIndexes:  make([]*CompositeIndex, 0),
	}
}

func (tb *TableBuilder) Create(fn func()) error {
	tb.operation = "create"
	currentBuilder = tb
	fn()

	query := tb.dialect.BuildCreateTable(tb)
	_, err := tb.db.Exec(query)
	if err != nil {
		return err
	}

	return executeIndexStatements(tb)
}

func (tb *TableBuilder) Modify(fn func()) error {
	tb.operation = "modify"
	currentBuilder = tb
	fn()

	sqls := tb.dialect.BuildModifyTable(tb)
	for _, query := range sqls {
		if _, err := tb.db.Exec(query); err != nil {
			return err
		}
	}

	return executeIndexStatements(tb)
}

// indexExecutor is an optional interface a Dialect may implement to take full
// control of index statement execution (e.g. MySQL idempotency pre-checks).
type indexExecutor interface {
	ExecuteIndexStatements(db *sql.DB, tb *TableBuilder) error
}

func executeIndexStatements(tb *TableBuilder) error {
	if ex, ok := tb.dialect.(indexExecutor); ok {
		return ex.ExecuteIndexStatements(tb.db, tb)
	}
	for _, stmt := range tb.dialect.BuildIndexStatements(tb) {
		if _, err := tb.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

// PreviewCreate returns the SQL statements that would be executed by Create(),
// without actually executing them. Useful for dry-run and debugging.
func (tb *TableBuilder) PreviewCreate(fn func()) []string {
	tb.operation = "create"
	currentBuilder = tb
	fn()

	var sqls []string
	sqls = append(sqls, tb.dialect.BuildCreateTable(tb))
	sqls = append(sqls, tb.dialect.BuildIndexStatements(tb)...)
	return sqls
}

// PreviewModify returns the SQL statements that would be executed by Modify(),
// without actually executing them. Useful for dry-run and debugging.
func (tb *TableBuilder) PreviewModify(fn func()) []string {
	tb.operation = "modify"
	currentBuilder = tb
	fn()

	var sqls []string
	sqls = append(sqls, tb.dialect.BuildModifyTable(tb)...)
	sqls = append(sqls, tb.dialect.BuildIndexStatements(tb)...)
	return sqls
}

func (tb *TableBuilder) Drop() error {
	// For MySQL, disable foreign key checks temporarily
	if _, isMySQL := tb.dialect.(*MySQLDialect); isMySQL {
		if _, err := tb.db.Exec("SET FOREIGN_KEY_CHECKS = 0"); err != nil {
			return fmt.Errorf("failed to disable foreign key checks: %w", err)
		}
		defer func() {
			_, _ = tb.db.Exec("SET FOREIGN_KEY_CHECKS = 1")
		}()
	}

	query := tb.dialect.BuildDropTable(tb.tableName)
	_, err := tb.db.Exec(query)
	return err
}

func (tb *TableBuilder) DropColumn(columnName string) error {
	query := tb.dialect.BuildDropColumn(tb.tableName, columnName)
	_, err := tb.db.Exec(query)
	return err
}

// DropColumns drops multiple columns from the table.
func (tb *TableBuilder) DropColumns(columnNames ...string) error {
	for _, name := range columnNames {
		if err := tb.DropColumn(name); err != nil {
			return err
		}
	}
	return nil
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

func Ulid(name string) *ColumnBuilder {
	col := &Column{
		name:     name,
		dataType: "ulid",
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

func TinyInteger(name string) *ColumnBuilder {
	col := &Column{
		name:     name,
		dataType: "tinyint",
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

func Enum(name string, values ...string) *ColumnBuilder {
	col := &Column{
		name:       name,
		dataType:   "enum",
		nullable:   false,
		enumValues: values,
	}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

func SmallInteger(name string) *ColumnBuilder {
	col := &Column{name: name, dataType: "smallint", nullable: false}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

func MediumInteger(name string) *ColumnBuilder {
	col := &Column{name: name, dataType: "mediumint", nullable: false}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

func UnsignedInteger(name string) *ColumnBuilder {
	col := &Column{name: name, dataType: "integer", nullable: false, unsigned: true}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

func UnsignedBigInteger(name string) *ColumnBuilder {
	col := &Column{name: name, dataType: "bigint", nullable: false, unsigned: true}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

func UnsignedSmallInteger(name string) *ColumnBuilder {
	col := &Column{name: name, dataType: "smallint", nullable: false, unsigned: true}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

func UnsignedTinyInteger(name string) *ColumnBuilder {
	col := &Column{name: name, dataType: "tinyint", nullable: false, unsigned: true}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

func Float(name string) *ColumnBuilder {
	col := &Column{name: name, dataType: "float", nullable: false}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

func Double(name string) *ColumnBuilder {
	col := &Column{name: name, dataType: "double", nullable: false}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

func Char(name string, length int) *ColumnBuilder {
	col := &Column{name: name, dataType: "char", nullable: false, length: length}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

func MediumText(name string) *ColumnBuilder {
	col := &Column{name: name, dataType: "mediumtext", nullable: false}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

func LongText(name string) *ColumnBuilder {
	col := &Column{name: name, dataType: "longtext", nullable: false}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

func Binary(name string) *ColumnBuilder {
	col := &Column{name: name, dataType: "binary", nullable: false}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

func Time(name string) *ColumnBuilder {
	col := &Column{name: name, dataType: "time", nullable: false}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

func Year(name string) *ColumnBuilder {
	col := &Column{name: name, dataType: "year", nullable: false}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

// ForeignId creates a UUID column with an inline foreign key reference.
// Equivalent to: Uuid("user_id").References("id").On("users")
// Usage: ForeignId("user_id").Constrained() or ForeignId("user_id").References("id").On("users")
func ForeignId(name string) *ColumnBuilder {
	col := &Column{name: name, dataType: "uuid", nullable: false}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

// Increments creates an auto-incrementing integer primary key column
func Increments(name string) *ColumnBuilder {
	col := &Column{
		name:          name,
		dataType:      "integer",
		nullable:      false,
		primary:       true,
		autoIncrement: true,
	}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

// BigIncrements creates an auto-incrementing bigint primary key column
func BigIncrements(name string) *ColumnBuilder {
	col := &Column{
		name:          name,
		dataType:      "bigint",
		nullable:      false,
		primary:       true,
		autoIncrement: true,
	}
	if currentBuilder != nil {
		currentBuilder.columns = append(currentBuilder.columns, col)
	}
	return &ColumnBuilder{column: col}
}

// Unique creates a composite unique constraint on multiple columns
func Unique(columns ...string) *UniqueConstraintBuilder {
	uc := &UniqueConstraint{
		columns: columns,
	}
	if currentBuilder != nil {
		currentBuilder.uniqueConstraints = append(currentBuilder.uniqueConstraints, uc)
	}
	return &UniqueConstraintBuilder{uc: uc}
}

type UniqueConstraintBuilder struct {
	uc *UniqueConstraint
}

// Name sets a custom name for the unique constraint
func (ucb *UniqueConstraintBuilder) Name(name string) *UniqueConstraintBuilder {
	ucb.uc.name = name
	return ucb
}

func Timestamps() {
	Timestamp("created_at").Nullable()
	Timestamp("updated_at").Nullable()
}

func TimestampsTz() {
	Timestamp("created_at").Nullable()
	Timestamp("updated_at").Nullable()
}

func SoftDeletes() {
	Timestamp("deleted_at").Nullable()
}

func SoftDeletesTz() {
	Timestamp("deleted_at").Nullable()
}

// Morphs adds polymorphic columns: {name}_id (UUID) and {name}_type (VARCHAR).
func Morphs(name string) {
	Uuid(name + "_id")
	String(name + "_type").Index()
}

// NullableMorphs adds nullable polymorphic columns.
func NullableMorphs(name string) {
	Uuid(name + "_id").Nullable()
	String(name + "_type").Nullable().Index()
}

// RememberToken adds a remember_token column for authentication.
func RememberToken() {
	String("remember_token").Length(100).Nullable()
}

// CompIndex creates a composite index on multiple columns.
// Usage: CompIndex("col1", "col2") or CompIndex("col1", "col2").Name("custom_name")
func CompIndex(columns ...string) *CompositeIndexBuilder {
	ci := &CompositeIndex{columns: columns}
	if currentBuilder != nil {
		currentBuilder.compositeIndexes = append(currentBuilder.compositeIndexes, ci)
	}
	return &CompositeIndexBuilder{ci: ci}
}

type CompositeIndexBuilder struct {
	ci *CompositeIndex
}

func (cib *CompositeIndexBuilder) Name(name string) *CompositeIndexBuilder {
	cib.ci.name = name
	return cib
}

// Desc marks the given columns as descending in the composite index.
// Columns not listed remain ascending.
func (cib *CompositeIndexBuilder) Desc(columns ...string) *CompositeIndexBuilder {
	if cib.ci.desc == nil {
		cib.ci.desc = make(map[string]bool, len(columns))
	}
	for _, c := range columns {
		cib.ci.desc[c] = true
	}
	return cib
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

// Length sets a custom length for string/char columns.
// String("name").Length(100) produces VARCHAR(100) instead of VARCHAR(255).
func (cb *ColumnBuilder) Length(length int) *ColumnBuilder {
	cb.column.length = length
	return cb
}

// Unsigned marks an integer column as unsigned (MySQL).
func (cb *ColumnBuilder) Unsigned() *ColumnBuilder {
	cb.column.unsigned = true
	return cb
}

// First places the column at the beginning of the table (MySQL only).
func (cb *ColumnBuilder) First() *ColumnBuilder {
	cb.column.firstColumn = true
	return cb
}

// Comment adds a comment to the column.
func (cb *ColumnBuilder) Comment(comment string) *ColumnBuilder {
	cb.column.comment = comment
	return cb
}

// Constrained is a shorthand for defining a foreign key reference using naming conventions.
// ForeignId("user_id").Constrained() is equivalent to:
// Uuid("user_id").References("id").On("users")
// It derives the table name by removing the _id suffix and pluralizing.
func (cb *ColumnBuilder) Constrained() *ColumnBuilder {
	name := cb.column.name
	if len(name) > 3 && name[len(name)-3:] == "_id" {
		tableName := name[:len(name)-3] + "s"
		cb.column.refColumn = "id"
		cb.column.refTable = tableName
	}
	return cb
}

// CascadeOnDelete sets ON DELETE CASCADE for inline foreign key.
func (cb *ColumnBuilder) CascadeOnDelete() *ColumnBuilder {
	cb.column.onDelete = "cascade"
	return cb
}

// CascadeOnUpdate sets ON UPDATE CASCADE for inline foreign key.
func (cb *ColumnBuilder) CascadeOnUpdate() *ColumnBuilder {
	cb.column.onUpdate = "cascade"
	return cb
}

// RestrictOnDelete sets ON DELETE RESTRICT for inline foreign key.
func (cb *ColumnBuilder) RestrictOnDelete() *ColumnBuilder {
	cb.column.onDelete = "restrict"
	return cb
}

// RestrictOnUpdate sets ON UPDATE RESTRICT for inline foreign key.
func (cb *ColumnBuilder) RestrictOnUpdate() *ColumnBuilder {
	cb.column.onUpdate = "restrict"
	return cb
}

// NullOnDelete sets ON DELETE SET NULL for inline foreign key.
func (cb *ColumnBuilder) NullOnDelete() *ColumnBuilder {
	cb.column.onDelete = "set null"
	return cb
}

// Foreign key methods for inline definitions
func (cb *ColumnBuilder) References(column string) *ColumnBuilder {
	cb.column.refColumn = column
	return cb
}

func (cb *ColumnBuilder) On(table string) *ColumnBuilder {
	cb.column.refTable = table
	return cb
}

func (cb *ColumnBuilder) OnDelete(action string) *ColumnBuilder {
	cb.column.onDelete = action
	return cb
}

func (cb *ColumnBuilder) OnUpdate(action string) *ColumnBuilder {
	cb.column.onUpdate = action
	return cb
}

// Index methods for creating indexes on columns
func (cb *ColumnBuilder) Index() *ColumnBuilder {
	cb.column.hasIndex = true
	// Index name will be auto-generated as idx_tablename_columnname
	return cb
}

func (cb *ColumnBuilder) IndexWithName(name string) *ColumnBuilder {
	cb.column.hasIndex = true
	cb.column.indexName = name
	return cb
}

// Change marks this column definition as a modification to an existing column
// Use this in a Modify() closure to alter an existing column's type or properties
func (cb *ColumnBuilder) Change() *ColumnBuilder {
	cb.column.isChange = true
	return cb
}

type ForeignKeyBuilder struct {
	fk *ForeignKey
}

// Foreign creates a foreign key constraint on one or more columns.
// For single-column FKs: Foreign("business_id").References("id").On("businesses")
// For composite FKs: Foreign("user_id", "tenant_id").References("id", "tenant_id").On("users")
func Foreign(columnNames ...string) *ForeignKeyBuilder {
	fk := &ForeignKey{
		columns: columnNames,
	}
	if currentBuilder != nil {
		currentBuilder.foreignKeys = append(currentBuilder.foreignKeys, fk)
	}
	return &ForeignKeyBuilder{fk: fk}
}

// References specifies the referenced column(s) in the foreign table.
// For composite FKs, pass multiple column names matching the order of Foreign() columns.
func (fkb *ForeignKeyBuilder) References(columns ...string) *ForeignKeyBuilder {
	fkb.fk.refColumns = columns
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
