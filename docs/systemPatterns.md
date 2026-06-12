# System Patterns: sqlh

## Architecture Overview

`sqlh` follows a layered architecture with two main packages:

```txt
┌─────────────────────────────────────────────────────┐
│                   migrate package                      │
│  (Schema migrations: FromStruct, Diff, Raw, Apply)    │
├─────────────────────────────────────────────────────┤
│                   sqlh package                       │
│  (High-level CRUD: Insert, Get, List, Update,       │
│   Delete, Set, Create, with auto-transactions)      │
├─────────────────────────────────────────────────────┤
│                   query package                      │
│  (SQL query generation: Table, Insert, Select,      │
│   Update, Delete query builders, metadata cache)    │
├─────────────────────────────────────────────────────┤
│              database/sql (std library)              │
│  (Connection pool, raw query execution)             │
└─────────────────────────────────────────────────────┘
```

## Package Structure

```txt
sqlh/
├── sqlh_exec.go            # Core CRUD functions (Insert, Get, List, Update, Delete, Set)
├── sqlh_table.go           # Table[T] wrapper type with method-based API
├── sqlh_test.go            # Integration tests (SQLite)
├── sqlh_mysql_test.go      # MySQL-specific tests
├── sqlh_retry_test.go      # Lock-detection and retry unit tests
├── sqlh_update_test.go     # Batch Update regression test
├── sqlh_benchmark_test.go  # Performance benchmarks
├── table_test.go           # Table type tests
├── migrate/
│   ├── migrate.go           # Core types: Version, Migration, Plan, Options
│   ├── introspection.go     # Schema introspection (PRAGMA, SHOW COLUMNS, info_schema)
│   ├── fromstruct.go        # FromStruct[T]: CREATE TABLE IF NOT EXISTS
│   ├── diff.go              # Diff[T]: ALTER TABLE ADD COLUMN via struct comparison
│   ├── raw.go               # Raw: explicit SQL migration steps
│   ├── apply.go             # Apply: migration runner, _migrations, DryRun, Backup
│   └── migrate_test.go      # Integration tests (SQLite)
├── query/
│   ├── sqlh_query.go       # SQL query generation (Select, Insert, Update, Delete, Table)
│   ├── sqlh_meta_cache.go  # Cached reflection metadata
│   ├── sqlh_meta_cache_test.go  # Metadata cache unit tests
│   └── sqlh_test.go        # Query generation tests
├── CHANGELOG.md
├── README.md
└── ROADMAP.md
```

## Key Architectural Patterns

### 1. Generics-First Design (Go 1.25+)

All public functions are generic over type parameter `T any`, where `T` is a struct type representing a database table:

```go
func Insert[T any](db *sql.DB, rows ...T) (err error)
func Get[T any](db *sql.DB, wheres ...Where) (row *T, err error)
func List[T any](db querier, previous int, groupBy, orderBy string, listAttrs ...any) (rows []T, pagination int, err error)
func ListRows[T any](db querier, previous int, groupBy, orderBy string, numRows int, listAttrs ...any) (rows []T, pagination int, err error)
func ListRange[T any](db querier, offset int, groupBy, orderBy string, limit int, listAttrs ...any) iter.Seq2[int, T]
```

| Function | Returns | Page Size | Role |
|----------|---------|-----------|------|
| `List` | `([]T, int, error)` | Default (10, configurable) | Quick convenience — delegates to `ListRows` |
| `ListRows` | `([]T, int, error)` | Explicit `numRows` | **Preferred** for explicit pagination |
| `ListRange` | `iter.Seq2[int, T]` | Explicit `limit` | Core lazy iterator — memory-efficient, JOINs, context |
| `QueryRange` | `iter.Seq[T]` | Caller's SQL | Raw SQL escape hatch — bypasses query generation |
| `Table.List` | `iter.Seq2[int, T]` | Explicit `limit` | Wrapper delegation to `ListRange` |

Relationship: `List` → thin wrapper over `ListRows` with default page size. `ListRows` → collects from `ListRange` iterator into a slice. `ListRange` → core lazy iterator. `QueryRange` → separate raw SQL path. `Table.List` → delegates to `ListRange`.

