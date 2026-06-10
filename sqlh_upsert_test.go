// Copyright 2024-2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlh

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractColumn(t *testing.T) {
	tests := []struct {
		field    string
		expected string
	}{
		{"id=", "id"},
		{"name=", "name"},
		{"id>", "id"},
		{"age>=", "age"},
		{"score<=", "score"},
		{"name LIKE", "name"},
		{"id IN", "id"},
		{"field BETWEEN", "field"},
		{"id<>", "id"},
		{"deleted IS NULL", "deleted"},
		{"closed IS NOT NULL", "closed"},
		{"status=", "status"},
		{"count !=", "count"},
		{" raw  LIKE", "raw"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			assert.Equal(t, tt.expected, extractColumn(tt.field))
		})
	}
}

func TestBuildUpsertSQL(t *testing.T) {
	// PostgreSQL single-column conflict
	t.Run("postgres single key", func(t *testing.T) {
		sql, err := buildUpsertSQL[TestTable]([]string{"id"}, []string{"id", "name", "data"}, dialectPostgreSQL)
		require.NoError(t, err)
		assert.Equal(t, "INSERT INTO testtable(name,data) VALUES(?,?) ON CONFLICT (id) DO UPDATE SET id = EXCLUDED.id, name = EXCLUDED.name, data = EXCLUDED.data;", sql)
	})

	// PostgreSQL composite conflict key
	t.Run("postgres composite key", func(t *testing.T) {
		sql, err := buildUpsertSQL[TestTable]([]string{"name", "id"}, []string{"id", "name", "data"}, dialectPostgreSQL)
		require.NoError(t, err)
		assert.Equal(t, "INSERT INTO testtable(name,data) VALUES(?,?) ON CONFLICT (name, id) DO UPDATE SET id = EXCLUDED.id, name = EXCLUDED.name, data = EXCLUDED.data;", sql)
	})

	// PostgreSQL empty conflict key (implicit DO NOTHING-ish fallback, generates no conflict key)
	t.Run("postgres empty key", func(t *testing.T) {
		sql, err := buildUpsertSQL[TestTable]([]string{}, []string{"id", "name", "data"}, dialectPostgreSQL)
		require.NoError(t, err)
		assert.Equal(t, "INSERT INTO testtable(name,data) VALUES(?,?) ON CONFLICT DO UPDATE SET id = EXCLUDED.id, name = EXCLUDED.name, data = EXCLUDED.data;", sql)
	})

	// SQLite single-column conflict
	t.Run("sqlite single key", func(t *testing.T) {
		sql, err := buildUpsertSQL[TestTable]([]string{"id"}, []string{"id", "name", "data"}, dialectSQLite)
		require.NoError(t, err)
		assert.Equal(t, "INSERT INTO testtable(name,data) VALUES(?,?) ON CONFLICT (id) DO UPDATE SET id = excluded.id, name = excluded.name, data = excluded.data;", sql)
	})

	// MySQL ON DUPLICATE KEY UPDATE
	t.Run("mysql", func(t *testing.T) {
		sql, err := buildUpsertSQL[TestTable]([]string{"id"}, []string{"id", "name", "data"}, dialectMySQL)
		require.NoError(t, err)
		assert.Equal(t, "INSERT INTO testtable(name,data) VALUES(?,?) ON DUPLICATE KEY UPDATE id = VALUES(id), name = VALUES(name), data = VALUES(data);", sql)
	})

	// Unsupported dialect returns empty string (fallback path)
	t.Run("unsupported dialect", func(t *testing.T) {
		sql, err := buildUpsertSQL[TestTable]([]string{"id"}, []string{"id", "name", "data"}, "sqlserver")
		require.NoError(t, err)
		assert.Equal(t, "", sql)
	})

	// Unknown dialect returns empty string
	t.Run("unknown dialect", func(t *testing.T) {
		sql, err := buildUpsertSQL[TestTable]([]string{"id"}, []string{"id", "name", "data"}, "oracle")
		require.NoError(t, err)
		assert.Equal(t, "", sql)
	})

	// Autoincrement fields excluded from SET clause (fieldNames are already filtered)
	t.Run("autoincrement excluded", func(t *testing.T) {
		sql, err := buildUpsertSQL[TestTable]([]string{"id"}, []string{"name", "data"}, dialectPostgreSQL)
		require.NoError(t, err)
		assert.Contains(t, sql, "SET name = EXCLUDED.name, data = EXCLUDED.data;")
		assert.NotContains(t, sql, "id = EXCLUDED.id")
	})

	// Error from query.Insert[T] propagates
	t.Run("non-struct type error", func(t *testing.T) {
		type notAstruct int
		_, err := buildUpsertSQL[notAstruct]([]string{"id"}, []string{"id"}, dialectPostgreSQL)
		require.Error(t, err)
	})
}
