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

var (
	// cachedDialect is set by detectDialect whenever a *sql.DB is passed to
	// any sqlh function. It is used by rebindIfPG to decide whether ? → $N
	// placeholder conversion is needed.
	cachedDialect string
)

// detectDialect detects the database dialect from the driver type and sets
// the cachedDialect global. It's called at the beginning of every top-level
// function that receives a *sql.DB.
func detectDialect(db *sql.DB) {
	driverName := reflect.TypeOf(db.Driver()).String()
	driverName = strings.ToLower(driverName)

	switch {
	case strings.Contains(driverName, "postgres"),
		strings.Contains(driverName, "pq"),
		strings.Contains(driverName, "pgx"):
		cachedDialect = dialectPostgreSQL
	case strings.Contains(driverName, "mysql"):
		cachedDialect = dialectMySQL
	case strings.Contains(driverName, "sqlite"):
		cachedDialect = dialectSQLite
	case strings.Contains(driverName, "sqlserver"),
		strings.Contains(driverName, "mssql"):
		cachedDialect = dialectSQLServer
	default:
		cachedDialect = dialectSQLite
	}
}

// rebindIfPG converts ? placeholders to PostgreSQL $N style when the cached
// dialect is PostgreSQL. For all other dialects the SQL is returned unchanged.
func rebindIfPG(sql string) string {
	if cachedDialect == dialectPostgreSQL {
		return query.Rebind(sql)
	}
	return sql
}
