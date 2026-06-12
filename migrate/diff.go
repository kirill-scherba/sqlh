// Copyright 2026 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package migrate

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/kirill-scherba/sqlh/query"
)

// DiffOption configures Diff behavior.
type DiffOption func(*diffConfig)

type diffConfig struct {
	autoAdd bool
}

// AutoAdd is the default and only mode for Diff. It ensures only additive
// changes (ADD COLUMN) are generated. Destructive changes must use Raw.
func AutoAdd() DiffOption {
	return func(c *diffConfig) { c.autoAdd = true }
}

// Diff creates a migration step that compares a Go struct against the live
// database schema and generates ALTER TABLE ADD COLUMN for each missing column.
//
// Only additive changes are produced — no DROP, RENAME, or type alterations.
func Diff[T any](tableName string, v Version, opts ...DiffOption) Migration {
	cfg := &diffConfig{autoAdd: true}
	for _, o := range opts {
		o(cfg)
	}
	name := nameFromTableAndVersion(tableName, v)
	return &diffMigration[T]{
		name:      name,
		version:   v,
		tableName: tableName,
		config:    cfg,
	}
}

// diffColumn describes a column extracted from a Go struct.
type diffColumn struct {
	Name    string
	Type    string
	NotNull bool
}

// diffMigration compares struct fields against the live schema.
type diffMigration[T any] struct {
	name      string
	version   Version
	tableName string
	config    *diffConfig
}

func (d *diffMigration[T]) Name() string     { return d.name }
func (d *diffMigration[T]) Version() Version { return d.version }

// SQL implements the Migration interface for static dialect-only mode.
// For Diff, the real work happens in SQLWithDB.
func (d *diffMigration[T]) SQL(dialect Dialect) (string, error) {
	return "", fmt.Errorf("Diff requires Apply (live database) to generate SQL")
}

// SQLWithDB generates the ALTER TABLE ADD COLUMN statements by comparing the
// struct definition against the live database schema.
func (d *diffMigration[T]) SQLWithDB(db querier, dialect Dialect) (string, error) {
	if d.config.autoAdd {
		return d.sqlAutoAdd(db, dialect)
	}
	return "", fmt.Errorf("diff mode not supported")
}

// sqlAutoAdd generates ALTER TABLE ADD COLUMN for every struct field not
// present in the live database.
func (d *diffMigration[T]) sqlAutoAdd(db querier, dialect Dialect) (string, error) {
	// Build a dummy value just to get the concrete type.
	var zero T
	t := reflect.TypeOf(zero)
	if t == nil {
		// T may be a pointer — dereference.
		var ptrZero *T
		t = reflect.TypeOf(ptrZero).Elem()
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return "", fmt.Errorf("diff requires a struct type, got %s", t.Kind())
	}

	// Determine table name.
	table := d.tableName
	if table == "" {
		table = query.Name[T]()
	}

	// Extract columns from struct tags.
	structCols, err := structColumns(t, dialect)
	if err != nil {
		return "", err
	}

	// Retrieve live columns.
	liveCols, err := TableColumns(db, table, dialect)
	if err != nil {
		return "", fmt.Errorf("introspect table %q: %w", table, err)
	}

	// Build a set of live column names for fast lookup.
	liveSet := make(map[string]struct{}, len(liveCols))
	for _, c := range liveCols {
		liveSet[c.Name] = struct{}{}
	}

	// Generate ALTER TABLE ADD COLUMN for each missing column.
	var stmts []string
	for _, col := range structCols {
		if _, ok := liveSet[col.Name]; ok {
			continue // already exists
		}
		stmt := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, col.Name, col.Type)
		if col.NotNull {
			stmt += " NOT NULL"
		}
		stmt += ";"
		stmts = append(stmts, stmt)
	}

	if len(stmts) == 0 {
		return "", nil // no changes needed
	}
	return strings.Join(stmts, "\n"), nil
}

// structColumns extracts columns from a struct type using the same rules as
// query.getFieldType and query.getFieldName.
func structColumns(t reflect.Type, dialect Dialect) ([]diffColumn, error) {
	var cols []diffColumn
	for i := range t.NumField() {
		field := t.Field(i)

		// Skip sentinel fields.
		if field.Name == "_" {
			continue
		}

		// Skip fields with db:"-".
		dbTag := field.Tag.Get("db")
		if dbTag == "-" {
			continue
		}

		name := dbTag
		if name == "" {
			name = strings.ToLower(field.Name)
		}

		typ, err := goTypeToSQL(field, dialect)
		if err != nil {
			return nil, err
		}

		dbKey := field.Tag.Get("db_key")
		notNull := strings.Contains(strings.ToLower(dbKey), "not null")

		cols = append(cols, diffColumn{
			Name:    name,
			Type:    typ,
			NotNull: notNull,
		})
	}
	return cols, nil
}

// goTypeToSQL maps a Go struct field to its SQL type, respecting db_type tag.
func goTypeToSQL(field reflect.StructField, dialect Dialect) (string, error) {
	// db_type tag takes highest priority.
	if ft := field.Tag.Get("db_type"); ft != "" {
		return ft, nil
	}

	ft := field.Type
	if ft.Kind() == reflect.Pointer {
		ft = ft.Elem()
	}

	switch ft.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "integer", nil
	case reflect.Uint8:
		if dialect == PostgreSQL {
			return "smallint", nil
		}
		return "tinyint", nil
	case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "bigint", nil
	case reflect.Float32, reflect.Float64:
		if dialect == PostgreSQL {
			return "double precision", nil
		}
		return "double", nil
	case reflect.Bool:
		if dialect == PostgreSQL {
			return "boolean", nil
		}
		return "bit", nil
	case reflect.String:
		return "text", nil
	case reflect.Slice:
		if ft.Elem().Kind() == reflect.Uint8 {
			if dialect == PostgreSQL {
				return "bytea", nil
			}
			return "blob", nil
		}
		return "", fmt.Errorf("unsupported slice type: %s", ft)
	case reflect.Struct:
		if ft == reflect.TypeOf(time.Time{}) {
			return "timestamp", nil
		}
		return "", fmt.Errorf("unsupported struct type: %s", ft)
	case reflect.Complex64, reflect.Complex128:
		if dialect == PostgreSQL {
			return "bytea", nil
		}
		return "blob", nil
	default:
		return "", fmt.Errorf("unsupported type: %s", ft.Kind())
	}
}
