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
	"strings"
	"time"

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

// Eq returns a Where clause for equality: field = value.
func Eq(field string, value any) Where {
	return Where{Field: field + "=", Value: value}
}

// Ne returns a Where clause for not-equal: field <> value.
func Ne(field string, value any) Where {
	return Where{Field: field + "<>", Value: value}
}

// Gt returns a Where clause for greater-than: field > value.
func Gt(field string, value any) Where {
	return Where{Field: field + ">", Value: value}
}

// Gte returns a Where clause for greater-than-or-equal: field >= value.
func Gte(field string, value any) Where {
	return Where{Field: field + ">=", Value: value}
}

// Lt returns a Where clause for less-than: field < value.
func Lt(field string, value any) Where {
	return Where{Field: field + "<", Value: value}
}

// Lte returns a Where clause for less-than-or-equal: field <= value.
func Lte(field string, value any) Where {
	return Where{Field: field + "<=", Value: value}
}

// Like returns a Where clause for LIKE pattern matching.
func Like(field string, value any) Where {
	return Where{Field: field + " LIKE", Value: value}
}

// In returns a Where clause for IN operator with variadic values.
// The values are expanded into individual bind parameters at query time.
// Callers must ensure the slice is non-empty; an empty slice produces
// "field IN ()" which is a SQL syntax error.
func In(field string, values ...any) Where {
	return Where{Field: field + " IN", Value: values}
}

// IsNull returns a Where clause for IS NULL.
func IsNull(field string) Where {
	return Where{Field: field + " IS NULL", Value: nil}
}

// IsNotNull returns a Where clause for IS NOT NULL.
func IsNotNull(field string) Where {
	return Where{Field: field + " IS NOT NULL", Value: nil}
}

// processWhere converts a Where struct into a SQL fragment string and
// corresponding bind arguments. It handles IN expansion and NULL predicates.
func processWhere(w Where) (fragment string, args []any) {
	if w.Value == nil {
		return w.Field, nil
	}

	// Detect IN operator
	field := strings.TrimSpace(w.Field)
	if strings.HasSuffix(field, " IN") {
		rv := reflect.ValueOf(w.Value)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			if rv.Len() == 0 {
				return field + " ()", nil
			}
			places := make([]string, rv.Len())
			for i := range rv.Len() {
				places[i] = "?"
				args = append(args, rv.Index(i).Interface())
			}
			return field + " (" + strings.Join(places, ", ") + ")", args
		}
		// Single value: treat as single-element IN
		return field + " (?)", []any{w.Value}
	}

	return field + " ?", []any{w.Value}
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

// Create creates the SQL table for the T type.
//
// It takes a database connection as a parameter and returns an error if the
// table could not be created.
//
// The function does not start a transaction, so it is up to the caller to manage
// transactions if needed.
func Create[T any](db *sql.DB) (err error) {

	// Detect dialect (needed for PG-specific DDL)
	dialect := detectDialect(db)

	// Create the SQL table for the T type
	var createStm string
	if dialect == dialectPostgreSQL {
		createStm, err = query.TablePG[T]()
	} else {
		createStm, err = query.Table[T]()
	}
	if err != nil {
		return
	}

	// Create the SQL table for the T type
	_, err = execDb(db, createStm, dialect)
	return
}

// Insert inserts rows into the T database table.
//
// It accepts a variadic number of rows of type T and inserts them into the
// corresponding database table. The function starts a transaction and prepares
// an insert statement. Each row is then inserted in a loop. If any error occurs,
// the transaction is rolled back. Otherwise, the transaction is committed.
func Insert[T any](db *sql.DB, rows ...T) (err error) {
	return InsertWithCallback(db, nil, rows...)
}

// InsertId inserts rows into the T database table and returns the last inserted
// row ID.
//
// It accepts a variadic number of rows of type T and inserts them into the
// database table. The function starts a transaction and prepares
// an insert statement. Each row is then inserted in a loop. If any error occurs,
// the transaction is rolled back. Otherwise, the transaction is committed.
// The last inserted row ID is returned as a result.
func InsertId[T any](db *sql.DB, rows ...T) (id int64, err error) {
	tableName := query.Name[T]()
	// Call insertWithCallback function
	err = InsertWithCallback(db,
		// Callback function which returns last inserted row ID
		func(db *sql.DB, tx *sql.Tx) error {
			id, err = getLastInsertID(db, tx, tableName)
			return err
		},
		// Rows to insert
		rows...)
	return
}

