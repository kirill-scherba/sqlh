// Copyright 2024 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlh

import (
	"database/sql"
	"encoding/json"
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

type TestTable2 struct {
	ID    int64 `db:"id" db_key:"not null primary key"`
	Value int64 `db:"value"`
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

	// Create table 2
	createStmt, err = query.Table[TestTable2]()
	require.NoError(t, err)
	_, err = db.Exec(createStmt)
	require.NoError(t, err, "failed to create table")

	// Test select and get from empty table
	t.Run("Select and Get from Empty table", func(t *testing.T) {

		usersStmt, err := query.Select[TestTable](nil)
		require.NoError(t, err)

		sqlRows, err := db.Query(usersStmt)
		require.NoError(t, err)

		for sqlRows.Next() {
			var user TestTable
			err := sqlRows.Scan(&user.ID, &user.Name, &user.Data)
			require.NoError(t, err)
		}
		require.NoError(t, sqlRows.Err())

		// Empty rows from ListRows returns empty array in JSON
		rows, _, err := ListRows[TestTable](db, 0, "name ASC", 0)
		require.NoError(t, err)
		rowsJson, err := json.Marshal(rows)
		require.NoError(t, err)
		t.Logf("rows: %s", rowsJson)
		require.Equal(t, string(rowsJson), "[]")

		// An empty array returns null in JSON
		var emptyArr []TestTable
		rowsJson, err = json.Marshal(emptyArr)
		require.NoError(t, err)
		t.Logf("rows: %s", rowsJson)
		require.Equal(t, string(rowsJson), "null")

		// Check Get with empty result, it should return nil and sql.ErrNoRows
		retrievedUser, err := Get[TestTable](db, Where{"name=", "Alice"})
		require.Error(t, err)
		require.Equal(t, err, sql.ErrNoRows)
		require.Nil(t, retrievedUser)
	})

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

		// Insert to test table 2
		testTable2 := TestTable2{ID: 1, Value: 42}
		err = Insert(db, testTable2)
		require.NoError(t, err)
		//
		testTable2 = TestTable2{ID: 2, Value: 75}
		err = Insert(db, testTable2)
		require.NoError(t, err)
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

	t.Run("List with Joins", func(t *testing.T) {

		type testTable struct {

			// Test table fields
			ID   int64  `db:"id" db_key:"not null primary key autoincrement"`
			Name string `db:"name"`
			Data []byte `db:"data"`

			// Test table 2 fields
			// ID2   int64 `db:"id"`
			// Value int64 `db:"value"`
		}

		// List all users
		users, _, err := ListRows[testTable](db, 0, "name ASC", 100,
			"t",
			query.Join{Name: "TestTable2", On: "t.id = o.id", Alias: "o"})
		require.NoError(t, err)
		// assert.Len(t, users, 2)

		for _, user := range users {
			t.Logf("user: %+v", user)
		}
	})

	t.Run("Query with Joins", func(t *testing.T) {

		// Make select attributes
		attr := &query.SelectAttr{
			// Where clauses
			Wheres: []string{"t.name <> ''", "t.id > 0"},

			// Alias of main table
			Alias: "t",

			// Joins with other tables
			Joins: []query.Join{query.MakeJoin[TestTable2](query.Join{
				Join:  "left",
				Alias: "o",
				On:    "t.id = o.id",
			})},
		}

		// Make Select Query
		selectQuery, err := query.Select[TestTable](attr)
		if err != nil {
			t.Fatalf("failed to make select query: %v", err)
		}
		t.Logf("selectQuery: %s", selectQuery)

		// Struct with 2 tables to use with QueryRange as return from range
		type result struct {
			*TestTable
			*TestTable2
		}

		// Get records in range and append to rows slice
		var rows []result
		for s := range QueryRange[result](db, selectQuery, func(errQueryRange error) {
			t.Logf("QueryRange error: %v", errQueryRange)
			err = errQueryRange
		}) {
			t.Logf("Query row: %+v %+v", s.TestTable, s.TestTable2)
			rows = append(rows, s)
		}

		// Check the slice
		for _, row := range rows {
			t.Logf("Query row: %+v %+v", row.TestTable, row.TestTable2)
		}

		require.NoError(t, err)
	})

	t.Run("Query with Joins Select ", func(t *testing.T) {
		// To create query with join with select add Join attributes with
		// Select and Fields
		attr := &query.SelectAttr{
			// Alias of main table
			Alias: "t",

			// Joins with select
			Joins: []query.Join{{
				Join:   "left",
				Alias:  "o",
				Select: "select id, value from TestTable2",
				Fields: []string{"o.id", "o.value"},
				On:     "t.id = o.id",
			}},
		}

		// Make Select Query
		selectQuery, err := query.Select[TestTable](attr)
		if err != nil {
			t.Fatalf("failed to make select query: %v", err)
		}
		t.Logf("selectQuery: %s", selectQuery)

		// Execute query range
		for s := range QueryRange[struct {
			*TestTable
			*TestTable2
		}](db, selectQuery, func(errQueryRange error) {
			t.Logf("QueryRange error: %v", errQueryRange)
			err = errQueryRange
			require.NoError(t, err)
		}) {
			t.Logf("Query row: %+v %+v", s.TestTable, s.TestTable2)
		}

	})

	// 6. Test list range with pointer
	t.Run("ListRange", func(t *testing.T) {
		// List with where clause
		for _, row := range ListRange[TestTable](db, 0, "name ASC", 0,
			Where{"name=", "Bob"}, func(e error) {
				assert.NoError(t, e)
			}) {
			assert.Equal(t, "Bob", row.Name)
		}

		// List with where clause
		for _, row := range ListRange[TestTable](db, 0, "name ASC", 0, 
		Where{"name=", "Alice"}, func(e error) {
			assert.NoError(t, e)
		}) {
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
