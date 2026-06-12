// Copyright 2026 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package migrate

import (
	"fmt"
	"strings"

	"github.com/kirill-scherba/sqlh/query"
)

// FromStruct creates a CREATE TABLE IF NOT EXISTS migration from struct T.
//
// The tableName parameter overrides the struct-derived table name; when
// empty, the name is taken from struct tags (db_table_name, TableName()
// method, or lower-cased type name) via query.Name[T].
func FromStruct[T any](tableName string, v Version) Migration {
	return &fromStruct[T]{
		name:      nameFromTableAndVersion(tableName, v),
		version:   v,
		tableName: tableName,
	}
}

// nameFromTableAndVersion builds a default migration name.
func nameFromTableAndVersion(tableName string, v Version) string {
	if tableName == "" {
		return fmt.Sprintf("v%d", v)
	}
	return fmt.Sprintf("%s_v%d", tableName, v)
}

type fromStruct[T any] struct {
	name      string
	version   Version
	tableName string
}

func (s *fromStruct[T]) Name() string       { return s.name }
func (s *fromStruct[T]) Version() Version   { return s.version }

func (s *fromStruct[T]) SQL(dialect Dialect) (string, error) {
	var sql string
	var err error

	if dialect == PostgreSQL {
		sql, err = query.TablePG[T]()
	} else {
		sql, err = query.Table[T]()
	}
	if err != nil {
		return "", err
	}

	// Override table name if explicitly provided.
	if s.tableName != "" {
		defaultName := query.Name[T]()
		sql = strings.Replace(sql, "CREATE TABLE IF NOT EXISTS "+defaultName, "CREATE TABLE IF NOT EXISTS "+s.tableName, 1)
	}

	return sql, nil
}