// InsertWithCallback inserts rows into the T database table and calls the
// callback function after the rows are successfully inserted.
//
// The function accepts a database connection, a callback function, and a
// variadic number of rows of type T. The callback function is called after the
// rows are successfully inserted. If any error occurs, the transaction is
// rolled back. Otherwise, the transaction is committed.
//
// The callback function is called with the database connection and the
// transaction object as parameters instead of transaction. If the callback
// function returns an error, the transaction is rolled back. Otherwise, the
// transaction is committed.
//
// The function returns an error if any error occurs.
func InsertWithCallback[T any](
	// Database connection
	db *sql.DB,
	// Callback function which calls after rows are successfully inserted
	callback func(db *sql.DB, tx *sql.Tx) error,
	// Rows to insert
	rows ...T,
) (err error) {

	// Detect dialect (needed for PG placeholder rebinding)
	dialect := detectDialect(db)

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

	// Commit or rollback transaction depending on error and callback function
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}

		// Call callback function
		if callback != nil {
			err = callback(db, tx)
			if err != nil {
				tx.Rollback()
				return
			}
		}

		// Commit transaction
		err = tx.Commit()
	}()

	// Create prepared insert statement (rebind for PG)
	stmt, err := tx.Prepare(rebind(insertStmt, dialect))
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
		_, err = execStmt(stmt, args...)
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

	// Detect dialect (needed for PG placeholder rebinding)
	dialect := detectDialect(db)

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

	// Update rows. Each iteration uses its own prepared statement and closes
	// it before moving on so that statement handles do not pile up on the
	// transaction when many attrs are processed in one call.
	for _, attr := range attrs {
		if err = updateOne(tx, dialect, attr); err != nil {
			return
		}
	}

	return
}

// updateOne runs a single UPDATE within the given transaction. It is split
// out from Update so that the prepared statement is closed at the end of
// each iteration via a function-scoped defer, instead of accumulating one
// defer per attribute on the parent Update frame.
func updateOne[T any](tx *sql.Tx, dialect string, attr UpdateAttr[T]) (err error) {

	// Create where clause
	var whereFragments []string
	var whereArgs []any
	for _, where := range attr.Wheres {
		frag, a := processWhere(where)
		whereFragments = append(whereFragments, frag)
		whereArgs = append(whereArgs, a...)
	}

	// Create update statement
	updateStmt, err := query.Update[T](whereFragments...)
	if err != nil {
		return
	}

	// Create prepared update statement (rebind for PG)
	stmt, err := tx.Prepare(rebind(updateStmt, dialect))
	if err != nil {
		return
	}
	defer stmt.Close()

	// Create struct attr.Row field values array
	args, err := query.Args(attr.Row, forWrite)
	if err != nil {
		return
	}

	// Add where conditions to args array
	args = append(args, whereArgs...)

	// Execute update statement
	_, err = execStmt(stmt, args...)
	return
}

// extractColumn extracts the bare column name from a Where.Field string by
// stripping the trailing operator. It handles all documented operators:
// =, >, <, >=, <=, <>, !=, LIKE, IN, BETWEEN, IS NULL, IS NOT NULL.
//
// Longer operators are checked first to avoid partial matches (e.g. >= vs =).
// Examples:
//   "id=" → "id"
//   "age>=" → "age"
//   "name LIKE" → "name"
//   "deleted IS NULL" → "deleted"
func extractColumn(field string) string {
	field = strings.TrimSpace(field)
	operators := []string{
		" IS NOT NULL",
		" IS NULL",
		" NOT IN",
		" LIKE",
		" IN",
		" BETWEEN",
		">=",
		"<=",
		"<>",
		"!=",
		">",
		"<",
		"=",
	}
	for _, op := range operators {
		if strings.HasSuffix(field, op) {
			return strings.TrimSpace(field[:len(field)-len(op)])
		}
	}
	return field
}

