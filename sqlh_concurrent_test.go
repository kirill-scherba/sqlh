// Copyright 2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlh

import (
	"database/sql"
	"sync"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConcurrentMixedDialect verifies that concurrent calls to sqlh functions
// with different *sql.DB connections (simulating mixed-dialect usage) do not
// cause data races on dialect state. Each goroutine creates its own SQLite
// in-memory database and performs CRUD operations independently.
func TestConcurrentMixedDialect(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent test in short mode")
	}

	const numGoroutines = 8
	const opsPerGoroutine = 10

	var wg sync.WaitGroup

	for g := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Each goroutine gets its own isolated SQLite DB.
			db, err := sql.Open("sqlite3", ":memory:")
			require.NoError(t, err)
			defer db.Close()

			// Create table
			err = Create[TestTable](db)
			require.NoError(t, err)

			for range opsPerGoroutine {
				// Insert a row
				err := Insert(db, TestTable{
					Name: "Alice",
					Data: []byte("concurrent"),
				})
				require.NoError(t, err)

				// Read back via List
				rows, _, err := List[TestTable](db, 0, "", "name ASC")
				require.NoError(t, err)
				assert.NotEmpty(t, rows)

				// Read back via Count
				count, err := Count[TestTable](db)
				require.NoError(t, err)
				assert.Greater(t, count, 0)

				// Read back via Get
				row, err := Get[TestTable](db, Where{"name=", "Alice"})
				require.NoError(t, err)
				require.NotNil(t, row)

				// Update
				row.Data = []byte("updated")
				err = Update(db, UpdateAttr[TestTable]{
					Row:    *row,
					Wheres: []Where{{Field: "id=", Value: row.ID}},
				})
				require.NoError(t, err)

				// Delete
				err = Delete[TestTable](db, Where{"name=", "Alice"})
				require.NoError(t, err)
			}
		}(g)
	}

	wg.Wait()
}
