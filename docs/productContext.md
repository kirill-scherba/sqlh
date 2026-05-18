# Product Context: sqlh

## Why This Project Exists

Go developers working with SQL databases face a significant amount of repetitive boilerplate code: writing `CREATE TABLE` statements, manual `INSERT`/`UPDATE`/`SELECT`/`DELETE` queries, scanning rows into structs, managing transactions, and handling errors. While Go's `database/sql` package provides a solid foundation, it requires verbose, error-prone code for even simple CRUD operations.

`sqlh` was created to solve this problem by leveraging Go generics (introduced in Go 1.18 and matured in Go 1.25) to provide a type-safe, reflection-based ORM-like experience without the complexity and overhead of full ORM frameworks.

## Problems It Solves

### 1. Boilerplate SQL Generation
- **Problem**: Every CRUD operation requires writing and maintaining SQL statements
- **Solution**: Automatic generation of `CREATE TABLE`, `INSERT`, `UPDATE`, `SELECT`, `DELETE` from struct definitions with struct tags
- **Impact**: Reduces code volume by 60-80% for typical CRUD operations

### 2. Type Safety in Database Operations
- **Problem**: Raw SQL queries are stringly-typed—no compile-time checking of column names or types
- **Solution**: Go generics ensure type-safe operations; `Get[T]()` returns `*T`, `List[T]()` returns `[]T` plus the next pagination offset
- **Impact**: Catch type mismatches at compile time rather than runtime

### 3. Transaction Management Complexity
- **Problem**: Correctly managing transactions (Begin, Commit, Rollback on error) is tedious and error-prone
- **Solution**: All write operations (`Insert`, `Update`, `Delete`, `Set`) are automatically wrapped in transactions with proper rollback on error
- **Impact**: Eliminates whole class of bugs related to incomplete transactions and connection leaks

### 4. Database Lock Handling
- **Problem**: SQLite databases frequently encounter "database is locked" errors under concurrent access
- **Solution**: Built-in automatic retry mechanism (up to 20 attempts with 100ms intervals)
- **Impact**: Production-grade resilience for SQLite-based applications

### 5. Struct-to-Schema Mapping
- **Problem**: Manually maintaining separate schema definitions and Go struct definitions that can drift apart
- **Solution**: Single-source-of-truth via struct tags (`db`, `db_type`, `db_key`)
- **Impact**: Schema always matches code; eliminates schema drift

## User Experience Goals

1. **Minimal Configuration**: Define a struct with tags, call `query.Table[T]()` to get `CREATE TABLE`, use `sqlh.Insert`, `sqlh.Get`, etc. — no configuration files, no migrations to write
2. **Intuitive API**: Function names mirror SQL operations (`Insert`, `Update`, `Get`, `List`, `Delete`)
3. **Predictable Behavior**: Write operations are always transactional; errors are wrapped with clear context
4. **Database-Agnostic by Default**: Works with SQLite out of the box; supports MySQL; extensible to PostgreSQL and SQL Server
5. **Progressive Enhancement**: Start with basic CRUD, add pagination (`ListRows`/`ListRange`), WHERE conditions (`Where` struct), JOINs as needed

## Key Differentiators from Alternatives

| Feature | sqlh | GORM | sqlx |
|---------|------|------|------|
| Generics-based | ✅ Native | ❌ Interface-based | ❌ Interface-based |
| Query generation | ✅ Full CRUD | ✅ Full CRUD | ❌ Manual SQL |
| Transaction auto-wrap | ✅ Always | ✅ Always | ❌ Manual |
| Database lock retry | ✅ Built-in | ❌ | ❌ |
| Compile-time type safety | ✅ High | ✅ Medium | ❌ Low |
| Learning curve | ✅ Low | ❌ High | ✅ Medium |
| Reflection overhead | ✅ Optimized | ❌ Heavy | ✅ Minimal |

## Project Maturity

Current version v0.5.1 plus active `feature/metadata_cache` branch work — active development with a clear roadmap. The package is functional for basic and intermediate use cases, with planned enhancements for advanced query features, native UPSERT, aggregate functions, and schema migrations.
