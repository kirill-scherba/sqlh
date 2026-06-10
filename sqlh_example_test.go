// Copyright 2024-2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlh

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// ExampleUser represents the users table for examples.
type ExampleUser struct {
	ID    int64  `db:"id" db_key:"not null primary key autoincrement"`
	Name  string `db:"name" db_key:"unique"`
	Email string `db:"email"`
}

// Example demonstrates basic CRUD operations: Create, Insert, Get, List, Update, Delete.
func Example() {
	// Open in-memory SQLite database
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create table from struct
	if err := Create[ExampleUser](db); err != nil {
		log.Fatal(err)
	}

	// Insert users
	if err := Insert(db, ExampleUser{Name: "Alice", Email: "alice@example.com"}); err != nil {
		log.Fatal(err)
	}
	if err := Insert(db, ExampleUser{Name: "Bob", Email: "bob@example.com"}); err != nil {
		log.Fatal(err)
	}

	// Get user by name
	user, err := Get[ExampleUser](db, Where{Field: "name=", Value: "Alice"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found: %s <%s>\n", user.Name, user.Email)

	// List all users
	users, _, err := List[ExampleUser](db, 0, "", "name ASC")
	if err != nil {
		log.Fatal(err)
	}
	for _, u := range users {
		fmt.Printf("User: %s <%s>\n", u.Name, u.Email)
	}

	// Update user's email
	if err := Update(db, UpdateAttr[ExampleUser]{
		Row:    ExampleUser{ID: user.ID, Name: "Alice", Email: "alice.new@example.com"},
		Wheres: []Where{{Field: "id=", Value: user.ID}},
	}); err != nil {
		log.Fatal(err)
	}

	// Delete user
	if err := Delete[ExampleUser](db, Where{Field: "name=", Value: "Bob"}); err != nil {
		log.Fatal(err)
	}

	// Output:
	// Found: Alice <alice@example.com>
	// User: Alice <alice@example.com>
	// User: Bob <bob@example.com>
}

// ExampleListRows demonstrates explicit paginated listing using ListRows.
func ExampleListRows() {
	// Open in-memory SQLite database
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create table and insert sample data
	if err := Create[ExampleUser](db); err != nil {
		log.Fatal(err)
	}
	names := []string{"Alice", "Bob", "Charlie", "Dave", "Eve"}
	for _, name := range names {
		if err := Insert(db, ExampleUser{Name: name, Email: name + "@example.com"}); err != nil {
			log.Fatal(err)
		}
	}

	// Paginate through users in pages of 3
	offset := 0
	page := 1
	for {
		users, next, err := ListRows[ExampleUser](db, offset, "", "name ASC", 3)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Page %d (offset=%d): %d users\n", page, offset, len(users))
		for _, u := range users {
			fmt.Printf("  - %s\n", u.Name)
		}
		if len(users) < 3 {
			break
		}
		offset = next
		page++
	}

	// Output:
	// Page 1 (offset=0): 3 users
	//   - Alice
	//   - Bob
	//   - Charlie
	// Page 2 (offset=3): 2 users
	//   - Dave
	//   - Eve
}

// ExampleListRange demonstrates lazy iteration using the Go 1.25 iterator.
func ExampleListRange() {
	// Open in-memory SQLite database
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create table and insert sample data
	if err := Create[ExampleUser](db); err != nil {
		log.Fatal(err)
	}
	for _, name := range []string{"Alice", "Bob", "Charlie"} {
		if err := Insert(db, ExampleUser{Name: name, Email: name + "@example.com"}); err != nil {
			log.Fatal(err)
		}
	}

	// Iterate lazily with an error callback
	for i, user := range ListRange[ExampleUser](db, 0, "", "name ASC", 0,
		func(e error) { log.Fatal(e) },
	) {
		fmt.Printf("%d: %s\n", i, user.Name)
	}

	// Output:
	// 0: Alice
	// 1: Bob
	// 2: Charlie
}
