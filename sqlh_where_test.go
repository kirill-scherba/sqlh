// Copyright 2024-2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlh

import (
	"database/sql"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

// Test helper constructors produce correct Where structs.

func TestWhereHelpers_Constructors(t *testing.T) {
	t.Run("Eq", func(t *testing.T) {
		w := Eq("id", 42)
		require.Equal(t, "id=", w.Field)
		require.Equal(t, 42, w.Value)
	})

	t.Run("Ne", func(t *testing.T) {
		w := Ne("status", "deleted")
		require.Equal(t, "status<>", w.Field)
		require.Equal(t, "deleted", w.Value)
	})

	t.Run("Gt", func(t *testing.T) {
		w := Gt("age", 18)
		require.Equal(t, "age>", w.Field)
		require.Equal(t, 18, w.Value)
	})

	t.Run("Gte", func(t *testing.T) {
		w := Gte("age", 21)
		require.Equal(t, "age>=", w.Field)
		require.Equal(t, 21, w.Value)
	})

	t.Run("Lt", func(t *testing.T) {
		w := Lt("price", 100.0)
		require.Equal(t, "price<", w.Field)
		require.Equal(t, 100.0, w.Value)
	})

	t.Run("Lte", func(t *testing.T) {
		w := Lte("price", 50.0)
		require.Equal(t, "price<=", w.Field)
		require.Equal(t, 50.0, w.Value)
	})

	t.Run("Like", func(t *testing.T) {
		w := Like("name", "%Alice%")
		require.Equal(t, "name LIKE", w.Field)
		require.Equal(t, "%Alice%", w.Value)
	})

	t.Run("In", func(t *testing.T) {
		w := In("id", 1, 2, 3)
		require.Equal(t, "id IN", w.Field)
		vals, ok := w.Value.([]any)
		require.True(t, ok)
		require.Equal(t, []any{1, 2, 3}, vals)
	})

	t.Run("IsNull", func(t *testing.T) {
		w := IsNull("deleted_at")
		require.Equal(t, "deleted_at IS NULL", w.Field)
		require.Nil(t, w.Value)
	})

	t.Run("IsNotNull", func(t *testing.T) {
		w := IsNotNull("created_at")
		require.Equal(t, "created_at IS NOT NULL", w.Field)
		require.Nil(t, w.Value)
	})
}

// Test processWhere for SQL fragment generation and argument binding.

func TestWhereHelpers_ProcessWhere(t *testing.T) {

	t.Run("standard comparison generates single placeholder", func(t *testing.T) {
		frag, args := processWhere(Where{Field: "id=", Value: 42})
		require.Equal(t, "id=?", frag)
		require.Equal(t, []any{42}, args)
	})

	t.Run("nil value generates no placeholder", func(t *testing.T) {
		frag, args := processWhere(Where{Field: "deleted_at IS NULL", Value: nil})
		require.Equal(t, "deleted_at IS NULL", frag)
		require.Empty(t, args)
	})

	t.Run("IN expands slice values", func(t *testing.T) {
		frag, args := processWhere(In("id", 1, 2, 3))
		require.Equal(t, "id IN (?, ?, ?)", frag)
		require.Equal(t, []any{1, 2, 3}, args)
	})

	t.Run("IN with empty slice", func(t *testing.T) {
		frag, args := processWhere(In("id"))
		require.Equal(t, "id IN ()", frag)
		require.Empty(t, args)
	})

	t.Run("IN with single value", func(t *testing.T) {
		frag, args := processWhere(Where{Field: "id IN", Value: []any{42}})
		require.Equal(t, "id IN (?)", frag)
		require.Equal(t, []any{42}, args)
	})

	t.Run("LIKE generates single placeholder", func(t *testing.T) {
		frag, args := processWhere(Like("name", "%foo%"))
		require.Equal(t, "name LIKE?", frag)
		require.Equal(t, []any{"%foo%"}, args)
	})
}

// Test IN with typed slices via spread.

func TestWhereHelpers_InTypedSlice(t *testing.T) {
	ids := []any{10, 20, 30}
	w := In("id", ids...)
	frag, args := processWhere(w)
	require.Equal(t, "id IN (?, ?, ?)", frag)
	require.Equal(t, []any{10, 20, 30}, args)
}

// Integration tests with a real SQLite database.

type whereTestTable struct {
	ID    int64  `db:"id" db_key:"not null primary key autoincrement"`
	Name  string `db:"name" db_key:"unique"`
	Age   int    `db:"age"`
	Email string `db:"email"`
}

func TestWhereHelpers_CRUD(t *testing.T) {
	// Open in-memory SQLite database
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	require.NoError(t, err)
	defer db.Close()

	// Create table
	err = Create[whereTestTable](db)
	require.NoError(t, err)

	// Insert test data
	err = Insert(db,
		whereTestTable{Name: "Alice", Age: 30, Email: "alice@example.com"},
		whereTestTable{Name: "Bob", Age: 25, Email: "bob@example.com"},
		whereTestTable{Name: "Charlie", Age: 35, Email: "charlie@example.com"},
		whereTestTable{Name: "Dave", Age: 28, Email: "dave@example.com"},
	)
	require.NoError(t, err)

	// Test Eq — Get by name
	t.Run("Get with Eq", func(t *testing.T) {
		user, err := Get[whereTestTable](db, Eq("name", "Alice"))
		require.NoError(t, err)
		require.Equal(t, "Alice", user.Name)
		require.Equal(t, 30, user.Age)
	})

	// Test Gt — List adults (age > 25)
	t.Run("List with Gt", func(t *testing.T) {
		users, _, err := List[whereTestTable](db, 0, "", "name ASC",
			Gt("age", 25),
		)
		require.NoError(t, err)
		names := make([]string, len(users))
		for i, u := range users {
			names[i] = u.Name
		}
		require.ElementsMatch(t, []string{"Alice", "Charlie", "Dave"}, names)
	})

	// Test Lt — List young users
	t.Run("List with Lt", func(t *testing.T) {
		users, _, err := List[whereTestTable](db, 0, "", "name ASC",
			Lt("age", 30),
		)
		require.NoError(t, err)
		require.Len(t, users, 2)
		require.Equal(t, "Bob", users[0].Name)
		require.Equal(t, "Dave", users[1].Name)
	})

	// Test Gte/Lte together
	t.Run("List with Gte and Lte", func(t *testing.T) {
		users, _, err := List[whereTestTable](db, 0, "", "name ASC",
			Gte("age", 25),
			Lte("age", 30),
		)
		require.NoError(t, err)
		require.Len(t, users, 3)
	})

	// Test Like
	t.Run("List with Like", func(t *testing.T) {
		users, _, err := List[whereTestTable](db, 0, "", "name ASC",
			Like("name", "A%"),
		)
		require.NoError(t, err)
		require.Len(t, users, 1)
		require.Equal(t, "Alice", users[0].Name)
	})

	// Test In
	t.Run("Delete with In", func(t *testing.T) {
		err := Delete[whereTestTable](db, In("name", "Bob", "Charlie"))
		require.NoError(t, err)

		// Verify deletion
		_, err = Get[whereTestTable](db, Eq("name", "Bob"))
		require.ErrorIs(t, err, sql.ErrNoRows)

		_, err = Get[whereTestTable](db, Eq("name", "Charlie"))
		require.ErrorIs(t, err, sql.ErrNoRows)
	})

	// Test Count with Gte
	t.Run("Count with Gte", func(t *testing.T) {
		count, err := Count[whereTestTable](db, Gte("age", 25))
		require.NoError(t, err)
		require.Equal(t, 2, count) // Alice (30) and Dave (28)
	})

	// Test Update with Eq
	t.Run("Update with Eq", func(t *testing.T) {
		err := Update(db, UpdateAttr[whereTestTable]{
			Row:    whereTestTable{Name: "Alice Updated", Age: 31, Email: "alice.new@example.com"},
			Wheres: []Where{Eq("name", "Alice")},
		})
		require.NoError(t, err)

		// Verify
		user, err := Get[whereTestTable](db, Eq("name", "Alice Updated"))
		require.NoError(t, err)
		require.Equal(t, 31, user.Age)
	})

	// Test backward compatibility — raw Where still works
	t.Run("backward compat raw Where", func(t *testing.T) {
		users, _, err := List[whereTestTable](db, 0, "", "name ASC",
			Where{Field: "age>=", Value: 28},
		)
		require.NoError(t, err)
		require.Len(t, users, 2) // Alice Updated (31) and Dave (28)
	})

	// Test Set with Eq
	t.Run("Set with Eq", func(t *testing.T) {
		err := Set(db,
			whereTestTable{Name: "Eve", Age: 40, Email: "eve@example.com"},
			Eq("name", "Eve"),
		)
		require.NoError(t, err)

		// Verify insert via Set
		user, err := Get[whereTestTable](db, Eq("name", "Eve"))
		require.NoError(t, err)
		require.Equal(t, 40, user.Age)

		// Update via Set
		err = Set(db,
			whereTestTable{Name: "Eve", Age: 41, Email: "eve@example.com"},
			Eq("name", "Eve"),
		)
		require.NoError(t, err)

		user, err = Get[whereTestTable](db, Eq("name", "Eve"))
		require.NoError(t, err)
		require.Equal(t, 41, user.Age)
	})

	// Test IsNull / IsNotNull
	t.Run("IsNull and IsNotNull", func(t *testing.T) {
		// With no nullable column in our test struct, we test processWhere directly
		frag, args := processWhere(IsNull("deleted_at"))
		require.Equal(t, "deleted_at IS NULL", frag)
		require.Empty(t, args)

		frag, args = processWhere(IsNotNull("created_at"))
		require.Equal(t, "created_at IS NOT NULL", frag)
		require.Empty(t, args)
	})
}

// Test processWhere with Ne (not equal).
func TestWhereHelpers_Ne(t *testing.T) {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	require.NoError(t, err)
	defer db.Close()

	err = Create[whereTestTable](db)
	require.NoError(t, err)

	err = Insert(db,
		whereTestTable{Name: "Alice", Age: 30, Email: "alice@example.com"},
		whereTestTable{Name: "Bob", Age: 25, Email: "bob@example.com"},
	)
	require.NoError(t, err)

	users, _, err := List[whereTestTable](db, 0, "", "name ASC",
		Ne("name", "Alice"),
	)
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, "Bob", users[0].Name)
}

