# sqlh: A Go SQL Helper Package

[![Go Report Card](https://goreportcard.com/badge/github.com/kirill-scherba/sqlh)](https://goreportcard.com/report/github.com/kirill-scherba/sqlh)
[![GoDoc](https://godoc.org/github.com/kirill-scherba/sqlh?status.svg)](https://godoc.org/github.com/kirill-scherba/sqlh/)
[![Test](https://github.com/kirill-scherba/sqlh/actions/workflows/test.yml/badge.svg)](https://github.com/kirill-scherba/sqlh/actions/workflows/test.yml)

`sqlh` is a lightweight helper package for Go that simplifies interactions with SQL databases. It leverages **Go generics (Go 1.25+)** to provide type-safe CRUD functions (`Insert`, `Get`, `List`, `Update`, `Delete`, `Set`) that work directly with your Go structs, automatically generating SQL queries from struct definitions using struct tags — reducing boilerplate code by 60-80%.

## Features

- **Generic Type-Safe API:** Work with any struct type `T any` — no manual SQL writing, no type assertions.
- **Automatic Query Generation:** Auto-generates `CREATE TABLE`, `INSERT`, `UPDATE`, `SELECT`, and `DELETE` statements from struct definitions.
- **Struct Tag-Based Mapping:** Use `db` (column name), `db_type` (SQL type override), and `db_key` (constraints) tags to control table and column definitions.
- **Autoincrement Support:** Automatically excludes fields marked with `autoincrement` from `INSERT` and `UPDATE` statements.
- **Built-in Transactions:** All write operations (`Insert`, `Update`, `Delete`, `Set`) are automatically wrapped in transactions with proper rollback on error.
- **Database Lock Retry:** Built-in retry mechanism (up to 20 attempts with 100ms delay) for "database is locked" errors — ideal for SQLite.
- **Go 1.25 Iterators:** `ListRange` returns `iter.Seq2[int, T]` for lazy iteration over query results.
- **Pagination:** `ListRows` and `ListRange` support explicit offset/limit pagination.
- **JOIN Support:** Basic JOIN support with composite struct scanning.
- **DISTINCT, Alias, Custom Table Names:** Flexible query attributes for advanced SELECT queries.
- **Standardized Error Handling:** Returns `sql.ErrNoRows` and exported package errors (`ErrWhereClauseRequired`, `ErrMultipleRowsFound`, etc.) for easy checking with `errors.Is`.
- **Context Support:** Functions optionally accept `context.Context` for timeouts and cancellations.

## Database Support

| Database  | Status       | CI  | Notes                        |
|-----------|--------------|-----|------------------------------|
| SQLite    | **Tested**   | ✅  | Full CRUD tested on every CI |
| MySQL     | **Tested**   | ✅  | Opt-in via `SQLH_MYSQL_TEST` / service container |
| PostgreSQL| **Tested**   | ✅  | Opt-in via `SQLH_TEST_POSTGRES=1` / service container |
| SQL Server| Experimental | ❌   | `getLastInsertID` support only; no integration tests |

## Installation

```bash
go get github.com/kirill-scherba/sqlh
```

## Quick Start

Here's a quick example of how to use `sqlh` with an in-memory SQLite database.

### 1. Define Your Struct

Define a Go struct that represents your database table. Use struct tags to define column names, types, and keys.

```go
package main

import (
    "database/sql"
    "errors"
    "fmt"
    "log"

    "github.com/kirill-scherba/sqlh"
    _ "github.com/mattn/go-sqlite3"
)

// User represents the users table.
type User struct {
    ID    int64  `db:"id" db_key:"not null primary key autoincrement"`
    Name  string `db:"name" db_key:"unique"`
    Email string `db:"email"`
}
```

### 2. Connect and Create Table

Use `sqlh.Create` to generate and execute a `CREATE TABLE` statement from your struct in one call.

```go
func main() {
    // Open in-memory SQLite database for this example
    db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
    if err != nil {
        log.Fatalf("failed to open database: %v", err)
    }
    defer db.Close()

    // Create table from struct
    if err := sqlh.Create[User](db); err != nil {
        log.Fatalf("failed to create table: %v", err)
    }
    fmt.Println("Table 'user' created successfully.")

    // Insert a new user
    alice := User{Name: "Alice", Email: "alice@example.com"}
    if err := sqlh.Insert(db, alice); err != nil {
        log.Fatalf("failed to insert user: %v", err)
    }
    fmt.Println("Inserted Alice.")

    // Insert with returned ID
    bob := User{Name: "Bob", Email: "bob@example.com"}
    bobID, err := sqlh.InsertId(db, bob)
    if err != nil {
        log.Fatalf("failed to insert user: %v", err)
    }
    fmt.Printf("Inserted Bob with ID=%d.\n", bobID)

    // Get user by name
    retrievedUser, err := sqlh.Get[User](db, sqlh.Where{Field: "name=", Value: "Alice"})
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            log.Println("User not found.")
        } else {
            log.Fatalf("failed to get user: %v", err)
        }
        return
    }
    fmt.Printf("Retrieved User: ID=%d, Name=%s, Email=%s\n",
        retrievedUser.ID, retrievedUser.Name, retrievedUser.Email)

    // Update Alice's email
    retrievedUser.Email = "alice.new@example.com"
    updateAttr := sqlh.UpdateAttr[User]{
        Row:    *retrievedUser,
        Wheres: []sqlh.Where{{Field: "id=", Value: retrievedUser.ID}},
    }
    if err := sqlh.Update(db, updateAttr); err != nil {
        log.Fatalf("failed to update user: %v", err)
    }
    fmt.Println("Updated Alice's email.")

    // List all users (quick convenience — 10 rows default)
    users, pagination, err := sqlh.List[User](db, 0, "", "name ASC")
    if err != nil {
        log.Fatalf("failed to list users: %v", err)
    }
    fmt.Printf("Listed %d users, next offset=%d.\n", len(users), pagination)

    // Iterate with ListRange (Go 1.25 iterator)
    for i, user := range sqlh.ListRange[User](db, 0, "", "name ASC", 0,
        func(err error) { log.Fatalf("failed to iterate: %v", err) },
    ) {
        fmt.Printf("  #%d User: ID=%d, Name=%s, Email=%s\n",
            i, user.ID, user.Name, user.Email)
    }

    // Delete user
    if err := sqlh.Delete[User](db, sqlh.Where{Field: "id=", Value: bobID}); err != nil {
        log.Fatalf("failed to delete user: %v", err)
    }
    fmt.Println("Deleted Bob.")
}
```

## Table Wrapper API

For convenience, you can use the method-based `Table[T]` API:

```go
// Create table wrapper
userTable, err := sqlh.CreateTable[User](db)
if err != nil {
    log.Fatalf("failed to create table: %v", err)
}

// Use methods
userTable.Insert(User{Name: "Charlie", Email: "charlie@example.com"})
charlie, _ := userTable.Get(sqlh.Where{Field: "name=", Value: "Charlie"})
fmt.Println(charlie.Name)
for _, user := range userTable.List(0, "", "name ASC", 0) {
    fmt.Println(user.Name)
}
```

> **Note:** `Table[T]` is a lightweight wrapper over a shared `*sql.DB` connection pool.
> It does **not** own the database connection — the pool is managed by the caller
> who created it. `Table.Close()` exists but is intentionally a no-op for
> backward compatibility. Resource cleanup is done by closing the original
> `*sql.DB` handle (`db.Close()`).

## Query Attributes

`List`, `ListRows`, and `ListRange` accept variadic query attributes for advanced queries:

```go
// Pagination
users, nextOffset, err := sqlh.ListRows[User](db, 10, "", "name ASC", 5)

// WHERE with OR
users, _, err := sqlh.List[User](db, 0, "", "name ASC",
    sqlh.Where{Field: "name=", Value: "Alice"},
    sqlh.Where{Field: "name=", Value: "Bob"},
    sqlh.SetWheresJoinOr(),
)

// SELECT DISTINCT
users, _, err := sqlh.List[User](db, 0, "", "name ASC",
    sqlh.SetDistinct(),
)

// Table alias
users, _, err := sqlh.List[User](db, 0, "", "name ASC",
    sqlh.SetAlias("u"),
)

// JOIN with ListRows (supports composite structs)
type UserWithProfile struct {
    User    User
    Profile Profile
}
users, _, _ := sqlh.ListRows[UserWithProfile](db, 0, "", "", 10,
    // Set main table alias
    sqlh.SetAlias("t"),
    // Join with MakeJoin: automatically sets name and fields from struct
    query.MakeJoin[Profile](query.Join{
        Join:  "LEFT",
        On:    "t.id = o.user_id",
        Alias: "o",
    }),
)

// Context
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
users, _, err := sqlh.List[User](db, 0, "", "name ASC", ctx)
```

## Choosing the Right List API

| Function | Returns | When to use |
|----------|---------|-------------|
| `List` | `([]T, int, error)` | Quick convenience with default page size (10 rows) |
| `ListRows` | `([]T, int, error)` | Explicit page-size control for pagination loops |
| `ListRange` | `iter.Seq2[int, T]` | Lazy/streaming iteration, JOIN queries, context cancellation |
| `QueryRange` | `iter.Seq[T]` | Raw SQL queries beyond generated query coverage |

See [docs/list-api-guidance.md](docs/list-api-guidance.md) for detailed guidance.

## Set (Upsert)

`Set` performs an atomic upsert: it selects a row matching WHERE conditions, then either updates it (if found) or inserts a new row (if not found).

```go
err := sqlh.Set(db, User{Name: "Dave", Email: "dave@example.com"},
    sqlh.Where{Field: "name=", Value: "Dave"})
```

## Custom Table Name

Override the auto-generated snake_case table name using a `db_table_name`
struct tag on a sentinel `_` field, or define a `TableName() string` method on
your struct.

> **Why `_`?** Fields named `_` are ignored by sqlh as columns — they carry
> only struct tags. The actual Go type of the sentinel field does not matter;
> `any`, `string`, and `bool` all behave identically. This keeps table-name
> overrides self-contained and backward-compatible.

**Priority order (highest to lowest):**

1. **`TableName()` method** — highest priority
2. **`db_table_name` struct tag** — on a `_` sentinel field (any Go type)
3. **Auto-generated snake_case** from type name (e.g. `MyTable` → `my_table`)

### Using `db_table_name` tag

```go
type Product struct {
    _    any       `db_table_name:"inventory"`
    ID   int64     `db:"id" db_key:"primary key autoincrement"`
    Name string    `db:"name"`
    Cost float64   `db:"cost"`
}
// Generates: CREATE TABLE IF NOT EXISTS inventory (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, cost REAL)
```

> `string` and `bool` are also valid sentinel types — they behave identically.

### Using `TableName()` method

```go
type Product struct {
    ID   int64     `db:"id" db_key:"primary key autoincrement"`
    Name string    `db:"name"`
    Cost float64   `db:"cost"`
}

func (Product) TableName() string { return "my_products" }
// Generates: CREATE TABLE IF NOT EXISTS my_products (...)
```

## SQL Fragment Safety

sqlh parameterizes **values**, but not **SQL identifiers or SQL fragments**.

The following fields are embedded directly into SQL and **must be trusted constants**
(never user-supplied without validation):

- `Where.Field` — column name and operator (e.g. `"name="`, `"id IN"`)
- `orderBy` — ORDER BY clause
- `groupBy` — GROUP BY clause
- `Join.On` — JOIN ON condition
- `SetAlias` — table alias
- `SetName` — table name override

User-provided values must go through `Where.Value` or standard query arguments.

## Documentation

The [docs](docs/) directory contains comprehensive documentation about the project architecture, progress, and context:

- [projectbrief.md](docs/projectbrief.md) — Project overview and core capabilities
- [productContext.md](docs/productContext.md) — Problems solved and user experience goals
- [systemPatterns.md](docs/systemPatterns.md) — Architecture and design patterns
- [techContext.md](docs/techContext.md) — Technology stack and API surface
- [activeContext.md](docs/activeContext.md) — Current development focus and roadmap
- [progress.md](docs/progress.md) — Feature completeness and release history

## Changelog

For a detailed list of changes, please see the [CHANGELOG.md](CHANGELOG.md) file.

## Licence

BSD
