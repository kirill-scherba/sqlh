# sqlh: A Go SQL Helper Package

[![Go Report Card](https://goreportcard.com/badge/github.com/kirill-scherba/sqlh)](https://goreportcard.com/report/github.com/kirill-scherba/sqlh)
[![GoDoc](https://godoc.org/github.com/kirill-scherba/sqlh?status.svg)](https://godoc.org/github.com/kirill-scherba/sqlh/)

`sqlh` is a lightweight helper package for Go that simplifies interactions with SQL databases. It leverages **Go generics (Go 1.25+)** to provide type-safe CRUD functions (`Insert`, `Get`, `List`, `Update`, `Delete`, `Set`) that work directly with your Go structs, automatically generating SQL queries from struct definitions using struct tags — reducing boilerplate code by 60-80%.

## Features

- **Generic Type-Safe API:** Work with any struct type `T any` — no manual SQL writing, no type assertions.
- **Automatic Query Generation:** Auto-generates `CREATE TABLE`, `INSERT`, `UPDATE`, `SELECT`, and `DELETE` statements from struct definitions.
- **Struct Tag-Based Mapping:** Use `db` (column name), `db_type` (SQL type override), and `db_key` (constraints) tags to control table and column definitions.
- **Autoincrement Support:** Automatically excludes fields marked with `autoincrement` from `INSERT` and `UPDATE` statements.
- **Built-in Transactions:** All write operations (`Insert`, `Update`, `Delete`, `Set`) are automatically wrapped in transactions with proper rollback on error.
- **Database Lock Retry:** Built-in retry mechanism (up to 20 attempts with 100ms delay) for "database is locked" errors — ideal for SQLite.
- **Go 1.25 Iterators:** `ListRange` returns `iter.Seq2[T, error]` for lazy iteration over query results.
- **Pagination:** `List` supports offset/limit pagination via `query.Paginator`.
- **JOIN Support:** Basic JOIN support with composite struct scanning.
- **DISTINCT, Alias, Custom Table Names:** Flexible query attributes for advanced SELECT queries.
- **Standardized Error Handling:** Returns `sql.ErrNoRows` and exported package errors (`ErrWhereClauseRequired`, `ErrMultipleRowsFound`, etc.) for easy checking with `errors.Is`.
- **Context Support:** Functions optionally accept `context.Context` for timeouts and cancellations.
- **Database-Agnostic:** Works with SQLite, MySQL, PostgreSQL, and SQL Server (driver-detected `last_insert_rowid`).

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
    "github.com/kirill-scherba/sqlh/query"
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

    // List all users
    users, err := sqlh.List[User](db, 0, "", "", 0)
    if err != nil {
        log.Fatalf("failed to list users: %v", err)
    }
    fmt.Printf("Listed %d users.\n", len(users))

    // Iterate with ListRange (Go 1.25 iterator)
    for user, err := range sqlh.ListRange[User](db, 0, "", "", 0) {
        if err != nil {
            log.Fatalf("failed to iterate: %v", err)
        }
        fmt.Printf("  User: ID=%d, Name=%s, Email=%s\n", user.ID, user.Name, user.Email)
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
defer userTable.Close()

// Use methods
userTable.Insert(User{Name: "Charlie", Email: "charlie@example.com"})
charlie, _ := userTable.Get(sqlh.Where{Field: "name=", Value: "Charlie"})
users, _ := userTable.List(0, "", "", 0)
```

## Query Attributes

`List` and `Get` accept variadic query attributes for advanced queries:

```go
// Pagination
users, _ := sqlh.List[User](db, 0, "", "", 0,
    &query.Paginator{Offset: 10, Limit: 5},
)

// WHERE with OR
users, _ := sqlh.List[User](db, 0, "", "", 0,
    sqlh.Where{Field: "name=", Value: "Alice"},
    sqlh.Where{Field: "name=", Value: "Bob"},
    sqlh.SetWheresJoinOr(),
)

// SELECT DISTINCT
users, _ := sqlh.List[User](db, 0, "", "", 0,
    sqlh.SetDistinct(),
)

// Table alias
users, _ := sqlh.List[User](db, 0, "", "", 0,
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
users, _ := sqlh.List[User](db, 0, "", "", 0, ctx)
```

## Set (Upsert)

`Set` performs an atomic upsert: it selects a row matching WHERE conditions, then either updates it (if found) or inserts a new row (if not found).

```go
err := sqlh.Set(db, User{Name: "Dave", Email: "dave@example.com"},
    sqlh.Where{Field: "name=", Value: "Dave"})
```

## Custom Table Name

Override the auto-generated snake_case table name using a `db_table_name` struct tag on a `_ bool` field, or define a `TableName() string` method on your struct.

**Priority order (highest to lowest):**

1. **`TableName()` method** — highest priority
2. **`db_table_name` struct tag** — on `_ bool` field
3. **Auto-generated snake_case** from type name (e.g. `MyTable` → `my_table`)

### Using `db_table_name` tag

```go
type Product struct {
    _    bool      `db_table_name:"inventory"`
    ID   int64     `db:"id" db_key:"primary key autoincrement"`
    Name string    `db:"name"`
    Cost float64   `db:"cost"`
}
// Generates: CREATE TABLE IF NOT EXISTS inventory (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, cost REAL)
```

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
