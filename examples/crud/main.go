package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/kirill-scherba/sqlh"
	_ "github.com/mattn/go-sqlite3"
)

// User represents the users table.
type User struct {
	ID    int64  `db:"id" db_key:"not null primary key autoincrement"`
	Name  string `db:"name" db_key:"unique"`
	Email string `db:"email"`
	Age   int    `db:"age"`
}

func main() {
	// Open in-memory SQLite database
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// ---- 1. Create table from struct ----
	if err := sqlh.Create[User](db); err != nil {
		log.Fatalf("failed to create table: %v", err)
	}
	fmt.Println("✓ Table 'user' created")

	// ---- 2. Insert ----
	if err := sqlh.Insert(db, User{Name: "Alice", Email: "alice@example.com", Age: 30}); err != nil {
		log.Fatalf("failed to insert: %v", err)
	}
	fmt.Println("✓ Inserted Alice")

	bobID, err := sqlh.InsertId(db, User{Name: "Bob", Email: "bob@example.com", Age: 25})
	if err != nil {
		log.Fatalf("failed to insert: %v", err)
	}
	fmt.Printf("✓ Inserted Bob with ID=%d\n", bobID)

	if err := sqlh.Insert(db, User{Name: "Charlie", Email: "charlie@example.com", Age: 35}); err != nil {
		log.Fatalf("failed to insert: %v", err)
	}
	fmt.Println("✓ Inserted Charlie")

	// ---- 3. Get by ID (returns *T, err) ----
	userPtr, err := sqlh.Get[User](db, sqlh.Where{Field: "id=", Value: bobID})
	if err != nil {
		if err == sql.ErrNoRows {
			log.Fatal("user not found")
		}
		log.Fatalf("failed to get user: %v", err)
	}
	fmt.Printf("✓ Get by ID: ID=%d, Name=%s, Email=%s, Age=%d\n",
		userPtr.ID, userPtr.Name, userPtr.Email, userPtr.Age)

	// ---- 4. Get by name (unique field) ----
	userPtr, err = sqlh.Get[User](db, sqlh.Where{Field: "name=", Value: "Alice"})
	if err != nil {
		log.Fatalf("failed to get user: %v", err)
	}
	fmt.Printf("✓ Get by name: ID=%d, Name=%s, Age=%d\n",
		userPtr.ID, userPtr.Name, userPtr.Age)

	// ---- 5. List all users (previous=0, groupBy="", orderBy="name ASC") ----
	users, pagination, err := sqlh.List[User](db, 0, "", "name ASC")
	if err != nil {
		log.Fatalf("failed to list users: %v", err)
	}
	fmt.Printf("✓ List: %d users, pagination=%d\n", len(users), pagination)
	for _, u := range users {
		fmt.Printf("   - %s (%s), age %d\n", u.Name, u.Email, u.Age)
	}

	// ---- 6. List with WHERE ----
	adults, _, err := sqlh.List[User](db, 0, "", "name ASC",
		sqlh.Where{Field: "age>=", Value: 30})
	if err != nil {
		log.Fatalf("failed to list adults: %v", err)
	}
	fmt.Printf("✓ Filtered list (age>=30): %d users\n", len(adults))
	for _, u := range adults {
		fmt.Printf("   - %s, age %d\n", u.Name, u.Age)
	}

	// ---- 7. ListRange (lazy iterator — Seq2[int, T]) ----
	fmt.Println("✓ ListRange (sorted by age DESC):")
	for i, user := range sqlh.ListRange[User](db, 0, "", "age DESC", 0) {
		fmt.Printf("   [%d] %s, age %d\n", i, user.Name, user.Age)
	}

	// ---- 8. Update (uses UpdateAttr with Row + Wheres) ----
	if err := sqlh.Update(db,
		sqlh.UpdateAttr[User]{
			Row:    User{Name: "Alice", Email: "alice.new@example.com", Age: 31},
			Wheres: []sqlh.Where{{Field: "name=", Value: "Alice"}},
		},
	); err != nil {
		log.Fatalf("failed to update: %v", err)
	}
	fmt.Println("✓ Updated Alice's email and age")

	// Verify update
	userPtr, err = sqlh.Get[User](db, sqlh.Where{Field: "name=", Value: "Alice"})
	if err != nil {
		fmt.Printf("  Failed to get updated Alice: %v\n", err)
	} else {
		fmt.Printf("  Alice now: Email=%s, Age=%d\n", userPtr.Email, userPtr.Age)
	}

	// ---- 9. Delete ----
	if err := sqlh.Delete[User](db, sqlh.Where{Field: "id=", Value: bobID}); err != nil {
		log.Fatalf("failed to delete: %v", err)
	}
	fmt.Println("✓ Deleted Bob")

	// Verify deletion
	_, err = sqlh.Get[User](db, sqlh.Where{Field: "id=", Value: bobID})
	if err == sql.ErrNoRows {
		fmt.Println("  Bob is gone (sql.ErrNoRows)")
	}

	// ---- 10. Set (upsert) — insert or update ----
	// Set Dave (new user — will insert)
	err = sqlh.Set(db,
		User{Name: "Dave", Email: "dave@example.com", Age: 28},
		sqlh.Where{Field: "name=", Value: "Dave"})
	if err != nil {
		log.Fatalf("failed to set Dave: %v", err)
	}
	fmt.Println("✓ Set (inserted) Dave")

	// Set Dave again with different age (will update)
	err = sqlh.Set(db,
		User{Name: "Dave", Email: "dave@example.com", Age: 29},
		sqlh.Where{Field: "name=", Value: "Dave"})
	if err != nil {
		log.Fatalf("failed to set Dave again: %v", err)
	}
	fmt.Println("✓ Set (updated) Dave's age")

	userPtr, err = sqlh.Get[User](db, sqlh.Where{Field: "name=", Value: "Dave"})
	if err != nil {
		fmt.Printf("  Failed to get Dave: %v\n", err)
	} else {
		fmt.Printf("  Dave now: Age=%d\n", userPtr.Age)
	}

	// ---- 11. Final list ----
	allUsers, _, _ := sqlh.List[User](db, 0, "", "name ASC")
	fmt.Printf("\n✓ Final users (%d):\n", len(allUsers))
	for _, u := range allUsers {
		fmt.Printf("   ID=%d  %-10s %-25s age=%d\n",
			u.ID, u.Name, u.Email, u.Age)
	}
}
