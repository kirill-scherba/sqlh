// Copyright 2024-2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.


package sqlh

import (
	"database/sql"
	"iter"
)

// Table is an sqlh type that works with a single SQL table.
// The methods are implemented for the Table[T] type and is wrappers for
// standard sqlh methods.
type Table[T any] struct {
	db *sql.DB
}

// CreateTable creates the SQL table for the T type.
//
// Parameters:
//
// - db - the database to create the table in.
//
// Returns:
//
// - t - the Table[T] type that works with the created table.
// - err - an error if the table could not be created.
func CreateTable[T any](db *sql.DB) (t *Table[T], err error) {
	return &Table[T]{db}, Create[T](db)
}

// Close is a no-op for backward compatibility.
//
// It does not close the underlying database connection because *sql.DB is a
// connection pool that may be shared between multiple tables, other Table[T]
// instances, and other parts of the application. The pool should be closed by
// the caller who created it (e.g. via db.Close()).
//
// Do not rely on Close() for resource cleanup — it is intentionally empty.
// New code should not call Close() on Table[T] at all; it is retained only
// to avoid breaking existing callers.
func (t *Table[T]) Close() {
	// Intentionally empty: do not close a shared *sql.DB pool.
}

// Insert inserts rows into the T database table.
//
// Parameters:
//
// - rows - a variadic number of rows of type T to insert into the table.
//
// Returns:
//
// - err - an error if the rows could not be inserted.
func (t *Table[T]) Insert(rows ...T) (err error) {
	return Insert(t.db, rows...)
}

// InsertId inserts rows into the T database table and returns the last inserted row ID.
//
// Parameters:
//
// - rows - a variadic number of rows of type T to insert into the table.
//
// Returns:
//
// - id - the last inserted row ID.
// - err - an error if the rows could not be inserted.
func (t *Table[T]) InsertId(rows ...T) (id int64, err error) {
	return InsertId(t.db, rows...)
}

// Update updates rows in the T database table.
//
// Parameters:
//
// - attrs - a list of UpdateAttr objects which contains row and where condition.
//
// Returns:
//
// - err - an error if the rows could not be updated.
func (t *Table[T]) Update(attrs ...UpdateAttr[T]) (err error) {
	return Update(t.db, attrs...)
}

// Set sets a row in T database table.
//
// The function is atomic and uses a transaction.
// The function takes a list of Where condition as input parameter.
// The function checks if the row is found in the database.
// If the row is not found, the function inserts a new row.
// If the row is found, the function updates the row.
// If multiple rows are found, the function returns an error with message "multiple rows found".
func (t *Table[T]) Set(row T, wheres ...Where) (err error) {
	return Set(t.db, row, wheres...)
}

// Get returns a row from T database table.
//
// Parameters:
//
// - wheres - a variadic list of Where conditions to filter the row.
//
// Returns:
//
// - row - the found row, or nil if no row is found.
// - err - an error if the row could not be found.
func (t *Table[T]) Get(wheres ...Where) (row *T, err error) {
	return Get[T](t.db, wheres...)
}

// Delete deletes rows from the T database table.
//
// Parameters:
//
// - wheres - a variadic list of Where conditions to filter the rows to delete.
//
// Returns:
//
// - err - an error if the rows could not be deleted.
func (t *Table[T]) Delete(wheres ...Where) (err error) {
	return Delete[T](t.db, wheres...)
}

// Count returns the number of rows from the T database table that match the given where conditions.
//
// Parameters:
//
// - wheres - a variadic list of Where conditions to filter the rows.
//
// Returns:
//
// - count - the number of rows that match the where conditions.
// - err - an error if the rows could not be counted.
func (t *Table[T]) Count(wheres ...Where) (count int, err error) {
	return Count[T](t.db, wheres...)
}

// List returns an iter.Seq2[int, T] that lazily streams rows from the database.
// It delegates to ListRange. Use this method when you want memory-efficient
// iteration via the Table[T] wrapper API.
//
// For explicit paginated listing with a materialised slice, use ListRows
// directly; for raw SQL queries use QueryRange.
func (t *Table[T]) List(offset int, groupBy, orderBy string, limit int,
	listAttrs ...any) iter.Seq2[int, T] {
	return ListRange[T](t.db, offset, groupBy, orderBy, limit, listAttrs...)
}
