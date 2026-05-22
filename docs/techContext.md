# Technical Context: sqlh

## Technology Stack

- **Language**: Go 1.25.2+ (requires generics support)
- **Standard Library**: `database/sql`, `reflect`, `context`, `iter` (Go 1.25+)
- **Databases Supported**:
  - SQLite (via `github.com/mattn/go-sqlite3 v1.14.28`)
  - MySQL (via `github.com/go-sql-driver/mysql v1.9.3`)
  - PostgreSQL (partial, via `last_insert_rowid` detection)
  - SQL Server (partial, via `SCOPE_IDENTITY` detection)
- **Testing**: `github.com/stretchr/testify v1.10.0`

## Development Environment

- **Go Version**: 1.25.2 (from `go.mod`)
- **Module Path**: `github.com/kirill-scherba/sqlh`
- **Repository**: `git@github.com:kirill-scherba/sqlh.git`
- **Latest Main Tag**: `v0.5.1`
- **Active Branch**: `feature/metadata_cache`

## Project Dependencies

```gomod
# Direct
github.com/go-sql-driver/mysql v1.9.3
github.com/mattn/go-sqlite3 v1.14.44
github.com/stretchr/testify v1.10.0

# Indirect
filippo.io/edwards25519 v1.1.0
github.com/davecgh/go-spew v1.1.1
github.com/pmezard/go-difflib v1.0.0
gopkg.in/yaml.v3 v3.0.1
```

## API Surface

### sqlh Package (Public Functions)

```go
// Table creation
func Create[T any](db *sql.DB) error

// CRUD operations
func Insert[T any](db *sql.DB, rows ...T) error
func InsertId[T any](db *sql.DB, rows ...T) (int64, error)
func InsertWithCallback[T any](db *sql.DB, callback func(*sql.DB, *sql.Tx) error, rows ...T) error
func Get[T any](db *sql.DB, wheres ...Where) (*T, error)
func List[T any](db querier, previous int, groupBy, orderBy string, listAttrs ...any) ([]T, int, error)
func ListRows[T any](db querier, previous int, groupBy, orderBy string, numRows int, listAttrs ...any) ([]T, int, error)
func ListRange[T any](db querier, offset int, groupBy, orderBy string, limit int, listAttrs ...any) iter.Seq2[int, T]
func QueryRange[T any](db querier, selectQuery string, queryArgs ...any) iter.Seq[T]
func Update[T any](db *sql.DB, attrs ...UpdateAttr[T]) error
func Delete[T any](db *sql.DB, wheres ...Where) error
func Set[T any](db *sql.DB, row T, wheres ...Where) error

// Table wrapper
func CreateTable[T any](db *sql.DB) (*Table[T], error)
type Table[T any] struct { ... }
func (t *Table[T]) Insert(rows ...T) error
func (t *Table[T]) InsertId(rows ...T) (int64, error)
func (t *Table[T]) Update(attrs ...UpdateAttr[T]) error
func (t *Table[T]) Set(row T, wheres ...Where) error
func (t *Table[T]) Get(wheres ...Where) (*T, error)
func (t *Table[T]) Delete(wheres ...Where) error
func (t *Table[T]) Count(wheres ...Where) (int, error)
func (t *Table[T]) List(offset int, groupBy, orderBy string, limit int, listAttrs ...any) iter.Seq2[int, T]
func (t *Table[T]) Close()

// Utilities
func SetNumRows(n int)
func GetNumRows() int
func SetWheresJoinOr() WheresJoinOr
func SetWheresJoinAnd() WheresJoinOr
func SetDistinct() Distinct
func SetAlias(alias string) Alias
func SetName(name string) Name
```

### query Package (Public Functions)

```go
func Table[T any]() (string, error)
func Insert[T any]() (string, error)
func Select[T any](attr *SelectAttr) (string, error)
func Update[T any](attr *UpdateAttr) (string, error)
func Delete[T any](attr *DeleteAttr) (string, error)
func Args(value any, forWrite bool) ([]any, error)
func ArgsApply(dest any, args []any) error   // ⬅ new, preferred
func ArgsAppay(dest any, args []any) error   // Deprecated, use ArgsApply
func MakeJoin[T any](join Join) Join
func SetNumRows(n int)
func GetNumRows() int
```

### Key Types

```go
// Where clause: field name with operator + value
type Where struct {
    Field string  // e.g., "id=", "name LIKE", "age>="
    Value any
}

// Update attribute: row data + WHERE conditions
type UpdateAttr[T any] struct {
    Row    T
    Wheres []Where
}

// Query configuration types (used as variadic attributes)
type WheresJoinOr bool   // OR-join WHERE conditions
type Distinct bool        // SELECT DISTINCT
type Alias string         // Table alias
type Name *string         // Custom table name

// Query package types
type SelectAttr struct {
    Wheres     []string
    WheresJoinOr bool
    Joins      []Join
    GroupBy    string
    OrderBy    string
    Paginator  *Paginator
    Alias      string
    Distinct   bool
    Name       *string
}
type Join struct {
    Join   string   // "left", "inner", etc.
    Name   string   // table name
    Alias  string   // table alias
    On     string   // ON condition
    Fields []string // selected join fields
    Select string   // optional subquery
}
type Paginator struct {
    Offset int
    Limit  int
}
```

## Database Compatibility

### Detected Drivers

| Driver | Driver Name Pattern | last_insert_rowid Query |
| -------- | ------------------- | ------------------------ |
| SQLite | `sqlite` | `SELECT last_insert_rowid()` |
| MySQL | `mysql` | `SELECT LAST_INSERT_ID()` |
| PostgreSQL | `postgres` | `SELECT currval(...)` |
| SQL Server | `sqlserver` | `SELECT SCOPE_IDENTITY()` |

### SQL Type Mapping

| Go Type | SQL Type (default) |
| --------- | ------------------- |
| int, int8, int16, int32, int64 | integer |
| uint8 | tinyint |
| uint, uint16, uint32, uint64 | bigint |
| float32, float64 | double |
| bool | bit |
| string | text |
| []byte | blob |
| time.Time | timestamp |
| complex64, complex128 | blob |

All types can be overridden via `db_type` struct tag.

## Testing

Tests cover:

1. **SQLite in-memory database** (primary test target in `sqlh_test.go`)
2. **MySQL database** (environment-specific in `sqlh_mysql_test.go`)
3. **Query generation** (in `query/sqlh_test.go`)
4. **Table wrapper** (in `table_test.go`)

Run tests:

```bash
go test ./...
# MySQL tests are gated behind SQLH_MYSQL_TEST:
SQLH_MYSQL_TEST=1 go test -run TestMySQL ./...
```

## Current Limitations

1. `context.Context` support is partially implemented (available in read paths
   via variadic attributes, not fully propagated to write operations)
2. No native `UPSERT` (uses `Set` with SELECT-then-INSERT/UPDATE pattern)
3. No aggregate functions (GROUP BY, HAVING, SUM, AVG)
4. No schema migration support (ALTER TABLE)
5. No raw SQL fragment injection for edge cases
6. MySQL test requires a running Docker container and is unconditional
   (planned: gate behind `SQLH_MYSQL_TEST` env var)