// buildUpsertSQL generates a database-native UPSERT SQL statement.
// Returns a non-empty string when dialect is one of the supported drivers
// (postgres, mysql, sqlite). For unsupported drivers it returns "" so
// the caller falls back to the legacy SELECT-then-INSERT/UPDATE path.
func buildUpsertSQL[T any](conflictFields, fieldNames []string, dialect string) (string, error) {
	insertStmt, err := query.Insert[T]()
	if err != nil {
		return "", err
	}
	// Drop trailing semicolon so we can append the conflict clause.
	if len(insertStmt) > 0 && insertStmt[len(insertStmt)-1] == ';' {
		insertStmt = insertStmt[:len(insertStmt)-1]
	}

	switch dialect {
	case dialectPostgreSQL:
		var assigns []string
		for _, f := range fieldNames {
			assigns = append(assigns, fmt.Sprintf("%s = EXCLUDED.%s", f, f))
		}
		conflict := strings.Join(conflictFields, ", ")
		if conflict == "" {
			return fmt.Sprintf("%s ON CONFLICT DO UPDATE SET %s;",
				insertStmt, strings.Join(assigns, ", ")), nil
		}
		return fmt.Sprintf("%s ON CONFLICT (%s) DO UPDATE SET %s;",
			insertStmt, conflict, strings.Join(assigns, ", ")), nil

	case dialectSQLite:
		var assigns []string
		for _, f := range fieldNames {
			assigns = append(assigns, fmt.Sprintf("%s = excluded.%s", f, f))
		}
		conflict := strings.Join(conflictFields, ", ")
		if conflict == "" {
			return fmt.Sprintf("%s ON CONFLICT DO UPDATE SET %s;",
				insertStmt, strings.Join(assigns, ", ")), nil
		}
		return fmt.Sprintf("%s ON CONFLICT (%s) DO UPDATE SET %s;",
			insertStmt, conflict, strings.Join(assigns, ", ")), nil

	case dialectMySQL:
		var assigns []string
		for _, f := range fieldNames {
			assigns = append(assigns, fmt.Sprintf("%s = VALUES(%s)", f, f))
		}
		return fmt.Sprintf("%s ON DUPLICATE KEY UPDATE %s;",
			insertStmt, strings.Join(assigns, ", ")), nil

	default:
		return "", nil
	}
}

