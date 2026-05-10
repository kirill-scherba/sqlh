# Project Brief: sqlh

## Overview

`sqlh` (SQL Helper) is a lightweight Go package that simplifies interactions with SQL databases. It leverages Go generics (Go 1.25+) to provide a set of intuitive type-safe CRUD functions (`Insert`, `Update`, `Get`, `List`, `Delete`, `Set`) that work directly with user-defined Go structs, reducing boilerplate SQL code.

## Core Mission

Eliminate repetitive SQL query writing for Go developers by automatically generating SQL statements from struct definitions using struct tags, while maintaining database-agnostic compatibility and transactional safety.

## Key Stakeholders

- **Author/Maintainer**: Kirill Scherba (kirill@scherba.ru)
- **Target Users**: Go developers working with SQL databases (SQLite, MySQL, PostgreSQL) who want to avoid writing boilerplate CRUD SQL

## Core Capabilities

1. **Automatic Query Generation**: Auto-generates `CREATE TABLE`, `INSERT`, `UPDATE`, `SELECT`, and `DELETE` statements from Go structs
2. **Generic Type-Safe API**: Uses Go generics (`T any`) for compile-time type safety
3. **Struct Tag Mapping**: Uses `db`, `db_type`, and `db_key` tags to define column names, types, and constraints
4. **Autoincrement Support**: Automatically excludes `autoincrement` fields from INSERT and UPDATE statements
5. **Transactional Writes**: All write operations (`Insert`, `Update`, `Delete`, `Set`) are wrapped in transactions
6. **Database Lock Retry**: Built-in retry mechanism (up to 20 attempts with 100ms delay) for "database is locked" errors
7. **Standardized Errors**: Returns `sql.ErrNoRows` and exported package errors (`ErrWhereClauseRequired`, `ErrMultipleRowsFound`, etc.)
8. **Context Support**: Functions accept `context.Context` for timeouts and cancellations
9. **Pagination**: `List` function supports pagination with offset/limit via `Paginator` structure
10. **Iterators**: Uses Go 1.25 iterators (`iter.Seq`) for row iteration in `ListRange`
11. **JOIN Support**: Basic JOIN support via `query.Join` attribute
12. **DISTINCT, Alias, Name**: Flexible query attributes for advanced SELECT queries

## Current Version

v0.2.2 (released 2025-10-26)

## License

BSD-style license