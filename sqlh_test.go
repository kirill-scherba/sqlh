// Copyright 2024 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlh

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kirill-scherba/sqlh/query"
	_ "github.com/mattn/go-sqlite3"
)

// TestTable is the test table structure
type TestTable struct {
	ID   int64  `db:"id" db_key:"not null primary key autoincrement"`
	Name string `db:"name"`
	Data []byte `db:"data"`
}

func TestSQLOperations(t *testing.T) {

	// Open in-memory SQLite database for testing
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	require.NoError(t, err, "failed to open database")
	defer db.Close()

	// Create table
	createStmt, err := query.Table[TestTable]()
	require.NoError(t, err)
	_, err = db.Exec(createStmt)
	require.NoError(t, err, "failed to create table")

	// 1. Test Insert with autoincrement
	t.Run("Insert and Verify Autoincrement", func(t *testing.T) {
		user1 := TestTable{Name: "Alice", Data: []byte("data1")} // ID is 0
		err := Insert(db, user1)
		require.NoError(t, err)

		// Retrieve to verify
		retrievedUser, err := Get[TestTable](db, Where{"name=", "Alice"})
		require.NoError(t, err)
		require.NotNil(t, retrievedUser)

		// Check if autoincrement ID was assigned
		assert.Equal(t, int64(1), retrievedUser.ID)
		assert.Equal(t, "Alice", retrievedUser.Name)
		assert.Equal(t, []byte("data1"), retrievedUser.Data)
	})

	// 2. Test Update
	t.Run("Update", func(t *testing.T) {
		// The user with ID 1 should exist from the previous test
		userToUpdate := TestTable{ID: 1, Name: "Alicia", Data: []byte("data_updated")}
		err := Update(db, UpdateAttr[TestTable]{
			Row:    userToUpdate,
			Wheres: []Where{{"id=", 1}},
		})
		require.NoError(t, err)

		// Retrieve to verify update
		updatedUser, err := Get[TestTable](db, Where{"id=", 1})
		require.NoError(t, err)
		require.NotNil(t, updatedUser)
		assert.Equal(t, "Alicia", updatedUser.Name)
		assert.Equal(t, []byte("data_updated"), updatedUser.Data)
	})

	// 3. Test List
	t.Run("List", func(t *testing.T) {
		// Insert another user to have multiple rows
		user2 := TestTable{Name: "Bob", Data: []byte("data2")}
		err := Insert(db, &user2)
		require.NoError(t, err)

		// List all users
		users, _, err := ListRows[TestTable](db, 0, "name ASC", 100)
		require.NoError(t, err)
		assert.Len(t, users, 2)

		// List with where clause
		bobs, _, err := ListRows[TestTable](db, 0, "name ASC", 100, Where{"name=", "Bob"})
		require.NoError(t, err)
		assert.Len(t, bobs, 1)
		assert.Equal(t, "Bob", bobs[0].Name)
	})

	// 6. Test list range with pointer
	t.Run("ListRange", func(t *testing.T) {
		// List with where clause
		for row := range ListRange[TestTable](db, 0, "name ASC", 0, Where{"name=", "Bob"}) {
			assert.Equal(t, "Bob", row.Name)
		}

		// List with where clause
		for row := range ListRange[TestTable](db, 0, "name ASC", 0, Where{"name=", "Alice"}) {
			assert.Equal(t, "Alice", row.Name)
		}
	})

	// 4. Test Delete
	t.Run("Delete", func(t *testing.T) {
		// Delete user with ID 1
		err := Delete[TestTable](db, Where{"id=", 1})
		require.NoError(t, err)

		// Verify deletion
		_, err = Get[TestTable](db, Where{"id=", 1})
		require.Error(t, err)
		assert.ErrorIs(t, err, sql.ErrNoRows, "expected sql.ErrNoRows after deleting")

		// Check remaining rows
		remainingUsers, _, err := ListRows[TestTable](db, 0, "name ASC", 100)
		require.NoError(t, err)
		assert.Len(t, remainingUsers, 1)
		assert.Equal(t, "Bob", remainingUsers[0].Name)
	})

	// 5. Test Get error cases
	t.Run("Get Errors", func(t *testing.T) {
		// No where clause
		_, err := Get[TestTable](db)
		assert.ErrorIs(t, err, ErrWhereClauseRequired)

		// Multiple rows found
		_ = Insert(db, TestTable{Name: "Charlie", Data: []byte("data3")})
		_ = Insert(db, TestTable{Name: "Charlie", Data: []byte("data4")})
		_, err = Get[TestTable](db, Where{"name=", "Charlie"})
		assert.ErrorIs(t, err, ErrMultipleRowsFound)
	})
}