// Set sets a row in T database table.
//
// The function is atomic and uses a transaction.
// The function takes a list of Where condition as input parameter.
// For PostgreSQL, SQLite, and MySQL it uses database-native UPSERT
// (ON CONFLICT / ON DUPLICATE KEY UPDATE), providing true atomicity
// without the race window of SELECT-then-INSERT/UPDATE.
// For unsupported or unknown drivers it falls back to the legacy
// SELECT-then-INSERT/UPDATE behaviour.
func Set[T any](db *sql.DB, row T, wheres ...Where) (err error) {

	// Detect dialect (needed for PG placeholder rebinding)
	dialect := detectDialect(db)

	// Extract conflict columns from Where fields.
	var conflictFields []string
	for _, w := range wheres {
		conflictFields = append(conflictFields, extractColumn(w.Field))
	}

	// Attempt to generate native UPSERT SQL.
	fieldNames := query.Fields[T]()
	upsertSQL, upsertErr := buildUpsertSQL[T](conflictFields, fieldNames, dialect)
	if upsertErr != nil {
		return upsertErr
	}

	if upsertSQL != "" {
		// Native UPSERT supported -- execute atomically.
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer func() {
			if err != nil {
				tx.Rollback()
				return
			}
			err = tx.Commit()
		}()
		args, errArgs := query.Args(row, forWrite)
		if errArgs != nil {
			return errArgs
		}
		_, err = execTx(tx, upsertSQL, dialect, args...)
		return err
	}

	// Fallback: legacy SELECT-then-INSERT/UPDATE path.
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
	// Use the internal listRows with explicit dialect so that *sql.Tx is handled correctly.
	rows, _, err := listRows[T](tx, 0, "", "", 2, dialect, wheresToAttrs(wheres)...)
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
		_, err = execTx(tx, insertStmt, dialect, args...)
		if err != nil {
			return // Rollback
		}

	case 1:
		// One row found, update row within the transaction
		var whereFragments []string
		var whereArgs []any
		for _, w := range wheres {
			frag, a := processWhere(w)
			whereFragments = append(whereFragments, frag)
			whereArgs = append(whereArgs, a...)
		}

		updateStmt, errUpdate := query.Update[T](whereFragments...)
		if errUpdate != nil {
			err = errUpdate
			return // Rollback
		}

		args, errArgs := query.Args(row, forWrite)
		if errArgs != nil {
			err = errArgs
			return // Rollback
		}
		args = append(args, whereArgs...)

		_, err = execTx(tx, updateStmt, dialect, args...)
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

	// Detect dialect (needed for PG placeholder rebinding)
	dialect := detectDialect(db)

	// Check if the where clause is required
	if len(wheres) == 0 {
		err = ErrWhereClauseRequired
		return nil, err // Return nil pointer on error
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

	// Get rows from database. Limit to 2 to detect multiple rows.
	// Use the internal listRows with explicit dialect so that *sql.Tx is handled correctly.
	rows, _, err := listRows[T](tx, 0, "", "", 2, dialect, wheresToAttrs(wheres)...)
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

	// Detect dialect (needed for PG placeholder rebinding)
	dialect := detectDialect(db)

	// Prepare where clauses and arguments
	var whereArgs []any
	var whereFields []string
	for _, w := range wheres {
		frag, args := processWhere(w)
		whereArgs = append(whereArgs, args...)
		whereFields = append(whereFields, frag)
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

	// Create prepared delete statement (rebind for PG)
	stmt, err := tx.Prepare(rebind(deleteStmt, dialect))
	if err != nil {
		return
	}
	defer stmt.Close()

	// Execute delete statement with where arguments
	_, err = execStmt(stmt, whereArgs...)
	return
}

// Count returns the number of rows from the selected T table in the database.
//
// The function accepts a variadic list of Where conditions to filter the rows.
// It constructs a SQL COUNT statement and executes it using the provided
// database connection. The count of rows is returned along with any error
// encountered during the execution.
func Count[T any](db querier, wheres ...Where) (count int, err error) {

	dialect := dialectFromQuerier(db)

	var attr = &query.SelectAttr{}
	var selectArgs []any

	// Construct where clauses and corresponding arguments
	for _, w := range wheres {
		frag, args := processWhere(w)
		attr.Wheres = append(attr.Wheres, frag)
		selectArgs = append(selectArgs, args...)
	}

	// Create SQL COUNT statement
	selectStmt, err := query.Count[T](attr)
	if err != nil {
		return
	}

	// Execute the query (rebind for PG)
	sqlRows, err := db.Query(rebind(selectStmt, dialect), selectArgs...)
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

// List returns up to query.GetNumRows() (default 10) rows as a materialized
// slice. It is a convenience wrapper that delegates to ListRows with the default
// page size.
//
// Prefer ListRows for explicit page-size control, or ListRange for lazy/
// streaming iteration. For JOIN queries use ListRange or QueryRange directly.
func List[T any](db querier, previous int, groupBy, orderBy string, listAttrs ...any) (
	rows []T, pagination int, err error) {

	// Call ListRows function with default number of rows
	return ListRows[T](db, previous, groupBy, orderBy, query.GetNumRows(), listAttrs...)
}

// ListRows returns up to numRows rows as a materialized slice. This is the
// preferred API for explicit pagination — pass the returned pagination value
// as the starting offset on the next call.
//
// For lazy/streaming iteration use ListRange, which ListRows wraps internally.
// For raw SQL queries use QueryRange.
func ListRows[T any](db querier, previous int, groupBy, orderBy string, numRows int,
	listAttrs ...any) (rows []T, pagination int, err error) {
	return listRows[T](db, previous, groupBy, orderBy, numRows, dialectFromQuerier(db),
		listAttrs...)
}

// listRows is the internal version of ListRows that accepts an explicit
// dialect string.
func listRows[T any](db querier, previous int, groupBy, orderBy string, numRows int,
	dialect string, listAttrs ...any) (rows []T, pagination int, err error) {

	// Function to process errors on ListRange
	listAttrs = append(listAttrs, func(e error) { err = e })

	// Prepare rows slice
	rows = make([]T, 0, numRows)

	// Iterate over list records and append it to rows slice
	for _, row := range listRange[T](db, previous, groupBy, orderBy, numRows, dialect,
		listAttrs...) {
		rows = append(rows, row)
	}

	// Calculate result pagination
	pagination = previous + len(rows)

	return
}

// ListRange is the core lazy iterator for reading rows from the database.
// It returns an iter.Seq2[int, T] that yields (index, row) pairs, so rows are
// produced on-demand without materialising the full result set in memory.
//
// Use ListRange when you need:
//   - Memory-efficient streaming over large datasets.
//   - Early termination (break out of the range).
//   - JOIN queries with composite structs.
//   - Context-driven cancellation.
//
// Parameters:
//   - offset — starting row position (0 for the first page).
//   - groupBy — GROUP BY expression (empty string to skip).
//   - orderBy — ORDER BY expression (empty string to skip).
//   - limit — maximum rows to yield (0 means unlimited).
//   - listAttrs — variadic attributes such as Where, Join, Alias, Distinct,
//     Name, or an error callback func(error).
//
// List and ListRows are convenience wrappers that collect ListRange results
// into a materialised slice. QueryRange is the low-level raw-SQL sibling.
//
// Example with JOIN:
//
//	for i, row := range ListRange[struct{
//	    *User    // Main table
//	    *Profile // Joined table
//	}](db, 0, "", "u.name ASC", 100,
//	    SetAlias("u"),
//	    query.MakeJoin[Profile](query.Join{On: "u.id = p.user_id", Alias: "p"}),
//	) {
//	    fmt.Printf("%d: %s\n", i, row.User.Name)
//	}
//
// The dialect is auto-detected from the querier. Callers that pass a *sql.Tx
// and know the dialect should use the non-exported listRange overload.
func ListRange[T any](db querier, offset int, groupBy, orderBy string, limit int,
	listAttrs ...any) iter.Seq2[int, T] {
	return listRange[T](db, offset, groupBy, orderBy, limit, dialectFromQuerier(db),
		listAttrs...)
}

// listRange is the internal version of ListRange that accepts an explicit
// dialect string.
func listRange[T any](db querier, offset int, groupBy, orderBy string, limit int,
	dialect string, listAttrs ...any) iter.Seq2[int, T] {

	// Get errorFunc and ctx from listAttrs
	listAttrs, errFunc, ctx := getErrfuncAndCtx(listAttrs)

	// Check ListRange is with join
	var withJoin bool
	for _, attr := range listAttrs {
		if _, ok := attr.(query.Join); ok {
			withJoin = true
			break
		}
	}

	// Return iterator
	return func(yield func(i int, row T) bool) {

		// Create select statement and get select arguments
		stmt, args, err := listStatement[T](offset, groupBy, orderBy, limit, listAttrs...)
		if err != nil {
			errFunc(err)
			return
		}

		// Add error function and ctx to arguments
		args = append(args, errFunc, ctx)

		// Iterate over rows in request with join
		if withJoin {
			var i = offset
			for row := range queryRange[T](db, stmt, dialect, args...) {
				if !yield(i, row) {
					break
				}
				i++
			}
			return
		}

		// Iterate over rows in request without join
		var i = offset
		for row := range queryRange[struct{ In T }](db, stmt, dialect, args...) {
			if !yield(i, row.In) {
				break
			}
			i++
		}
	}
}

// QueryRange returns an iterator over the rows in the database for an arbitrary
// SELECT query. The caller provides the complete SQL statement and its arguments;
// sqlh handles struct scanning via reflection.
//
// Use QueryRange when the auto-generated query (Where / Join attribute system)
// is insufficient and you need full control over the SELECT statement. For
// auto-generated queries prefer ListRange or ListRows.
//
// The returned iter.Seq[T] yields each scanned row. Stop iterating early by
// using break in the range statement.
//
// Error handling: include a func(error) in queryArgs to receive scan or
// execution errors.
//
// The dialect is auto-detected from the querier. Callers that pass a *sql.Tx
// and know the dialect should use the non-exported queryRange overload.
func QueryRange[T any](db querier, selectQuery string, queryArgs ...any) iter.Seq[T] {
	return queryRange[T](db, selectQuery, dialectFromQuerier(db), queryArgs...)
}

// queryRange is the internal version of QueryRange that accepts an explicit
// dialect string. It is used by functions that know the dialect from a
// surrounding *sql.DB call and pass a *sql.Tx as the querier.
func queryRange[T any](db querier, selectQuery string, dialect string,
	queryArgs ...any) iter.Seq[T] {

	// Get errorFunc and ctx from listAttrs
	queryArgs, errFunc, ctx := getErrfuncAndCtx(queryArgs)

	// Return iterator
	return func(yield func(row T) bool) {

		// Execute query (rebind for PG)
		sqlRows, err := db.QueryContext(ctx, rebind(selectQuery, dialect), queryArgs...)
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
				if err := query.ArgsApply(fieldPtr.Interface(), argsByStruct[i]); err != nil {
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

// getLastInsertID returns the last inserted row ID for the given database
// connection and transaction. It supports SQLite, MySQL, PostgreSQL and
// SQL Server. The tableName argument is required for PostgreSQL, which has
// no global "last insert id" and must look the value up via the table's
// serial sequence (pg_get_serial_sequence). If the database driver is not
// supported, it returns an error.
func getLastInsertID(db *sql.DB, tx *sql.Tx, tableName string) (id int64, err error) {

	// Get driver name
	driverName := reflect.TypeOf(db.Driver()).String()

	// Get last inserted row ID
	switch {
	case strings.Contains(driverName, "sqlite"):
		err = tx.QueryRow("SELECT last_insert_rowid()").Scan(&id)
	case strings.Contains(driverName, "mysql"):
		err = tx.QueryRow("SELECT LAST_INSERT_ID()").Scan(&id)
	case strings.Contains(driverName, "postgres"),
		strings.Contains(driverName, "pq"),
		strings.Contains(driverName, "pgx"):
		// PostgreSQL has no session-wide LastInsertId. We look up the
		// table's serial sequence at runtime instead of hardcoding a name.
		// The "id" column name is assumed by sqlh convention; tables with a
		// differently named auto-increment column should use
		// InsertWithCallback with an explicit RETURNING query instead.
		if tableName == "" {
			err = fmt.Errorf("sqlh: PostgreSQL InsertId requires a table name")
			return
		}
		err = tx.QueryRow(
			"SELECT currval(pg_get_serial_sequence($1, 'id'))",
			tableName,
		).Scan(&id)
	case strings.Contains(driverName, "sqlserver"):
		err = tx.QueryRow("SELECT SCOPE_IDENTITY()").Scan(&id)
	default:
		err = fmt.Errorf("unsupported database driver")
	}

	return
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
func listStatement[T any](previous int, groupBy, orderBy string, numRows int,
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
		frag, args := processWhere(w)
		attr.Wheres = append(attr.Wheres, frag)
		selectArgs = append(selectArgs, args...)
	}

	// Group by
	attr.GroupBy = groupBy

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

// Number of retries and delay when database is locked
const numRetries = 20
const retryDelay = 100 * time.Millisecond

func execDb(db *sql.DB, query string, dialect string, args ...any) (result sql.Result, err error) {
	return execRetries(func() (sql.Result, error) {
		return db.Exec(rebind(query, dialect), args...)
	})
}

// execStmt executes a query with the given arguments on the given statement.
// It takes a statement object, a query string, and a variadic list of arguments.
// The function returns a pointer to the result of the query and an error if encountered.
// If the query execution fails, the function returns an error immediately.
// If the query execution is successful, the function returns a pointer to the result of the query.
// If the query execution fails due to a "database is locked" error, the function
// retries the query execution up to numRetries times with a retryDelay delay between retries.
func execStmt(stmt *sql.Stmt, args ...any) (result sql.Result, err error) {
	return execRetries(func() (sql.Result, error) {
		return stmt.Exec(args...)
	})
}

// execTx executes a query with the given arguments on the given transaction.
// It takes a transaction object, a query string, and a variadic list of arguments.
// The function returns a pointer to the result of the query and an error if encountered.
// If the query execution fails, the function returns an error immediately.
// If the query execution is successful, the function returns a pointer to the result of the query.
// If the query execution fails due to a "database is locked" error, the function
// retries the query execution up to numRetries times with a retryDelay delay between retries.
func execTx(tx *sql.Tx, query string, dialect string, args ...any) (result sql.Result, err error) {
	return execRetries(func() (sql.Result, error) {
		return tx.Exec(rebind(query, dialect), args...)
	})
}

// execRetries is a helper function to execute a function that
// returns a sql.Result and error, retrying up to numRetries times
// in case of a transient "database is locked" / busy error. It sleeps
// for retryDelay between retries. Errors that are not lock-related are
// returned immediately without retrying.
func execRetries(f func() (sql.Result, error)) (result sql.Result, err error) {
	for range numRetries {
		result, err = f()
		if err == nil {
			return
		}
		if !isLockError(err) {
			return
		}
		time.Sleep(retryDelay)
	}
	return
}

// isLockError reports whether err looks like a transient lock or busy
// error from any supported driver. The check is intentionally string-based
// because the concrete driver error types are imported as side-effect only
// blank imports, and we do not want to pull driver packages into the public
// API of sqlh. It accepts wrapped errors by inspecting the full Error()
// text.
func isLockError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "database is locked"):
		return true
	case strings.Contains(msg, "database table is locked"):
		return true
	case strings.Contains(msg, "SQLITE_BUSY"):
		return true
	}
	return false
}
