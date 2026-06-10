# List API Guidance

This guide explains the roles of the five list/read APIs in `sqlh` and helps you choose the right one for your use case.

## Quick Decision Guide

| You want to... | Use |
| --- | --- |
| Get a quick page of results (default 10 rows) with minimal code | [`List`](https://pkg.go.dev/github.com/kirill-scherba/sqlh#List) |
| Control page size for explicit paginated listing | [`ListRows`](https://pkg.go.dev/github.com/kirill-scherba/sqlh#ListRows) |
| Stream results lazily (memory-efficient, supports JOINs) | [`ListRange`](https://pkg.go.dev/github.com/kirill-scherba/sqlh#ListRange) |
| Run a custom `SELECT` query not expressible via the builder | [`QueryRange`](https://pkg.go.dev/github.com/kirill-scherba/sqlh#QueryRange) |
| Use the `Table[T]` wrapper method-based API | [`Table.List`](https://pkg.go.dev/github.com/kirill-scherba/sqlh#Table.List) |

## When to Use Each API

### `List[T]` — Quick Convenience

Returns a **materialized slice** of up to the default number of rows (10 by default, configurable with `SetNumRows`).

Use `List` when you just need a quick list of items and the default page size is sufficient. It is the shortest path to a `[]T`.

```go
users, nextPage, err := sqlh.List[User](db, 0, "", "name ASC")
```

Under the hood `List` delegates to `ListRows` with the default page size.
For production code that paginates, prefer `ListRows`.

### `ListRows[T]` — Explicit Paginated Listing

Returns a **materialized slice** of up to `numRows` items.
Use `ListRows` when you need explicit control over the page size.
It is the preferred API for paginated listings.

```go
users, nextPage, err := sqlh.ListRows[User](db, offset, "", "name ASC", 20)
```

The `nextPage` return value (`pagination`) equals `offset + len(users)`. Use it as the `offset` for the next call.

### `ListRange[T]` — Streaming/Lazy Iterator

Returns a Go 1.25 `iter.Seq2[int, T]` — a lazy iterator that yields `(index, row)` pairs.

Use `ListRange` when you want:

- **Memory efficiency** — rows are yielded one at a time; you never hold the full result set in memory
- **JOIN queries** — the most natural API for scanning composite structs
- **Context cancellation** — the iterator respects context timeouts and cancellations
- **Early termination** — `break` out of the `range` stops the underlying query early

```go
for i, user := range sqlh.ListRange[User](db, 0, "", "id ASC", 0,
    func(e error) { log.Fatal(e) }) {
    fmt.Printf("%d: %s\n", i, user.Name)
}
```

This is the **core lazy iterator**. `List` and `ListRows` are materialized convenience wrappers built on top of `ListRange`.

### `QueryRange[T]` — Raw SQL Iterator

Returns a Go 1.25 `iter.Seq[T]` for an arbitrary `SELECT` statement.

Use `QueryRange` when you need to run a custom `SELECT` query that cannot be expressed through the `Where` / `Join` attribute system. You provide the raw SQL and query arguments; `sqlh` handles struct scanning.

```go
const rawSQL = `SELECT * FROM users WHERE email LIKE ?`
for _, user := range sqlh.QueryRange[User](db, rawSQL, "%@example.com",
    func(e error) { log.Fatal(e) }) {
    fmt.Println(user.Name)
}
```

### `Table[T].List` — Wrapper Method

`Table[T]` is a convenience wrapper around a `*sql.DB`. Its `List` method delegates to `ListRange`.

```go
table, _ := sqlh.CreateTable[User](db)
for _, user := range table.List(0, "", "name ASC", 0) {
    fmt.Println(user.Name)
}
```

## Relationship Diagram

```
┌──────────────┐  wraps (default page size)  ┌──────────────┐
│     List     │ ──────────────────────────► │   ListRows   │
└──────────────┘                             └──────────────┘
                                                           │
                                                           │ collects into slice
                                                           │ via listRange
                                                           ▼
┌──────────────┐  wraps (explicit limit)    ┌──────────────┐
│ Table.List   │ ─────────────────────────► │  ListRange   │ ◄── core iterator
└──────────────┘                            └──────────────┘
                                              │
                                              │ delegates to
                                              │ for JOINs: queryRange[T]
                                              │ without JOIN: queryRange[struct{In T}]
                                              ▼
                                          ┌──────────────┐
                                          │  QueryRange  │ ◄── raw SQL scanner
                                          └──────────────┘
```

## Recommendations

| Use case | Recommended API | Reason |
| --- | --- | --- |
| Quick listing in examples / prototypes | `List` | Least typing, sensible defaults |
| Paginated REST API endpoint | `ListRows` | Explicit page-size control, straightforward `[]T` |
| Processing large datasets | `ListRange` | Memory-efficient streaming, can `break` early |
| ETL / report generation | `ListRange` | JOIN-friendly, lazy evaluation |
| Custom analytics queries | `QueryRange` | Full SQL control when the builder is insufficient |
| Table-wrapper style code | `Table.List` | Delegates to `ListRange`, keeps code fluent |

## Parameter Naming Note

The first parameter is called `previous` in `List` and `ListRows`, and `offset` in `ListRange` and `Table.List`. They are the same concept — the starting position of the result window. The historical difference in naming is retained for backward compatibility.
