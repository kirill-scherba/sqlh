// Copyright 2024-2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Sqlh is a SQL Helper package contains helper functions to execute SQL
// requests. It provides such functions as Execute, Select, Insert, Update and
// Delete.
package sqlh

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"iter"
	"reflect"

	"github.com/kirill-scherba/sqlh/query"
)

// querier is an interface for sql.DB and sql.Tx
type querier interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
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

// WheresJoinOr is a type for query.Args function to join wheres with OR
type WheresJoinOr bool

// Distinct is a type for query.Args function to add DISTINCT clause
type Distinct bool

// Alias is a type for query.Args function to set table alias
type Alias string

// Name is a type for query.Args function to set table name
type Name *string

// SetWheresJoinOr returns a WheresJoinOr type set to true.
// It's used to join wheres conditions with OR instead of AND.
// It's used in the List function.
func SetWheresJoinOr() WheresJoinOr {
	return WheresJoinOr(true)
}

// SetWheresJoinAnd returns a WheresJoinOr type set to false.
// It's used to join wheres conditions with AND instead of OR.
// It's used in the List function.
// The join wheres conditions with AND is the default behavior.
func SetWheresJoinAnd() WheresJoinOr {
	return WheresJoinOr(false)
}

// SetDistinct returns a Distinct type set to true.
// It's used to add DISTINCT clause to the select statement.
func SetDistinct() Distinct {
	return Distinct(true)
}

// SetAlias returns a Alias type with the given alias.
// It's used to set table alias in the select statement.
func SetAlias(alias string) Alias {
	return Alias(alias)
}

// SetName returns a Name type with the given name.
// It's used to set table name in the select statement.
func SetName(name string) Name {
	return Name(&name)
}

// SetNumRows sets numer of rows in List function. It may be get by GetNumRows.
// By default, it is 10.
func SetNumRows(n int) {
	query.SetNumRows(n)
}

