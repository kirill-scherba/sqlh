# Progress: sqlh

## What Works

### Core CRUD Operations
- **Create**: `Create[T]()` generates and executes `CREATE TABLE IF NOT EXISTS` from struct tags — ✅ functional
- **Insert**: `Insert[T]()` inserts one or more rows with auto-transaction support — ✅ functional
- **InsertId**: `InsertId[T]()` returns the last inserted ID — ✅ functional
- **Get**: `Get[T]()` retrieves a single row with WHERE clause — ✅ functional
- **List**: `List[T]()` retrieves multiple rows with pagination, ordering, and WHERE — ✅ functional
- **ListRange**: `ListRange[T]()` lazy iterator version of List — ✅ functional (v0.2.2)
- **Update**: `Update[T]()` updates rows matching WHERE conditions — ✅ functional
- **Delete**: `Delete[T]()` deletes rows matching WHERE conditions — ✅ functional
- **Set**: `Set[T]()` upserts (SELECT-then-INSERT/UPDATE) — ✅ functional

### Query Generation
- **CREATE TABLE**: Auto-generates from struct tags — ✅ tested
- **INSERT**: Auto-generates with field list and VALUES placeholders — ✅ tested
- **SELECT**: Auto-generates with WHERE, ORDER BY, LIMIT, OFFSET — ✅ tested
- **UPDATE**: Auto-generates with SET clauses — ✅ tested
- **DELETE**: Auto-generates with WHERE — ✅ tested

### Transaction Management
- All write operations auto-wrapped in transactions with rollback on error — ✅ functional
- Atomic upsert in `Set` (SELECT + INSERT/UPDATE in single transaction) — ✅ fixed in v0.2.0

### Database Lock Retry
- `execRetries` with 20 attempts × 100ms delay — ✅ functional
- Three execution layers (`execDb`, `execStmt`, `execTx`) all use retry — ✅ functional

### Struct Tag Support
- `db` tag for column name — ✅ functional
- `db_type` tag for SQL type override — ✅ functional
- `db_key` tag for constraints (`primary key`, `autoincrement`, `unique`, `not null`, etc.) — ✅ functional
- Nested struct support for JOINs — ✅ functional

### Table Wrapper API
- Generic `Table[T]` struct with method-based API — ✅ functional

### Database Abstraction
- SQLite driver detection and compatibility — ✅ tested
- MySQL driver detection and compatibility — ✅ tested (with external instance)
- PostgreSQL/SQL Server `last_insert_rowid` detection — ✅ partial

## What's In Progress

### Phase 1: Core Query Enhancements (HIGH)
- ❌ **Flexible SELECT queries**: Column-specific SELECT instead of `SELECT *`
- ❌ **Advanced WHERE conditions**: `OR` operator, `IN` operator, improved `LIKE`, `IS NULL`/`IS NOT NULL`
- ❌ **Context propagation**: `context.Context` to all database query functions

## What's Planned

### Phase 2: Advanced Features & Data Integrity (MEDIUM)
- ❌ **JOIN support**: LEFT JOIN with composite struct scanning
- ❌ **Native UPSERT**: `ON CONFLICT DO UPDATE`
- ❌ **Aggregate functions**: GROUP BY, HAVING, SUM, AVG, MIN, MAX

### Phase 3: Schema Management (LOW)
- ❌ **Schema migrations**: ALTER TABLE support
- ❌ **CREATE INDEX generation**

### Phase 4: Developer Experience (LOW)
- ❌ **Raw SQL fragments**: Allow raw SQL injection into generated queries
- ❌ **Transactional reads**: Support `*sql.Tx` in Get/List

## Known Issues

| Issue | Severity | Status |
|-------|----------|--------|
| Database lock retry uses fragile string matching | Medium | Not fixed |
| MySQL tests require external MySQL instance | Low | Not fixed |
| Context support partially implemented | Medium | Partially done |
| JOIN support requires manual composite struct setup | Low | Not fixed |
| PostgreSQL `last_insert_rowid` hardcodes sequence name | Medium | Not fixed |

## Feature Completeness

| Feature | Status | Version |
|---------|--------|---------|
| Basic CRUD (Insert, Get, List, Update, Delete) | ✅ Complete | v0.1.0 |
| Set (upsert) | ✅ Complete | v0.1.0 |
| Transaction auto-wrap | ✅ Complete | v0.1.0 |
| Database lock retry | ✅ Complete | v0.1.0 |
| Struct tag mapping | ✅ Complete | v0.1.0 |
| Table wrapper API | ✅ Complete | v0.1.0 |
| Create table from struct | ✅ Complete | v0.1.0 |
| InsertId (return inserted ID) | ✅ Complete | v0.1.0 |
| Autoincrement field detection | ✅ Complete | v0.2.0 |
| ErrWhereClauseRequiredForUpdate | ✅ Complete | v0.2.0 |
| Set atomicity fix | ✅ Complete | v0.2.0 |
| Delete uses tx.Prepare | ✅ Complete | v0.2.0 |
| Transaction close fix | ✅ Complete | v0.2.1 |
| Bool field scanning fix | ✅ Complete | v0.2.1 |
| ListRange (Go 1.25 iterator) | ✅ Complete | v0.2.2 |
| Expanded arg types | ✅ Complete | v0.2.2 |
| Context propagation to all functions | ❌ Not started | — |
| Advanced WHERE (OR, IN, LIKE, IS NULL) | ❌ Not started | — |
| Flexible SELECT columns | ❌ Not started | — |
| Native UPSERT (ON CONFLICT DO UPDATE) | ❌ Not started | — |
| JOIN support (composite struct scanning) | ❌ Not started | — |
| Aggregate functions | ❌ Not started | — |
| Schema migrations | ❌ Not started | — |
| Raw SQL fragments | ❌ Not started | — |
| Transactional reads | ❌ Not started | — |

## Quality Metrics

- **Test Coverage**: SQLite tests pass, Query generation tests pass, Table wrapper tests pass
- **MySQL Tests**: ⚠️ Requires manual external setup
- **Documentation**: CHANGELOG.md, README.md, ROADMAP.md present
- **Backward Compatibility**: Public API changes limited; deprecated items tracked in CHANGELOG

## Next Milestones

1. **v0.3.0 target**: Core query enhancements (context propagation, advanced WHERE, flexible SELECT)
2. **v0.4.0 target**: Advanced features (JOIN, UPSERT, aggregates)
3. **v1.0.0 target**: Stable API with schema management and full database compatibility

## Release History

| Version | Date | Highlights |
|---------|------|------------|
| v0.1.0 | 2025-06-05 | Initial release |
| v0.1.1 | 2025-06-21 | Added Update function |
| v0.2.0 | 2025-06-21 | Atomic operations, autoincrement fix |
| v0.2.1 | 2025-06-23 | Transaction close fix, bool handling fix |
| v0.2.2 | 2025-10-26 | ListRange iterator, expanded arg types |