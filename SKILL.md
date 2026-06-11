---
name: sqlh
description: SQL Helper — Go library for auto-generating SQL queries from struct tags. Provides CRUD, ListRange iterator, Set (upsert), Table wrapper, JOIN support, auto-transaction management. Use when working with sqlh database layer.
---

# sqlh — Go SQL Helper

## Overview

**sqlh** is a Go library that auto-generates SQL queries from Go struct tags. Supports SQLite, MySQL, PostgreSQL (opt-in integration tests), and SQL Server (experimental).

**Key principles:**
- **Struct tags define the schema** — `db`, `db_key`, `db_type`
- **No rows.Scan needed** — use `ListRange` iterator or `List`/`Get` which return structs directly
- **Auto-transactions** — all write operations are wrapped in transactions with rollback on error
- **Go 1.25 iterators** — `ListRange` can be used in `for range` loops

## Preferred API Choices

- Use `Get[T]` for one row by unique key; it returns `(*T, error)`.
- Use `List[T]` for normal list pages with the package default row count.
- Use `ListRows[T]` when the page size must be explicit.
- Use `ListRange[T]` for streaming/lazy iteration and large result sets.
- Use `QueryRange[T]` only when a custom `SELECT` string is already needed.
- Use `Table[T]` when several operations target the same table in one component.
- Use `Set` only for SELECT-then-INSERT/UPDATE upsert semantics. For database-native UPSERT, write explicit SQL until sqlh adds native support.

## Core Functions

### Create Table
```go
sqlh.Create[User](db)  // CREATE TABLE IF NOT EXISTS from struct tags
```

### Insert
```go
sqlh.Insert(db, User{Name: "Alice", Age: 30})
sqlh.InsertId(db, User{Name: "Bob"})  // returns last inserted ID
```

### Get (single row)
```go
user, err := sqlh.Get[User](db, sqlh.Eq("id", 1))
```

### List (multiple rows)
```go
users, nextOffset, err := sqlh.List[User](db, 0, "", "name ASC", where...)
users, nextOffset, err := sqlh.ListRows[User](db, 20, "", "name ASC", 10, where...)
```

### ListRange (lazy iterator — no rows.Scan!)
```go
// Preferred way to iterate query results. The iterator wraps rows.Next automatically.
var listErr error
for i, user := range sqlh.ListRange[User](db, 0, "", "name ASC", 0,
    sqlh.Eq("active", true),
    func(e error) { listErr = e },
    context.Background()) {
    fmt.Println(i, user.Name)
}
if listErr != nil {
    return listErr
}
```

### Update
```go
err := sqlh.Update(db, sqlh.UpdateAttr[User]{
    Row:    User{Name: "Alice Updated"},
    Wheres: []sqlh.Where{sqlh.Eq("id", 1)},
})
```

### Delete
```go
sqlh.Delete[User](db, sqlh.Eq("id", 1))
```

### Set (upsert)
```go
// The 'name' field has db_key:"unique", so this becomes an upsert:
sqlh.Set(db, Product{Name: "Laptop", Price: 999}, sqlh.Eq("name", "Laptop"))
```

## Table Wrapper API

```go
tbl, err := sqlh.CreateTable[User](db)
tbl.Insert(User{Name: "Alice"})
tbl.Get(sqlh.Eq("id", 1))
tbl.Update(sqlh.UpdateAttr[User]{
    Row:    User{Name: "Bob"},
    Wheres: []sqlh.Where{sqlh.Eq("id", 1)},
})
tbl.Delete(sqlh.Eq("id", 1))
for _, user := range tbl.List(0, "", "name ASC", 0, errFunc, ctx) {
    fmt.Println(user.Name)
}
```

## JOIN Support

Use a composite wrapper type plus `query.MakeJoin`. The first field is the main table projection; joined tables are added as pointer fields and selected through the join attributes.

```go
type UserTable struct {
    ID   int64  `db:"id" db_key:"primary key autoincrement"`
    Name string `db:"name"`
}

type OrderTable struct {
    ID     int64   `db:"id" db_key:"primary key autoincrement"`
    UserID int64   `db:"user_id"`
    Total  float64 `db:"total"`
}

type UserWithOrders struct {
    *UserTable  // main table
    *OrderTable // joined table
}

join := query.MakeJoin[OrderTable](query.Join{
    Join:  "left",
    Alias: "o",
    On:    "t.id = o.user_id",
})

for _, row := range sqlh.ListRange[UserWithOrders](db, 0, "", "t.name ASC", 0,
    sqlh.SetAlias("t"),
    join,
    func(err error) { log.Fatal(err) },
) {
    if row.OrderTable != nil {
        fmt.Println(row.UserTable.Name, row.OrderTable.Total)
    }
}
```

For aggregate joins (`COUNT`, `SUM`, etc.), prefer an explicit `query.Select` plus `QueryRange` or raw SQL when the result shape is not a direct table-composite scan.

## Custom Table Name

Override the table name using `db_table_name` on a `_ bool` field, or define a `TableName()` method:

```go
type SomeTable struct {
    _    bool      `db_table_name:"custom_table"`
    Name string    `db:"name"`
    Cost float64   `db:"cost"`
}
// Generates: CREATE TABLE IF NOT EXISTS custom_table (name text, cost double)
```

Priority order (highest to lowest):
1. **`TableName()` method** — `func (t *SomeTable) TableName() string { return "my_table" }`
2. **`db_table_name` struct tag** — `_ bool \`db_table_name:"custom_table"\``
3. **Auto-generated snake_case** from type name (e.g. `SomeTable` → `some_table`)

