// Copyright 2024-2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package query

import (
	"testing"
	"time"
)

type benchmarkQueryUser struct {
	ID        int64     `db:"id" db_key:"primary key autoincrement"`
	Name      string    `db:"name"`
	Email     string    `db:"email"`
	Active    bool      `db:"active"`
	Score     float64   `db:"score"`
	CreatedAt time.Time `db:"created_at"`
	Data      []byte    `db:"data"`
}

var (
	benchmarkQueryArgs []any
	benchmarkQuerySQL  string
	benchmarkQueryErr  error
)

func BenchmarkArgsWrite(b *testing.B) {
	row := benchmarkQueryUser{
		ID:        42,
		Name:      "Alice",
		Email:     "alice@example.com",
		Active:    true,
		Score:     98.5,
		CreatedAt: time.Unix(1_700_000_000, 0),
		Data:      []byte("payload"),
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		benchmarkQueryArgs, benchmarkQueryErr = Args(row, true)
		if benchmarkQueryErr != nil {
			b.Fatal(benchmarkQueryErr)
		}
	}
}

func BenchmarkArgsReadAndApply(b *testing.B) {
	src := benchmarkQueryUser{}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		args, err := Args(src, false)
		if err != nil {
			b.Fatal(err)
		}

		*args[0].(*any) = int64(42)
		*args[1].(*any) = "Alice"
		*args[2].(*any) = "alice@example.com"
		*args[3].(*any) = true
		*args[4].(*any) = 98.5
		*args[5].(*any) = time.Unix(1_700_000_000, 0)
		*args[6].(*any) = []byte("payload")

		var dst benchmarkQueryUser
		benchmarkQueryErr = ArgsApply(&dst, args)
		if benchmarkQueryErr != nil {
			b.Fatal(benchmarkQueryErr)
		}
	}
}

func BenchmarkSelect(b *testing.B) {
	attr := &SelectAttr{
		Wheres:  []string{"active=?", "score>?"},
		OrderBy: "created_at DESC",
		Paginator: &Paginator{
			Offset: 100,
			Limit:  20,
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		benchmarkQuerySQL, benchmarkQueryErr = Select[benchmarkQueryUser](attr)
		if benchmarkQueryErr != nil {
			b.Fatal(benchmarkQueryErr)
		}
	}
}
