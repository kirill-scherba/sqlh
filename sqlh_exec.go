// Copyright 2024 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Sqlh is a SQL Helper package contains helper functions to execute SQL
// requests. It provides such functions as Execute, Select, Insert, Update and
// Delete.
package sqlh

import (
	"database/sql"
	"errors"

	"github.com/kirill-scherba/sqlh/query"
)

var numRows = 10 // number of rows to get in select query

// Exported errors
var (
	ErrWhereClauseRequired = errors.New("sqlh: the where clause is required")
	ErrMultipleRowsFound   = errors.New("sqlh: multiple rows found")
)

// UpdateAttr struct contains row and where condition and used in Update
// function as attrs parameter.
type UpdateAttr[T any] struct {

	// Row value to be updated
	Row T

	// Where condition
	Wheres []Where
}

// Where struct contains where condition as field and value.
type Where struct {

	// Database table field Name and Condition Operator, f.e. "id="
	// 	=	Equal
	// 	>	Greater than
	// 	<	Less than
	// 	>=	Greater than or equal
	// 	<=	Less than or equal
	// 	<>	Not equal. In some versions of SQL it may be written as !=
	// 	BETWEEN	Between a certain range
	// 	LIKE	Search for a pattern
	// 	IN	To specify multiple possible values for a column
	Field string

	// Field value
	Value any
}

// SetNumRows sets numer of rows in List function.
func SetNumRows(n int) {
	numRows = n
}

