// Copyright 2024-2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bench

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/kirill-scherba/sqlh"
	_ "github.com/mattn/go-sqlite3"
)

// BenchmarkSqlh_Insert measures inserting a single row using sqlh.
func BenchmarkSqlh_Insert(b *testing.B) {
	db := newSqlhDB(b)
	defer db.Close()
	createSqlhTable(b, db)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		err := sqlh.Insert(db, SqlhUser{
			Name:  fmt.Sprintf("user-%06d", i),
			Email: fmt.Sprintf("user-%06d@example.com", i),
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSqlh_GetByPK measures retrieving a single row by primary key.
func BenchmarkSqlh_GetByPK(b *testing.B) {
	db := newSqlhDB(b)
	defer db.Close()
	createSqlhTable(b, db)
	seedSqlhUsers(b, db, 100)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		id := int64(i%100 + 1)
		u, err := sqlh.Get[SqlhUser](db, sqlh.Eq("id", id))
		if err != nil {
			b.Fatal(err)
		}
		if u == nil {
			b.Fatal("expected user, got nil")
		}
	}
}

// BenchmarkSqlh_ListAll measures selecting and scanning 100 rows.
func BenchmarkSqlh_ListAll(b *testing.B) {
	db := newSqlhDB(b)
	defer db.Close()
	createSqlhTable(b, db)
	seedSqlhUsers(b, db, 100)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		users, _, err := sqlh.ListRows[SqlhUser](db, 0, "", "name ASC", 100)
		if err != nil {
			b.Fatal(err)
		}
		if len(users) != 100 {
			b.Fatalf("expected 100 users, got %d", len(users))
		}
	}
}

// BenchmarkSqlh_ListWithLimit measures paginated selection (10 rows offset 50).
func BenchmarkSqlh_ListWithLimit(b *testing.B) {
	db := newSqlhDB(b)
	defer db.Close()
	createSqlhTable(b, db)
	seedSqlhUsers(b, db, 100)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		users, _, err := sqlh.ListRows[SqlhUser](db, 50, "", "name ASC", 10)
		if err != nil {
			b.Fatal(err)
		}
		if len(users) != 10 {
			b.Fatalf("expected 10 users, got %d", len(users))
		}
	}
}

// BenchmarkSqlh_Update measures updating a single row by primary key.
func BenchmarkSqlh_Update(b *testing.B) {
	db := newSqlhDB(b)
	defer db.Close()
	createSqlhTable(b, db)
	seedSqlhUsers(b, db, 100)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		id := int64(i%100 + 1)
		err := sqlh.Update(db, sqlh.UpdateAttr[SqlhUser]{
			Row:    SqlhUser{ID: id, Name: fmt.Sprintf("user-%03d", id-1), Email: fmt.Sprintf("updated-%d@example.com", i)},
			Wheres: []sqlh.Where{sqlh.Eq("id", id)},
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSqlh_Delete measures deleting a single row.
// Uses the StopTimer/StartTimer pattern to insert a fresh row before each deletion.
func BenchmarkSqlh_Delete(b *testing.B) {
	db := newSqlhDB(b)
	defer db.Close()
	createSqlhTable(b, db)

	b.ReportAllocs()
	for i := range b.N {
		b.StopTimer()
		err := sqlh.Insert(db, SqlhUser{
			Name:  fmt.Sprintf("del-%06d", i),
			Email: fmt.Sprintf("del-%06d@example.com", i),
		})
		if err != nil {
			b.Fatal(err)
		}
		// sqlh Insert doesn't return ID directly; look up by name
		u, err := sqlh.Get[SqlhUser](db, sqlh.Eq("name", fmt.Sprintf("del-%06d", i)))
		if err != nil {
			b.Fatal(err)
		}
		if u == nil {
			b.Fatal("expected inserted user, got nil")
		}
		b.StartTimer()

		err = sqlh.Delete[SqlhUser](db, sqlh.Eq("id", u.ID))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ─────────────────── helpers ───────────────────

func newSqlhDB(tb testing.TB) *sql.DB {
	db, err := sql.Open("sqlite3", "file::memory:")
	if err != nil {
		tb.Fatal(err)
	}
	return db
}

func createSqlhTable(tb testing.TB, db *sql.DB) {
	if err := sqlh.Create[SqlhUser](db); err != nil {
		tb.Fatal(err)
	}
}

func seedSqlhUsers(tb testing.TB, db *sql.DB, count int) {
	for i := range count {
		if err := sqlh.Insert(db, SqlhUser{
			Name:  fmt.Sprintf("user-%03d", i),
			Email: fmt.Sprintf("user-%03d@example.com", i),
		}); err != nil {
			tb.Fatal(err)
		}
	}
}
