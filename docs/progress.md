# Progress: sqlh

## What Works

### Core CRUD Operations
- **Create**: `Create[T]()` generates and executes `CREATE TABLE IF NOT EXISTS`
  from struct tags — ✅ functional
- **Insert**: `Insert[T]()` inserts one or more rows with auto-transaction
  support — ✅ functional
- **InsertId**: `InsertId[T]()` returns the last inserted ID — ✅ functional
- **Get**: `Get[T]()` retrieves a single row with WHERE clause — ✅ functional
- **List**: `List[T]()` retrieves multiple rows with default page size and
  returns the next offset — ✅ functional
- **ListRows**: `ListRows[T]()` retrieves multiple rows with explicit
  limit/offset — ✅ functional
- **ListRange**: `ListRange[T]()` lazy iterator version returning
  `(index, row)` — ✅ functional
- **Update**: `Update[T]()` updates rows matching WHERE conditions — ✅ functional
- **Delete**: `Delete[T]()` deletes rows matching WHERE conditions — ✅ functional
- **Set**: `Set[T]()` native database upsert — ✅ functional
- **Pagination**: `ListRows` and `ListRange` use explicit offset/limit
  arguments — ✅ functional

### Advanced WHERE Conditions
- **OR operator**: `SetWheresJoinOr()` enables OR-joining of WHERE conditions —
  ✅ functional
- **IN operator**: `sqlh.In("id", ...)` supports list parameters with automatic
  `?` placeholder expansion — ✅ functional
- **LIKE / IS NULL / IS NOT NULL**: `sqlh.Like("name", "%foo%")`,
  `sqlh.IsNull("deleted_at")`, `sqlh.IsNotNull("created_at")` — ✅ functional
- **Type-safe WHERE helpers**: `Eq`, `Ne`, `Gt`, `Gte`, `Lt`, `Lte` constructors
  — ✅ functional

### Query Generation
- **CREATE TABLE**: Auto-generates from struct tags — ✅ tested
- **INSERT**: Auto-generates with field list and VALUES placeholders — ✅ tested
- **SELECT**: Auto-generates with WHERE, ORDER BY, LIMIT, OFFSET — ✅ tested
- **SELECT with DISTINCT**: `SetDistinct()` — ✅ tested
- **SELECT with Alias**: `SetAlias("t")` for table aliases — ✅ tested
- **SELECT with custom Name**: `SetName("custom_table")` — ✅ tested
- **UPDATE**: Auto-generates with SET clauses — ✅ tested
- **DELETE**: Auto-generates with WHERE — ✅ tested
- **Metadata cache**: Caches table names, field lists, autoincrement flags, and
  scan metadata by `reflect.Type` — ✅ tested

### JOIN Support
- **MakeJoin[T]**: Construct JOIN attributes from a struct type — ✅ functional
- **LEFT / RIGHT / INNER / OUTER**: All join types support — ✅ functional
- **QueryRange with composite structs**: Scan results into composite wrappers
  (`struct{ *MainTable; *JoinedTable }`) — ✅ functional

### Transaction Management
- All write operations auto-wrapped in transactions with rollback on error —
  ✅ functional
- Atomic upsert in `Set` (SELECT + INSERT/UPDATE in single transaction) —
  ✅ fixed in v0.2.0

### Database Lock Retry
- `execRetries` with 20 attempts × 100ms delay — ✅ functional
- `isLockError` detects `database is locked`, `database table is locked`,
  and `SQLITE_BUSY` using substring matching — ✅ functional
- Works with wrapped errors (unlike the previous exact-string match) —
  ✅ fixed in Stage 1
- Three execution layers (`execDb`, `execStmt`, `execTx`) all use retry —
  ✅ functional

### Struct Tag Support
- `db` tag for column name — ✅ functional
- `db_type` tag for SQL type override — ✅ functional
- `db_key` tag for constraints (`primary key`, `autoincrement`, `unique`,
  `not null`, etc.) — ✅ functional
