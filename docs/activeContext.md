# Active Context: sqlh

## Current State

The project is at version **v0.7.1** on `main`. All features for **v0.8.0** are
merged and awaiting the final annotated git tag and GitHub release. No active
development branches are in flight.

## Recent Changes (v0.8.0 — 2026-06-12)

### Type-safe WHERE Helpers (issue #14)
- **Added**: `sqlh.Eq`, `sqlh.Ne`, `sqlh.Gt`, `sqlh.Gte`, `sqlh.Lt`, `sqlh.Lte`,
  `sqlh.Like`, `sqlh.In`, `sqlh.IsNull`, `sqlh.IsNotNull` — fluent helpers
  for building WHERE conditions without raw SQL strings.
- **Fixed**: Backward compatibility restored for exported `query.Update` and
  `query.Delete` wrappers. Fixed missing space before `?` in `processWhere`.

### Documentation & Examples (issues #25, #26, #27, #28, #29)
- **Animated GIF demo**: `docs/demo.gif` + `examples/demo/` for README. (#25)
- **Code comparison**: `examples/comparison/` side-by-side raw sql / sqlx / sqlh. (#26)
- **pkg.go.dev badge**: Official Go Reference badge replacing deprecated godoc.org. (#27)
- **Benchmarks**: 24 benchmark functions in `bench/` vs raw sql, sqlx, GORM. (#28)
- **Table[T] examples**: `Example_` functions in `sqlh_example_test.go` for pkg.go.dev. (#29)

## Prior Releases

### v0.7.1 — 2026-06-11
- **Native UPSERT**: PostgreSQL (`ON CONFLICT DO UPDATE`), SQLite (`ON CONFLICT DO
  UPDATE`), MySQL (`ON DUPLICATE KEY UPDATE`) with fallback to SELECT-then-INSERT/UPDATE.
- **`query.Fields[T]()`**: Exported public field name access.
- **`db_table_name` ergonomics**: `any`/`string`/`bool` sentinel types all valid.
- **List API guidance**: `docs/list-api-guidance.md` clarifies `List`/`ListRows`/
  `ListRange`/`QueryRange` roles.
- **SQL Server docs**: Marked experimental/partial; no CI.

### v0.7.0 — 2026-05-23
- **PostgreSQL integration tests**: 10 tests, opt-in via `SQLH_TEST_POSTGRES=1`.
- **PostgreSQL DDL**: `query.TablePG[T]()` with `SERIAL`/`BIGSERIAL`, `bytea`, etc.
- **Placeholder rebinding**: `?` → `$N` for PostgreSQL drivers.
- **CI matrix**: GitHub Actions with MySQL + PostgreSQL service containers.
- **Concurrency fix**: `cachedDialect` removed from global state.

### v0.6.0 / v0.6.1 — 2026-05-23
- **Critical bug fixes**: MySQL `AUTO_INCREMENT`, PostgreSQL `pg_get_serial_sequence`,
  lock-retry robustness, `Update` statement-leak.
- **API hygiene**: `ArgsAppay` → `ArgsApply` with deprecation alias.
- **Performance**: Zero-alloc read path via addressable struct pointers
  (4 allocs/op, −69 % vs prior).
- **Metadata cache**: Struct reflection cached by `reflect.Type`.

## Active Development Focus

### Completed

1. ✅ Critical bug fixes — Stage 1 (v0.6.0)
2. ✅ ArgsAppay → ArgsApply rename — Stage 2 (v0.6.0)
3. ✅ Zero-alloc read path — Stage 3 (v0.6.0)
4. ✅ PostgreSQL support — Stage 4 (v0.7.0)
5. ✅ Native UPSERT — Stage 5 (v0.7.1)
6. ✅ Type-safe WHERE helpers — Stage 6 (v0.8.0)
7. ✅ Benchmarks, comparisons, animated GIF — Stage 6 (v0.8.0)

### Remaining Short-term items

1. **Tag v0.8.0**: Create annotated git tag and GitHub release notes

### Medium-term (Phase 2)

1. **Aggregate functions**: GROUP BY, HAVING, SUM, AVG, MIN, MAX
2. **Schema migrations**: ALTER TABLE support
3. **Batch operations**: Multi-row insert/update in a single query

## Known Issues

| # | Issue | Severity | Status |
|---|-------|----------|--------|
| 1 | Context support partially implemented in write paths | Medium | Known |
| 2 | JOIN composite struct setup requires manual convention | Low | Known |
| 3 | Lock-retry `isLockError` still uses string matching | Low | ✅ Mitigated |
| 4 | SQL Server support is experimental, no CI | Medium | Known |

## Current Design Decisions Being Evaluated

1. ✅ **Issue #17 resolved** — List API guidance clarified in v0.7.1.
2. Whether to use a builder pattern for complex queries (similar to GORM's
   chainable API) or continue with attribute-based configuration
3. How to implement aggregate functions with database-specific SQL syntax
4. Memory Bank file naming — current names differ from AGENTS.md convention.
   Rename requires discussion due to external link impact.

## Next Milestones

1. **Short-term**: Tag v0.8.0 and publish GitHub release notes
2. **Medium-term**: Aggregate functions, schema migrations, batch operations
3. **v1.0.0**: Stable API with full database compatibility

## Testing Status

- SQLite tests: ✅ Passing (primary test suite)
- MySQL tests: ✅ Gated behind `SQLH_MYSQL_TEST=1`
- PostgreSQL tests: ✅ Gated behind `SQLH_TEST_POSTGRES=1`
- Query generation tests: ✅ Passing
- Table wrapper tests: ✅ Passing
- Metadata cache tests: ✅ Passing
- Retry logic tests: ✅ Passing
- Batch Update test (200 rows): ✅ Passing

## Release Cadence

| Version | Date | Highlights |
|---------|------|------------|
| v0.1.0 | 2025-06-05 | Initial release |
| v0.1.1 | 2025-06-21 | Added Update function |
| v0.2.0 | 2025-06-21 | Atomic operations, autoincrement fix |
| v0.2.1 | 2025-06-23 | Transaction close fix, bool handling fix |
| v0.2.2 | 2025-10-26 | ListRange iterator, expanded arg types |
| v0.4.0 | 2025-11-15 | Lock retry, metadata cache, JOIN, flexible SELECT, advanced WHERE |
| v0.5.0 | 2025-12-01 | `Table[T]` wrapper API |
| v0.5.1 | 2026-01-15 | Custom table name, CRUD example updates |
| v0.6.0 | 2026-05-23 | Critical fixes, ArgsApply rename, zero-alloc read path, metadata cache |
| v0.6.1 | 2026-05-23 | Patch (no functional changes) |
| v0.7.0 | 2026-05-23 | PostgreSQL integration tests, PG DDL, placeholder rebinding, CI matrix |
| v0.7.1 | 2026-06-11 | Native UPSERT, Fields[T](), List API guidance, SQL Server docs |
| v0.8.0 | 2026-06-12 | Type-safe WHERE helpers, benchmarks, comparisons, animated GIF demo |