// Test that invalid field names still pass through (escape hatch behaviour).
func TestWhereHelpers_InvalidField_EscapeHatch(t *testing.T) {
	// The helpers are thin wrappers — they do not validate field names.
	// Invalid names are passed through as-is, matching the raw Where contract.
	w := Eq("1=1; DROP TABLE users; --", "foo")
	require.Equal(t, "1=1; DROP TABLE users; --=", w.Field)

	frag, args := processWhere(w)
	require.Equal(t, "1=1; DROP TABLE users; --=?", frag)
	require.Equal(t, []any{"foo"}, args)
}

// Test ListRange with helpers.
func TestWhereHelpers_ListRange(t *testing.T) {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	require.NoError(t, err)
	defer db.Close()

	err = Create[whereTestTable](db)
	require.NoError(t, err)

	err = Insert(db,
		whereTestTable{Name: "Alice", Age: 30, Email: "alice@example.com"},
		whereTestTable{Name: "Bob", Age: 25, Email: "bob@example.com"},
		whereTestTable{Name: "Charlie", Age: 35, Email: "charlie@example.com"},
	)
	require.NoError(t, err)

	var names []string
	var listErr error
	for _, u := range ListRange[whereTestTable](db, 0, "", "name ASC", 0,
		Gt("age", 25),
		func(e error) { listErr = e },
	) {
		names = append(names, u.Name)
	}
	require.NoError(t, listErr)
	require.ElementsMatch(t, []string{"Alice", "Charlie"}, names)
}

