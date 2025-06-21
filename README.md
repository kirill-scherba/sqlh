# sqlh: A Go SQL Helper Package

[![Go Report Card](https://goreportcard.com/badge/github.com/kirill-scherba/sqlh)](https://goreportcard.com/report/github.com/kirill-scherba/sqlh)
[![GoDoc](https://godoc.org/github.com/kirill-scherba/sqlh?status.svg)](https://godoc.org/github.com/kirill-scherba/sqlh/)

`sqlh` is a lightweight helper package for Go that simplifies interactions with SQL databases. It leverages generics to provide a set of intuitive functions (`Insert`, `Update`, `Get`, `List`, `Delete`) that work directly with your Go structs, reducing boilerplate code.

The package automatically generates SQL queries from your struct definitions, using struct tags for customization.

## Features

- **Generic Functions:** Work with any of your custom structs without needing to write specific SQL for each.
- **Automatic Query Generation:** Automatically creates `CREATE TABLE`, `INSERT`, `UPDATE`, `SELECT`, and `DELETE` statements.
- **Struct Tag-Based Mapping:** Use `db`, `db_type`, and `db_key` tags to control table and column definitions.
- **Autoincrement Support:** Automatically excludes fields marked with `autoincrement` from `INSERT` and `UPDATE` statements.
- **Built-in Transactions:** All write operations (`Insert`, `Update`, `Delete`, `Set`) are wrapped in transactions for data integrity.
- **Standardized Error Handling:** Returns standard errors like `sql.ErrNoRows` and exported package errors for easy checking with `errors.Is`.

## Installation

```bash
go get github.com/kirill-scherba/sqlh
```

## Quick Start

Here's a quick example of how to use `sqlh` with an in-memory SQLite database.

### 1. Define Your Struct

Define a Go struct that represents your database table. Use struct tags to define column names, types, and keys.

```go
package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/kirill-scherba/sqlh"
	"github.com/kirill-scherba/sqlh/query"
	_ "github.com/mattn/go-sqlite3"
)

// User represents the users table.
type User struct {
	ID    int64  `db:"id" db_key:"not null primary key autoincrement"`
	Name  string `db:"name" db_key:"unique"`
	Email string `db:"email"`
}
```

### 2. Connect and Create Table

Use the `query.Table` function to generate a `CREATE TABLE` statement from your struct.

```go
func main() {
	// Open in-memory SQLite database for this example
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Generate and execute CREATE TABLE statement
	createStmt, err := query.Table[User]()
	if err != nil {
		log.Fatalf("failed to create table query: %v", err)
	}
	if _, err := db.Exec(createStmt); err != nil {
		log.Fatalf("failed to execute create table statement: %v", err)
	}
	fmt.Println("Table 'user' created successfully.")

	// Insert a new user
	alice := User{Name: "Alice", Email: "alice@example.com"}
	if err := sqlh.Insert(db, alice); err != nil {
		log.Fatalf("failed to insert user: %v", err)
	}
	fmt.Println("Inserted Alice.")

	// Get the user we just inserted
	retrievedUser, err := sqlh.Get[User](db, sqlh.Where{Field: "name=", Value: "Alice"})
	if err != nil {
		// Check for a specific "not found" error
		if errors.Is(err, sql.ErrNoRows) {
			log.Println("User not found.")
		} else {
			log.Fatalf("failed to get user: %v", err)
		}
		return
	}
	fmt.Printf("Retrieved User: ID=%d, Name=%s, Email=%s\n", retrievedUser.ID, retrievedUser.Name, retrievedUser.Email)

	// Update Alice's email
	retrievedUser.Email = "alice.new@example.com"
	updateAttr := sqlh.UpdateAttr[User]{
		Row:    *retrievedUser,
		Wheres: []sqlh.Where{{Field: "id=", Value: retrievedUser.ID}},
	}
	if err := sqlh.Update(db, updateAttr); err != nil {
		log.Fatalf("failed to update user: %v", err)
	}
	fmt.Println("Updated Alice's email.")
}
```

## Changelog

For a detailed list of changes, please see the [CHANGELOG.md](CHANGELOG.md) file.

## Licence

BSD