---
title: "Why I Built a Go SQL Helper"
published: false
tags: go, sql, database, opensource
---

# Why I Built a Go SQL Helper

> *Zero-boilerplate SQL for Go. Define a struct with tags. That's it.*

If you write Go and talk to SQL databases, you know the pain. Every CRUD operation requires writing a raw SQL string, carefully mapping columns to struct fields with `rows.Scan`, manually wrapping writes in `Begin/Commit/Rollback`, and keeping your schema definitions in sync with your code. It's tedious, error-prone, and the boilerplate never ends.

This is the story of `sqlh` — a library I built to eliminate all of that, while staying in the "sweet spot" between raw SQL (too much work) and heavy ORMs (too much magic).

## The Problem: Go + SQL = Death by a Thousand `rows.Scan`s

Go's `database/sql` package is excellent. It gives you a solid, portable foundation for talking to any SQL database. But it intentionally leaves the hard work to you.

Here's what a simple CRUD workflow looks like with raw `database/sql`:

```go
// 1. Create table — raw DDL string
_, err := db.Exec(`CREATE TABLE IF NOT EXISTS user (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE,
    email TEXT,
    age INTEGER
)`)

// 2. Insert — explicit placeholders and args
_, err = db.Exec(
    "INSERT INTO user (name, email, age) VALUES (?, ?, ?)",
    "Alice", "alice@example.com", 30,
)

// 3. Get by ID — QueryRow + manual Scan
var u User
err = db.QueryRow("SELECT id, name, email, age FROM user WHERE id = ?", 1).
    Scan(&u.ID, &u.Name, &u.Email, &u.Age)

// 4. List all — Query + rows.Next + rows.Scan loop
rows, err := db.Query("SELECT id, name, email, age FROM user ORDER BY name ASC")
var users []User
for rows.Next() {
    var u User
    if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Age); err != nil {
        log.Fatal(err)
    }
    users = append(users, u)
}
rows.Close()

// 5. Update — raw SQL with placeholders
_, err = db.Exec(
    "UPDATE user SET email = ?, age = ? WHERE id = ?",
    "alice.new@example.com", 31, 1,
)

// 6. Delete — raw SQL
_, err = db.Exec("DELETE FROM user WHERE id = ?", 1)
```

That is **~115 lines of code** for six basic operations. And every time you add a column, you must update the `CREATE TABLE` string, the `INSERT` columns, the `SELECT` columns, and the `rows.Scan` call. One typo in any of those, and you get a runtime error — no compile-time safety.

### The Pain Points

| Pain | Why it hurts |
|------|-------------|
| Manual SQL writing | Every CRUD operation needs a raw SQL string — no compile-time check |
| `rows.Scan` verbosity | 4–5 lines per query result just to map columns to struct fields |
| Transaction boilerplate | `db.Begin()` + `defer tx.Rollback()` + `tx.Commit()` — repeated everywhere |
| No schema traceability | Table DDL lives in migration files, structs in Go — they drift |
| Error-prone column ordering | Adding a column means updating SQL strings *and* `Scan` calls |

## Existing Solutions: sqlx, GORM, and the Missing Middle

The Go ecosystem offers two well-known paths to reduce this pain. Both have trade-offs.

### sqlx: Better, But Still Manual