// Insert inserts rows into the T database table.
//
// It accepts a variadic number of rows of type T and inserts them into the
// corresponding database table. The function starts a transaction and prepares
// an insert statement. Each row is then inserted in a loop. If any error occurs,
// the transaction is rolled back. Otherwise, the transaction is committed.
func Insert[T any](db *sql.DB, rows ...T) (err error) {

	// Create insert statement
	insertStmt, err := query.Insert[T]()
	if err != nil {
		return
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return
	}

	// Create prepared insert statement
	stmt, err := tx.Prepare(insertStmt)
	if err != nil {
		tx.Rollback()
		return
	}
	defer stmt.Close()

	// Insert rows
	for _, row := range rows {
		// Get arguments from the row
		args, err := query.Args(row)
		if err != nil {
			tx.Rollback()
			return err
		}
		// Execute insert statement with arguments
		_, err = stmt.Exec(args...)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// Commit transaction and return
	err = tx.Commit()
	return
}

// Update updates rows in T database table.
//
// The function takes a list of UpdateAttr as input parameter.
// UpdateAttr contains row and where condition.
// The function executes UPDATE statement for each UpdateAttr in the list.
//
// The function returns error if something failed during the update process.
func Update[T any](db *sql.DB, attrs ...UpdateAttr[T]) (err error) {

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return
	}

	// Update rows
	for _, attr := range attrs {

		// Create where clause
		var wheres []string
		for _, where := range attr.Wheres {
			wheres = append(wheres, where.Field)
		}

		// Create update statement
		updateStmt, err := query.Update[T](wheres...)
		if err != nil {
			tx.Rollback()
			return err
		}

		// Create prepared update statement
		stmt, err := tx.Prepare(updateStmt)
		if err != nil {
			tx.Rollback()
			return err
		}
		defer stmt.Close()

		// Create struct attr.Row field values array
		args, err := query.Args(attr.Row)
		if err != nil {
			tx.Rollback()
			return err
		}

		// Add where conditions to args array
		for _, where := range attr.Wheres {
			args = append(args, where.Value)
		}

		// Execute update statement
		_, err = stmt.Exec(args...)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// Commit transaction and return
	err = tx.Commit()

	return
}

// Set sets a row in T database table.
//
// The function takes a list of Where condition as input parameter.
// The function checks if the row is found in the database.
// If the row is not found, the function inserts a new row.
// If the row is found, the function updates the row.
// If multiple rows are found, the function returns an error with message "multiple rows found".
func Set[T any](db *sql.DB, row T, wheres ...Where) (err error) {
	// Get rows from database. Limit to 2 to detect multiple rows
	rows, _, err := ListRows[T](db, 0, "", 2, wheres...)
	if err != nil {
		return err
	}

	// Check if the row is found
	switch len(rows) {
	case 0:
		// No rows found, insert new row
		err = Insert(db, row)
		if err != nil {
			return err
		}

	case 1:
		// One row found, update row
		err = Update(db, UpdateAttr[T]{Row: row, Wheres: wheres})
		if err != nil {
			return err
		}

	default:
		// Multiple rows found, return error
		err = ErrMultipleRowsFound
	}

	return
}


// Get returns a row from T database table.
//
// The function takes a list of Where condition as input parameter.
// The function executes SELECT statement with the given where conditions.
// If the row is found, the function returns the row and nil as error.
// If the row is not found, the function returns a default value for row and
// an error with message "not found".
// If multiple rows are found, the function returns a default value for row and
// an error with message "multiple rows found". It returns a pointer to the row.
func Get[T any](db *sql.DB, wheres ...Where) (row *T, err error) {

	// Check if the where clause is required
	if len(wheres) == 0 {
		err = ErrWhereClauseRequired
		return nil, err // Return nil pointer on error
	}

	// Get rows from database
	rows, _, err := ListRows[T](db, 0, "", 2, wheres...) // Limit to 2 to detect multiple rows
	if err != nil {
		return nil, err // Return nil pointer on error
	}

	// Check if the row is found
	switch len(rows) {
	case 0:
		err = sql.ErrNoRows // No rows found, return nil pointer and sql.ErrNoRows
	case 1:
		row = &rows[0] // One row found, return pointer to the row
	default:
		err = ErrMultipleRowsFound // Multiple rows found, return nil pointer and ErrMultipleRowsFound
	}

	return
}

// Delete deletes rows from the T database table.
//
// The function takes a variadic list of Where conditions to specify which
// rows to delete. It constructs a DELETE SQL statement with the given
// conditions, starts a database transaction, prepares the DELETE statement,
// and executes it. If any error occurs during the process, the transaction
// is rolled back. Otherwise, the transaction is committed.
func Delete[T any](db *sql.DB, wheres ...Where) (err error) {

	// Prepare where clauses and arguments
	var whereArgs []any
	var whereFields []string
	for _, w := range wheres {
		whereArgs = append(whereArgs, w.Value)
		whereFields = append(whereFields, w.Field)
	}

	// Create delete statement
	deleteStmt, err := query.Delete[T](whereFields...)
	if err != nil {
		return
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return
	}

	// Create prepared delete statement
	stmt, err := db.Prepare(deleteStmt)
	if err != nil {
		tx.Rollback()
		return
	}
	defer stmt.Close()

	// Execute delete statement with where arguments
	_, err = stmt.Exec(whereArgs...)
	if err != nil {
		tx.Rollback()
		return
	}

	// Commit transaction and return
	err = tx.Commit()
	return
}

// List returns rows from T database table.
//
// The function takes a list of Where condition as input parameter.
// The function executes SELECT statement with the given where conditions.
// If the rows are found, the function returns the rows and nil as error.
// If the rows are not found, the function returns a default value for rows and
// an error with message "not found".
func List[T any](db *sql.DB, previous int, orderBy string, wheres ...Where) (
	rows []T, pagination int, err error) {

	// Call ListRows function with numRows as number of rows
	return ListRows[T](db, previous, orderBy, numRows, wheres...)
}
func ListRows[T any](db *sql.DB, previous int, orderBy string, numRows int, wheres ...Where) (
	rows []T, pagination int, err error) {

	var attr = &query.SelectAttr{}
	var selectArgs []any

	// Where clauses
	for _, w := range wheres {
		if w.Value == nil {
			attr.Wheres = append(attr.Wheres, w.Field)
			continue
		}
		attr.Wheres = append(attr.Wheres, w.Field+"?")
		selectArgs = append(selectArgs, w.Value)
	}

	// Order by
	attr.OrderBy = orderBy

	// Limit and offset
	attr.Paginator = &query.Paginator{
		Offset: previous,
		Limit:  numRows,
	}

	// Create select statement
	selectStmt, _ := query.Select[T](attr)

	// Execute select statement
	sqlRows, err := db.Query(selectStmt, selectArgs...)
	if err != nil {
		return
	}
	defer sqlRows.Close()

	// Get rows
	for sqlRows.Next() {
		var row T
		args, _ := query.Args(row)
		if err = sqlRows.Scan(args...); err != nil {
			return
		}

		// Apply scanned arguments to the row struct fields
		err = query.ArgsAppay(&row, args)
		if err != nil {
			return // Return if ArgsAppay fails
		}
		rows = append(rows, row)
	}
	if err = sqlRows.Err(); err != nil {
		return
	}
	pagination = previous + len(rows)

	return
}

// Count returns the number of rows from the selected T table in the database.
//
// The function accepts a variadic list of Where conditions to filter the rows.
// It constructs a SQL COUNT statement and executes it using the provided
// database connection. The count of rows is returned along with any error
// encountered during the execution.
func Count[T any](db *sql.DB, wheres ...Where) (count int, err error) {

	var attr = &query.SelectAttr{}
	var selectArgs []any

	// Construct where clauses and corresponding arguments
	for _, w := range wheres {
		attr.Wheres = append(attr.Wheres, w.Field+"?")
		selectArgs = append(selectArgs, w.Value)
	}

	// Create SQL COUNT statement
	selectStmt, err := query.Count[T](attr)
	if err != nil {
		return
	}

	// Execute the query
	sqlRows, err := db.Query(selectStmt, selectArgs...)
	if err != nil {
		return
	}
	defer sqlRows.Close()

	// Retrieve the row count
	if sqlRows.Next() {
		err = sqlRows.Scan(&count)
		if err != nil {
			return
		}
	}

	return
}