// Test Count with In.
func TestWhereHelpers_CountWithIn(t *testing.T) {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	require.NoError(t, err)
	defer db.Close()

	err = Create[whereTestTable](db)
	require.NoError(t, err)

	err = Insert(db,
		whereTestTable{Name: "Alice", Age: 30, Email: "alice@example.com"},
		whereTestTable{Name: "Bob", Age: 25, Email: "bob@example.com"},
		whereTestTable{Name: "Charlie", Age: 35, Email: "charlie@example.com"},
	)
	require.NoError(t, err)

	count, err := Count[whereTestTable](db, In("name", "Alice", "Bob"))
	require.NoError(t, err)
	require.Equal(t, 2, count)
}

// Test Delete with a single-element In.
func TestWhereHelpers_DeleteSingleIn(t *testing.T) {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	require.NoError(t, err)
	defer db.Close()

	err = Create[whereTestTable](db)
	require.NoError(t, err)

	err = Insert(db,
		whereTestTable{Name: "Alice", Age: 30, Email: "alice@example.com"},
		whereTestTable{Name: "Bob", Age: 25, Email: "bob@example.com"},
	)
	require.NoError(t, err)

	err = Delete[whereTestTable](db, In("name", "Alice"))
	require.NoError(t, err)

	_, err = Get[whereTestTable](db, Eq("name", "Alice"))
	require.ErrorIs(t, err, sql.ErrNoRows)

	// Bob still exists
	user, err := Get[whereTestTable](db, Eq("name", "Bob"))
	require.NoError(t, err)
	require.Equal(t, "Bob", user.Name)
}

