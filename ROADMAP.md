# `sqlh` Development Roadmap

This document outlines the planned features and improvements for the `sqlh` package. It is a living document and may be adjusted based on priorities and feedback.

## Phase 1: Core Query Enhancements

*   **1.1. Flexible `SELECT` Queries:**
    *   **Select Specific Columns:** Modify `query.Select` to generate a list of columns based on the struct fields instead of `SELECT *`. This will allow for efficient selection of only the necessary data.
    *   **`DISTINCT` Support:** Add the ability to perform `SELECT DISTINCT ...`.

*   **1.2. Advanced `WHERE` Conditions:**
    *   **`OR` Operator:** Implement a way to combine conditions using `OR` in addition to `AND`.
    *   **`IN` Operator:** Add a convenient construct for queries like `WHERE id IN (?, ?, ?)`.
    *   **Improved `LIKE`, `IS NULL` / `IS NOT NULL` support.**

*   **1.3. `context.Context` Propagation:**
    *   Add `context.Context` to all functions that execute database queries (`Get`, `List`, `Insert`, etc.). This is critical for managing timeouts and cancellations in real-world applications.

## Phase 2: Advanced Features & Data Integrity

*   **2.1. `JOIN` Support:**
    *   The most requested feature for working with relational data. A good starting point would be implementing `LEFT JOIN` and scanning the result into a composite struct.

*   **2.2. Native `UPSERT`:**
    *   Replace the current `Set` logic (`SELECT` + `INSERT`/`UPDATE`) with native database commands (e.g., `ON CONFLICT DO UPDATE` for PostgreSQL/SQLite) to make the operation atomic and faster.

*   **2.3. Aggregate Functions:**
    *   Add support for `GROUP BY`, `HAVING`, and functions like `SUM()`, `AVG()`, `MIN()`, `MAX()`.

## Phase 3: Schema Management

*   **3.1. Schema Migrations:**
    *   Add basic support for `ALTER TABLE` to modify existing tables (add/remove columns).
    *   Implement `CREATE INDEX` generation to speed up queries.

## Phase 4: Developer Experience

*   **4.1. Raw SQL Fragments:**
    *   Provide a way to insert raw SQL fragments into generated queries for complex or non-standard cases.

*   **4.2. Transactional Reads:**
    *   Allow `Get` and `List` calls to be executed within an existing transaction by passing a `*sql.Tx` instead of a `*sql.DB`.