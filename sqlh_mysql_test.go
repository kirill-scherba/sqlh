// Copyright 2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlh

import (
	"bytes"
	"database/sql"
	"os"
	"os/exec"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kirill-scherba/sqlh/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Start MySQL server with docker:
//
//	docker run --rm --name mysql_test -e MYSQL_ROOT_PASSWORD=password -p 3306:3306 -d mysql
//
// Run tests with MySQL:
//
//	SQLH_MYSQL_TEST=1 go test -run TestMySQL ./...
//
// The test is gated behind the SQLH_MYSQL_TEST environment variable so that
// a plain `go test ./...` does not require a running Docker MySQL instance.

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
	// Check if a container named 'mysql_test' is already running.
	checkCmd := exec.Command("docker", "ps", "-q", "-f", "name=mysql_test")
	var out bytes.Buffer
	checkCmd.Stdout = &out
	err := checkCmd.Run()
	if err != nil {
		t.Fatalf("Failed to check running containers: %v", err)
	}

	if out.String() != "" {
		t.Log("MySQL container is already running.")
		return
	}

	t.Log("Starting MySQL container...")
	runCmd := exec.Command("docker", "run", "--rm", "--name", "mysql_test",
		"-e", "MYSQL_ROOT_PASSWORD=password",
		"-p", "3306:3306",
		"-d", "mysql",
	)
	if err := runCmd.Run(); err != nil {
		t.Fatalf("Failed to start MySQL container: %v", err)
	}
}

// waitMySQLReady pings the MySQL server until it responds or a 90-second
// timeout expires. This replaces the old approach of sleeping for a fixed
// duration or looping on a CREATE DATABASE error string.
func waitMySQLReady(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	const maxAttempts = 90
	const sleep = 1 * time.Second

	for i := 0; i < maxAttempts; i++ {
		db, err := sql.Open("mysql", dsn)
		if err == nil {
			if err = db.Ping(); err == nil {
				return db
			}
			db.Close()
		}
		if i == 0 {
			t.Logf("Waiting for MySQL to become available (up to %ds)...", maxAttempts)
		}
		time.Sleep(sleep)
	}
	t.Fatalf("MySQL did not become ready within %ds", maxAttempts)
	return nil
}

// newMySQLDB opens a MySQL connection pointed at the test database, creates
// the test database if it does not exist, creates the test tables, and returns
// the connection.
func newMySQLDB(driverName, dataSourceName string) (db *sql.DB, err error) {
	db, err = sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}

	// Create database
	if _, err = db.Exec("CREATE DATABASE IF NOT EXISTS test"); err != nil {
		return nil, err
	}

	// Execute the USE command to select the database
	if _, err = db.Exec("USE test"); err != nil {
		return nil, err
	}

	// Create table 1
	createStmt, err := query.Table[TestMySQLTable]()
	if err != nil {
		return nil, err
	}
	if _, err = db.Exec(createStmt); err != nil {
		return nil, err
	}

	// Create table 2
	createStmt, err = query.Table[TestMySQLTable2]()
	if err != nil {
		return nil, err
	}
	if _, err = db.Exec(createStmt); err != nil {
		return nil, err
	}

	return db, nil
}

func TestMySQL(t *testing.T) {
	// The test requires Docker + MySQL. Gate behind SQLH_MYSQL_TEST so that
	// a plain 'go test ./...' never fails on CI or on machines without Docker.
	if os.Getenv("SQLH_MYSQL_TEST") == "" {
		t.Skip("MySQL tests disabled; set SQLH_MYSQL_TEST=1 to enable")
	}

	startMySQLContainer(t)

	// Open a connection to the *running* MySQL (root@localhost:3306) and
	// wait for readiness. The /mysql database exists by default in the
	// official mysql image.
	dsn := "root:password@tcp(localhost:3306)/mysql"
	readyDB := waitMySQLReady(t, dsn)
	readyDB.Close()

	// Create test database and tables, then run sub-tests.
	db, err := newMySQLDB("mysql", dsn)
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
