// Copyright 2026 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package migrate

import (
	"database/sql"
	"fmt"
	"time"
)

// querier is the common interface used by *sql.DB and *sql.Tx.
type querier interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// migrationWithDB is an internal interface for migrations that need a live
// database connection to generate their SQL (e.g. Diff).
type migrationWithDB interface {
	Migration
	SQLWithDB(db querier, dialect Dialect) (string, error)
}

// Apply runs all pending migrations in the plan inside a transaction.
//
// It creates the _migrations tracking table if it does not exist, validates
// the plan, skips already-applied versions, and executes pending migrations
// in ascending version order.
//
// When opts.DryRun is true, Apply prints SQL to stdout and returns nil
// without executing anything.
//
// When opts.Backup is set, it is called once before the transaction begins.
// If Backup returns an error, Apply aborts before any database changes.
func Apply(db *sql.DB, plan Plan, opts Options) error {
	// Validate plan.
	if err := validatePlan(plan); err != nil {
		return err
	}

	// Detect dialect.
	dialect := DetectDialect(db)

	// DryRun: print SQL and return.
	if opts.DryRun {
		return applyDryRun(db, plan, opts, dialect)
	}

	// Ensure _migrations table exists (outside transaction for portability).
	if err := ensureMigrationsTable(db, dialect); err != nil {
		return fmt.Errorf("create _migrations table: %w", err)
	}

	// Query applied versions.
	applied, err := appliedVersions(db)
	if err != nil {
		return fmt.Errorf("query applied versions: %w", err)
	}

	// Filter pending.
	pending := filterPending(plan, applied)
	if len(pending) == 0 {
		return nil // nothing to do
	}

	// Begin transaction.
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute each pending migration.
	for _, m := range pending {
		// Backup hook called before each migration step.
		if opts.Backup != nil {
			if err := opts.Backup(db); err != nil {
				return fmt.Errorf("backup hook before v%d: %w", m.Version(), err)
			}
		}

		sql, err := migrationSQL(m, tx, dialect)
		if err != nil {
			return fmt.Errorf("migration v%d %q: generate SQL: %w", m.Version(), m.Name(), err)
		}

		// Skip empty SQL (e.g. Diff with no changes).
		if sql == "" {
			if err := recordMigration(tx, m, dialect); err != nil {
				return fmt.Errorf("migration v%d %q: record: %w", m.Version(), m.Name(), err)
			}
			continue
		}

		// Execute migration SQL.
		if _, err := tx.Exec(sql); err != nil {
			return fmt.Errorf("migration v%d %q: execute SQL: %w", m.Version(), m.Name(), err)
		}

		// Record in _migrations.
		if err := recordMigration(tx, m, dialect); err != nil {
			return fmt.Errorf("migration v%d %q: record: %w", m.Version(), m.Name(), err)
		}
	}

	// Commit.
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// applyDryRun prints pending SQL to stdout without executing.
func applyDryRun(db *sql.DB, plan Plan, opts Options, dialect Dialect) error {
	fmt.Println("--- Dry run ---")

	for _, m := range plan {
		sql, err := migrationSQL(m, db, dialect)
		if err != nil {
			return fmt.Errorf("migration v%d %q: %w", m.Version(), m.Name(), err)
		}
		if sql != "" {
			fmt.Printf("[V%d] %s\n%s\n", m.Version(), m.Name(), sql)
		} else {
			fmt.Printf("[V%d] %s\n(no changes needed)\n", m.Version(), m.Name())
		}
	}

	fmt.Println("--- Dry run complete (no changes made) ---")
	return nil
}

// migrationSQL generates SQL for a migration, using the DB-aware interface
// when available.
func migrationSQL(m Migration, db querier, dialect Dialect) (string, error) {
	if md, ok := m.(migrationWithDB); ok {
		return md.SQLWithDB(db, dialect)
	}
	return m.SQL(dialect)
}

// ensureMigrationsTable creates the _migrations tracking table.
func ensureMigrationsTable(db querier, dialect Dialect) error {
	var sql string
	switch dialect {
	case PostgreSQL:
		sql = `CREATE TABLE IF NOT EXISTS _migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP NOT NULL
		);`
	default:
		sql = `CREATE TABLE IF NOT EXISTS _migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP NOT NULL
		);`
	}
	_, err := db.Exec(sql)
	return err
}

// migrationsRow represents a single row in _migrations.
type migrationsRow struct {
	Version   int
	Name      string
	AppliedAt time.Time
}

// appliedVersions returns the set of already-applied migration versions.
func appliedVersions(db querier) (map[Version]struct{}, error) {
	rows, err := db.Query("SELECT version FROM _migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[Version]struct{})
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[Version(v)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return applied, nil
}

// filterPending returns migrations from plan that are not in applied.
func filterPending(plan Plan, applied map[Version]struct{}) Plan {
	var pending Plan
	for _, m := range plan {
		if _, ok := applied[m.Version()]; !ok {
			pending = append(pending, m)
		}
	}
	return pending
}

// recordMigration inserts a row into _migrations.
func recordMigration(tx querier, m Migration, dialect Dialect) error {
	sql := "INSERT INTO _migrations (version, name, applied_at) VALUES (?, ?, ?)"
	if dialect == PostgreSQL {
		sql = "INSERT INTO _migrations (version, name, applied_at) VALUES ($1, $2, $3)"
	}
	_, err := tx.Exec(
		sql,
		int(m.Version()), m.Name(), time.Now().UTC(),
	)
	return err
}
