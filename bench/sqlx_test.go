// Copyright 2024-2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bench

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// BenchmarkSqlx_Insert measures inserting a single row using sqlx.
func BenchmarkSqlx_Insert(b *testing.B) {
	dbx := newSqlxDB(b)
	defer dbx.Close()
	createSqlxTable(b, dbx)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		_, err := dbx.NamedExec(
			"INSERT INTO sqlx_users (name, email) VALUES (:name, :email)",
			&SqlxUser{
				Name:  fmt.Sprintf("user-%06d", i),
				Email: fmt.Sprintf("user-%06d@example.com", i),
			},
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSqlx_GetByPK measures retrieving a single row by primary key.
func BenchmarkSqlx_GetByPK(b *testing.B) {
	dbx := newSqlxDB(b)
	defer dbx.Close()
	createSqlxTable(b, dbx)
	seedSqlxUsers(b, dbx, 100)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		id := int64(i%100 + 1)
		var u SqlxUser
		err := dbx.Get(&u,
			"SELECT id, name, email FROM sqlx_users WHERE id = ?", id)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSqlx_ListAll measures selecting and scanning 100 rows.
func BenchmarkSqlx_ListAll(b *testing.B) {
	dbx := newSqlxDB(b)
	defer dbx.Close()
	createSqlxTable(b, dbx)
	seedSqlxUsers(b, dbx, 100)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		var users []SqlxUser
		err := dbx.Select(&users,
			"SELECT id, name, email FROM sqlx_users ORDER BY name ASC")
		if err != nil {
			b.Fatal(err)
		}
		if len(users) != 100 {
			b.Fatalf("expected 100 users, got %d", len(users))
		}
	}
}

// BenchmarkSqlx_ListWithLimit measures paginated selection (10 rows offset 50).
func BenchmarkSqlx_ListWithLimit(b *testing.B) {
	dbx := newSqlxDB(b)
	defer dbx.Close()
	createSqlxTable(b, dbx)
	seedSqlxUsers(b, dbx, 100)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		var users []SqlxUser
		err := dbx.Select(&users,
			"SELECT id, name, email FROM sqlx_users ORDER BY name ASC LIMIT ? OFFSET ?",
			10, 50)
		if err != nil {
			b.Fatal(err)
		}
		if len(users) != 10 {
			b.Fatalf("expected 10 users, got %d", len(users))
		}
	}
}

// BenchmarkSqlx_Update measures updating a single row by primary key.
func BenchmarkSqlx_Update(b *testing.B) {
	dbx := newSqlxDB(b)
	defer dbx.Close()
	createSqlxTable(b, dbx)
	seedSqlxUsers(b, dbx, 100)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		id := int64(i%100 + 1)
		_, err := dbx.NamedExec(
			"UPDATE sqlx_users SET email = :email WHERE id = :id",
			&SqlxUser{
				ID:    id,
				Email: fmt.Sprintf("updated-%d@example.com", i),
			},
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSqlx_Delete measures deleting a single row.
// Uses the StopTimer/StartTimer pattern to insert a fresh row before each deletion.
func BenchmarkSqlx_Delete(b *testing.B) {
	dbx := newSqlxDB(b)
	defer dbx.Close()
	createSqlxTable(b, dbx)

	b.ReportAllocs()
	for i := range b.N {
		b.StopTimer()
		res, err := dbx.NamedExec(
			"INSERT INTO sqlx_users (name, email) VALUES (:name, :email)",
			&SqlxUser{
				Name:  fmt.Sprintf("del-%06d", i),
				Email: fmt.Sprintf("del-%06d@example.com", i),
			},
		)
		if err != nil {
			b.Fatal(err)
		}
		id, _ := res.LastInsertId()
		b.StartTimer()

		_, err = dbx.Exec("DELETE FROM sqlx_users WHERE id = ?", id)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ─────────────────── helpers ───────────────────

func newSqlxDB(tb testing.TB) *sqlx.DB {
	db, err := sql.Open("sqlite3", "file::memory:")
	if err != nil {
		tb.Fatal(err)
	}
	return sqlx.NewDb(db, "sqlite3")
}

func createSqlxTable(tb testing.TB, dbx *sqlx.DB) {
	_, err := dbx.Exec(`CREATE TABLE IF NOT EXISTS sqlx_users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE,
		email TEXT
	)`)
	if err != nil {
		tb.Fatal(err)
	}
}

func seedSqlxUsers(tb testing.TB, dbx *sqlx.DB, count int) {
	for i := range count {
		_, err := dbx.NamedExec(
			"INSERT INTO sqlx_users (name, email) VALUES (:name, :email)",
			&SqlxUser{
				Name:  fmt.Sprintf("user-%03d", i),
				Email: fmt.Sprintf("user-%03d@example.com", i),
			},
		)
		if err != nil {
			tb.Fatal(err)
		}
	}
}
