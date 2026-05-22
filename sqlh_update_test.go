// Copyright 2026 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlh

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kirill-scherba/sqlh/query"
	_ "github.com/mattn/go-sqlite3"
)

// TestUpdate_batchManyAttrs is a regression test for the per-iteration
// statement leak that used to happen in Update: defer stmt.Close() was
// stacked on the parent frame and only released after the whole batch
// committed. The current implementation closes each statement at the end
// of its own iteration via updateOne. The test runs a 200-row batch and
// expects no resource exhaustion or error.
func TestUpdate_batchManyAttrs(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	createStmt, err := query.Table[TestTable]()
	require.NoError(t, err)
	_, err = db.Exec(createStmt)
	require.NoError(t, err)

	// Seed 200 rows
	const n = 200
	rows := make([]TestTable, 0, n)
	for i := 0; i < n; i++ {
		rows = append(rows, TestTable{Name: fmt.Sprintf("user_%d", i), Data: []byte("seed")})
	}
	require.NoError(t, Insert(db, rows...))

	// Build a batch of 200 UpdateAttr targeting each row by name
	attrs := make([]UpdateAttr[TestTable], 0, n)
	for i := 0; i < n; i++ {
		attrs = append(attrs, UpdateAttr[TestTable]{
			Row: TestTable{
				Name: fmt.Sprintf("user_%d", i),
				Data: []byte(fmt.Sprintf("updated_%d", i)),
			},
			Wheres: []Where{{Field: "name=", Value: fmt.Sprintf("user_%d", i)}},
		})
	}

	// All updates must succeed in a single Update call
	require.NoError(t, Update(db, attrs...))

	// Spot-check a few rows
	for _, i := range []int{0, n / 2, n - 1} {
		name := fmt.Sprintf("user_%d", i)
		row, err := Get[TestTable](db, Where{Field: "name=", Value: name})
		require.NoError(t, err)
		require.NotNil(t, row)
		require.Equal(t, fmt.Sprintf("updated_%d", i), string(row.Data))
	}
}
