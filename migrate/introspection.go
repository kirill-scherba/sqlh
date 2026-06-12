// Copyright 2026 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package migrate

import (
	"database/sql"
	"fmt"
	"strings"
)

// ColumnInfo describes a column from the live database schema.
type ColumnInfo struct {
	Name    string // column name
	Type    string // SQL type (lower-cased for comparison)
	NotNull bool   // true when the column is NOT NULL
}

// TableColumns returns the current column list for a table.
// The dialect is detected from db so the correct introspection query is used.
func TableColumns(db querier, tableName string, dialect Dialect) ([]ColumnInfo, error) {
	switch dialect {
	case SQLite:
		return tableColumnsSQLite(db, tableName)
	case MySQL:
		return tableColumnsMySQL(db, tableName)
	case PostgreSQL:
		return tableColumnsPostgreSQL(db, tableName)
	default:
		return nil, fmt.Errorf("unsupported dialect for introspection: %s", dialect)
	}
}

// tableColumnsSQLite uses PRAGMA table_info.
func tableColumnsSQLite(db querier, tableName string) ([]ColumnInfo, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info('%s')", tableName))
	if err != nil {
		return nil, fmt.Errorf("pragma table_info failed: %w", err)
	}
	defer rows.Close()

	var cols []ColumnInfo
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			return nil, fmt.Errorf("scan pragma row failed: %w", err)
		}
		cols = append(cols, ColumnInfo{
			Name:    name,
			Type:    strings.ToLower(typ),
			NotNull: notNull != 0,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// PRAGMA table_info returns empty set for non-existent tables; detect that.
	if len(cols) == 0 {
		var exists int
		err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&exists)
		if err != nil {
			return nil, err
		}
		if exists == 0 {
			return nil, fmt.Errorf("table %q does not exist", tableName)
		}
	}
	return cols, nil
}

// tableColumnsMySQL uses SHOW COLUMNS FROM.
func tableColumnsMySQL(db querier, tableName string) ([]ColumnInfo, error) {
	rows, err := db.Query(fmt.Sprintf("SHOW COLUMNS FROM `%s`", tableName))
	if err != nil {
		return nil, fmt.Errorf("show columns failed: %w", err)
	}
	defer rows.Close()

	var cols []ColumnInfo
	for rows.Next() {
		var field, typ, null, key, extra string
		var defaultVal sql.NullString
		if err := rows.Scan(&field, &typ, &null, &key, &defaultVal, &extra); err != nil {
			return nil, fmt.Errorf("scan show columns row failed: %w", err)
		}
		cols = append(cols, ColumnInfo{
			Name:    field,
			Type:    strings.ToLower(typ),
			NotNull: strings.ToUpper(null) == "NO",
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return cols, nil
}

// tableColumnsPostgreSQL uses information_schema.columns.
func tableColumnsPostgreSQL(db querier, tableName string) ([]ColumnInfo, error) {
	query := `
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_name = $1
		ORDER BY ordinal_position`
	rows, err := db.Query(query, tableName)
	if err != nil {
		return nil, fmt.Errorf("information_schema query failed: %w", err)
	}
	defer rows.Close()

	var cols []ColumnInfo
	for rows.Next() {
		var name, typ, nullable string
		if err := rows.Scan(&name, &typ, &nullable); err != nil {
			return nil, fmt.Errorf("scan info_schema row failed: %w", err)
		}
		cols = append(cols, ColumnInfo{
			Name:    name,
			Type:    strings.ToLower(typ),
			NotNull: strings.ToUpper(nullable) == "NO",
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return cols, nil
}