Example with method:

```go
type SomeTable struct {
    Name string    `db:"name" db_table_name:"custom_table"`
    Cost float64   `db:"cost"`
}

func (t *SomeTable) TableName() string {
    return "highest_priority_name"  // this wins over the tag
}
```

## Struct Tags

| Tag | Purpose | Example |
|-----|---------|---------|
| `db` | Column name | `db:"user_name"` |
| `db_key` | Constraints, indexes, foreign keys | `db_key:"primary key autoincrement"` |
| `db_type` | SQL type override | `db_type:"TEXT"` |
| `db_table_name` | Table name override | `db_table_name:"custom_table"` |

### db_key advanced examples

Use `_ string` with `db:"-"` to add KEY (index) and FOREIGN KEY constraints:

```go
type UserAccount struct {
    ID       int64  `db:"id" db_key:"primary key autoincrement"`
    Username string `db:"username" db_key:"unique"`

    // KEY index on username
    _ string `db:"-" db_key:"KEY username (username)"`

    // FOREIGN KEY with CASCADE delete
    _ string `db:"-" db_key:"CONSTRAINT useraccount_ibfk_1 FOREIGN KEY (username) REFERENCES user (username) ON DELETE CASCADE"`
}
```

This generates `CREATE TABLE` with:

```sql
CREATE TABLE IF NOT EXISTS user_account (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE,
    KEY username (username),
    CONSTRAINT useraccount_ibfk_1 FOREIGN KEY (username) REFERENCES user (username) ON DELETE CASCADE
)
```

## Where Clause

```go
sqlh.Eq("name", "Alice")              // name = ?
sqlh.Ne("status", "deleted")          // status <> ?
sqlh.Gt("id", 5)                      // id > 5
sqlh.Gte("age", 18)                   // age >= 18
sqlh.Lt("price", 100.0)               // price < 100.0
sqlh.Lte("price", 100.0)              // price <= 100.0
sqlh.Like("name", "%Alice%")          // name LIKE '%Alice%'
sqlh.In("id", 1, 2, 3)               // id IN (1, 2, 3)
sqlh.IsNull("deleted_at")              // deleted_at IS NULL
sqlh.IsNotNull("created_at")           // created_at IS NOT NULL

// Raw Where{Field, Value} is still available as a low-level escape hatch:
sqlh.Where{Field: "custom_operator", Value: 42}
```

## Pagination

```go
offset := 0
for {
    users, nextOffset, err := sqlh.ListRows[User](db, offset, "", "name ASC", 10)
    if err != nil {
        return err
    }
    for _, user := range users {
        fmt.Println(user.Name)
    }
    if len(users) < 10 {
        break
    }
    offset = nextOffset
}
```

## Context and Errors

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

var listErr error
for _, user := range sqlh.ListRange[User](db, 0, "", "name ASC", 0,
    ctx,
    func(err error) { listErr = err },
) {
    fmt.Println(user.Name)
}
if listErr != nil {
    return listErr
}
```

`ListRange` and `QueryRange` do not yield `error` as the second range value. They report errors through an optional `func(error)` argument.

## Common Mistakes

- Do not write `user, ok, err := sqlh.Get[...]`; `Get` returns only `(*T, error)`.
- Do not pass `limit` to `List`; use `ListRows` for explicit page size.
- Do not write `for row, err := range ListRange`; the yielded values are `(index, row)`.
- Do not call `sqlh.Update(db, row, where...)`; wrap updates in `sqlh.UpdateAttr[T]`.
- Do not expect `Table[T].List` to return a slice; it returns an iterator.
- Do not use manual `rows.Scan` unless you intentionally bypass sqlh with custom raw SQL.
- Do not close shared `*sql.DB` through `Table[T]`; `Table.Close()` is intentionally a no-op.
- Do not tag service fields as real DB columns. Use `db:"-"` or `_` fields for constraints/index declarations.

## Important Rules

1. **DO NOT use `rows.Scan()` or `for rows.Next()`** — sqlh's `ListRange` iterator handles this internally
2. **DO use struct tags** — schema is defined by tags, not SQL files
3. **All write ops are auto-transacted** — no need to wrap in transactions manually
4. **Use `Set` for upsert** — it's atomic (SELECT + INSERT/UPDATE in one transaction)
5. **ListRange errors are callback-based** — provide `func(error)` when errors matter

## Examples Directory

See `examples/` for runnable programs:
- `basic/` — Insert, Get, List, Update, Delete
- `crud/` — Full CRUD workflow example
- `join/` — JOIN queries with nested structs
- `paginator/` — Pagination with `ListRows`
- `set/` — Upsert via `Set`
- `iterators/` — `ListRange` with Go 1.25 iterators
- `context/` — Context cancellation with `ListRange`

## Test Files

- `sqlh_test.go` — SQLite integration tests (CRUD, joins, errors)
- `sqlh_retry_test.go` — Lock-detection and retry unit tests
- `sqlh_update_test.go` — Batch Update regression test (200-row)
- `sqlh_benchmark_test.go` — Performance benchmarks
- `sqlh_mysql_test.go` — MySQL tests (set `SQLH_MYSQL_TEST=1` to enable)
- `query/sqlh_test.go` — Query generation unit tests
- `query/sqlh_meta_cache_test.go` — Metadata cache unit tests
