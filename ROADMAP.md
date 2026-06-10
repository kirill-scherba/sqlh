# `sqlh` Development Roadmap

This document outlines the planned features and improvements for the `sqlh` package. It is a living document and may be adjusted based on priorities and feedback.

## ✅ Completed

### Core Query Enhancements

- **Custom Table Name:** Override auto-generated snake_case table name via `db_table_name` struct tag or `TableName()` method.
- **Flexible `SELECT` Queries:**
  - **Select Specific Columns:** `query.Select` generates a column list from struct fields instead of `SELECT *`.
  - **`DISTINCT` Support:** `SetDistinct()` option for `SELECT DISTINCT`.
- **Advanced `WHERE` Conditions:**
  - **`OR` Operator:** `SetWheresJoinOr()` combines conditions with `OR`.
  - **`IN` Operator:** `Where{Field: "id IN", Value: ...}` supports list parameters.
  - **`LIKE`, `IS NULL` / `IS NOT NULL`:** Supported through `Where{Field: "name LIKE", Value: "%foo%"}`.
- **`context.Context` Propagation:** All query functions accept `context.Context` for timeouts and cancellations.

### Advanced Features & Data Integrity

- **`JOIN` Support:** `MakeJoin[T]` for LEFT/RIGHT/INNER/OUTER JOINs with composite struct scanning via `ListRows`.
- **Atomic Upsert:** `Set` performs atomic upsert (SELECT then INSERT/UPDATE).
- **Go 1.25 Iterators:** `ListRange` returns `iter.Seq2[int, T]` for lazy iteration over query results. `QueryRange` returns `iter.Seq[T]`.
- **Pagination:** `ListRows` and `ListRange` accept explicit `offset` and `limit` parameters.
- **Table Wrapper API:** `Table[T]` provides method-based API for all CRUD operations.
- **Database Lock Retry:** Built-in retry for "database is locked" errors (SQLite).
- **Database Support:** SQLite and MySQL tested; PostgreSQL tested (opt-in); SQL Server experimental.
- **Composite Types:** Supports `complex64`, `complex128`, `[]byte`, `time.Time` struct fields.

## Future Directions

### Schema Management

- **Schema Migrations:** Add `ALTER TABLE` to modify existing tables (add/remove columns).
- **`CREATE INDEX` / `FOREIGN KEY`:** Already supported via `db_key` struct tag:

  ```go
  // KEY (index)
  _ string `db:"-" db_key:"KEY username (username)"`

  // FOREIGN KEY with CASCADE
  _ string `db:"-" db_key:"CONSTRAINT fk_user FOREIGN KEY (username) REFERENCES user (username) ON DELETE CASCADE"`
  ```

  No future work needed — the `db_key` tag already generates these in `CREATE TABLE`.

### API Unification (v1.0.0 consideration)

- **Options-based List API:** Consider a single `List` function with an options
  struct for v1.0.0, replacing the three-materialized-vs-iterator split:
  ```go
  type ListOpts struct {
      Offset   int
      Limit    int
      GroupBy  string
      OrderBy  string
      Materialize bool  // true -> []T; false -> iter.Seq2[int, T]
  }
  func List[T any](db querier, opts ListOpts, listAttrs ...any) (...)
  ```
  This would consolidate `List`/`ListRows`/`ListRange` into one entry point
  without losing expressiveness. Requires careful migration path for existing
  callers.

### Performance

- **Batch Operations:** Batch insert/update multiple rows in a single query.
- **Connection Pool Tuning:** Built-in helpers for connection pool configuration.
- **Prepare Statement Cache:** Cache frequently used prepared statements.