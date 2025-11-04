// Copyright 2024 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package query

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestSQLQuery(t *testing.T) {

	t.Run("TestQueryArgs", func(t *testing.T) {

		type SomeStruct struct {
			Name string    `db:"name"`
			Cost float64   `db:"cost"`
			Age  int32     `db:"age"`
			Time time.Time `db:"time"`
		}

		var someStruct = SomeStruct{
			Name: "John",
			Cost: 100.0,
			Age:  20,
			Time: time.Now(),
		}

		// Create args
		args, err := Args(someStruct, false)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("someStruct: %+v", someStruct)

		// Update args
		*args[0].(*any) = "Jane"
		*args[1].(*any) = float32(200.0)
		*args[2].(*any) = int8(30)
		*args[3].(*any) = time.Now()

		// Applay args
		err = ArgsAppay(&someStruct, args)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("someStruct: %+v", someStruct)
	})

}

func TestSelect(t *testing.T) {

	var selectQuery string
	var err error

	type SomeTable struct {
		Name string    `db:"name"`
		Cost float64   `db:"cost"`
		Age  int32     `db:"age"`
		Time time.Time `db:"time"`
	}

	type OtherTable struct {
		Name string  `db:"name"`
		Cost float64 `db:"cost"`
	}

	// Create test db
	createDB := func() (db *sql.DB, err error) {
		// Open db
		db, err = sql.Open("sqlite3", "file::memory:?cache=shared")
		if err != nil {
			err = fmt.Errorf("failed to open database: %v", err)
			return
		}

		// Create table SomeTable
		tbl, _ := Table[SomeTable]()
		_, err = db.Exec(tbl)
		if err != nil {
			err = fmt.Errorf("failed to create table: %v", err)
			return
		}

		// Insert data
		ins, _ := Insert[SomeTable]()
		_, err = db.Exec(ins, "John", 100.0, 20, time.Now())
		if err != nil {
			err = fmt.Errorf("failed to insert data: %v", err)
			return
		}

		// Create table OtherTable
		tbl, _ = Table[OtherTable]()
		_, err = db.Exec(tbl)
		if err != nil {
			err = fmt.Errorf("failed to create table: %v", err)
			return
		}

		// Insert data
		ins, _ = Insert[OtherTable]()
		_, err = db.Exec(ins, "John", 200.0)
		if err != nil {
			err = fmt.Errorf("failed to insert data: %v", err)
			return
		}

		return
	}

	t.Run("TestSelect", func(t *testing.T) {

		attr := &SelectAttr{
			Wheres: []string{"name = ?", "cost > ?"},
			// Alias:  "t",
		}

		selectQuery, err = Select[SomeTable](attr)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(selectQuery)
	})

	t.Run("TestSelectExecute", func(t *testing.T) {

		// Create db
		db, err := createDB()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		// Execute query
		sqlRows, err := db.Query(selectQuery, "John", 0)
		if err != nil {
			t.Fatal(err)
		}
		defer sqlRows.Close()

		// Get rows
		for sqlRows.Next() {
			row := SomeTable{}

			// Get arguments and scan row
			args, _ := Args(row, false)
			if err = sqlRows.Scan(args...); err != nil {
				err = fmt.Errorf("failed to scan row: %v", err)
				t.Fatal(err)
			}

			// Apply scanned arguments to the row struct fields
			err = ArgsAppay(&row, args)
			if err != nil {
				err = fmt.Errorf("failed to apply arguments: %v", err)
				t.Fatal(err)
			}

			t.Logf("row: %+v", row)
		}
	})

	t.Run("TestSelectJoin", func(t *testing.T) {

		attr := &SelectAttr{
			Wheres: []string{"t.name = ?", "t.cost > ?"},
			Alias:  "t",
			Joins: []Join{MakeJoin[OtherTable](Join{
				Join:  "left",
				Alias: "o",
				On:    "t.name = o.name",
			})},
		}

		selectQuery, err = Select[SomeTable](attr)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(selectQuery)
	})

	t.Run("TestSelectJsonExecute", func(t *testing.T) {

		// Create db
		db, err := createDB()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		// Execute query
		sqlRows, err := db.Query(selectQuery, "John", 0)
		if err != nil {
			t.Fatal(err)
		}
		defer sqlRows.Close()

		// Get rows
		var l = 0
		for sqlRows.Next() {
			// Row structs to get values from scanned row
			someTable := SomeTable{}
			otherTable := OtherTable{}

			// Get arguments from structs
			args1, _ := Args(someTable, false)
			args2, _ := Args(otherTable, false)
			args := append(args1, args2...)

			// Scan row
			if err = sqlRows.Scan(args...); err != nil {
				err = fmt.Errorf("failed to scan row: %v", err)
				t.Fatal(err)
			}

			// Apply scanned arguments to the structs
			err = ArgsAppay(&someTable, args1)
			if err != nil {
				err = fmt.Errorf("failed to apply arguments: %v", err)
				t.Fatal(err)
			}
			err = ArgsAppay(&otherTable, args2)
			if err != nil {
				err = fmt.Errorf("failed to apply arguments: %v", err)
				t.Fatal(err)
			}

			// Log scanned row by struct
			t.Logf("row someTable: %+v", someTable)
			t.Logf("row otherTable: %+v", otherTable)
			l++
		}

		t.Logf("sqlRows len: %v", l)
	})
}