[`sqlx`](https://github.com/jmoiron/sqlx) is a popular enhancement over `database/sql`. It adds `StructScan`, `Get`, `Select`, and named query parameters. You still write raw SQL, but `rows.Scan` is automated.

```go
// sqlx: still manual SQL, but StructScan eliminates Scan
var u User
dbx.Get(&u, "SELECT id, name, email, age FROM user WHERE id = ?", 1)
```

sqlx saves about **30% of the boilerplate** (down to ~80 lines). But you still write every `CREATE TABLE`, `INSERT`, `SELECT`, `UPDATE`, and `DELETE` by hand. SQL generation is not its job.

### GORM: Full ORM, Full Magic

[GORM](https://gorm.io/) is the heavyweight champion. It generates everything — schema, queries, migrations — and provides a rich chainable API. But it comes with a cost:

- **Heavy reflection overhead** at runtime
- **Steep learning curve** — tags, hooks, scopes, associations
- **~4 MB binary size increase** just for the ORM
- **Magic that hides complexity** — until it doesn't, and you spend hours debugging

For large teams with dedicated DBAs and complex domain models, GORM is a solid choice. For CLI tools, startups, and small-to-medium services, it's overkill.

### sqlh: The Sweet Spot

| Feature | `database/sql` | sqlx | GORM | **sqlh** |
|---|---|---|---|---|
| SQL generation | ❌ Manual | ❌ Manual | ✅ Full | ✅ Full |
| `rows.Scan` needed | ✅ Yes | ❌ `StructScan` | ❌ Auto | ❌ Auto |
| Type-safe (generics) | ❌ | ❌ | ❌ | ✅ |
| Auto-transactions | ❌ | ❌ | ✅ | ✅ |
| Lock retry | ❌ | ❌ | ❌ | ✅ |
| Learning curve | Medium | Medium | High | **Low** |
| Binary size overhead | 0 | ~200 KB | ~4 MB | ~200 KB |

sqlh lives between sqlx and GORM:
- **Zero-boilerplate CRUD** — struct tags generate all SQL
- **Type-safe via Go generics** — `Get[User]()` returns `*User`, not `interface{}`
- **No magic** — what you see in the struct is what you get in the database
- **Lightweight** — minimal reflection, cached metadata, no hidden complexity

## How sqlh Works: Struct Tags as Single Source of Truth

The core idea is simple: **your Go struct is your schema**.

```go
type User struct {
    ID    int64  `db:"id" db_key:"not null primary key autoincrement"`
    Name  string `db:"name" db_key:"unique"`
    Email string `db:"email"`
    Age   int    `db:"age"`
}
```

Three struct tags control everything:

| Tag | Purpose | Example |
|-----|---------|---------|
| `db` | Column name | `db:"user_name"` |
| `db_key` | Constraints, indexes | `db_key:"primary key autoincrement"` |
| `db_type` | SQL type override | `db_type:"TEXT"` |

From this single struct definition, sqlh generates:

- **CREATE TABLE** — `sqlh.Create[User](db)` generates and executes `CREATE TABLE IF NOT EXISTS user (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT UNIQUE, email TEXT, age INTEGER)`
- **INSERT** — `sqlh.Insert(db, User{Name: "Alice"})` generates `INSERT INTO user (name, email, age) VALUES (?, ?, ?)`
- **SELECT** — `sqlh.Get[User](db, ...)` generates `SELECT id, name, email, age FROM user WHERE ... LIMIT 2`
- **UPDATE** — `sqlh.Update(db, ...)` generates `UPDATE user SET name=?, email=?, age=? WHERE ...`
- **DELETE** — `sqlh.Delete[User](db, ...)` generates `DELETE FROM user WHERE ...`

### Architecture

```
┌─────────────────────────────────────────────┐
│  sqlh package                               │
│  Insert, Get, List, Update, Delete, Set,    │
│  Create — with auto-transactions            │
├─────────────────────────────────────────────┤
│  query package                              │
│  SQL generation, metadata cache, JOINs      │
├─────────────────────────────────────────────┤
│  database/sql (stdlib)                      │
│  Connection pool, raw query execution       │
└─────────────────────────────────────────────┘
```

### Key Design Decisions

1. **Generics-first (Go 1.25+)** — `Get[User]()` returns `*User` with compile-time type safety. No `interface{}`, no type assertions.
2. **Reflection at call-time** — Struct metadata is parsed once and cached in a `sync.Map` keyed by `reflect.Type`. Subsequent calls reuse table names, field lists, and scan metadata.
3. **Auto-transactions on writes** — Every `Insert`, `Update`, `Delete`, and `Set` is automatically wrapped in `BEGIN...COMMIT` with `ROLLBACK` on error. You never forget a transaction again.
4. **Database lock retry** — SQLite "database is locked" errors are automatically retried up to 20 times with 100ms backoff. Production-grade resilience out of the box.
5. **Multi-database support** — SQLite (primary), MySQL, PostgreSQL (both CI-tested), and SQL Server (experimental).

## CRUD in 50 Lines: The Quick Start

Here's the complete CRUD workflow with sqlh. Same operations as the raw SQL example above — **~57% less code**:

```go
package main

import (
    "database/sql"
    "fmt"

    "github.com/kirill-scherba/sqlh"
    _ "github.com/mattn/go-sqlite3"
)

type User struct {
    ID    int64  `db:"id" db_key:"not null primary key autoincrement"`
    Name  string `db:"name" db_key:"unique"`
    Email string `db:"email"`
    Age   int    `db:"age"`
}

func main() {
    db, _ := sql.Open("sqlite3", "file::memory:?cache=shared")
    defer db.Close()

    // 1. Create table from struct
    sqlh.Create[User](db)

    // 2. Insert
    sqlh.Insert(db, User{Name: "Alice", Email: "alice@example.com", Age: 30})
    bobID, _ := sqlh.InsertId(db, User{Name: "Bob", Email: "bob@example.com", Age: 25})

    // 3. Get by ID — returns *User, not interface{}
    u, _ := sqlh.Get[User](db, sqlh.Eq("id", bobID))
    fmt.Println(u.Name) // "Bob"

    // 4. List all — returns []User + next offset
    users, _, _ := sqlh.List[User](db, 0, "", "name ASC")
    fmt.Println(len(users)) // 2

    // 5. Update — pass full struct to avoid zeroing other columns
    sqlh.Update(db, sqlh.UpdateAttr[User]{
        Row:    User{Name: "Alice", Email: "alice.new@example.com", Age: 31},
        Wheres: []sqlh.Where{sqlh.Eq("id", 1)},
    })

    // 6. Delete
    sqlh.Delete[User](db, sqlh.Eq("id", bobID))
}
```

**~50 lines.** No raw SQL. No `rows.Scan`. No transaction management. No column ordering errors.

### Side-by-Side Comparison

| Operation | Raw `database/sql` | sqlx | **sqlh** |
|-----------|------------------|------|----------|
| CREATE TABLE | Raw SQL string | Raw SQL string | `sqlh.Create[User](db)` |
| INSERT | `Exec(?,?,?)` | `NamedExec` | `Insert(T)` |
| GET | `QueryRow + Scan` | `Get(&T)` | `Get[T](where)` |
| LIST | `rows.Next + Scan` | `Select` | `List[T](...)` |
| UPDATE | `Exec(?,?,?,?)` | `NamedExec` | `Update(attr)` |
| DELETE | `Exec(?)` | `Exec(?)` | `Delete[T](where)` |
| COUNT | `QueryRow + Scan` | `Get(&int)` | `Count[T]()` |

| | Lines of code | Reduction |
|---|---|---|
| Raw `database/sql` | ~115 | baseline |
| sqlx | ~80 | −30% |
| **sqlh** | **~50** | **−57%** |

## Benchmarks: The Numbers

How fast is sqlh in practice? I created a standalone `bench/` module comparing raw `database/sql`, `sqlx`, GORM, and sqlh on the same CRUD workload. All benchmarks use in-memory SQLite — zero external setup.

Reproduce on your machine:

```bash
cd bench && go test -bench=. -benchmem -benchtime=1s
```

### CRUD Throughput (ops/sec)

| Operation | raw sql | sqlx | GORM | **sqlh** |
|-----------|---------|------|------|----------|
| **Insert** | 158,856 | 131,337 | 35,288 | **85,631** |
| **Get by PK** | 169,090 | 150,082 | 77,489 | **73,601** |
| **List all** | 11,857 | 9,076 | 6,775 | **7,607** |
| **List limit 10** | 51,000 | 43,381 | 37,666 | **44,204** |
| **Update** | 226,963 | 177,242 | 65,828 | **84,083** |
| **Delete** | 170,503 | 163,185 | 41,375 | **60,503** |

### Memory Allocations (bytes/op, allocs/op)

| Operation | raw sql | sqlx | GORM | **sqlh** |
|-----------|---------|------|------|----------|
| **Insert** | 328 B, 12 | 721 B, 20 | 5,536 B, 82 | **1,274 B, 39** |
| **Get by PK** | 792 B, 27 | 976 B, 31 | 3,952 B, 66 | **2,593 B, 78** |
| **List all** | 23,744 B, 528 | 26,376 B, 632 | 27,669 B, 946 | **26,394 B, 745** |
| **List limit** | 3,120 B, 76 | 3,624 B, 91 | 6,145 B, 141 | **3,958 B, 115** |
| **Update** | 296 B, 10 | 680 B, 19 | 5,079 B, 68 | **1,393 B, 43** |
| **Delete** | 216 B, 7 | 216 B, 7 | 5,483 B, 67 | **1,139 B, 37** |

### What the Numbers Tell Us

- **GORM** has the highest latency and allocation footprint across all operations, reflecting its rich feature set and internal reflection overhead.
- **sqlh** sits between raw sql/sqlx and GORM. The moderate overhead comes from auto-generated SQL, struct tag parsing, and built-in transaction wrapping for writes.
- **sqlh trades raw speed for correctness**: every write is auto-transacted with rollback on error, eliminating an entire class of bugs at the cost of ~2–6x latency versus raw sql for single-row mutations.
- **ListAll** is dominated by the cost of scanning 100 rows. All libraries show similar performance here.

> **Environment:** Linux AMD Ryzen 9 3900, Go 1.26.3, SQLite in-memory.
> Run `cd bench && go test -bench=. -benchmem -benchtime=1s` on your own hardware for an apples-to-apples comparison.

## When to Use sqlh

sqlh is not a silver bullet. Here's where it shines and where you might want something else:

| Use case | Recommendation |
|----------|---------------|
| CLI tools & utilities | ✅ Perfect — zero migration files, single binary |
| Startups & MVPs | ✅ Ship faster, refactor later |
| Microservices with simple schemas | ✅ Low overhead, type-safe |
| High-throughput OLTP (>100K writes/sec) | ⚠️ Test first — raw sql may be needed |
| Complex multi-table analytics | ⚠️ Prefer raw SQL or a query builder |
| Large teams with dedicated DBAs | ⚠️ GORM or sqlx may fit better |
| Learning Go + SQL | ✅ Great teaching tool — low cognitive load |

## Getting Started

```bash
go get github.com/kirill-scherba/sqlh
```

- 📖 [README & Quick Start](https://github.com/kirill-scherba/sqlh)
- 📦 [pkg.go.dev reference](https://pkg.go.dev/github.com/kirill-scherba/sqlh)
- 🏗️ [Source code](https://github.com/kirill-scherba/sqlh)

## What’s Next

sqlh is actively developed. As of v0.8.0 (June 2026), the library supports:

- ✅ Full CRUD with auto-transactions
- ✅ Native UPSERT (PostgreSQL, SQLite, MySQL)
- ✅ JOIN queries with composite struct scanning
- ✅ Go 1.25 iterators (`ListRange`) for lazy streaming
- ✅ Type-safe WHERE helpers (`Eq`, `Ne`, `Gt`, `Like`, `In`, etc.)
- ✅ Database lock retry for SQLite
- ✅ Multi-database support (SQLite, MySQL, PostgreSQL)

On the roadmap: aggregate functions (`SUM`, `AVG`), schema migrations, batch operations, and transactional reads. The API is stabilizing toward v1.0.0.

If you’re building a Go project that talks to SQL and you’re tired of writing the same boilerplate over and over — give sqlh a try. Define your struct. That’s it.

---

*Written by [Kirill Scherba](https://github.com/kirill-scherba). sqlh is open source under the BSD license. Contributions welcome.*
