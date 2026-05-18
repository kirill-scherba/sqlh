# Active Context: sqlh

## Current State

The project is at version **v0.5.1** on `main`, with active work on `feature/metadata_cache`. It provides a functional CRUD abstraction over SQL databases with auto-generated queries, automatic transaction management, database lock retry logic, a `Table[T]` wrapper, examples, and cached struct metadata for reflection-heavy paths.

## Recent Changes (feature/metadata_cache)

- **Added**: `query` metadata cache keyed by `reflect.Type`
- **Fixed**: Metadata cache compatibility for named composite JOIN wrapper structs
- **Fixed**: `time.Time` fields are treated as ordinary columns, not composite JOIN structs
- **Added**: Unit tests for metadata cache hits, same-name types in different packages, composite projection compatibility, and `time.Time` handling

## Recent Changes (v0.5.1)

- **Added**: `examples/crud/` runnable example
- **Added**: Custom table name documentation, examples, and pkg.go.dev examples
- **Added**: Memory Bank documentation in `docs/`
- **Changed**: `Table[T].Close()` is a no-op so it does not close shared `*sql.DB` pools

## Recent Changes (v0.5.0)

- **Added**: `CreateTable[T]` and the `Table[T]` wrapper API
- **Added**: `Table[T].Insert`, `InsertId`, `Update`, `Set`, `Get`, `Delete`, `Count`, and iterator-style `List`

## Recent Changes (v0.4.0)

- **Added**: Database lock retry logic
- **Changed**: `Get` uses a transaction

## Earlier Changes (v0.2.2)

- **Added**: `ListRange` function using Go 1.25 iterators for lazy row iteration in range loops
- **Changed**: Expanded argument types for broader compatibility

## Recent Changes (v0.2.1)

- **Fixed**: Bug in `Set` function where transaction was not properly closed, potentially causing database locks
- **Fixed**: Boolean (`bool`) fields in `ArgsAppay` were misinterpreted as integers

## Recent Changes (v0.2.0)

- **Changed**: `INSERT` and `UPDATE` now skip `autoincrement` fields
- **Added**: `ErrWhereClauseRequiredForUpdate` exported error
- **Added**: Internal `forWrite`/`forRead` constants for `query.Args`
- **Fixed**: `Delete` operations now correctly use `tx.Prepare` instead of `db.Prepare`
- **Fixed**: `Set` is now atomic (uses transaction for SELECT + INSERT/UPDATE)

## Active Development Focus

Based on the ROADMAP, current priorities are:

### Phase 1: Core Query Enhancements (HIGH priority)
1. **Documentation alignment**: Keep README, Memory Bank, and assistant skills synchronized with the current API
2. **Metadata cache stabilization**: Finish compatibility tests and performance checks for cached reflection metadata
3. **Advanced WHERE conditions**: `IN` operator, improved `LIKE`, `IS NULL`/`IS NOT NULL`
4. **Context propagation**: Add `context.Context` to write operations without breaking existing API

### Phase 2: Advanced Features & Data Integrity (MEDIUM priority)
1. **JOIN support**: Improve ergonomics around composite struct scanning
2. **Native UPSERT**: Replace `Set` with `ON CONFLICT DO UPDATE`
3. **Aggregate functions**: GROUP BY, HAVING, SUM, AVG, MIN, MAX

### Phase 3: Schema Management (LOW priority)
1. **Schema migrations**: ALTER TABLE support
2. **CREATE INDEX generation**

### Phase 4: Developer Experience (LOW priority)
1. **Raw SQL fragments**: Allow raw SQL injection into generated queries
2. **Transactional reads**: Support `*sql.Tx` in Get/List

## Known Issues

1. **Database lock retry**: Uses string matching (`"database is locked"`) which is fragile and driver-specific
2. **MySQL test dependency**: `sqlh_mysql_test.go` starts or reuses a Docker MySQL container and may require a long startup wait
3. **Context support**: Partially implemented; some functions accept `context.Context` but not all
4. **JOIN support**: Basic; requires manual composite struct setup
5. **Last insert ID**: The `getLastInsertID` function has a PostgreSQL query that hardcodes the sequence name (`table_name_id_seq`), which will fail for tables with custom sequences or non-standard naming

## Current Design Decisions Being Evaluated

1. How to implement context propagation across all functions without breaking existing API
2. Whether to use a builder pattern for complex queries (similar to GORM's chainable API) or continue with attribute-based configuration
3. How to implement native UPSERT with database-specific SQL syntax abstraction

## Next Steps

1. **Short-term**: Keep documentation and assistant skills synchronized with current signatures
2. **Short-term**: Complete metadata cache review and merge `feature/metadata_cache`
3. **Short-term**: Add `IN` operator and explicit NULL helpers
4. **Medium-term**: Implement native UPSERT using `ON CONFLICT DO UPDATE`
5. **Long-term**: Improve JOIN support with clearer composite struct conventions

## Testing Status

- SQLite tests: ✅ Passing (primary test suite)
- MySQL tests: ⚠️ Requires Docker/MySQL startup wait
- Query generation tests: ✅ Passing
- Table wrapper tests: ✅ Passing

## Release Cadence

- v0.1.0: Initial release (2025-06-05)
- v0.1.1: Added Update function (2025-06-21)
- v0.2.0: Atomic operations, autoincrement fix (2025-06-21)
- v0.2.1: Transaction close fix, bool handling fix (2025-06-23)
- v0.2.2: ListRange iterator (2025-10-26)
- v0.4.0: Database lock retry and transactional Get
- v0.5.0: `Table[T]` wrapper API
- v0.5.1: CRUD example and documentation updates
