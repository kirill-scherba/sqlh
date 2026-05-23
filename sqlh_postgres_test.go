// Copyright 2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlh

import (
	"bytes"
	"database/sql"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/kirill-scherba/sqlh/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Start PostgreSQL server with docker:
//
//	docker run --rm --name postgres_test -e POSTGRES_PASSWORD=password -p 5432:5432 -d postgres:16
//
// Run tests with PostgreSQL:
//
//	SQLH_TEST_POSTGRES=1 go test -run TestPostgreSQL ./...
//
// The test is gated behind the SQLH_TEST_POSTGRES environment variable so that
// a plain `go test ./...` does not require a running Docker PostgreSQL instance.

// TestPGTable is a test table for PostgreSQL integration tests.
// Uses SERIAL for auto-increment (PostgreSQL native).
type TestPGTable struct {
	ID   int64  `db:"id" db_type:"SERIAL"`
	Name string `db:"name" db_key:"not null"`
	Data []byte `db:"data"`
	_    bool   `db_key:"unique (name)"`
}

// TestPGTable2 is a second test table for join tests with PostgreSQL.
type TestPGTable2 struct {
	ID    int64   `db:"id"`
	Value float64 `db:"value"`
}

// startPostgresContainer starts a Docker PostgreSQL container if one is not
// already running.
func startPostgresContainer(t *testing.T) {
	t.Helper()

	checkCmd := exec.Command("docker", "ps", "-q", "-f", "name=postgres_test")
	var out bytes.Buffer
	checkCmd.Stdout = &out
	if err := checkCmd.Run(); err != nil {
		t.Fatalf("Failed to check running containers: %v", err)
	}

	if out.String() != "" {
		t.Log("PostgreSQL container is already running.")
		return
	}

	t.Log("Starting PostgreSQL container...")
	runCmd := exec.Command("docker", "run", "--rm", "--name", "postgres_test",
		"-e", "POSTGRES_PASSWORD=password",
		"-p", "5432:5432",
		"-d", "postgres:16",
	)
	if err := runCmd.Run(); err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}
}

// waitPostgresReady pings the PostgreSQL server until it responds or a 90-second
// timeout expires.
func waitPostgresReady(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	const maxAttempts = 90
	const sleep = 1 * time.Second

	for i := 0; i < maxAttempts; i++ {
		db, err := sql.Open("postgres", dsn)
		if err == nil {
			if err = db.Ping(); err == nil {
				return db
			}
			db.Close()
		}
		if i == 0 {
			t.Logf("Waiting for PostgreSQL to become available (up to %ds)...", maxAttempts)
		}
		time.Sleep(sleep)
	}
	t.Fatalf("PostgreSQL did not become ready within %ds", maxAttempts)
	return nil
}

// replaceDBName replaces the database name in a PostgreSQL DSN with the given
// new name. Supports both key=value and URI-style DSNs:
//
//	key=value: "host=localhost dbname=postgres" → "host=localhost dbname=test"
//	URI:       "postgres://user:pass@host/db"   → "postgres://user:pass@host/test"
func replaceDBName(dsn, newDBName string) string {
	// Try URI-style first (contains ://)
	if strings.Contains(dsn, "://") {
		// Replace the last path segment (the database name).
		// A URI like postgres://user:pass@host:5432/postgres?sslmode=disable
		// should become postgres://user:pass@host:5432/test?sslmode=disable
		idx := strings.LastIndex(dsn, "/")
		if idx >= 0 {
			rest := dsn[idx+1:]
			// Find where the query string starts (if any)
			qIdx := strings.Index(rest, "?")
			if qIdx >= 0 {
				return dsn[:idx+1] + newDBName + rest[qIdx:]
			}
			return dsn[:idx+1] + newDBName
		}
		return dsn
	}

	// key=value format: replace dbname=<current> with dbname=<new>
	re := regexp.MustCompile(`\bdbname=\S+`)
	return re.ReplaceAllString(dsn, "dbname="+newDBName)
}

