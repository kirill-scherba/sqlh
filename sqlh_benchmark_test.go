// Copyright 2024-2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlh

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

type benchmarkListUser struct {
	ID    int64  `db:"id" db_key:"not null primary key autoincrement"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

var (
	benchmarkListRows []benchmarkListUser
	benchmarkListNext int
	benchmarkListErr  error
)

func BenchmarkListRows(b *testing.B) {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { db.Close() })

	if err := Create[benchmarkListUser](db); err != nil {
		b.Fatal(err)
	}

	for i := range 100 {
		err := Insert(db, benchmarkListUser{
			Name:  fmt.Sprintf("user-%03d", i),
			Email: fmt.Sprintf("user-%03d@example.com", i),
		})
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		benchmarkListRows, benchmarkListNext, benchmarkListErr = ListRows[benchmarkListUser](
			db, 0, "", "name ASC", 20,
			Where{Field: "id>", Value: 0},
		)
		if benchmarkListErr != nil {
			b.Fatal(benchmarkListErr)
		}
		if len(benchmarkListRows) != 20 || benchmarkListNext != 20 {
			b.Fatalf("unexpected result: len=%d next=%d", len(benchmarkListRows), benchmarkListNext)
		}
	}
}
