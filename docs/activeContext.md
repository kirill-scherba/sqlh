# Active Context: sqlh

## Current State

The project is at version **v0.5.1** on `main`, with active development on
`feature/metadata_cache` and three stacked fix branches stacked on top:

| Branch | Stage | Status | 
|--------|-------|--------|
| `feature/metadata_cache` | — | Base (merged-ready) |
| `fix/critical-bugs` | 1 | ✅ Under review |
| `refactor/api-cleanup` | 2 | ✅ Under review |
| `perf/args-allocs` | 3 | ✅ Under review |
| `docs/sync-status` | 4 | 🚧 This branch |

## Recent Changes (Stage 1–3, 2026-05-22)

### Stage 1: Critical Bugs (`fix/critical-bugs`)

- **Fixed**: `query.isAutoIncrement` now detects MySQL-style `AUTO_INCREMENT`
  tags. The previous double-lower-case logic never matched `AUTO_INCREMENT`.
- **Fixed**: `getLastInsertID` for PostgreSQL no longer hardcodes the sequence
  name `table_name_id_seq`. The sequence is resolved at runtime through
  `pg_get_serial_sequence(\$1, 'id')`.
- **Fixed**: `execRetries` now detects transient database-lock errors through
  `isLockError()` with substring matching, working with wrapped errors.
  The previous exact-string match missed wrapped errors and the
  `database table is locked` variant.
- **Fixed**: `Update` no longer accumulates `defer stmt.Close()` calls on the
  parent frame. Per-iteration work is extracted into `updateOne` so each
  prepared statement is closed before the next attribute is processed.

### Stage 2: API Hygiene (`refactor/api-cleanup`)

- **Added**: `query.ArgsApply` — correctly spelled replacement for the
  long-standing misspelling `query.ArgsAppay`.
- **Deprecated**: `query.ArgsAppay` is now a thin wrapper marked for removal in
  v1.0.0. All internal callers migrated to `ArgsApply`.

### Stage 3: Performance (`perf/args-allocs`)

- **Optimised**: `query.Args(row, false)` now uses the addressability of the
  struct to pass typed pointers directly to struct fields instead of boxing
  values into `interface{}` and copying. The addressable fast path reduces
  the read+apply pipeline from 13 allocs/op to 4 allocs/op (—69 %), and to
  approximately 2 allocs/op in the production `QueryRange` loop.
- **Optimised**: `query.ArgsApply` rewritten to dispatch on
  `reflect.Value.Kind()` instead of extracting values via `.Interface()` for a
  type switch, avoiding heap boxing for `string`, `time.Time`, and `[]byte`
  values. Benefits both the addressable and non-addressable paths.

## Recent Changes (feature/metadata_cache)

- **Added**: `query` metadata cache keyed by `reflect.Type`
- **Fixed**: Metadata cache compatibility for named composite JOIN wrapper
  structs
- **Fixed**: `time.Time` fields are treated as ordinary columns, not composite
  JOIN structs
- **Added**: Unit tests for metadata cache hits, same-name types in different
  packages, composite projection compatibility, and `time.Time` handling

## Active Development Focus

### Completed (Stages 1–3)

1. ✅ **Critical bug fixes** — isAutoIncrement case-folding, PostgreSQL
   getLastInsertID, lock-retry robustness, Update statement-leak
2. ✅ **ArgsAppay → ArgsApply rename** with deprecation alias
3. ✅ **Zero-alloc read path** — addressable structs skip per-field heap
   allocations in the Args scan/apply pipeline

### Remaining Short-term items

1. **Documentation alignment** (this branch): Keep Memory Bank synchronised
   with current API and implementation state
2. **Merge stage branches**: Land `fix/critical-bugs`, `refactor/api-cleanup`,
   `perf/args-allocs` into `feature/metadata_cache`
3. **MySQL test stability**: Gate MySQL tests behind `SQLH_MYSQL_TEST` env var
   to avoid constant failures on CI

### Medium-term (Phase 2)

1. **Native UPSERT**: Replace `Set` with `ON CONFLICT DO UPDATE`
2. **Aggregate functions**: GROUP BY, HAVING, SUM, AVG, MIN, MAX
3. **Advanced WHERE helpers**: Dedicated `IN` operator, explicit NULL clauses

## Known Issues

| # | Issue | Severity | Status |
|---|-------|----------|--------|
| 1 | MySQL test starts Docker container unconditionally | Low | ✅ Fixed (gated behind `SQLH_MYSQL_TEST`, Stage 5) |
| 2 | Context support partially implemented in write paths | Medium | Known |
| 3 | JOIN composite struct setup requires manual convention | Low | Known |
| 4 | Lock-retry `isLockError` still uses string matching | Medium | ✅ Fixed |
| 5 | PostgreSQL `getLastInsertID` no longer hardcodes sequence | Medium | ✅ Fixed |
| 6 | No native UPSERT (Set uses SELECT-then-INSERT/UPDATE) | Medium | Planned |

## Current Design Decisions Being Evaluated

1. Whether to use a builder pattern for complex queries (similar to GORM's
   chainable API) or continue with attribute-based configuration
2. How to implement native UPSERT with database-specific SQL syntax abstraction
3. Memory Bank file naming — current names (`activeContext.md`, `progress.md`,
   etc.) differ from the AGENTS.md convention (`CONTEXT.md`, `STATUS.md`,
   `DESIGN.md`). Rename requires discussion due to external link impact.

## Next Milestones

1. **Short-term**: Merge `docs/sync-status` → `perf/args-allocs`
2. **Short-term**: Merge `fix/critical-bugs` → `feature/metadata_cache`
3. **Short-term**: Merge `refactor/api-cleanup` → `feature/metadata_cache`
4. **Short-term**: Merge `perf/args-allocs` → `feature/metadata_cache`
5. **Medium-term**: MySQL test gate (Stage 5)
6. **Long-term**: Native UPSERT, aggregate functions, schema management

## Testing Status

- SQLite tests: ✅ Passing (primary test suite)
- MySQL tests: ✅ Gated behind `SQLH_MYSQL_TEST=1`; runs against Docker container
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
| v0.4.0 | — | Database lock retry, transactional Get |
| v0.5.0 | — | `Table[T]` wrapper API |
| v0.5.1 | — | CRUD example and documentation updates |
