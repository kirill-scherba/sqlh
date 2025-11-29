// Copyright 2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlh

import (
	"bytes"
	"database/sql"
	"fmt"
	"os/exec"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kirill-scherba/sqlh/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Start MySQL server with docker:
//   docker run --rm --name mysql_test -e MYSQL_ROOT_PASSWORD=password -p 3306:3306 -d mysql

type TestMySQLTable struct {
	ID   int64  `db:"id" db_key:"not null primary key AUTO_INCREMENT"`
	Name string `db:"name" db_key:"not null"`
	Data []byte `db:"data"`
	// Add key for name column
	_ bool `db_key:"unique KEY(name(255))"`
}

type TestMySQLTable2 struct {
	ID    int64   `db:"id" db_key:"unique not null primary key"`
	Value float64 `db:"value"`
}

func startMySQLContainer(t *testing.T) {
	// Check if the container with the name 'mysql' is running
	checkCmd := exec.Command("docker", "ps", "-q", "-f", "name=mysql_test")
	var out bytes.Buffer
	checkCmd.Stdout = &out
	err := checkCmd.Run()
	if err != nil {
		t.Fatalf("Failed to check running containers: %v", err)
	}

	if out.String() != "" {
		t.Log("Container 'mysql' is already running.")
	} else {
		t.Log("Container 'mysql' is not running. Starting...")
		runCmd := exec.Command("docker", "run", "--rm", "--name", "mysql_test", "-e", "MYSQL_ROOT_PASSWORD=password", "-p", "3306:3306", "-d", "mysql")
		err := runCmd.Run()
		if err != nil {
			t.Fatalf("Failed to start MySQL container: %v", err)
		}
		// t.Log("MySQL container starting...")
		// time.Sleep(60 * time.Second)
	}
}

func newMySQLDB(driverName, dataSourceName string) (db *sql.DB, err error) {

	db, err = sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}

	// Create database
	for {
		_, err = db.Exec("CREATE DATABASE IF NOT EXISTS test")
		if err != nil {
			if err.Error() == "invalid connection" {
				// Wait for MySQL server to start
				fmt.Println("Waiting 5 seconds for MySQL server to start...")
				time.Sleep(5 * time.Second)
				continue
			}
			return nil, err
		}
		break
	}

	// Execute the USE command to select the database
	_, err = db.Exec("USE test")
	if err != nil {
		return nil, err
	}

	// Create table 1
	createStmt, err := query.Table[TestMySQLTable]()
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(createStmt)
	if err != nil {
		return nil, err
	}

	// Create table 2
	createStmt, err = query.Table[TestMySQLTable2]()
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(createStmt)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func TestMySQL(t *testing.T) {

	startMySQLContainer(t)

	// Create test mysql database
	db, err := newMySQLDB("mysql", "root:password@tcp(localhost:3306)/mysql")
	if err != nil {
		t.Fatalf("Error creating test database: %v", err)
	}
	defer db.Close()

	// Run test inserts
	t.Run("Insert and Verify Get", func(t *testing.T) {

		// Insert
		user1 := TestMySQLTable{Name: "Alice", Data: []byte("data1")}
		err := Insert(db, user1)
		require.NoError(t, err)

		// Get and verify
		retrievedUser, err := Get[TestMySQLTable](db, Where{"name=", "Alice"})
		require.NoError(t, err)
		require.NotNil(t, retrievedUser)

		user1.ID = retrievedUser.ID
		assert.Equal(t, &user1, retrievedUser)

		// Delete
		err = Delete[TestMySQLTable](db, Where{"name=", "Alice"})
		require.NoError(t, err)
	})

	t.Run("Check join query with alias", func(t *testing.T) {

		// Insert to table 1
		user1 := TestMySQLTable{Name: "Alice", Data: []byte("data1")}
		err := Insert(db, user1)
		require.NoError(t, err)
		require.NotNil(t, user1)
		defer Delete[TestMySQLTable](db, Where{"name=", user1.Name})
		retrievedUser, err := Get[TestMySQLTable](db, Where{"name=", user1.Name})
		require.NoError(t, err)
		require.NotNil(t, retrievedUser)
		user1.ID = retrievedUser.ID
		//
		user2 := TestMySQLTable{Name: "Mike", Data: []byte("data2")}
		err = Insert(db, user2)
		require.NoError(t, err)
		require.NotNil(t, user2)
		defer Delete[TestMySQLTable](db, Where{"name=", user2.Name})
		retrievedUser, err = Get[TestMySQLTable](db, Where{"name=", user2.Name})
		require.NoError(t, err)
		require.NotNil(t, retrievedUser)
		user2.ID = retrievedUser.ID

		// Insert to table 2
		tbl2 := TestMySQLTable2{ID: user1.ID, Value: 1.23}
		err = Insert(db, tbl2)
		require.NoError(t, err)
		defer Delete[TestMySQLTable2](db, Where{"id=", user1.ID})
		//
		tbl2 = TestMySQLTable2{ID: user2.ID, Value: 4.56}
		err = Insert(db, tbl2)
		require.NoError(t, err)
		defer Delete[TestMySQLTable2](db, Where{"id=", user2.ID})

		// Create select attributes and select query with join
		attr := &query.SelectAttr{
			Alias: "tbl1",
			Joins: []query.Join{query.MakeJoin[TestMySQLTable2](query.Join{
				Join:  "left",
				Alias: "tbl2",
				On:    "tbl1.id = tbl2.id",
			})},
		}
		selectQuery, err := query.Select[TestMySQLTable](attr)
		require.NoError(t, err)
		t.Logf("selectQuery: %s", selectQuery)

		// Execute select query range
		for row := range QueryRange[struct {
			*TestMySQLTable
			*TestMySQLTable2
		}](db, selectQuery, func(e error) {
			t.Logf("QueryRange error: %v", e)
			err = e
		}) {
			t.Logf("Query row: %+v %+v", row.TestMySQLTable, row.TestMySQLTable2)
		}

	})
}
