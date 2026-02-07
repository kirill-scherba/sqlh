// Copyright 2024-2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package query

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestSQLQuery(t *testing.T) {

	t.Run("TestQueryArgs", func(t *testing.T) {

		type SomeStruct struct {
			Name string    `db:"name"`
			Cost float64   `db:"cost"`
			Age  int16     `db:"age"`
			Time time.Time `db:"time"`
		}

		var someStruct = SomeStruct{
			Name: "John",
			Cost: 100.0,
			Age:  20,
			Time: time.Now(),
		}
		t.Logf("someStruct: %+v", someStruct)

		// Create args for reading
		args, err := Args(someStruct, false)
		require.NoError(t, err)
		for i := range args {
			t.Logf("args[%d]: %+v %T", i, *args[i].(*any), *args[i].(*any))
		}

		// Update args
		*args[0].(*any) = "Jane"         // Name
		*args[1].(*any) = float32(200.0) // Cost
		*args[2].(*any) = 33             // Age
		*args[3].(*any) = time.Now()     // Time
		for i := range args {
			t.Logf("args[%d]: %+v %T", i, *args[i].(*any), *args[i].(*any))
		}

		// Applay args
		err = ArgsAppay(&someStruct, args)
		require.NoError(t, err)
		t.Logf("someStruct: %+v", someStruct)
	})
}

func TestTable(t *testing.T) {

	type SomeTable struct {
		Name   string    `db:"name" db_type:"varchar(36)" db_key:"not null primary key"`
		DishId int64     `db_key:"references dishtypes(id)"`
		Cost   float64   `db:"cost"`
		Age    int32     `db:"age"`
		Time   time.Time `db:"time"`
		Compl  complex128
		_      bool `db_table_name:"some_table"`
	}

	t.Run("Test table with db_table_name tag and complex type struct field", func(t *testing.T) {

		// Open db
		db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
		require.NoError(t, err, "failed to open database: %v", err)
		defer db.Close()

		// Create table query
		createQuery, err := Table[SomeTable]()
		require.NoError(t, err, "failed to create table query: %v", err)
		t.Log(createQuery)

		// Create table in db
		_, err = db.Exec(createQuery)
		require.NoError(t, err, "failed to create table: %v", err)

		// Insert row query
		insertQuery, err := Insert[SomeTable]()
		require.NoError(t, err, "failed to create insert query: %v", err)
		t.Log(insertQuery)

		// Create row
		var row = SomeTable{
			Name:   "John",
			DishId: 1,
			Cost:   100.0,
			Age:    20,
			Time:   time.Now(),
			Compl:  3.14 + 2i,
		}

		// Get args for write row
		args, err := Args(row, true)
		require.NoError(t, err, "failed to get args: %v", err)

		// Insert row
		_, err = db.Exec(insertQuery, args...)
		require.NoError(t, err, "failed to insert row: %v", err)

		// Select row query
		selectQuery, err := Select[SomeTable](&SelectAttr{
			Paginator: &Paginator{0, 1},
		})
		require.NoError(t, err, "failed to get row: %v", err)
		t.Log(selectQuery)

		// Get args for read row
		var row2 SomeTable
		args, err = Args(row2, false)
		require.NoError(t, err, "failed to get args: %v", err)

		// Read row
		err = db.QueryRow(selectQuery).Scan(args...)
		require.NoError(t, err, "failed to get row: %v", err)

		// Applay args to row2 struct
		err = ArgsAppay(&row2, args)
		require.NoError(t, err, "failed to apply args: %v", err)

		// Print row2 struct
		t.Log(row2)
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

	t.Run("Test Select", func(t *testing.T) {

		attr := &SelectAttr{
			Wheres: []string{"t.name = ?", "t.cost > ?"},
			Alias:  "t",
		}

		selectQuery, err = Select[SomeTable](attr)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(selectQuery)

		// Check query
		if selectQuery != "SELECT t.name, t.cost, t.age, t.time FROM sometable t WHERE t.name = ? AND t.cost > ?;" {
			t.Fatal("Invalid query")
		}
	})

	t.Run("Test Select Execute", func(t *testing.T) {

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

	t.Run("Test Select Join", func(t *testing.T) {

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

	t.Run("Test Select Json Execute", func(t *testing.T) {

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

func TestTableName(t *testing.T) {

	t.Run("Default", func(t *testing.T) {

		type SomeTable struct {
			Name string    `db:"name"`
			Cost float64   `db:"cost"`
			Age  int32     `db:"age"`
			Time time.Time `db:"time"`
		}

		name := Name[SomeTable]()
		require.Equal(t, "sometable", name)

	})

	t.Run("By tag", func(t *testing.T) {

		type SomeTable struct {
			Name string    `db:"name" db_table_name:"some_table"`
			Cost float64   `db:"cost"`
			Age  int32     `db:"age"`
			Time time.Time `db:"time"`
		}

		name := Name[SomeTable]()
		require.Equal(t, "some_table", name)

	})

	t.Run("By method", func(t *testing.T) {

		name := Name[SomeTable]()
		require.Equal(t, "some_table_name", name)

	})

}

type SomeTable struct {
	Name string    `db:"name" db_table_name:"some_table"`
	Cost float64   `db:"cost"`
	Age  int32     `db:"age"`
	Time time.Time `db:"time"`
}

func (t *SomeTable) TableName() string {
	return "some_table_name"
}
