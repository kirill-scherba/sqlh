<!--
This file follows the principles of Keep a Changelog (https://keepachangelog.com/en/1.0.0/).
It's intended to be a human-readable history of changes.
-->

# Changelog

## [Unreleased]

## [v0.8.0] - 2026-06-12

### Added

- **Type-safe WHERE helper API**: `query.Eq`, `query.Ne`, `query.Gt`, `query.Gte`, `query.Lt`, `query.Lte`, `query.Like`, `query.In`, `query.IsNull`, `query.IsNotNull` — fluent, chainable helpers for building WHERE conditions without raw SQL strings. (#14)
- **Code comparison examples**: Side-by-side CRUD comparison of raw `database/sql`, `sqlx`, and `sqlh` in `examples/comparison/`, demonstrating the 60-80 % boilerplate reduction. (#26)
- **Performance benchmarks**: 24 benchmark functions in `bench/` comparing `sqlh` vs raw `database/sql`, `sqlx`, and GORM across Insert, Get, List, Update, and Delete operations. (#28)
- **Animated GIF demo**: `docs/demo.gif` showing zero-boilerplate CRUD workflow from struct definition to database operations. (#25)
- **`pkg.go.dev` badge**: Official Go Reference badge replacing deprecated `godoc.org` link in README. (#27)
- **`Table[T]` wrapper API examples**: `Example_` functions in `sqlh_example_test.go` for `pkg.go.dev` rendering. (#29)

### Fixed

- **Backward compatibility for `query.Update` / `query.Delete`**: Restored exported `query.Update` and `query.Delete` wrappers that were accidentally removed. Fixed missing space before `?` in `processWhere`. (#14)

### Changed

- **README benchmark tables**: Corrected units (ops/sec, allocs/op), values, and formatting for accuracy. (#28)
- **Comparison UPDATE semantics**: Made the `UPDATE` operation in `examples/comparison/` semantically equivalent across all three approaches for fair benchmarking. (#26)

## [v0.7.1] - 2026-06-11

### Added

- **Native database UPSERT for `Set`**: PostgreSQL (`ON CONFLICT ... DO UPDATE`), SQLite (`ON CONFLICT ... DO UPDATE`), MySQL (`ON DUPLICATE KEY UPDATE`). Falls back to the legacy SELECT-then-INSERT/UPDATE path for unsupported or unknown drivers. Includes `extractColumn` helper and comprehensive tests for all three dialects. (#13)
- **`query.Fields[T]()`**: Exported function for public field name access. (#13)
- **`db_table_name` ergonomics**: Support `any` and `string` (in addition to `bool`) as sentinel types for table-name override. The Go type of a `_` sentinel field is irrelevant; all are skipped identically. (#18)

### Changed

- **List API guidance**: Clarified `List`, `ListRows`, `ListRange`, `QueryRange` roles in documentation. `List` remains a convenience helper; `ListRows` is the **preferred** API for explicit pagination; `ListRange` is the **core lazy iterator** — memory-efficient streaming, JOINs, context; `QueryRange` is the **raw SQL escape hatch** — bypasses query generation. New file: `docs/list-api-guidance.md`. (#17)
- **SQL Server support documentation**: Explicitly marked as experimental/partial with no integration tests or CI; added ROADMAP entry for optional future SQL Server CI. (#15)
- **`Table[T]` documentation**: Removed `Table.Close()` no-op from recommended documentation flow — `Table[T]` does not own the `*sql.DB` pool, only the caller should close it. (#16)

### Fixed

- **Memory Bank alignment**: Restored ROADMAP items and softened `ListOpts` wording per review feedback. (#17)

## [v0.7.0] - 2026-05-23

### Added

- **PostgreSQL integration test suite**: 10 tests covering full CRUD (`Insert`, `InsertId`, `Get`, `List`, `ListRows`, `ListRange`, `Update`, `Delete`, `Set`, `Count`), `time.Time` fields, `SERIAL` auto-increment, and JOIN with composite structs. Opt-in via `SQLH_TEST_POSTGRES=1`. (#2)
- **PostgreSQL DDL generation**: `query.TablePG[T]()` generates PostgreSQL-compatible CREATE TABLE statements (`SERIAL`/`BIGSERIAL`, `bytea`, `boolean`, `double precision`). (#2)
- **Placeholder rebinding**: `query.Rebind()` converts `?` to PostgreSQL `$N` style. Automatically applied when a PostgreSQL driver is detected. (#2)
- **`SQLH_MYSQL_DSN` / `SQLH_POSTGRES_DSN`** environment variables allow connecting to existing MySQL/PostgreSQL instances instead of starting Docker containers from the tests. (#2)
- **CI matrix**: GitHub Actions workflow (`test.yml`) runs SQLite tests on every push/PR, plus opt-in MySQL and PostgreSQL jobs via service containers. (#2)

### Changed

- `isAutoIncrement` now detects `serial`, `bigserial`, and `smallserial` `db_type` tags, enabling PostgreSQL-style auto-increment alongside `autoincrement`/`auto_increment` in `db_key`. (#2)
- Database support claims in documentation updated: PostgreSQL is now listed as tested (opt-in); SQL Server is experimental. (#2)
- `cachedDialect` removed from global state; dialect detection is now per-call, fixing concurrency issues. (#11)

### Fixed

- **Driver detection for PostgreSQL** now matches `lib/pq` (`*pq.Driver`) and `pgx` (`*pgx.Driver`) driver types, not just strings containing `"postgres"`. (#2)

## [v0.6.1] - 2026-05-23

Patch release — no functional changes.

## [v0.6.0] - 2026-05-23

### Added

- `query.ArgsApply` — correctly spelled replacement for the misspelled `query.ArgsAppay`. Both functions have identical behaviour; new code should use `ArgsApply`. (#5)
- Unit tests for `isAutoIncrement` case-insensitivity, lock-error detection, `execRetries` behaviour, a 200-row `Update` batch regression test, and a compatibility test for the deprecated `ArgsAppay` alias.

### Deprecated

- `query.ArgsAppay` is now a thin wrapper around `query.ArgsApply` and is marked `Deprecated:` for removal in v1.0.0. Existing callers continue to work without changes. (#5)

### Performance

- `query.Args(row, false)` now uses the addressability of the struct to pass typed pointers directly to struct fields instead of boxing values into `interface{}` and copying. When the struct is addressable (passed by pointer, the common case through `QueryRange`), this eliminates all per-field heap allocations, reducing the read+apply pipeline from 13 to 4 allocs/op (−69 %) on the benchmark and to approximately 2 allocs/op in the production `QueryRange` code path. The non-addressable fallback (by-value struct) also benefits from the `reflect.Kind`-based dispatch in `ArgsApply`, which avoids an intermediate `interface{}` boxing step. (#6)

### Fixed

- `query.isAutoIncrement` now detects MySQL-style `AUTO_INCREMENT` tags. The previous implementation lower-cased the tag and then compared it against the literal `"AUTO_INCREMENT"`, which never matched, causing MySQL auto-increment columns to be included in INSERT/UPDATE column lists. (#4)
- `getLastInsertID` for PostgreSQL no longer references the hardcoded sequence name `table_name_id_seq`. The sequence is now resolved at runtime through `pg_get_serial_sequence` using the table name derived from the generic type. Tables with a non-`id` auto-increment column should use `InsertWithCallback` with an explicit `RETURNING` query. (#4)
- `execRetries` now detects transient "database is locked" / busy errors through substring matching and works with wrapped errors. The previous exact-string comparison missed wrapped errors and the `database table is locked` variant. The retry loop also exits immediately on any non-lock error. (#4)
- `Update` no longer accumulates `defer stmt.Close()` calls on the parent frame. Per-iteration work is extracted into `updateOne` so that each prepared statement is closed before the next attribute is processed, preventing handle exhaustion on large batches. (#4)

## [v0.5.1] - 2026-01-15

### Added

- `Custom Table Name`: Override auto-generated snake_case table name via `db_table_name` struct tag or `TableName()` method.
- CRUD example and documentation updates.

## [v0.5.0] - 2025-12-01

### Added

- **`Table[T]` Wrapper API**: Method-based API for all CRUD operations (`Insert`, `Get`, `List`, `Update`, `Delete`, `Set`, `Count`, `InsertId`).
- `CreateTable[T]()` convenience constructor for `Table[T]`.

## [v0.4.0] - 2025-11-15

### Added

- **Database lock retry**: Built-in `execRetries` with 20 attempts × 100ms delay for transient "database is locked" / busy errors.
- **Transactional `Get`**: `Get` wraps read in a transaction for consistency.
- **Metadata cache**: Struct reflection metadata cached by `reflect.Type` for repeated query generation and scan/apply operations.
- **JOIN support**: `MakeJoin[T]` and composite struct scanning for LEFT/RIGHT/INNER/OUTER JOINs.
- **Flexible SELECT**: `DISTINCT`, table aliases, custom table names.
- **`SetWheresJoinOr`**: OR-joining of WHERE conditions.
- **Expanded WHERE operators**: `IN`, `LIKE`, `IS NULL`, `IS NOT NULL`.

## [v0.2.2] - 2025-10-26

### Added

- The ListRange function created to use in range loops

### Changed

- Argument types expanded

## [v0.2.1] - 2025-06-23

### Fixed
- Fixed a bug in the `Set` function where a transaction was not properly closed, potentially causing database locks.
- Corrected the handling of boolean (`bool`) fields in `ArgsAppay`, which were previously misinterpreted as integers.

## [v0.2.0] - 2025-06-21

### Changed
- `INSERT` and `UPDATE` operations now automatically skip fields tagged with `autoincrement` in their `db_key`. This prevents errors when inserting records into tables with auto-generating primary keys.
- Refactored internal argument generation. The `query.Args` function now intelligently handles arguments for both read (`SELECT`) and write (`INSERT`/`UPDATE`) operations.

### Added
- New exported error `sqlh.ErrWhereClauseRequiredForUpdate` for better error handling in `Update` statements.
- Re-exported `sqlh.ErrTypeIsNotStruct` from the `query` package for easier access.
- New internal constants `forWrite` and `forRead` to improve readability when calling `query.Args`.

### Fixed
- Fixed a critical bug in `Delete` where operations were not correctly performed within a transaction due to using `db.Prepare` instead of `tx.Prepare`.
- The `Set` function is now atomic. It uses a transaction to prevent race conditions between checking for a record's existence and performing an `INSERT` or `UPDATE`.

## [v0.1.1] - 2025-06-21

### Added
- New generic function `sqlh.Update[T any]` to update records in the database based on specified conditions.

## [v0.1.0] - 2025-06-05

### Changed

- **Breaking Change:** The `Get` function now returns a pointer (`*T`) to the found struct instead of the struct value (`T`).
  - *Reason:* This provides a clearer way to indicate when a record was not found (by returning `nil`) compared to returning a zero-value struct.
  - *Impact:* Code calling `sqlh.Get` must be updated to expect a pointer (`*T`) and handle the `nil` case.

- **Breaking Change:** Error handling in the `Get` function has been standardized and improved.
  - `Get` now returns the standard `sql.ErrNoRows` error when no record is found. Previously, it returned `fmt.Errorf("not found")`.
  - Two new exported errors have been added: `sqlh.ErrWhereClauseRequired` (returned if `Get` is called without `Where` conditions) and `sqlh.ErrMultipleRowsFound` (returned if `Get` finds more than one record).
  - *Impact:* Code checking for specific errors from `sqlh.Get` (especially using `err.Error() == "not found"`) must be updated to use `errors.Is(err, sql.ErrNoRows)` and potentially check for the new exported errors.

### Added

- Exported errors `ErrWhereClauseRequired` and `ErrMultipleRowsFound` for specific error checking.
- Added a limit of 2 to the internal `ListRows` call within `Get` for minor performance optimization when checking for multiple rows.

### Fixed

- Corrected the return signature and logic of `Get` to consistently return `*T` or `nil` on error/not found.

[Unreleased]: https://github.com/kirill-scherba/sqlh/compare/v0.8.0...HEAD
[v0.8.0]: https://github.com/kirill-scherba/sqlh/compare/v0.7.1...v0.8.0
[v0.7.1]: https://github.com/kirill-scherba/sqlh/compare/v0.7.0...v0.7.1
[v0.7.0]: https://github.com/kirill-scherba/sqlh/compare/v0.6.1...v0.7.0
[v0.6.1]: https://github.com/kirill-scherba/sqlh/compare/v0.6.0...v0.6.1
[v0.6.0]: https://github.com/kirill-scherba/sqlh/compare/v0.5.1...v0.6.0
[v0.5.1]: https://github.com/kirill-scherba/sqlh/compare/v0.5.0...v0.5.1
[v0.5.0]: https://github.com/kirill-scherba/sqlh/compare/v0.4.0...v0.5.0
[v0.4.0]: https://github.com/kirill-scherba/sqlh/compare/v0.2.2...v0.4.0
[v0.2.2]: https://github.com/kirill-scherba/sqlh/compare/v0.2.1...v0.2.2
[v0.2.1]: https://github.com/kirill-scherba/sqlh/compare/v0.2.0...v0.2.1
[v0.2.0]: https://github.com/kirill-scherba/sqlh/compare/v0.1.1...v0.2.0
[v0.1.1]: https://github.com/kirill-scherba/sqlh/compare/v0.1.0...v0.1.1
[v0.1.0]: https://github.com/kirill-scherba/sqlh/releases/tag/v0.1.0
