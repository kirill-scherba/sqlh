---
name: sqlh
description: SQL Helper — Go library for auto-generating SQL queries from struct tags. Provides CRUD, ListRange iterator, Set (upsert), Table wrapper, JOIN support, auto-transaction management. Use when working with sqlh database layer.
---

# sqlh — Go SQL Helper

## Overview

**sqlh** is a Go library that auto-generates SQL queries from Go struct tags. Supports SQLite and MySQL.

**Key principles:**
- **Struct tags define the schema** — `db`, `db_key`, `db_type`
- **No rows.Scan needed** — use `ListRange` iterator or `List`/`Get` which return structs directly
- **Auto-transactions** — all write operations are wrapped in transactions with rollback on error
- **Go 1.25 iterators** — `ListRange` can be used in `for range` loops

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
user, ok, err := sqlh.Get[User](db, sqlh.Where{Field: "id=", Value: 1})
```

### List (multiple rows)
```go
users, count, err := sqlh.List[User](db, limit, offset, "name ASC", where...)
```

### ListRange (lazy iterator — no rows.Scan!)
```go
// Preferred way to iterate query results. The iterator wraps rows.Next automatically.
for _, user := range sqlh.ListRange[User](db, 0, "", "name ASC", 0,
    sqlh.Where{},
    func(e error) { log.Fatal(e) },
    context.Background()) {
    fmt.Println(user.Name)
}
```

### Update
```go
sqlh.Update(db, User{Name: "Alice Updated"}, sqlh.Where{Field: "id=", Value: 1})
```

### Delete
```go
sqlh.Delete[User](db, sqlh.Where{Field: "id=", Value: 1})
```

### Set (upsert)
```go
// The 'name' field has db_key:"unique", so this becomes an upsert:
sqlh.Set(db, Product{Name: "Laptop", Price: 999}, sqlh.Where{Field: "name=", Value: "Laptop"})
```

## Table Wrapper API

```go
tbl := sqlh.NewTable[User](db)
tbl.Create()
tbl.Insert(User{Name: "Alice"})
tbl.List(0, "", "name ASC")
tbl.Get(sqlh.Where{Field: "id=", Value: 1})
tbl.Update(User{Name: "Bob"}, sqlh.Where{Field: "id=", Value: 1})
tbl.Delete(sqlh.Where{Field: "id=", Value: 1})
tbl.ListRange(0, "", "name ASC", 0, sqlh.Where{}, errFunc, ctx)
```

## JOIN Support

Use nested structs with `db` tag prefix:

```go
type OrderItem struct {
    OrderID   int64   `db:"order_id"`
    ItemName  string  `db:"item_name"`
    OrderDate string  `db:"order_date"`
    Price     float64 `db:"price"`

    // JOIN with Customer
    Customer struct {
        ID   int64  `db:"customer_id"`
        Name string `db:"customer_name"`
        Email string `db:"customer_email"`
    }
}
```

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
sqlh.Where{Field: "name=", Value: "Alice"}           // name = ?
sqlh.Where{Field: "id>", Value: 5}                    // id > 5
sqlh.Where{Field: "age>=", Value: 18}                 // age >= 18
sqlh.Where{Field: "name LIKE", Value: "%Alice%"}      // name LIKE '%Alice%'
```

## Paginator

```go
p := sqlh.NewPaginator(1, 10)  // page 1, 10 per page
users, count, _ := sqlh.List[User](db, p.Limit, p.Offset, "name ASC")
p.SetTotal(count)              // sets pagination metadata
fmt.Println(p.Pages())         // total pages
```

## Important Rules

1. **DO NOT use `rows.Scan()` or `for rows.Next()`** — sqlh's `ListRange` iterator handles this internally
2. **DO use struct tags** — schema is defined by tags, not SQL files
3. **All write ops are auto-transacted** — no need to wrap in transactions manually
4. **Use `Set` for upsert** — it's atomic (SELECT + INSERT/UPDATE in one transaction)
5. **ListRange requires error handler function** — must provide `func(error)` callback

## Examples Directory

See `examples/` for runnable programs:
- `basic/` — Insert, Get, List, Update, Delete
- `join/` — JOIN queries with nested structs
- `paginator/` — Pagination with `NewPaginator`
- `set/` — Upsert via `Set`
- `iterators/` — `ListRange` with Go 1.25 iterators
- `context/` — Context cancellation with `ListRange`

## Test Files

- `sqlh_test.go` — SQLite tests
- `sqlh_mysql_test.go` — MySQL tests (requires external instance)