# Progress: sqlh

## What Works

### Core CRUD Operations
- **Create**: `Create[T]()` generates and executes `CREATE TABLE IF NOT EXISTS` from struct tags — ✅ functional
- **Insert**: `Insert[T]()` inserts one or more rows with auto-transaction support — ✅ functional
- **InsertId**: `InsertId[T]()` returns the last inserted ID — ✅ functional
- **Get**: `Get[T]()` retrieves a single row with WHERE clause — ✅ functional
- **List**: `List[T]()` retrieves multiple rows with default page size and returns the next offset — ✅ functional
- **ListRows**: `ListRows[T]()` retrieves multiple rows with explicit limit/offset — ✅ functional
- **ListRange**: `ListRange[T]()` lazy iterator version returning `(index, row)` — ✅ functional
- **Update**: `Update[T]()` updates rows matching WHERE conditions — ✅ functional
- **Delete**: `Delete[T]()` deletes rows matching WHERE conditions — ✅ functional
- **Set**: `Set[T]()` upserts (SELECT-then-INSERT/UPDATE) — ✅ functional
- **Pagination**: `ListRows` and `ListRange` use explicit offset/limit arguments — ✅ functional

### Documentation & Examples
- **Example functions**: `sqlh_example_test.go` with `Example_` functions for pkg.go.dev — ✅ created
- **SKILL.md**: AI-assistant user guide with quick reference and important rules — ✅ created
- **examples/basic/**: Insert, Get, List, Update, Delete demo — ✅ created
- **examples/join/**: JOIN queries with nested structs — ✅ created
- **examples/paginator/**: Pagination with `ListRows` offset/limit — ✅ created
- **examples/set/**: Upsert via `Set` — ✅ created
- **examples/iterators/**: `ListRange` with Go 1.25 iterators — ✅ created
- **examples/context/**: Context cancellation with `ListRange` — ✅ created

### Query Generation
- **CREATE TABLE**: Auto-generates from struct tags — ✅ tested
- **INSERT**: Auto-generates with field list and VALUES placeholders — ✅ tested
- **SELECT**: Auto-generates with WHERE, ORDER BY, LIMIT, OFFSET — ✅ tested
- **UPDATE**: Auto-generates with SET clauses — ✅ tested
- **DELETE**: Auto-generates with WHERE — ✅ tested
- **Metadata cache**: Caches table names, field lists, autoincrement flags, and scan metadata by `reflect.Type` — ✅ tested on `feature/metadata_cache`

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
| MySQL tests require Docker/MySQL startup wait | Low | Not fixed |
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
| ListRows explicit pagination | ✅ Complete | v0.2.2 |
| Table wrapper API | ✅ Complete | v0.5.0 |
| SKILL.md (AI-assistant user guide) | ✅ Complete | — |
| Examples (basic, join, paginator, set, iterators, context) | ✅ Complete | — |
| Metadata cache | 🚧 In progress | feature/metadata_cache |
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
- **MySQL Tests**: ⚠️ Requires Docker/MySQL startup wait
- **Documentation**: CHANGELOG.md, README.md, ROADMAP.md, SKILL.md present
- **Examples**: 6 runnable programs in examples/ directory
- **Backward Compatibility**: Public API changes limited; deprecated items tracked in CHANGELOG

## Performance Baseline

Benchmarks added on `feature/metadata_cache` provide a baseline for reflection metadata caching and scan/apply overhead:

| Benchmark | Time | Allocated | Allocs |
|-----------|------|-----------|--------|
| `BenchmarkArgsWrite` | ~196 ns/op | 224 B/op | 2 allocs/op |
| `BenchmarkArgsReadAndApply` | ~818 ns/op | 416 B/op | 13 allocs/op |
| `BenchmarkSelect` | ~1.15 us/op | 528 B/op | 13 allocs/op |
| `BenchmarkListRows` | ~46.7 us/op | 8465 B/op | 314 allocs/op |

Interpretation: metadata lookup and write argument generation are cheap. The next performance target is the read scan/apply pipeline, especially allocations in `Args(row, false)`, `ArgsAppay`, `QueryRange`, and `ListRows`. Future benchmarks should compare sqlh read paths against manual `rows.Scan` and add JOIN/composite wrapper coverage.

## Next Milestones

1. **Current branch target**: Merge metadata cache after compatibility review
2. **Next target**: Documentation/API consistency and advanced WHERE helpers
3. **v1.0.0 target**: Stable API with schema management and full database compatibility

## Release History

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
