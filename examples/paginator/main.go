// Copyright 2024-2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Paginator example: demonstrates paginated listing using sqlh.
package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/kirill-scherba/sqlh"
	_ "github.com/mattn/go-sqlite3"
)

// Item represents an items table with many rows.
type Item struct {
	ID    int64  `db:"id" db_key:"not null primary key autoincrement"`
	Value string `db:"value"`
}

func main() {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create table
	if err := sqlh.Create[Item](db); err != nil {
		log.Fatal(err)
	}

	// Insert 25 items
	for i := range 25 {
		if err := sqlh.Insert(db, Item{Value: fmt.Sprintf("item-%d", i+1)}); err != nil {
			log.Fatal(err)
		}
	}
	fmt.Println("Inserted 25 items")

	// List with pagination: page size = 10 (default)
	offset := 0
	for {
		items, pagination, err := sqlh.ListRows[Item](db, offset, "", "id ASC", 10)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Page offset=%d: got %d items\n", offset, len(items))
		for _, item := range items {
			fmt.Printf("  - %s\n", item.Value)
		}
		if len(items) < 10 {
			break // Last page
		}
		offset = pagination
	}
}