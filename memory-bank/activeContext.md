# Active Context: sqlh

## Current State

The project is at version **v0.2.2**, released 2025-10-26. It provides a functional CRUD abstraction over SQL databases with auto-generated queries, automatic transaction management, and database lock retry logic.

## Recent Changes (v0.2.2)

- **Added**: `ListRange` function using Go 1.25 iterators (`iter.Seq2[T, error]`) for lazy row iteration in range loops
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
1. **Flexible SELECT queries**: Column-specific SELECT instead of `SELECT *`
2. **Advanced WHERE conditions**: `OR` operator, `IN` operator, improved `LIKE`, `IS NULL`/`IS NOT NULL`
3. **Context propagation**: Add `context.Context` to all database query functions

### Phase 2: Advanced Features & Data Integrity (MEDIUM priority)
1. **JOIN support**: LEFT JOIN with composite struct scanning
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
2. **MySQL test dependency**: `sqlh_mysql_test.go` requires a running MySQL instance, not easily run in CI
3. **Context support**: Partially implemented; some functions accept `context.Context` but not all
4. **JOIN support**: Basic; requires manual composite struct setup
5. **Last insert ID**: The `getLastInsertID` function has a PostgreSQL query that hardcodes the sequence name (`table_name_id_seq`), which will fail for tables with custom sequences or non-standard naming

## Current Design Decisions Being Evaluated

1. How to implement context propagation across all functions without breaking existing API
2. Whether to use a builder pattern for complex queries (similar to GORM's chainable API) or continue with attribute-based configuration
3. How to implement native UPSERT with database-specific SQL syntax abstraction

## Next Steps

1. **Short-term**: Implement full `context.Context` propagation across `Get`, `List`, `Insert`, `Update`, `Delete`, `Set` functions
2. **Short-term**: Add `OR` WHERE clause support and `IN` operator
3. **Medium-term**: Implement native UPSERT using `ON CONFLICT DO UPDATE`
4. **Medium-term**: Add SELECT specific column support
5. **Long-term**: Implement JOIN support with composite struct auto-scanning

## Testing Status

- SQLite tests: ✅ Passing (primary test suite)
- MySQL tests: ⚠️ Requires external MySQL instance
- Query generation tests: ✅ Passing
- Table wrapper tests: ✅ Passing

## Release Cadence

- v0.1.0: Initial release (2025-06-05)
- v0.1.1: Added Update function (2025-06-21)
- v0.2.0: Atomic operations, autoincrement fix (2025-06-21)
- v0.2.1: Transaction close fix, bool handling fix (2025-06-23)
- v0.2.2: ListRange iterator (2025-10-26)