// newPostgresDB opens a PostgreSQL connection, creates the test database if it
// does not exist, drops existing tables, creates fresh test tables, and
// returns the connection.
func newPostgresDB(driverName, dataSourceName string) (db *sql.DB, err error) {
	// Connect to the default 'postgres' database first to create our test DB.
	db, err = sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}

	// Create database (ignore "already exists" error).
	_, err = db.Exec("CREATE DATABASE test")
	if err != nil && !strings.Contains(err.Error(), "42P04") {
		db.Close()
		return nil, err
	}
	db.Close()

	// Reconnect to the test database, preserving host/port/user/password/sslmode
	// from the original DSN by replacing only the database name.
	testDSN := replaceDBName(dataSourceName, "test")
	db, err = sql.Open(driverName, testDSN)
	if err != nil {
		return nil, err
	}

	// Drop existing tables to ensure a clean state.
	_, _ = db.Exec("DROP TABLE IF EXISTS testpgtable2")
	_, _ = db.Exec("DROP TABLE IF EXISTS testpgtable")
	_, _ = db.Exec("DROP TABLE IF EXISTS timetest")

	// Create table 1
	createStmt, err := query.TablePG[TestPGTable]()
	if err != nil {
		db.Close()
		return nil, err
	}
	if _, err = db.Exec(createStmt); err != nil {
		db.Close()
		return nil, err
	}

	// Create table 2 (no auto-increment for join test control)
	createStmt2, err := query.TablePG[TestPGTable2]()
	if err != nil {
		db.Close()
		return nil, err
	}
	if _, err = db.Exec(createStmt2); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func TestPostgreSQL(t *testing.T) {
	// The test requires Docker + PostgreSQL. Gate behind SQLH_TEST_POSTGRES so
	// that a plain 'go test ./...' never fails unexpectedly.
	if os.Getenv("SQLH_TEST_POSTGRES") == "" {
		t.Skip("PostgreSQL tests disabled; set SQLH_TEST_POSTGRES=1 to enable")
	}

	// Use custom DSN from env if provided, or start Docker container.
	dsn := os.Getenv("SQLH_POSTGRES_DSN")
	if dsn == "" {
		startPostgresContainer(t)
		dsn = "host=localhost port=5432 user=postgres password=password dbname=postgres sslmode=disable"
	}
	readyDB := waitPostgresReady(t, dsn)
	readyDB.Close()

	// Create test database and tables, then run sub-tests.
	db, err := newPostgresDB("postgres", dsn)
	if err != nil {
		t.Fatalf("Error creating test database: %v", err)
	}
	defer db.Close()

	t.Run("Insert and Verify Get", func(t *testing.T) {
		row1 := TestPGTable{Name: "Alice", Data: []byte("data1")}
		err := Insert(db, row1)
		require.NoError(t, err)

		retrieved, err := Get[TestPGTable](db, Where{"name=", "Alice"})
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		row1.ID = retrieved.ID
		assert.Equal(t, &row1, retrieved)

		err = Delete[TestPGTable](db, Where{"name=", "Alice"})
		require.NoError(t, err)

		_, err = Get[TestPGTable](db, Where{"name=", "Alice"})
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("InsertId with auto-increment", func(t *testing.T) {
		row := TestPGTable{Name: "Bob", Data: []byte("data2")}
		id, err := InsertId(db, row)
		require.NoError(t, err)
		assert.Greater(t, id, int64(0))

		retrieved, err := Get[TestPGTable](db, Where{"name=", "Bob"})
		require.NoError(t, err)
		assert.Equal(t, id, retrieved.ID)
		assert.Equal(t, "Bob", retrieved.Name)

		err = Delete[TestPGTable](db, Where{"name=", "Bob"})
		require.NoError(t, err)
	})

	t.Run("List and ListRows", func(t *testing.T) {
		names := []string{"Charlie", "Diana", "Eve"}
		for _, name := range names {
			err := Insert(db, TestPGTable{Name: name, Data: []byte("list")})
			require.NoError(t, err)
		}
		defer func() {
			for _, name := range names {
				_ = Delete[TestPGTable](db, Where{"name=", name})
			}
		}()

		rows, nextOffset, err := List[TestPGTable](db, 0, "", "name ASC")
		require.NoError(t, err)
		assert.Len(t, rows, 3)
		assert.Equal(t, 3, nextOffset)

		rows, _, err = ListRows[TestPGTable](db, 0, "", "name ASC", 2)
		require.NoError(t, err)
		assert.Len(t, rows, 2)
	})

	t.Run("ListRange iteration", func(t *testing.T) {
		names := []string{"Frank", "Grace"}
		for _, name := range names {
			err := Insert(db, TestPGTable{Name: name, Data: []byte("range")})
			require.NoError(t, err)
		}
		defer func() {
			for _, name := range names {
				_ = Delete[TestPGTable](db, Where{"name=", name})
			}
		}()

		var collected []string
		var iterErr error
		for _, row := range ListRange[TestPGTable](db, 0, "", "name ASC", 0,
			func(e error) { iterErr = e },
		) {
			collected = append(collected, row.Name)
		}
		require.NoError(t, iterErr)
		assert.ElementsMatch(t, names, collected)
	})

	t.Run("Update", func(t *testing.T) {
		row := TestPGTable{Name: "UpdateUser", Data: []byte("original")}
		err := Insert(db, row)
		require.NoError(t, err)
		defer Delete[TestPGTable](db, Where{"name=", "UpdateUser"})

		retrieved, err := Get[TestPGTable](db, Where{"name=", "UpdateUser"})
		require.NoError(t, err)

		updated := TestPGTable{ID: retrieved.ID, Name: "UpdateUser", Data: []byte("modified")}
		err = Update(db, UpdateAttr[TestPGTable]{
			Row:    updated,
			Wheres: []Where{{Field: "id=", Value: retrieved.ID}},
		})
		require.NoError(t, err)

		retrieved, err = Get[TestPGTable](db, Where{"id=", retrieved.ID})
		require.NoError(t, err)
		assert.Equal(t, []byte("modified"), retrieved.Data)
	})

	t.Run("Delete", func(t *testing.T) {
		row := TestPGTable{Name: "DeleteUser", Data: nil}
		err := Insert(db, row)
		require.NoError(t, err)

		err = Delete[TestPGTable](db, Where{"name=", "DeleteUser"})
		require.NoError(t, err)

		_, err = Get[TestPGTable](db, Where{"name=", "DeleteUser"})
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("Set (upsert)", func(t *testing.T) {
		err := Set(db, TestPGTable{Name: "SetUser", Data: []byte("new")},
			Where{"name=", "SetUser"})
		require.NoError(t, err)
		defer Delete[TestPGTable](db, Where{"name=", "SetUser"})

		retrieved, err := Get[TestPGTable](db, Where{"name=", "SetUser"})
		require.NoError(t, err)
		assert.Equal(t, []byte("new"), retrieved.Data)

		err = Set(db, TestPGTable{Name: "SetUser", Data: []byte("upserted")},
			Where{"name=", "SetUser"})
		require.NoError(t, err)

		retrieved, err = Get[TestPGTable](db, Where{"name=", "SetUser"})
		require.NoError(t, err)
		assert.Equal(t, []byte("upserted"), retrieved.Data)
	})

	t.Run("time.Time field", func(t *testing.T) {
		type TimeTest struct {
			ID   int64     `db:"id" db_type:"SERIAL"`
			Name string    `db:"name" db_key:"not null"`
			TS   time.Time `db:"ts"`
		}

		createStmt, err := query.TablePG[TimeTest]()
		require.NoError(t, err)
		_, err = db.Exec(createStmt)
		require.NoError(t, err)
		defer db.Exec("DROP TABLE IF EXISTS timetest")

		now := time.Now().UTC().Truncate(time.Microsecond)
		row := TimeTest{Name: "TimeRow", TS: now}
		err = Insert(db, row)
		require.NoError(t, err)

		retrieved, err := Get[TimeTest](db, Where{"name=", "TimeRow"})
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, "TimeRow", retrieved.Name)
		assert.Equal(t, now, retrieved.TS.UTC().Truncate(time.Microsecond))

		err = Delete[TimeTest](db, Where{"name=", "TimeRow"})
		require.NoError(t, err)
	})

	t.Run("Join with composite struct", func(t *testing.T) {
		type Composite struct {
			*TestPGTable
			*TestPGTable2
		}

		// Insert main table row
		t1 := TestPGTable{Name: "JoinUser", Data: []byte("join")}
		err := Insert(db, t1)
		require.NoError(t, err)

		retrieved, err := Get[TestPGTable](db, Where{"name=", "JoinUser"})
		require.NoError(t, err)
		t1.ID = retrieved.ID
		defer Delete[TestPGTable](db, Where{"id=", t1.ID})

		// Insert joined table row with matching ID
		t2 := TestPGTable2{ID: t1.ID, Value: 42.5}
		err = Insert(db, t2)
		require.NoError(t, err)
		defer Delete[TestPGTable2](db, Where{"id=", t2.ID})

		// Query with join
		var iterErr error
		found := false
		for _, row := range ListRange[Composite](db, 0, "", "t.id ASC", 0,
			SetAlias("t"),
			query.MakeJoin[TestPGTable2](query.Join{
				Join:  "left",
				Alias: "o",
				On:    "t.id = o.id",
			}),
			func(e error) { iterErr = e },
		) {
			if row.TestPGTable != nil && row.TestPGTable.Name == "JoinUser" {
				found = true
				if assert.NotNil(t, row.TestPGTable2, "joined table should exist") {
					assert.Equal(t, 42.5, row.TestPGTable2.Value)
				}
			}
		}
		require.NoError(t, iterErr)
		assert.True(t, found, "expected to find JoinUser in joined result")
	})

	t.Run("DirectReadsWithoutPriorWrite", func(t *testing.T) {
		// Create a fresh table for this sub-test so we start empty.
		type DirectReadTest struct {
			ID   int64  `db:"id" db_type:"SERIAL"`
			Name string `db:"name" db_key:"not null"`
		}
		createStmt, err := query.TablePG[DirectReadTest]()
		require.NoError(t, err)
		_, err = db.Exec(createStmt)
		require.NoError(t, err)
		defer db.Exec("DROP TABLE IF EXISTS directreadtest")

		// Insert data separately, then test direct reads.
		err = Insert(db, DirectReadTest{Name: "Alpha"})
		require.NoError(t, err)
		err = Insert(db, DirectReadTest{Name: "Beta"})
		require.NoError(t, err)

		// List — direct read after inserts in a separate call path.
		rows, _, err := List[DirectReadTest](db, 0, "", "name ASC")
		require.NoError(t, err)
		assert.Len(t, rows, 2)

		// Count — direct read.
		count, err := Count[DirectReadTest](db)
		require.NoError(t, err)
		assert.Equal(t, 2, count)

		// Get — direct read by name.
		row, err := Get[DirectReadTest](db, Where{"name=", "Alpha"})
		require.NoError(t, err)
		require.NotNil(t, row)
		assert.Equal(t, "Alpha", row.Name)

		// ListRange — direct read via iterator.
		var names []string
		var iterErr error
		for _, r := range ListRange[DirectReadTest](db, 0, "", "name ASC", 0,
			func(e error) { iterErr = e },
		) {
			names = append(names, r.Name)
		}
		require.NoError(t, iterErr)
		assert.ElementsMatch(t, []string{"Alpha", "Beta"}, names)
	})

	t.Run("Count", func(t *testing.T) {
		names := []string{"Count1", "Count2", "Count3"}
		for _, name := range names {
			err := Insert(db, TestPGTable{Name: name, Data: nil})
			require.NoError(t, err)
		}
		defer func() {
			for _, name := range names {
				_ = Delete[TestPGTable](db, Where{"name=", name})
			}
		}()

		count, err := Count[TestPGTable](db)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, len(names))
	})
}
