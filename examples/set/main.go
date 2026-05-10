// Copyright 2024-2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Set example: demonstrates InsertOrUpdate (upsert) operations using sqlh.
package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/kirill-scherba/sqlh"
	_ "github.com/mattn/go-sqlite3"
)

// Product represents a products table with a unique name.
type Product struct {
	ID    int64   `db:"id" db_key:"not null primary key autoincrement"`
	Name  string  `db:"name" db_key:"unique"`
	Price float64 `db:"price"`
	Stock int     `db:"stock"`
}

func main() {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create table
	if err := sqlh.Create[Product](db); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Table created")

	// Insert new product
	sqlh.Insert(db, Product{Name: "Laptop", Price: 999.99, Stock: 10})
	fmt.Println("Inserted Laptop")

	// Upsert: update stock if exists (Set = InsertOrUpdate)
	sqlh.Set(db, Product{Name: "Laptop", Price: 999.99, Stock: 15},
		sqlh.Where{Field: "name=", Value: "Laptop"})
	fmt.Println("Updated Laptop stock to 15")

	// Insert another product
	sqlh.Insert(db, Product{Name: "Mouse", Price: 29.99, Stock: 50})

	// List all products
	products, _, err := sqlh.List[Product](db, 0, "", "name ASC")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("All products:")
	for _, p := range products {
		fmt.Printf("  - %s: $%.2f (stock: %d)\n", p.Name, p.Price, p.Stock)
	}
}