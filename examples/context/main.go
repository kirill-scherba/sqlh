// Copyright 2024-2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Context example: demonstrates using context cancellation with sqlh queries.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/kirill-scherba/sqlh"
	_ "github.com/mattn/go-sqlite3"
)

// Task represents a tasks table.
type Task struct {
	ID   int64  `db:"id" db_key:"not null primary key autoincrement"`
	Name string `db:"name"`
}

func main() {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create table and insert data using sqlh
	if err := sqlh.Create[Task](db); err != nil {
		log.Fatal(err)
	}
	for _, name := range []string{"design", "implement", "test", "deploy", "monitor"} {
		sqlh.Insert(db, Task{Name: name})
	}
	fmt.Println("Inserted 5 tasks")

	// Use context with timeout for listing via ListRange iterator
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	fmt.Println("All tasks:")
	for _, t := range sqlh.ListRange[Task](db, 0, "", "", 0, ctx) {
		fmt.Printf("  - %s (ID=%d)\n", t.Name, t.ID)
	}

	// Example: cancelled context
	ctxCancelled, cancel2 := context.WithCancel(context.Background())
	cancel2() // immediate cancellation

	for range sqlh.ListRange[Task](db, 0, "", "", 0, ctxCancelled) {
	}
	fmt.Println("Cancelled context: query stopped successfully")
}
