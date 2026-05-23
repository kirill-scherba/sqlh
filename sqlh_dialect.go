// Copyright 2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlh

import (
	"database/sql"
	"reflect"
	"strings"

	"github.com/kirill-scherba/sqlh/query"
)

// Database dialect constants used internally for dialect-specific SQL
// generation and placeholder rebinding.
const (
	dialectSQLite     = "sqlite"
	dialectMySQL      = "mysql"
	dialectPostgreSQL = "postgres"
	dialectSQLServer  = "sqlserver"
)

// detectDialect detects the database dialect from the driver type. It
// inspects the registered driver name via reflection and returns one of the
// dialect constants. This is a pure function with no side effects.
func detectDialect(db *sql.DB) string {
	driverName := reflect.TypeOf(db.Driver()).String()
	driverName = strings.ToLower(driverName)

	switch {
	case strings.Contains(driverName, "postgres"),
		strings.Contains(driverName, "pq"),
		strings.Contains(driverName, "pgx"):
		return dialectPostgreSQL
	case strings.Contains(driverName, "mysql"):
		return dialectMySQL
	case strings.Contains(driverName, "sqlite"):
		return dialectSQLite
	case strings.Contains(driverName, "sqlserver"),
		strings.Contains(driverName, "mssql"):
		return dialectSQLServer
	default:
		return dialectSQLite
	}
}

// dialectFromQuerier attempts to extract the database dialect from a querier.
// If the querier is a *sql.DB, the dialect is detected via detectDialect.
// For *sql.Tx (which does not expose a Driver()) the function returns
// dialectSQLite as a safe default. Callers that know the dialect (e.g. from a
// surrounding *sql.DB call) should use the non-exported overloads instead.
func dialectFromQuerier(q querier) string {
	if db, ok := q.(*sql.DB); ok {
		return detectDialect(db)
	}
	return dialectSQLite
}

// rebind converts ? placeholders to PostgreSQL $N style when dialect is
// PostgreSQL. For all other dialects the SQL is returned unchanged.
func rebind(sql string, dialect string) string {
	if dialect == dialectPostgreSQL {
		return query.Rebind(sql)
	}
	return sql
}
