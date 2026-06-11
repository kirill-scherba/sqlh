// Copyright 2024-2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bench

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// BenchmarkRawSQL_Insert measures inserting a single row using raw database/sql.
func BenchmarkRawSQL_Insert(b *testing.B) {
	db := newRawDB(b)
	defer db.Close()
	createRawTable(b, db)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		_, err := db.Exec(
			"INSERT INTO raw_users (name, email) VALUES (?, ?)",
			fmt.Sprintf("user-%06d", i),
			fmt.Sprintf("user-%06d@example.com", i),
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRawSQL_GetByPK measures retrieving a single row by primary key.
func BenchmarkRawSQL_GetByPK(b *testing.B) {
	db := newRawDB(b)
	defer db.Close()
	createRawTable(b, db)
	seedRawUsers(b, db, 100)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		id := int64(i%100 + 1)
		var u RawSQLUser
		err := db.QueryRow(
			"SELECT id, name, email FROM raw_users WHERE id = ?", id,
		).Scan(&u.ID, &u.Name, &u.Email)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRawSQL_ListAll measures selecting and scanning 100 rows.
func BenchmarkRawSQL_ListAll(b *testing.B) {
	db := newRawDB(b)
	defer db.Close()
	createRawTable(b, db)
	seedRawUsers(b, db, 100)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		rows, err := db.Query("SELECT id, name, email FROM raw_users ORDER BY name ASC")
		if err != nil {
			b.Fatal(err)
		}
		var users []RawSQLUser
		for rows.Next() {
			var u RawSQLUser
			if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
				b.Fatal(err)
			}
			users = append(users, u)
		}
		if err := rows.Err(); err != nil {
			b.Fatal(err)
		}
		if len(users) != 100 {
			b.Fatalf("expected 100 users, got %d", len(users))
		}
		rows.Close()
	}
}

// BenchmarkRawSQL_ListWithLimit measures paginated selection (10 rows offset 50).
func BenchmarkRawSQL_ListWithLimit(b *testing.B) {
	db := newRawDB(b)
	defer db.Close()
	createRawTable(b, db)
	seedRawUsers(b, db, 100)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		rows, err := db.Query(
			"SELECT id, name, email FROM raw_users ORDER BY name ASC LIMIT ? OFFSET ?",
			10, 50,
		)
		if err != nil {
			b.Fatal(err)
		}
		var users []RawSQLUser
		for rows.Next() {
			var u RawSQLUser
			if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
				b.Fatal(err)
			}
			users = append(users, u)
		}
		if err := rows.Err(); err != nil {
			b.Fatal(err)
		}
		if len(users) != 10 {
			b.Fatalf("expected 10 users, got %d", len(users))
		}
		rows.Close()
	}
}

// BenchmarkRawSQL_Update measures updating a single row by primary key.
func BenchmarkRawSQL_Update(b *testing.B) {
	db := newRawDB(b)
	defer db.Close()
	createRawTable(b, db)
	seedRawUsers(b, db, 100)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		id := int64(i%100 + 1)
		_, err := db.Exec(
			"UPDATE raw_users SET email = ? WHERE id = ?",
			fmt.Sprintf("updated-%d@example.com", i), id,
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRawSQL_Delete measures deleting a single row.
// Uses the StopTimer/StartTimer pattern to insert a fresh row before each deletion.
func BenchmarkRawSQL_Delete(b *testing.B) {
	db := newRawDB(b)
	defer db.Close()
	createRawTable(b, db)

	b.ReportAllocs()
	for i := range b.N {
		b.StopTimer()
		// Insert a fresh row for this iteration (not timed)
		res, err := db.Exec(
			"INSERT INTO raw_users (name, email) VALUES (?, ?)",
			fmt.Sprintf("del-%06d", i),
			fmt.Sprintf("del-%06d@example.com", i),
		)
		if err != nil {
			b.Fatal(err)
		}
		id, _ := res.LastInsertId()
		b.StartTimer()

		// Timed deletion
		_, err = db.Exec("DELETE FROM raw_users WHERE id = ?", id)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ─────────────────── helpers ───────────────────

func newRawDB(tb testing.TB) *sql.DB {
	db, err := sql.Open("sqlite3", "file::memory:")
	if err != nil {
		tb.Fatal(err)
	}
	return db
}

func createRawTable(tb testing.TB, db *sql.DB) {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS raw_users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE,
		email TEXT
	)`)
	if err != nil {
		tb.Fatal(err)
	}
}

func seedRawUsers(tb testing.TB, db *sql.DB, count int) {
	for i := range count {
		_, err := db.Exec(
			"INSERT INTO raw_users (name, email) VALUES (?, ?)",
			fmt.Sprintf("user-%03d", i),
			fmt.Sprintf("user-%03d@example.com", i),
		)
		if err != nil {
			tb.Fatal(err)
		}
	}
}