// Test combining helpers with OR join.
func TestWhereHelpers_CombinedWithOr(t *testing.T) {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	require.NoError(t, err)
	defer db.Close()

	err = Create[whereTestTable](db)
	require.NoError(t, err)

	err = Insert(db,
		whereTestTable{Name: "Alice", Age: 30, Email: "alice@example.com"},
		whereTestTable{Name: "Bob", Age: 25, Email: "bob@example.com"},
		whereTestTable{Name: "Charlie", Age: 35, Email: "charlie@example.com"},
	)
	require.NoError(t, err)

	users, _, err := List[whereTestTable](db, 0, "", "name ASC",
		Eq("name", "Alice"),
		Eq("name", "Bob"),
		SetWheresJoinOr(),
	)
	require.NoError(t, err)
	require.Len(t, users, 2)
	names := []string{users[0].Name, users[1].Name}
	require.ElementsMatch(t, []string{"Alice", "Bob"}, names)
}

// Test that raw Where{Field, Value} still works alongside new helpers.
func TestWhereHelpers_BackwardCompat_Mixed(t *testing.T) {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	require.NoError(t, err)
	defer db.Close()

	err = Create[whereTestTable](db)
	require.NoError(t, err)

	err = Insert(db,
		whereTestTable{Name: "Alice", Age: 30, Email: "alice@example.com"},
		whereTestTable{Name: "Bob", Age: 25, Email: "bob@example.com"},
	)
	require.NoError(t, err)

	// Mix old-style Where with new helpers in a single query
	users, _, err := List[whereTestTable](db, 0, "", "name ASC",
		Eq("name", "Alice"),
		Where{Field: "age>", Value: 20},
	)
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, "Alice", users[0].Name)
}

// Test Table[T] wrapper with helpers.
func TestWhereHelpers_TableWrapper(t *testing.T) {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	require.NoError(t, err)
	defer db.Close()

	tbl, err := CreateTable[whereTestTable](db)
	require.NoError(t, err)

	err = tbl.Insert(whereTestTable{Name: "Alice", Age: 30, Email: "alice@example.com"})
	require.NoError(t, err)

	user, err := tbl.Get(Eq("name", "Alice"))
	require.NoError(t, err)
	require.Equal(t, "Alice", user.Name)

	count, err := tbl.Count(Eq("name", "Alice"))
	require.NoError(t, err)
	require.Equal(t, 1, count)

	err = tbl.Delete(Eq("name", "Alice"))
	require.NoError(t, err)

	_, err = tbl.Get(Eq("name", "Alice"))
	require.True(t, errors.Is(err, sql.ErrNoRows) || err == sql.ErrNoRows)
}