- `db_table_name` tag for custom table name override (any `_` sentinel type: `any`, `string`, `bool`, etc.) — ✅ functional
- `TableName()` interface for dynamic table name resolution — ✅ functional
- Nested struct support for JOINs — ✅ functional

### Table Wrapper API
- Generic `Table[T]` struct with method-based API (`Insert`, `Get`, `List`,
  `Update`, `Delete`, `Set`, `Count`, `InsertId`) — ✅ functional
- `CreateTable[T]()` convenience constructor — ✅ functional
- `Close()` is a backward-compatible no-op — `Table[T]` does not own the `*sql.DB` pool, only the caller should close it — ✅ functional (not recommended for teaching)

### Database Abstraction
- SQLite driver detection and compatibility — ✅ tested
- MySQL driver detection and compatibility — ✅ tested (with external instance)
- PostgreSQL driver detection and compatibility — ✅ tested (opt-in via `SQLH_TEST_POSTGRES=1`)
  - Full CRUD integration suite (10 tests)
  - `SERIAL`/`BIGSERIAL` auto-increment support
  - Automatic `?` → `$N` placeholder rebinding
  - PG-compatible DDL generation
- SQL Server `SCOPE_IDENTITY` — ✅ partial (driver detection and last-insert-ID only; no CRUD tests, no CI; not production-ready)

### Performance Optimisations
- **Metadata cache**: Struct reflection cached by `reflect.Type` — ✅ complete
- **Zero-alloc read path**: Addressable structs (via pointer) in `Args(row,
  false)` use direct field pointers instead of boxing + copy, eliminating
  per-field heap allocations — ✅ complete (Stage 3)

### Documentation & Examples
- **Memory Bank**: `docs/activeContext.md`, `docs/progress.md`,
  `docs/systemPatterns.md`, `docs/techContext.md`, `docs/productContext.md`,
  `docs/projectbrief.md`, `docs/list-api-guidance.md` — ✅ created
- **Example functions**: `sqlh_example_test.go` with `Example_` functions for
  pkg.go.dev — ✅ created
- **SKILL.md**: AI-assistant user guide with quick reference and important
  rules — ✅ created
