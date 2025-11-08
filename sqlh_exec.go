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
	"fmt"
	"iter"
	"reflect"

	"github.com/kirill-scherba/sqlh/query"
)

var numRows = 10 // number of rows to get in select query

// querier is an interface for sql.DB and sql.Tx
type querier interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

// Constants for query.Args function
const forWrite = true
const forRead = false

// Exported errors
var (
	ErrWhereClauseRequired = errors.New("sqlh: the where clause is required")
	ErrMultipleRowsFound   = errors.New("sqlh: multiple rows found")

	// Re-exported errors from the query package
	ErrTypeIsNotStruct              = query.ErrTypeIsNotStruct
	ErrWhereClauseRequiredForUpdate = query.ErrWhereClauseRequiredForUpdate
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

	// Commit or rollback transaction
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	// Create prepared insert statement
	stmt, err := tx.Prepare(insertStmt)
	if err != nil {
		return
	}
	defer stmt.Close()

	// Insert rows
	for _, row := range rows {
		// Get arguments from the row
		args, errArgs := query.Args(row, forWrite)
		if errArgs != nil {
			err = errArgs
			return
		}
		// Execute insert statement with arguments
		_, err = stmt.Exec(args...)
		if err != nil {
			return
		}
	}
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

	// Commit or rollback transaction
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	// Update rows
	for _, attr := range attrs {

		// Create where clause
		var wheres []string
		for _, where := range attr.Wheres {
			wheres = append(wheres, where.Field)
		}

		// Create update statement
		updateStmt, errUpdate := query.Update[T](wheres...)
		if errUpdate != nil {
			err = errUpdate
			return
		}

		// Create prepared update statement
		stmt, errPrepare := tx.Prepare(updateStmt)
		if errPrepare != nil {
			err = errPrepare
			return
		}
		defer stmt.Close()

		// Create struct attr.Row field values array
		args, errArgs := query.Args(attr.Row, forWrite)
		if errArgs != nil {
			err = errArgs
			return
		}

		// Add where conditions to args array
		for _, where := range attr.Wheres {
			args = append(args, where.Value)
		}

		// Execute update statement
		_, err = stmt.Exec(args...)
		if err != nil {
			return
		}
	}

	return
}

