// Copyright 2024-2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Demo program for sqlh animated GIF recording.
// Run: go run .
package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/kirill-scherba/sqlh"
	_ "github.com/mattn/go-sqlite3"
)

// User represents a users table. The schema is defined entirely
// through struct tags — no SQL migration files needed.
type User struct {
	ID    int64  `db:"id" db_key:"not null primary key autoincrement"`
	Name  string `db:"name" db_key:"unique"`
	Email string `db:"email"`
	Age   int    `db:"age"`
}

func main() {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 1. Create table from struct
	if err := sqlh.Create[User](db); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Table created")

	// 2. Insert
	if err := sqlh.Insert(db, User{Name: "Alice", Email: "alice@example.com", Age: 30}); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Inserted Alice")

	bobID, err := sqlh.InsertId(db, User{Name: "Bob", Email: "bob@example.com", Age: 25})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✓ Inserted Bob (ID=%d)\n", bobID)

	if err := sqlh.Insert(db, User{Name: "Charlie", Email: "charlie@example.com", Age: 35}); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Inserted Charlie")

	// 3. Get
	user, err := sqlh.Get[User](db, sqlh.Eq("id", bobID))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✓ Get: %s, age %d\n", user.Name, user.Age)

	// 4. List
	users, next, err := sqlh.List[User](db, 0, "", "name ASC")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✓ List: %d users (next=%d)\n", len(users), next)
	for _, u := range users {
		fmt.Printf("   %-10s %-22s age=%d\n", u.Name, u.Email, u.Age)
	}

	// 5. Update
	if err := sqlh.Update(db, sqlh.UpdateAttr[User]{
		Row:    User{Name: "Alice", Email: "alice.new@example.com", Age: 31},
		Wheres: []sqlh.Where{sqlh.Eq("name", "Alice")},
	}); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Updated Alice")

	updated, _ := sqlh.Get[User](db, sqlh.Eq("name", "Alice"))
	fmt.Printf("   → %s, age %d, email=%s\n", updated.Name, updated.Age, updated.Email)

	// 6. Delete
	if err := sqlh.Delete[User](db, sqlh.Eq("id", bobID)); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Deleted Bob")

	// Final count
	count, err := sqlh.Count[User](db)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\n✓ Remaining: %d user(s)\n", count)
}