The `Table[T]` wrapper provides a method-based API for convenience:

```go
type Table[T any] struct {
    db *sql.DB
}
func (t *Table[T]) Insert(rows ...T) (err error)
func (t *Table[T]) Get(wheres ...Where) (row *T, err error)
```

### 2. Reflection-Based Query Generation

The `query` package uses `reflect` to inspect struct fields at runtime and generate SQL statements. Reflection metadata is cached by `reflect.Type` so repeated query generation and scan/apply paths reuse table names, field lists, and field flags. Key reflection functions:

- **`getFieldName`**: Extracts column name from `db` struct tag (falls back to lowercase field name)
- **`getFieldType`**: Infers SQL type from Go type via `db_type` tag or automatic mapping (e.g., `int` → `integer`, `string` → `text`)
- **`getFieldKey`**: Processes `db_key` tag for SQL constraints (`primary key`, `autoincrement`, `unique`, `not null`)
- **`fieldsList`**: Collects all field names from a struct, supporting nested structs for JOINs
- **`Args` / `ArgsApply`**: Marshal/unmarshal struct fields to/from `[]any` for `database/sql` scanning
- **`getMeta`**: Returns cached struct metadata used by `Name`, `fields`, `Args`, and `ArgsApply`

### 3. Transactional Write Pattern

All write operations follow a consistent pattern:

```txt
Begin Transaction → Prepare Statement → Execute → Commit (or Rollback on error)
```

Implemented via closures and deferred rollback:

```go
tx, err := db.Begin()
if err != nil { return }
defer func() {
    if err != nil { tx.Rollback(); return }
    err = tx.Commit()
}()
```

### 4. Attribute-Based Query Configuration

Query behavior is configured via typed attributes passed as variadic arguments. This is a form of the **Builder pattern** using Go's type system:

```go
type Where struct { Field string; Value any }
type WheresJoinOr bool
type Distinct bool
type Alias string
type Name *string
type query.Join struct { Join string; Name string; Alias string; On string; Fields []string; Select string }
```

The `listStatement` function parses these attributes by type-switching:

```go
for _, listAttr := range listAttrs {
    switch v := listAttr.(type) {
    case Where: wheres = append(wheres, v)
    case query.Join: attr.Joins = append(attr.Joins, v)
    case Distinct: attr.Distinct = bool(v)
    // ...
    }
}
```

### 5. Error Wrapping and Export

- Standard `sql.ErrNoRows` for "not found" in `Get`
- Custom exported errors for specific conditions:
  - `ErrWhereClauseRequired` — when `Get` called without WHERE
  - `ErrMultipleRowsFound` — when `Get` finds > 1 row
  - `ErrWhereClauseRequiredForUpdate` — for Update without conditions
  - `ErrTypeIsNotStruct` — when T is not a struct
- Database-specific errors (e.g., "database is locked") are detected by string matching

### 6. Retry with Backoff for Database Locks

```go
const numRetries = 20
const retryDelay = 100 * time.Millisecond

func execRetries(f func() (sql.Result, error)) (result sql.Result, err error) {
    for range numRetries {
        result, err = f()
        if err == nil {
            return
        }
        if !isLockError(err) {
            return
        }
        time.Sleep(retryDelay)
    }
    return
}

// isLockError detects transient "database is locked" / "SQLITE_BUSY" errors
// using substring matching. Works with wrapped errors.
func isLockError(err error) bool { ... }
```

Three layers of execution (`execDb`, `execStmt`, `execTx`) all delegate to `execRetries`.

### 7. Iterator Pattern (Go 1.25)

`ListRange` uses Go 1.25's `iter.Seq2` for lazy iteration over query results. Errors are delivered through an optional `func(error)` attribute.

```go
func ListRange[T any](db querier, offset int, groupBy, orderBy string, limit int, listAttrs ...any) iter.Seq2[int, T] {
    return func(yield func(int, T) bool) {
        // Execute query, iterate rows.Scan, yield each row
    }
}
```

### 8. Callback Pattern for Insert Hooks

`InsertWithCallback` allows injecting custom logic after successful insertion but before transaction commit (used by `InsertId` to retrieve last inserted ID):

