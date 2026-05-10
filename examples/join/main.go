// Copyright 2024-2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// JOIN example: demonstrates querying across related tables using sqlh.
package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/kirill-scherba/sqlh"
	"github.com/kirill-scherba/sqlh/query"
	_ "github.com/mattn/go-sqlite3"
)

// UserTable is the users table struct.
type UserTable struct {
	ID   int64  `db:"id" db_key:"primary key autoincrement"`
	Name string `db:"name"`
}

// OrderTable is the orders table struct.
type OrderTable struct {
	ID     int64   `db:"id" db_key:"primary key autoincrement"`
	UserID int64   `db:"user_id"`
	Total  float64 `db:"total"`
}

func main() {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create tables using sqlh
	if err := sqlh.Create[UserTable](db); err != nil {
		log.Fatal(err)
	}
	if err := sqlh.Create[OrderTable](db); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Tables created")

	// Insert data using sqlh
	sqlh.Insert(db, UserTable{Name: "Alice"})
	sqlh.Insert(db, UserTable{Name: "Bob"})
	sqlh.Insert(db, UserTable{Name: "Charlie"})
	sqlh.Insert(db, OrderTable{UserID: 1, Total: 100})
	sqlh.Insert(db, OrderTable{UserID: 1, Total: 200})
	sqlh.Insert(db, OrderTable{UserID: 2, Total: 150})
	fmt.Println("Data inserted")

	// Method 1: Raw SQL with manual scan
	rows, err := db.Query(
		"SELECT u.id AS user_id, u.name, COUNT(o.id) AS orders " +
			"FROM usertable u LEFT JOIN ordertable o ON u.id = o.user_id " +
			"GROUP BY u.id ORDER BY u.name")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	type UserOrderCount struct {
		UserID int64
		Name   string
		Orders int
	}

	var results []UserOrderCount
	for rows.Next() {
		var r UserOrderCount
		if err := rows.Scan(&r.UserID, &r.Name, &r.Orders); err != nil {
			log.Fatal(err)
		}
		results = append(results, r)
	}
	fmt.Println("Users with order counts (raw SQL + manual scan):")
	for _, u := range results {
		fmt.Printf("  %s (ID=%d): %d orders\n", u.Name, u.UserID, u.Orders)
	}

	// Method 2: Using ListRange with embedded struct for JOIN
	type UserWithOrders struct {
		*UserTable
		*OrderTable
	}

	// Generate JOIN statement using query.Select
	join := query.MakeJoin[OrderTable](query.Join{
		Alias: "o",
		Join:  "left",
		On:    "t.id = o.user_id",
	})

	// Use ListRange with GROUP BY and COUNT — for simplicity iterate and group
	fmt.Println("\nAll users and their orders (ListRange with JOIN):")
	for i, row := range sqlh.ListRange[UserWithOrders](
		db, 0, "t.id", "t.name ASC", 10,
		sqlh.SetAlias("t"),
		join,
	) {
		fmt.Printf("  [%d] User: %s, OrderID: %d\n", i, row.UserTable.Name, row.OrderTable.ID)
	}
}
