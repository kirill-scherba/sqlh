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

// ExampleCreateTable demonstrates creating a SQL table via the Table[T] wrapper.
func ExampleCreateTable() {
	// Open in-memory SQLite database
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create table from struct and get a typed Table[T] handle
	tbl, err := CreateTable[ExampleUser](db)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Table created: %T\n", tbl)

	// Output:
	// Table created: *sqlh.Table[github.com/kirill-scherba/sqlh.ExampleUser]
}

// ExampleTable_Insert demonstrates inserting rows via the Table[T] wrapper.
func ExampleTable_Insert() {
	// Open in-memory SQLite database
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create table and get Table[T] wrapper
	tbl, err := CreateTable[ExampleUser](db)
	if err != nil {
		log.Fatal(err)
	}

	// Insert multiple rows through the wrapper
	if err := tbl.Insert(
		ExampleUser{Name: "Alice", Email: "alice@example.com"},
		ExampleUser{Name: "Bob", Email: "bob@example.com"},
	); err != nil {
		log.Fatal(err)
	}

	// Verify insertion by counting rows
	count, err := tbl.Count()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Inserted:", count)

	// Output:
	// Inserted: 2
}

// ExampleTable_Get demonstrates retrieving a single row via the Table[T] wrapper.
func ExampleTable_Get() {
	// Open in-memory SQLite database
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create table and get Table[T] wrapper
	tbl, err := CreateTable[ExampleUser](db)
	if err != nil {
		log.Fatal(err)
	}

	// Insert a sample row
	if err := tbl.Insert(ExampleUser{Name: "Alice", Email: "alice@example.com"}); err != nil {
		log.Fatal(err)
	}

	// Get the row by name
	user, err := tbl.Get(Where{Field: "name=", Value: "Alice"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found: %s <%s>\n", user.Name, user.Email)

	// Output:
	// Found: Alice <alice@example.com>
}

// ExampleTable_Set demonstrates upsert via the Table[T] wrapper.
// The first call inserts a new row; the second call updates it.
func ExampleTable_Set() {
	// Open in-memory SQLite database
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create table and get Table[T] wrapper
	tbl, err := CreateTable[ExampleUser](db)
	if err != nil {
		log.Fatal(err)
	}

	// Upsert: row does not exist yet, so Set performs an INSERT
	if err := tbl.Set(
		ExampleUser{Name: "Charlie", Email: "charlie@old.com"},
		Where{Field: "name=", Value: "Charlie"},
	); err != nil {
		log.Fatal(err)
	}
	u1, err := tbl.Get(Where{Field: "name=", Value: "Charlie"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("After insert:", u1.Email)

	// Upsert: row exists by unique key, so Set performs an UPDATE
	if err := tbl.Set(
		ExampleUser{Name: "Charlie", Email: "charlie@new.com"},
		Where{Field: "name=", Value: "Charlie"},
	); err != nil {
		log.Fatal(err)
	}
	u2, err := tbl.Get(Where{Field: "name=", Value: "Charlie"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("After update:", u2.Email)

	// Output:
	// After insert: charlie@old.com
	// After update: charlie@new.com
}

// ExampleTable_Delete demonstrates deleting rows via the Table[T] wrapper.
func ExampleTable_Delete() {
	// Open in-memory SQLite database
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create table and get Table[T] wrapper
	tbl, err := CreateTable[ExampleUser](db)
	if err != nil {
		log.Fatal(err)
	}

	// Insert a sample row
	if err := tbl.Insert(ExampleUser{Name: "Dave", Email: "dave@example.com"}); err != nil {
		log.Fatal(err)
	}

	// Delete by condition
	if err := tbl.Delete(Where{Field: "name=", Value: "Dave"}); err != nil {
		log.Fatal(err)
	}

	// Verify deletion
	count, err := tbl.Count()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Remaining:", count)

	// Output:
	// Remaining: 0
}

// ExampleTable_List demonstrates lazy iteration via the Table[T].List method.
func ExampleTable_List() {
	// Open in-memory SQLite database
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create table and get Table[T] wrapper
	tbl, err := CreateTable[ExampleUser](db)
	if err != nil {
		log.Fatal(err)
	}

	// Insert sample rows
	for _, name := range []string{"Alice", "Bob", "Charlie", "Dave", "Eve"} {
		if err := tbl.Insert(ExampleUser{Name: name, Email: name + "@example.com"}); err != nil {
			log.Fatal(err)
		}
	}

	// Iterate lazily with an error callback
	for i, user := range tbl.List(0, "", "name ASC", 0,
		func(e error) { log.Fatal(e) },
	) {
		fmt.Printf("%d: %s\n", i, user.Name)
	}

	// Output:
	// 0: Alice
	// 1: Bob
	// 2: Charlie
	// 3: Dave
	// 4: Eve
}

// ExampleTable_Update demonstrates updating rows via the Table[T] wrapper.
func ExampleTable_Update() {
	// Open in-memory SQLite database
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create table and get Table[T] wrapper
	tbl, err := CreateTable[ExampleUser](db)
	if err != nil {
		log.Fatal(err)
	}

	// Insert a sample row
	if err := tbl.Insert(ExampleUser{Name: "Eve", Email: "eve@old.com"}); err != nil {
		log.Fatal(err)
	}

	// Get the row so we know its ID
	u, err := tbl.Get(Where{Field: "name=", Value: "Eve"})
	if err != nil {
		log.Fatal(err)
	}

	// Update the row via wrapper
	if err := tbl.Update(UpdateAttr[ExampleUser]{
		Row:    ExampleUser{ID: u.ID, Name: "Eve", Email: "eve@new.com"},
		Wheres: []Where{{Field: "id=", Value: u.ID}},
	}); err != nil {
		log.Fatal(err)
	}

	// Verify update
	updated, err := tbl.Get(Where{Field: "id=", Value: u.ID})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Updated:", updated.Email)

	// Output:
	// Updated: eve@new.com
}
