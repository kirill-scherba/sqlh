// Copyright 2024-2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Basic CRUD example using the sqlh package.
package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/kirill-scherba/sqlh"
	_ "github.com/mattn/go-sqlite3"
)

// User represents a users table.
type User struct {
	ID    int64  `db:"id" db_key:"not null primary key autoincrement"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

func main() {
	// Open in-memory SQLite database
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create table from struct
	if err := sqlh.Create[User](db); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Table created")

	// Insert users
	alice := User{Name: "Alice", Email: "alice@example.com"}
	if err := sqlh.Insert(db, alice); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Inserted Alice")

	bob := User{Name: "Bob", Email: "bob@example.com"}
	if err := sqlh.Insert(db, bob); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Inserted Bob")

	// Get user by name
	user, err := sqlh.Get[User](db, sqlh.Eq("name", "Alice"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found: %s <%s> (ID=%d)\n", user.Name, user.Email, user.ID)

	// List users
	users, _, err := sqlh.List[User](db, 0, "", "name ASC")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("All users:")
	for _, u := range users {
		fmt.Printf("  - %s <%s>\n", u.Name, u.Email)
	}

	// Update user
	if err := sqlh.Update(db, sqlh.UpdateAttr[User]{
		Row:    User{ID: user.ID, Name: "Alice", Email: "alice.new@example.com"},
		Wheres: []sqlh.Where{sqlh.Eq("id", user.ID)},
	}); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Updated Alice's email")

	// Verify update
	updated, _ := sqlh.Get[User](db, sqlh.Eq("id", user.ID))
	fmt.Printf("Updated: %s <%s>\n", updated.Name, updated.Email)

	// Count users
	count, err := sqlh.Count[User](db)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Total users: %d\n", count)

	// Delete user
	if err := sqlh.Delete[User](db, sqlh.Eq("name", "Bob")); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Deleted Bob")

	// Verify deletion
	count, _ = sqlh.Count[User](db)
	fmt.Printf("Remaining users: %d\n", count)
}