// GetNumRows returns default number of rows. It may be set by SetNumRows. By
// default, it is 10.
func GetNumRows() int {
	return query.GetNumRows()
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
func Get[T any](db querier, wheres ...Where) (row *T, err error) {

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
func Count[T any](db querier, wheres ...Where) (count int, err error) {

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
// an error with message "not found". It returns number of rows limited to
// numRows. The default value for numRows is 10. The numRows may be set by
// SetNumRows and get by GetNumRows functions.
func List[T any](db querier, previous int, orderBy string, listAttrs ...any) (
	rows []T, pagination int, err error) {

	// Call ListRows function with default number of rows
	return ListRows[T](db, previous, orderBy, query.GetNumRows(), listAttrs...)
}

// ListRows returns rows from T database table.
//
// The function takes a list of Where condition as input parameter.
// The function executes SELECT statement with the given where conditions.
// If the rows are found, the function returns the rows and nil as error.
// If the rows are not found, the function returns a default value for rows and
// an error with message "not found". It returns number of rows limited to
// numRows.
//
// The listAttrs is a variadic list of Where conditions to filter the rows.
func ListRows[T any](db querier, previous int, orderBy string, numRows int,
	listAttrs ...any) (rows []T, pagination int, err error) {

	// Function to process errors on ListRange
	listAttrs = append(listAttrs, func(e error) { err = e })

	// Prepare rows slice
	rows = make([]T, 0, numRows)

	// Iterate over list records and append it to rows slice
	for _, row := range ListRange[T](db, previous, orderBy, numRows, listAttrs...) {
		rows = append(rows, row)
	}

	// Calculate result pagination
	pagination = previous + len(rows)

	return
}

// ListRange returns an iterator over the rows in the database. It takes a
// querier, a previous number of rows, order by string, number of rows to retrieve,
// and a variadic list of where conditions to filter the rows.
// The returned iterator yields each row in the database, and will stop yielding
// when all the rows have been retrieved or when the yield function returns false.
// The yielded value is a pointer to a struct of type T, and a new instance of
// the struct is created for each yielded value.
// To check for errors, add a function of type func(error) to the query
// arguments (listAttrs parameter of this function). The range will stop on any
// error returned by the function.
func ListRange[T any](db querier, offset int, orderBy string, limit int,
	listAttrs ...any) iter.Seq2[int, T] {

	// Get errorFunc and ctx from listAttrs
	listAttrs, errFunc, ctx := getErrfuncAndCtx(listAttrs)

	// Return iterator
	return func(yield func(i int, row T) bool) {

		// Create select statement and get select arguments
		stmt, args, err := listStatement[T](offset, orderBy, limit, listAttrs...)
		if err != nil {
			errFunc(err)
			return
		}

		// Add error function and ctx to arguments
		args = append(args, errFunc, ctx)

		// Iterate over rows
		var i = offset
		for row := range QueryRange[struct{ In T }](db, stmt, args...) {
			if !yield(i, row.In) {
				break
			}
			i++
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

	// Get errorFunc and ctx from listAttrs
	queryArgs, errFunc, ctx := getErrfuncAndCtx(queryArgs)

	// Return iterator
	return func(yield func(row T) bool) {

		// Execute query
		sqlRows, err := db.QueryContext(ctx, selectQuery, queryArgs...)
		if err != nil {
			err = fmt.Errorf("failed to execute query: %w", err)
			errFunc(err)
			return
		}
		defer sqlRows.Close()

		var yieldArg T
		yieldValue := reflect.ValueOf(&yieldArg).Elem()
		rowVal := reflect.New(yieldValue.Type()).Elem()

		// Iterate over rows
		for sqlRows.Next() {
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
			if err := sqlRows.Scan(scanArgs...); err != nil {
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

		// Check for errors in rows.Next
		if err := sqlRows.Err(); err != nil {
			// err = fmt.Errorf("failed to iterate rows: %w", err)
			errFunc(err)
		}
	}
}

// getErrfuncAndCtx gets func(error) and context from attrs and remove it from
// resut list of attrs. If func(error) and(or) context not found,
// return default values for them.
func getErrfuncAndCtx(attrs []any) (result []any, errFunc func(error),
	ctx context.Context) {

	// Set default values for errFunc and ctx
	errFunc = func(error) {}
	ctx = context.Background()

	// Range over attrs and get errFunc and ctx and create result
	for i := range attrs {
		switch v := attrs[i].(type) {
		case func(error):
			errFunc = v
		case context.Context:
			ctx = v
		default:
			result = append(result, v)
		}
	}

	return
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

// listStatement creates a SELECT statement for the given type T.
//
// It takes a previous number of rows, order by string, number of rows to retrieve,
// and a variadic list of where conditions to filter the rows.
// The function returns a pointer to the SELECT statement and a slice of arguments
// for the WHERE conditions, and an error if encountered.
// The returned pointer to the SELECT statement contains the rows retrieved from the database.
// The error is returned if the query creation fails.
//
// The listAttrs parameter is a variadic list of attributes, it may contain the
// following types:
//
//   - Where - represents a WHERE condition
//   - WheresJoinOr - represents a type of join wheres with OR instead of AND by default
//   - query.Join  - represents attributes for JOIN statement
//   - string - represents the alias for the SELECT table
//   - bool - represents a DISTINCT clause
//   - *string - represents the name of the SELECT table
func listStatement[T any](previous int, orderBy string, numRows int,
	listAttrs ...any) (selectStmt string, selectArgs []any, err error) {

	var attr = &query.SelectAttr{}
	var wheres []Where

	// Parse list attributes and set it to attr
	for _, listAttr := range listAttrs {
		switch v := listAttr.(type) {
		case Where:
			wheres = append(wheres, v)
		case WheresJoinOr:
			attr.WheresJoinOr = bool(v)
		case query.Join:
			attr.Joins = append(attr.Joins, v)
		case Alias:
			attr.Alias = string(v)
		case Distinct:
			attr.Distinct = bool(v)
		case Name:
			attr.Name = v
		default:
			err = fmt.Errorf("invalid list attribute type %T", listAttr)
			return
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
	selectStmt, err = query.Select[T](attr)
	return
}
