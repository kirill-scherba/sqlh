// Copyright 2026 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package migrate

import (
	"database/sql"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDB returns an isolated SQLite database file per test.
func testDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", t.TempDir()+"/test.db")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

// --- Structs for migration tests ---

type UserV1 struct {
	ID   int64  `db:"id" db_key:"primary key autoincrement"`
	Name string `db:"name"`
}

type UserV2 struct {
	ID    int64  `db:"id" db_key:"primary key autoincrement"`
	Name  string `db:"name"`
	Email string `db:"email" db_key:"unique"`
	Age   int    `db:"age"`
}

type UserV3 struct {
	ID        int64     `db:"id" db_key:"primary key autoincrement"`
	Name      string    `db:"name"`
	Email     string    `db:"email" db_key:"unique"`
	Age       int       `db:"age"`
	CreatedAt time.Time `db:"created_at"`
	_         string    `db:"-" db_key:"KEY idx_name (name)"`
}

// --- Tests ---

func TestDetectDialect(t *testing.T) {
	db := testDB(t)
	d := DetectDialect(db)
	assert.Equal(t, SQLite, d)
}

func TestValidatePlan(t *testing.T) {
	t.Run("empty plan", func(t *testing.T) {
		plan := Plan{}
		err := validatePlan(plan)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("duplicate version", func(t *testing.T) {
		plan := Plan{
			Raw("dup1", V(1), "SELECT 1"),
			Raw("dup2", V(1), "SELECT 2"),
		}
		err := validatePlan(plan)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate")
	})

	t.Run("unsorted versions", func(t *testing.T) {
		plan := Plan{
			Raw("step2", V(2), "SELECT 2"),
			Raw("step1", V(1), "SELECT 1"),
		}
		err := validatePlan(plan)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ascending")
	})

	t.Run("valid plan", func(t *testing.T) {
		plan := Plan{
			Raw("step1", V(1), "SELECT 1"),
			Raw("step2", V(2), "SELECT 2"),
		}
		err := validatePlan(plan)
		require.NoError(t, err)
	})
}

func TestFromStruct(t *testing.T) {
	m := FromStruct[UserV1]("", V(1))
	require.Equal(t, V(1), m.Version())
	require.Equal(t, "v1", m.Name())

	sql, err := m.SQL(SQLite)
	require.NoError(t, err)
	assert.Contains(t, sql, "CREATE TABLE IF NOT EXISTS userv1")
	assert.Contains(t, sql, "id integer primary key autoincrement")
	assert.Contains(t, sql, "name text")
}

func TestFromStructCustomName(t *testing.T) {
	m := FromStruct[UserV1]("users", V(1))
	require.Equal(t, "users_v1", m.Name())

	sql, err := m.SQL(SQLite)
	require.NoError(t, err)
	assert.Contains(t, sql, "CREATE TABLE IF NOT EXISTS users")
}

func TestRaw(t *testing.T) {
	m := Raw("idx_email", V(3), "CREATE INDEX IF NOT EXISTS idx_email ON users(email);")
	require.Equal(t, "idx_email", m.Name())
	require.Equal(t, V(3), m.Version())

	sql, err := m.SQL(SQLite)
	require.NoError(t, err)
	assert.Equal(t, "CREATE INDEX IF NOT EXISTS idx_email ON users(email);", sql)
}

func TestApplyFromStruct(t *testing.T) {
	db := testDB(t)
	plan := Plan{
		FromStruct[UserV1]("", V(1)),
	}

	err := Apply(db, plan, Options{})
	require.NoError(t, err)

	// Verify table exists by inserting.
	_, err = db.Exec("INSERT INTO userv1 (name) VALUES (?)", "Alice")
	require.NoError(t, err)

	// Verify _migrations record.
	var version int
	var name string
	err = db.QueryRow("SELECT version, name FROM _migrations WHERE version = 1").Scan(&version, &name)
	require.NoError(t, err)
	assert.Equal(t, 1, version)
	assert.Equal(t, "v1", name)
}

func TestApplyIdempotent(t *testing.T) {
	db := testDB(t)
	plan := Plan{
		FromStruct[UserV1]("", V(1)),
	}

	// First apply.
	err := Apply(db, plan, Options{})
	require.NoError(t, err)

	// Second apply — should be a no-op.
	err = Apply(db, plan, Options{})
	require.NoError(t, err)

	// Verify _migrations still has exactly one record.
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM _migrations").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestApplyDryRun(t *testing.T) {
	db := testDB(t)
	// Pre-create the V1 table so Diff has a live schema to compare against.
	_, err := db.Exec("CREATE TABLE userv2 (id integer primary key autoincrement, name text)")
	require.NoError(t, err)

	plan := Plan{
		Diff[UserV2]("userv2", V(2), AutoAdd()),
	}

	// Capture stdout.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = Apply(db, plan, Options{DryRun: true})
	_ = w.Close()
	os.Stdout = old
	require.NoError(t, err)

	out, _ := io.ReadAll(r)
	output := string(out)
	assert.Contains(t, output, "ALTER TABLE userv2 ADD COLUMN email text")
	assert.Contains(t, output, "ALTER TABLE userv2 ADD COLUMN age integer")
	assert.Contains(t, output, "Dry run complete")

	// Verify schema was NOT changed.
	cols, err := TableColumns(db, "userv2", SQLite)
	require.NoError(t, err)
	require.Len(t, cols, 2) // id, name — unchanged
}

func TestApplyBackupHook(t *testing.T) {
	db := testDB(t)
	callCount := 0

	plan := Plan{
		FromStruct[UserV1]("", V(1)),
		Raw("noop", V(2), "SELECT 1"),
	}

	err := Apply(db, plan, Options{
		Backup: func(db *sql.DB) error {
			callCount++
			return nil
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestApplyBackupHookError(t *testing.T) {
	db := testDB(t)
	plan := Plan{
		FromStruct[UserV1]("", V(1)),
		Raw("noop", V(2), "SELECT 1"),
	}

	callCount := 0
	err := Apply(db, plan, Options{
		Backup: func(db *sql.DB) error {
			callCount++
			if callCount == 2 {
				return errors.New("backup failed")
			}
			return nil
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "backup hook")
	assert.Equal(t, 2, callCount)
}

func TestApplyMultipleMigrations(t *testing.T) {
	db := testDB(t)
	// Create V1 table.
	_, err := db.Exec("CREATE TABLE users (id integer primary key autoincrement, name text)")
	require.NoError(t, err)

	// Apply V2 diff.
	plan := Plan{
		Diff[UserV2]("users", V(2), AutoAdd()),
	}
	err = Apply(db, plan, Options{})
	require.NoError(t, err)

	// Verify all columns exist.
	cols, err := TableColumns(db, "users", SQLite)
	require.NoError(t, err)

	colNames := make(map[string]struct{})
	for _, c := range cols {
		colNames[c.Name] = struct{}{}
	}
	assert.Contains(t, colNames, "id")
	assert.Contains(t, colNames, "name")
	assert.Contains(t, colNames, "email")
	assert.Contains(t, colNames, "age")

	// Verify _migrations has one record.
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM _migrations").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestDiffAutoAdd(t *testing.T) {
	db := testDB(t)

	// Create V1 table manually.
	_, err := db.Exec("CREATE TABLE userv2 (id integer primary key autoincrement, name text)")
	require.NoError(t, err)

	// Run Diff V2.
	m := Diff[UserV2]("userv2", V(2), AutoAdd())
	sql, err := m.(*diffMigration[UserV2]).SQLWithDB(db, SQLite)
	require.NoError(t, err)
	assert.Contains(t, sql, "ALTER TABLE userv2 ADD COLUMN email text")
	assert.Contains(t, sql, "ALTER TABLE userv2 ADD COLUMN age integer")
}

func TestDiffNoChanges(t *testing.T) {
	db := testDB(t)

	// Create table matching UserV1 exactly.
	_, err := db.Exec("CREATE TABLE userv1 (id integer primary key autoincrement, name text)")
	require.NoError(t, err)

	m := Diff[UserV1]("userv1", V(2), AutoAdd())
	sql, err := m.(*diffMigration[UserV1]).SQLWithDB(db, SQLite)
	require.NoError(t, err)
	assert.Empty(t, sql)
}

func TestApplyRollback(t *testing.T) {
	db := testDB(t)
	plan := Plan{
		FromStruct[UserV1]("", V(1)),
		Raw("bad", V(2), "INVALID SQL HERE"),
	}

	err := Apply(db, plan, Options{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "execute SQL")

	// Verify V1 was rolled back (transaction rolled back).
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM _migrations WHERE version = 1").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestIntrospectionSQLite(t *testing.T) {
	db := testDB(t)
	_, err := db.Exec("CREATE TABLE test_table (id integer PRIMARY KEY, name text NOT NULL, age integer)")
	require.NoError(t, err)

	cols, err := TableColumns(db, "test_table", SQLite)
	require.NoError(t, err)
	require.Len(t, cols, 3)

	assert.Equal(t, "id", cols[0].Name)
	assert.Equal(t, "integer", cols[0].Type)
	// SQLite PRAGMA table_info does NOT set notnull=1 for PRIMARY KEY columns.
	assert.False(t, cols[0].NotNull)

	assert.Equal(t, "name", cols[1].Name)
	assert.Equal(t, "text", cols[1].Type)
	assert.True(t, cols[1].NotNull)

	assert.Equal(t, "age", cols[2].Name)
	assert.Equal(t, "integer", cols[2].Type)
	assert.False(t, cols[2].NotNull)
}

func TestApplyWithRaw(t *testing.T) {
	db := testDB(t)
	plan := Plan{
		FromStruct[UserV1]("", V(1)),
		Raw("add index", V(2), "CREATE INDEX IF NOT EXISTS idx_name ON userv1(name);"),
	}

	err := Apply(db, plan, Options{})
	require.NoError(t, err)

	// Verify index exists.
	var name string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='index' AND name='idx_name'").Scan(&name)
	require.NoError(t, err)
	assert.Equal(t, "idx_name", name)
}

func TestApplyPlanValidation(t *testing.T) {
	db := testDB(t)
	plan := Plan{
		Raw("step2", V(2), "SELECT 1"),
		Raw("step1", V(1), "SELECT 1"),
	}
	err := Apply(db, plan, Options{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ascending")
}

func TestDiffAutoAddIndexes(t *testing.T) {
	db := testDB(t)

	// Create V2 table manually.
	_, err := db.Exec("CREATE TABLE userv3 (id integer primary key autoincrement, name text, email text, age integer, created_at timestamp)")
	require.NoError(t, err)

	// Run Diff V3 — should add the index from _ sentinel field.
	m := Diff[UserV3]("userv3", V(3), AutoAdd())
	sql, err := m.(*diffMigration[UserV3]).SQLWithDB(db, SQLite)
	require.NoError(t, err)
	assert.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_name ON userv3 (name)")
}

func TestDiffTableNotFound(t *testing.T) {
	db := testDB(t)
	m := Diff[UserV1]("nonexistent", V(1), AutoAdd())
	_, err := m.(*diffMigration[UserV1]).SQLWithDB(db, SQLite)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "introspect")
}