```go
func InsertWithCallback[T any](db *sql.DB, callback func(*sql.DB, *sql.Tx) error, rows ...T) error
```

### 9. Context Propagation

Functions optionally accept `context.Context` as an attribute. The `getErrfuncAndCtx` helper extracts context and error callback from variadic arguments, defaulting to `context.Background` and no-op error handler.

### 10. Metadata Cache

The `query` package caches reflection metadata in `sync.Map` keyed by `reflect.Type`. This avoids rebuilding table names, field lists, autoincrement flags, and scan metadata on every query. Composite JOIN wrapper compatibility is preserved: if the first field is a struct or pointer-to-struct, that first field defines the base table name and base projection. Ordinary `time.Time` fields are excluded from composite detection.

## Data Flow Example: `Get` Operation

```txt
User calls: sqlh.Get[User](db, Where{Field: "id=", Value: 1})

1. query.Select[User](attr)  →  generates "SELECT id, name, email FROM user WHERE id=? LIMIT 2"
2. db.QueryContext(ctx, selectStmt, args) → executes query with args [1]
3. rows.Next() → iterate result set
4. rows.Scan(&user.ID, &user.Name, &user.Email) → scan into struct fields
5. Return &user (pointer) or sql.ErrNoRows / ErrMultipleRowsFound
```

### 11. Schema Migration Pattern (Go 1.25+)

The `migrate` package adds a third layer for schema evolution, keeping the two-package architecture intact while providing additive-only migrations.

```go
var plan = migrate.Plan{
    migrate.FromStruct[MemoryV1]("memory", migrate.V(1)),   // CREATE TABLE IF NOT EXISTS
    migrate.Diff[MemoryV2]("memory", migrate.V(2), migrate.AutoAdd()), // ALTER TABLE ADD COLUMN
    migrate.Raw("add_index", migrate.V(3), `CREATE INDEX IF NOT EXISTS idx_key ON memory(key)`),
}

err := migrate.Apply(db, plan, migrate.Options{DryRun: false})
```

**Key patterns:**

- **Integer versioning** — `Version int`, applied in ascending order, tracked in `_migrations` table
- **querier interface** — unifies `*sql.DB` and `*sql.Tx` so `Diff` introspection works inside the Apply transaction
- **Safety by default** — `AutoAdd()` only generates `ADD COLUMN`; destructive changes require `migrate.Raw()`
- **DryRun mode** — prints SQL without executing; Diff migrations show a placeholder since they need live schema
- **Transaction wrapping** — pending migration execution + `_migrations` recording are transactional; `_migrations` table creation and version lookup happen outside the transaction for DDL portability. Rollback on any error.
- **Zero new dependencies** — reuses existing `query` package for DDL generation

**Data flow for Diff:**

```txt
1. structColumns(T, dialect)  →  extract {Name, Type, NotNull} from struct tags
2. TableColumns(db, table, dialect)  →  introspect live schema
3. set difference: structCols - liveCols  →  missing columns
4. generate ALTER TABLE ADD COLUMN for each missing column
```

## Design Decisions

| Decision | Rationale |
| ---------- | ----------- |
| Generics vs interface{} | Compile-time type safety, cleaner API, no type assertions |
| Reflection at call-time | Simplicity over code generation; acceptable for CRUD latency |
| `*T` return in Get | Clear sematics for "not found" vs zero-value; matches Go patterns |
| Variadic attributes | Extensible without breaking API; supports optional features |
| Auto-transactions | Safety by default; eliminates a common source of bugs |
| String-based lock detection | Substring match replaced exact-string match — still not `errors.Is`, but works with wrapped errors; kept driver-agnostic |
| Two packages (sqlh + query) | Separation of concerns: query generation vs execution logic |

## Planned Improvements (from ROADMAP)

- Context.Context propagation to all functions
- Native UPSERT (ON CONFLICT DO UPDATE)
- JOIN support ergonomics for composite structs
- Aggregate functions (GROUP BY, HAVING, SUM, AVG)
- Schema migrations (ALTER TABLE, CREATE INDEX) ✅ Experimental via `migrate` package
- Raw SQL fragment injection
- Transactional reads (pass *sql.Tx to Get/List)
