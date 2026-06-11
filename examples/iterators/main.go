// Copyright 2024-2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Iterator example: demonstrates scanning query results into structs using sqlh.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/kirill-scherba/sqlh"
	_ "github.com/mattn/go-sqlite3"
)

// LogEntry represents a log entry.
type LogEntry struct {
	ID     int    `db:"id" db_key:"primary key autoincrement"`
	Level  string `db:"level"`
	Msg    string `db:"message"`
	UserID int    `db:"user_id"`
}

func main() {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create table using sqlh
	if err := sqlh.Create[LogEntry](db); err != nil {
		log.Fatal(err)
	}

	// Insert sample data
	for i := range 5 {
		userID := i%2 + 1
		if err := sqlh.Insert(db, LogEntry{
			Level:  "INFO",
			Msg:    fmt.Sprintf("event-%d occurred", i+1),
			UserID: userID,
		}); err != nil {
			log.Fatal(err)
		}
	}
	fmt.Println("Inserted 5 log entries")

	// Query using raw SQL and scan rows with ListRange iterator. The where
	// clause, error handler and context are also provided as parameters.
	var entries []LogEntry
	fmt.Println("Log entries for user_id=1:")
	for _, e := range sqlh.ListRange[LogEntry](db, 0, "", "", 0,
		// Where clause
		sqlh.Eq("user_id", 1),
		// Func to handle Errors during execution and scan records
		func(e error) { log.Fatal("error:", e) },
		// Context
		context.Background()) {

		fmt.Printf("  [%s] %s\n", e.Level, e.Msg)
		entries = append(entries, e)

	}
	fmt.Printf("Total: %d entries\n", len(entries))
}
