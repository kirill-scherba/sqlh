// Copyright 2026 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package migrate provides schema migration for sqlh-managed tables.
//
// It supports three migration strategies:
//
//   - FromStruct[T] — CREATE TABLE IF NOT EXISTS from struct tags.
//   - Diff[T] — auto-detect missing columns and ADD COLUMN.
//   - Raw — explicit SQL for destructive or complex changes.
//
// Apply runs pending migrations in version order. When DryRun is true, it
// prints SQL to stdout without executing. An optional Backup hook allows
// pre-migration backups.
//
// The entire Apply run is wrapped in a single database transaction. If any
// migration step fails, the transaction is rolled back. For SQLite this is
// fully transactional; for MySQL and PostgreSQL some DDL statements may not
// be transactional — use the Backup hook in those cases.
//
// This package is experimental — the API may change.
package migrate

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

// Dialect identifies the database engine.
type Dialect string

// Supported dialects.
const (
	SQLite     Dialect = "sqlite"
	MySQL      Dialect = "mysql"
	PostgreSQL Dialect = "postgres"
	SQLServer  Dialect = "sqlserver"
)

// DetectDialect detects the database dialect from the driver type.
func DetectDialect(db *sql.DB) Dialect {
	driverName := reflect.TypeOf(db.Driver()).String()
	driverName = strings.ToLower(driverName)

	switch {
	case strings.Contains(driverName, "postgres"),
		strings.Contains(driverName, "pq"),
		strings.Contains(driverName, "pgx"):
		return PostgreSQL
	case strings.Contains(driverName, "mysql"):
		return MySQL
	case strings.Contains(driverName, "sqlite"):
		return SQLite
	case strings.Contains(driverName, "sqlserver"),
		strings.Contains(driverName, "mssql"):
		return SQLServer
	default:
		return SQLite
	}
}

// Version identifies a migration step. Integer versions are applied in
// ascending order.
type Version int

// V creates a Version shorthand.
func V(v int) Version { return Version(v) }

// Migration is a single step in a migration plan.
type Migration interface {
	// Name returns the human-readable migration name (stored in _migrations).
	Name() string

	// Version returns the migration version number.
	Version() Version

	// SQL generates the SQL to execute for the given dialect.
	SQL(dialect Dialect) (string, error)
}

// Plan is an ordered list of migration steps.
type Plan []Migration

// Options control Apply behavior.
type Options struct {
	// DryRun prints SQL to stdout without executing.
	DryRun bool

	// Backup is called before Apply starts (after DryRun check). If it
	// returns an error, Apply aborts.
	Backup func(*sql.DB) error
}

// validationError is returned when a Plan fails validation.
type validationError struct {
	msg string
}

func (e validationError) Error() string { return e.msg }

// validatePlan checks that a Plan has no duplicate versions and is sorted
// in ascending order.
func validatePlan(plan Plan) error {
	if len(plan) == 0 {
		return validationError{msg: "migration plan is empty"}
	}
	seen := make(map[Version]struct{}, len(plan))
	for i, m := range plan {
		if _, ok := seen[m.Version()]; ok {
			return validationError{msg: fmt.Sprintf("duplicate migration version %d", m.Version())}
		}
		seen[m.Version()] = struct{}{}
		if i > 0 && plan[i-1].Version() >= m.Version() {
			return validationError{msg: fmt.Sprintf("migration versions must be ascending: %d before %d", plan[i-1].Version(), m.Version())}
		}
	}
	return nil
}
