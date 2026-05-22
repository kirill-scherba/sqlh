// Copyright 2024-2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package query

import (
	"reflect"
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

// BenchmarkArgsReadApply_value benchmarks Args+ArgsApply when the source
// struct is passed by value (non-addressable). This exercises the fallback
// copy path and reflects the pre-optimisation baseline.
func BenchmarkArgsReadApply_value(b *testing.B) {
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

// BenchmarkArgsReadApply_addr benchmarks Args+ArgsApply when the source
// struct is passed by pointer (addressable). This exercises the fast path
// where Args returns typed pointers directly to the struct fields,
// eliminating per-field heap allocations. This is the typical code path
// taken inside QueryRange.
func BenchmarkArgsReadApply_addr(b *testing.B) {
	row := &benchmarkQueryUser{}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		args, err := Args(row, false)
		if err != nil {
			b.Fatal(err)
		}

		// Simulate sql.Rows.Scan: write typed values through the
		// pointers that Args returned.
		for i, arg := range args {
			elem := reflect.ValueOf(arg).Elem()
			switch i {
			case 0:
				elem.SetInt(42)
			case 1:
				elem.SetString("Alice")
			case 2:
				elem.SetString("alice@example.com")
			case 3:
				elem.SetBool(true)
			case 4:
				elem.SetFloat(98.5)
			case 5:
				elem.Set(reflect.ValueOf(time.Unix(1_700_000_000, 0)))
			case 6:
				elem.SetBytes([]byte("payload"))
			}
		}

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