- **examples/demo/**: Self-contained CRUD demo for animated GIF recording — ✅ created
- **examples/basic/**: Insert, Get, List, Update, Delete demo — ✅ created
- **examples/join/**: JOIN queries with nested structs — ✅ created
- **examples/paginator/**: Pagination with `ListRows` offset/limit — ✅ created
- **examples/set/**: Upsert via `Set` — ✅ created
- **examples/iterators/**: `ListRange` with Go 1.25 iterators — ✅ created
- **examples/context/**: Context cancellation with `ListRange` — ✅ created
- **examples/crud/**: Full CRUD workflow example — ✅ created
- **examples/comparison/**: Side-by-side CRUD comparison of raw `database/sql`, `sqlx`, and `sqlh` — ✅ created (issue #26)
- **bench/**: Comparative performance benchmarks (24 functions: raw sql, sqlx, GORM, sqlh) — ✅ created (issue #28)
- **docs/demo.gif**: Animated terminal recording for README — ✅ created

## What's Planned

### Phase 2: Advanced Features & Data Integrity (MEDIUM)
- ✅ **Native UPSERT**: PostgreSQL (`ON CONFLICT DO UPDATE`), SQLite
  (`ON CONFLICT DO UPDATE`), MySQL (`ON DUPLICATE KEY UPDATE`) — implemented
  in `Set[T]()` with automatic fallback to SELECT-then-INSERT/UPDATE for
  unsupported drivers. Includes `buildUpsertSQL[T]`, `extractColumn`, and
  comprehensive tests for all three dialects. See issue #13.
- ❌ **Aggregate functions**: GROUP BY, HAVING, SUM, AVG, MIN, MAX
- ❌ **Dedicated IN operator API**: Structured API for `WHERE id IN (?,?,?)`

### Phase 3: Schema Management (LOW)
- ❌ **Schema migrations**: ALTER TABLE support
- ❌ **CREATE INDEX generation**

### Phase 4: Developer Experience (LOW)
- ❌ **Raw SQL fragments**: Allow raw SQL injection into generated queries
- ❌ **Transactional reads**: Support `*sql.Tx` in Get/List
- ❌ **Batch operations**: Batch insert/update multiple rows in a single query
- ❌ **Connection pool tuning**: Built-in helpers for pool configuration

## Completed Milestones

| Milestone | Date | Details |
|-----------|------|---------|
| v0.6.0 release | 2026-05-23 | Critical bug fixes (isAutoIncrement, PostgreSQL getLastInsertID, lock-retry, Update leak), ArgsAppay → ArgsApply rename, zero-alloc read path, metadata cache, docs alignment. |
| v0.7.0 release | 2026-05-23 | PostgreSQL integration tests, PG DDL generation, `?` → `$N` rebinding, CI matrix with MySQL/PostgreSQL service containers, `cachedDialect` concurrency fix. |
| v0.7.1 release | 2026-06-11 | Native UPSERT for Set, `Fields[T]()`, `db_table_name` ergonomics, List API guidance, SQL Server docs, Table.Close docs fix. |
| v0.8.0 release | 2026-06-12 | Type-safe WHERE helpers, benchmarks, comparisons, animated GIF demo, pkg.go.dev badge. Annotated tag created; release notes pending. |

## Known Issues

| Issue | Severity | Status |
|-------|----------|--------|
| MySQL Docker test gated behind `SQLH_MYSQL_TEST` env var; `--network host` removed | Medium | ✅ Fixed |
| Context support partially implemented in write paths | Medium | Known |
| JOIN composite struct setup requires manual naming convention | Low | Known |
| PostgreSQL `last_insert_rowid` fixed via `pg_get_serial_sequence` | Medium | ✅ Fixed |
| Lock-retry uses substring match (less fragile, but still not `errors.Is`) | Low | ✅ Mitigated |
| `isAutoIncrement` now detects MySQL `AUTO_INCREMENT` | Medium | ✅ Fixed |
| No native UPSERT (Set uses SELECT-then-INSERT/UPDATE) | Medium | ✅ Fixed in v0.7.1 |

## Feature Completeness

| Feature | Status | Version |
|---------|--------|---------|
| Basic CRUD (Insert, Get, List, Update, Delete) | ✅ Complete | v0.1.0 |
| Set (upsert) | ✅ Complete | v0.1.0 |
| Transaction auto-wrap | ✅ Complete | v0.1.0 |
| Database lock retry | ✅ Complete | v0.1.0 / Stage 1 |
| Struct tag mapping | ✅ Complete | v0.1.0 |
| Create table from struct | ✅ Complete | v0.1.0 |
| InsertId (return inserted ID) | ✅ Complete | v0.1.0 |
| Autoincrement field detection | ✅ Complete | v0.2.0 / Stage 1 |
| ErrWhereClauseRequiredForUpdate | ✅ Complete | v0.2.0 |
| Set atomicity fix | ✅ Complete | v0.2.0 |
| Delete uses tx.Prepare | ✅ Complete | v0.2.0 |
| Transaction close fix | ✅ Complete | v0.2.1 |
| Bool field scanning fix | ✅ Complete | v0.2.1 |
| ListRange (Go 1.25 iterator) | ✅ Complete | v0.2.2 |
| Expanded arg types | ✅ Complete | v0.2.2 |
| ListRows explicit pagination | ✅ Complete | v0.2.2 |
| `Table[T]` wrapper API | ✅ Complete | v0.5.0 |
| Custom table name (tag + interface) | ✅ Complete | v0.5.1 |
| Metadata cache | ✅ Complete | v0.6.0 |
| Advanced WHERE (OR, IN, LIKE, IS NULL) | ✅ Complete | v0.2.2+ |
| Context propagation (read paths) | ✅ Complete | v0.2.2+ |
| JOIN support with composite structs | ✅ Complete | v0.6.0 |
| Flexible SELECT (DISTINCT, Alias, custom Name) | ✅ Complete | v0.2.2+ |
| MySQL `AUTO_INCREMENT` detection | ✅ Fixed | v0.6.0 |
| PostgreSQL `getLastInsertID` | ✅ Fixed | v0.6.0 |
| `isLockError` robust detection | ✅ Fixed | v0.6.0 |
| Update statement handle leak | ✅ Fixed | v0.6.0 |
| `ArgsAppay` → `ArgsApply` rename | ✅ Complete | v0.6.0 |
| Zero-alloc read path | ✅ Complete | v0.6.0 |
| Native UPSERT (ON CONFLICT DO UPDATE) | ✅ Complete | v0.7.1 |
| Type-safe WHERE helpers (Eq, Ne, Gt, etc.) | ✅ Complete | v0.8.0 |
| Aggregate functions | ❌ Not started | — |
| Schema migrations | ❌ Not started | — |
| Batch operations | ❌ Not started | — |
| Raw SQL fragments | ❌ Not started | — |
| Transactional reads | ❌ Not started | — |

## Performance Baseline (2026-06-12)

Benchmarks from the `bench/` module (v0.8.0). The addressable fast path (via
pointer) is the production code path used inside `QueryRange`.

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| `BenchmarkArgsWrite` | 195 | 224 | 2 |
| `BenchmarkArgsReadApply_value` (non-addressable fallback) | 711 | 392 | 12 |
| `BenchmarkArgsReadApply_addr` (addressable fast path) | 590 | 256 | 4 |
| `BenchmarkSelect` | 1068 | 528 | 13 |

### Improvement vs prior baseline

| Path | Before (Stage 0) | After (Stage 3) | Δ |
|------|-----------------|-----------------|---|
| Read+Apply (value path) | 759 ns, 416 B, 13 allocs | 711 ns, 392 B, 12 allocs | −6 % / −8 % |
| Read+Apply (addr path) | n/a (~same as value) | 590 ns, 256 B, 4 allocs | −22 % / −69 % |
| Write path | 195 ns, 224 B, 2 allocs | unchanged | — |

## Quality Metrics

- **Test Coverage**: SQLite tests pass, Query generation tests pass, Table
  wrapper tests pass, metadata cache tests pass, retry logic tests pass
- **MySQL Tests**: ✅ Gated behind `SQLH_MYSQL_TEST=1`; runs against local Docker
  container with readiness wait
- **PostgreSQL Tests**: ✅ Gated behind `SQLH_TEST_POSTGRES=1`; full CRUD suite
- **Documentation**: CHANGELOG.md, README.md, ROADMAP.md, SKILL.md, all 7
  Memory Bank files present
- **Examples**: 8 runnable programs in `examples/` directory (basic, join,
  paginator, set, iterators, context, crud, comparison) plus `ExampleListRows` and
  `ExampleListRange` in `sqlh_example_test.go` for pkg.go.dev
- **Backward Compatibility**: Public API changes limited; `ArgsAppay` deprecated
  for removal in v1.0.0; all else backward-compatible
- **Awesome-Go Submission**: PR #6401 submitted to avelino/awesome-go (SQL Query
  Builders section) — awaiting review. README badge deferred until upstream
  acceptance.

## Next Milestones

1. **v1.0.0**: Stable API with schema management and full database compatibility
2. **Aggregate functions**: GROUP BY, HAVING, SUM, AVG, MIN, MAX
3. **Batch operations**: Multi-row insert/update in a single query
4. **Coverage to 80%+**: Add tests for `getLastInsertID` (MySQL/PG branches), and
   `detectDialect` (non-SQLite paths). `Close` is an empty no-op by design —
   no testable statements inside.
