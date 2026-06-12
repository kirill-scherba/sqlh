---
title: "sqlh — Zero-boilerplate CRUD for Go. 57% less code vs raw sql. Benchmarked."
description: "Reddit post for r/golang announcing sqlh v0.8.0"
platform: reddit
subreddit: r/golang
---

# sqlh — Zero-boilerplate CRUD for Go. 57% less code vs raw sql. Benchmarked.

I built sqlh because I was tired of writing `rows.Scan` and manual SQL for every CRUD operation. Go's `database/sql` is great, but every struct change means updating CREATE TABLE, INSERT, SELECT, rows.Scan — and every typo is a runtime error, not compile-time.

sqlh eliminates all that with Go generics + struct tags:

```go
type User struct {
    ID    int64  `db:"id" db_key:"not null primary key autoincrement"`
    Name  string `db:"name" db_key:"unique"`
    Email string `db:"email"`
}

// That's it. Full CRUD:
sqlh.Create[User](db)
sqlh.Insert(db, User{Name: "Alice", Email: "alice@example.com"})
user, _ := sqlh.Get[User](db, sqlh.Eq("name", "Alice"))
users, _, _ := sqlh.List[User](db, 0, "", "name ASC")
```

No `rows.Scan`. No manual SQL. No transaction boilerplate.

## What's in v0.8.0

- ✅ Go 1.25 generics: `Get[User]()` → `*User`, not `interface{}`
- ✅ `ListRange` returns `iter.Seq2[int, T]` — lazy streaming, no `rows.Scan`
- ✅ Auto-transactions on every write (`Insert`, `Update`, `Delete`, `Set`)
- ✅ Native UPSERT: `INSERT ... ON CONFLICT DO UPDATE` (PostgreSQL, SQLite, MySQL)
- ✅ Type-safe WHERE helpers: `Eq`, `Ne`, `Gt`, `Like`, `In`, `IsNull`
- ✅ JOINs with composite struct scanning
- ✅ Database lock retry (20 attempts × 100ms) for SQLite
- ✅ Multi-DB: SQLite, MySQL, PostgreSQL — all CI-tested

## Benchmarks (in-memory SQLite)

| Operation      | raw sql  | sqlx     | GORM     | **sqlh** |
|----------------|----------|----------|----------|----------|
| **Insert**     | 158,856  | 131,337  | 35,288   | **85,631** |
| **Get by PK**  | 169,090  | 150,082  | 77,489   | **73,601** |
| **List all**   | 11,857   | 9,076    | 6,775    | **7,607** |
| **List limit** | 51,000   | 43,381   | 37,666   | **44,204** |
| **Update**     | 226,963  | 177,242  | 65,828   | **84,083** |
| **Delete**     | 170,503  | 163,185  | 41,375   | **60,503** |

sqlh trades raw speed for correctness: every write is auto-transacted with rollback on error, eliminating an entire class of bugs. For reads, performance remains in the same order of magnitude as raw sql while removing manual scanning and SQL boilerplate.

## When to use it

- CLI tools, startups, microservices with simple schemas → **perfect**
- High-throughput OLTP (100K+ writes/sec) → benchmark first
- Complex multi-table analytics → raw SQL or query builder preferred
- Large teams with dedicated DBAs → GORM may fit better

## Links

- https://github.com/kirill-scherba/sqlh
- https://pkg.go.dev/github.com/kirill-scherba/sqlh

Built this over the last year. Would love feedback — especially on the API surface, error handling patterns, and what you'd want before v1.0.0. Thanks!

---

**Posting instructions:**
1. Post as a **text post** (self post) on r/golang.
2. Best time: **Wednesday or Thursday, 14:00 UTC**.
3. Use flair "Show & Tell" if available.
4. Allocate 1-2 hours after posting to reply to comments.
5. Do NOT cross-post to other subreddits.
6. Do NOT use link shorteners — use full GitHub URL.