// Set sets a row in T database table.
//
// The function is atomic and uses a transaction.
// The function takes a list of Where condition as input parameter.
// The function checks if the row is found in the database.
// If the row is not found, the function inserts a new row.
// If the row is found, the function updates the row.
// If multiple rows are found, the function returns an error with message "multiple rows found".
func Set[T any](db *sql.DB, row T, wheres ...Where) (err error) {

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return
	}

	// Commit or rollback transaction
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	// Get rows from database using the transaction. Limit to 2 to detect multiple rows.
	rows, _, err := ListRows[T](tx, 0, "", 2, wheresToAttrs(wheres)...)
	if err != nil {
		return // Rollback will be called
	}

	// Check if the row is found and perform action within the transaction
	switch len(rows) {
	case 0:
		// No rows found, insert new row within the transaction
		insertStmt, errInsert := query.Insert[T]()
		if errInsert != nil {
			err = errInsert
			return // Rollback
		}
		args, errArgs := query.Args(row, forWrite)
		if errArgs != nil {
			err = errArgs
			return // Rollback
		}
		_, err = tx.Exec(insertStmt, args...)
		if err != nil {
			return // Rollback
		}

	case 1:
		// One row found, update row within the transaction
		var whereFields []string
		var whereValues []any
		for _, where := range wheres {
			whereFields = append(whereFields, where.Field)
			whereValues = append(whereValues, where.Value)
		}

		updateStmt, errUpdate := query.Update[T](whereFields...)
		if errUpdate != nil {
			err = errUpdate
			return // Rollback
		}

		args, errArgs := query.Args(row, forWrite)
		if errArgs != nil {
			err = errArgs
			return // Rollback
		}
		args = append(args, whereValues...)

		_, err = tx.Exec(updateStmt, args...)
		if err != nil {
			return // Rollback
		}

	default:
		// Multiple rows found, return error
		err = ErrMultipleRowsFound
		return // Rollback
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

	// Get rows from database. Limit to 2 to detect multiple rows
	rows, _, err := ListRows[T](db, 0, "", 2, wheresToAttrs(wheres)...)
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

	// Commit or rollback transaction
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	// Create prepared delete statement
	stmt, err := tx.Prepare(deleteStmt)
	if err != nil {
		return
	}
	defer stmt.Close()

	// Execute delete statement with where arguments
	_, err = stmt.Exec(whereArgs...)
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

// List returns rows from T database table.
//
// The function takes a list of Where condition as input parameter.
// The function executes SELECT statement with the given where conditions.
// If the rows are found, the function returns the rows and nil as error.
// If the rows are not found, the function returns a default value for rows and
// an error with message "not found".
func List[T any](db querier, previous int, orderBy string, listAttrs ...any) (
	rows []T, pagination int, err error) {

	// Call ListRows function with default number of rows
	return ListRows[T](db, previous, orderBy, numRows, listAttrs...)
}

// ListRows remains the public API, calling listRows with the *sql.DB
func ListRows[T any](db querier, previous int, orderBy string, numRows int,
	listAttrs ...any) (rows []T, pagination int, err error) {

	// Function to process errors on ListRange
	listAttrs = append(listAttrs, func(e error) {
		err = e
	})

	// Iterate over list records and append it to rows slice
	for row := range ListRange[T](db, previous, orderBy, numRows, listAttrs...) {
		rows = append(rows, row)
	}

	return
}

// ListRange returns an iterator for the given query and list of attributes.
// The iterator yields elements of type T until the end of the result set is reached.
// It takes a list of Where condition as input parameter.
// The function executes SELECT statement with the given where conditions.
// If the rows are found, the function returns an iterator for the rows and nil as error.
func ListRange[T any](db querier, previous int, orderBy string, numRows int,
	listAttrs ...any) iter.Seq[T] {

	// Get func(error) from queryArgs and remove it from queryArgs if present
	var errFunc = func() (errFunc func(error)) {
		errFunc = func(error) {}
		for i := range listAttrs {
			switch v := listAttrs[i].(type) {
			case func(error):
				errFunc = v
				listAttrs = append(listAttrs[:i], listAttrs[i+1:]...)
				return
			}
		}
		return
	}()

	// Return iterator
	return func(yield func(T) bool) {
		// Create and Execute select statement
		sqlRows, err := listQuery[T](db, previous, orderBy, numRows, listAttrs...)
		if err != nil {
			err = fmt.Errorf("failed to execute list query: %w", err)
			errFunc(err)
			return
		}
		defer sqlRows.Close()

		// Iterate over rows
		for sqlRows.Next() {
			// Create a new row
			var row = makeRow[T]()

			// Get arguments
			args, errArgs := query.Args(row, forRead)
			if errArgs != nil {
				// Stop iteration if Args fails
				err = fmt.Errorf("failed to get arguments: %w", errArgs)
				errFunc(err)
				break
			}

			// Scan into row
			if sqlRows.Scan(args...) != nil {
				// Stop iteration if Scan fails
				err = fmt.Errorf("failed to scan row: %w", sqlRows.Err())
				errFunc(err)
				break
			}

			// Apply scanned arguments to the row struct fields
			if err = query.ArgsAppay(&row, args); err != nil {
				// Return if ArgsAppay fails
				err = fmt.Errorf("failed to apply arguments: %w", err)
				errFunc(err)
				break
			}

			// Call yield for each element and check its return value
			if !yield(row) {
				// Stop iteration if yield returns false (e.g., due to a 'break'
				// in the range loop)
				break
			}
		}
	}
}

// QueryRange returns an iterator over the rows in the database. It takes a
// querier, a select query string and a variadic list of query arguments.
// The returned iterator yields each row in the database, and will stop yielding
// when all the rows have been retrieved or when the yield function returns false.
// The yielded value is a pointer to a struct of type T, and a new instance of
// the struct is created for each yielded value.
//
// To check for errors, add a function of type func(error) to the query
// arguments (queryArgs parameter of this function). The range will stop on any
// error returned by the function.
func QueryRange[T any](db querier, selectQuery string, queryArgs ...any) iter.Seq[T] {

	// Get func(error) from queryArgs and remove it from queryArgs if present
	var errFunc func(error) = func(error) {}
	func() {
		for i := range queryArgs {
			switch v := queryArgs[i].(type) {
			case func(error):
				errFunc = v
				queryArgs = append(queryArgs[:i], queryArgs[i+1:]...)
				return
			}
		}
	}()

	// Return iterator
	return func(yield func(row T) bool) {

		// Execute query
		rows, err := db.Query(selectQuery, queryArgs...)
		if err != nil {
			err = fmt.Errorf("failed to execute query: %w", err)
			errFunc(err)
			return
		}
		defer rows.Close()

		var yieldArg T
		yieldValue := reflect.ValueOf(&yieldArg).Elem()
		rowVal := reflect.New(yieldValue.Type()).Elem()

		for rows.Next() {
			// Create a new instance of the yield argument struct for each row.
			// This ensures that each yielded value is a distinct entity.
			scanArgs := make([]any, 0, rowVal.NumField())
			argsByStruct := make([][]any, 0, rowVal.NumField())

			// Prepare scan arguments for all fields in T
			for i := range rowVal.NumField() {
				field := rowVal.Field(i)

				// We need a pointer to the field to scan into it.
				// If the field itself is a pointer, we create a new object for it.
				// If it's a value, we get its address.
				// create new object if it's a pointer
				var fieldPtr reflect.Value
				if field.Kind() == reflect.Pointer {
					// Create new object for the pointer
					newValue := reflect.New(field.Type().Elem())
					// Set the field to point to the new object
					field.Set(newValue)
					// Get the address of the new object
					fieldPtr = newValue
				} else {
					fieldPtr = field.Addr()
				}

				// Get arguments
				args, err := query.Args(fieldPtr.Interface(), forRead)
				if err != nil {
					err = fmt.Errorf("failed to get arguments for field %s: %w", field, err)
					errFunc(err)
					return
				}
				scanArgs = append(scanArgs, args...)
				argsByStruct = append(argsByStruct, args)
			}

			// Scan row
			if err := rows.Scan(scanArgs...); err != nil {
				err = fmt.Errorf("failed to scan row: %w", err)
				errFunc(err)
				return
			}

			// Apply scanned values back to the structs
			for i := range rowVal.NumField() {

				// Apply scanned values
				fieldPtr := rowVal.Field(i).Addr()
				if err := query.ArgsAppay(fieldPtr.Interface(), argsByStruct[i]); err != nil {
					err = fmt.Errorf("failed to apply scanned values to field %s: %w", fieldPtr, err)
					errFunc(err)
					return
				}
			}

			// Yield row
			if !yield(rowVal.Interface().(T)) {
				// Stop iteration if yield returns false (e.g., due to a 'break'
				// in the range loop)
				break
			}
		}
	}
}

// wheresToAttrs converts a slice of Where conditions to a slice of any values.
// It's used to convert Where conditions to a slice of arguments for the
// Exec or Query functions.
func wheresToAttrs(wheres []Where) (listAttrs []any) {
	for _, where := range wheres {
		listAttrs = append(listAttrs, where)
	}
	return
}

// makeRow creates a new row of type T. If T is a pointer, it will create a new pointer
// with default values for its fields. If T is not a pointer, it will return a default
// value for T.
func makeRow[T any]() (row T) {
	rowType := reflect.TypeOf(row)
	if rowType.Kind() == reflect.Pointer {
		row = reflect.New(rowType.Elem()).Interface().(T)
	}
	return
}

// listQuery is an internal implementation for ListRows that works with a querier.
// It takes a querier, previous number of rows, order by string, number of rows to retrieve,
// and a variadic list of where conditions to filter the rows.
// The function returns a pointer to the sql.Rows and an error if encountered.
// The returned pointer to sql.Rows contains the rows retrieved from the database.
// The error is returned if the query execution fails.
func listQuery[T any](q querier, previous int, orderBy string, numRows int,
	listAttrs ...any) (sqlRows *sql.Rows, err error) {

	var attr = &query.SelectAttr{}
	var selectArgs []any
	var wheres []Where

	// Parse list attributes
	for _, listAttr := range listAttrs {
		switch v := listAttr.(type) {
		case Where:
			wheres = append(wheres, v)
		case query.Join:
			attr.Joins = append(attr.Joins, v)
		case string:
			attr.Alias = v
		}
	}

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
	sqlRows, err = q.Query(selectStmt, selectArgs...)
	return
}
