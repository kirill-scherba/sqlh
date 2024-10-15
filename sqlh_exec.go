package sqlh

import (
	"database/sql"
	"fmt"

	"gitlab.dev.redpad.games/dustland-server/dadmin/server/sqlh/query"
)

var numRows = 10 // number of rows to get in select query

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

// Insert inserts rows into T database table.
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
		args, err := query.Args(row)
		if err != nil {
			tx.Rollback()
			return err
		}
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
func Update[T any](db *sql.DB, attrs ...UpdateAttr[T]) (err error) {

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return
	}

	// Update rows
	for _, attr := range attrs {

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

// Get returns row from T database table.
func Get[T any](db *sql.DB, wheres ...Where) (row T, err error) {

	if len(wheres) == 0 {
		err = fmt.Errorf("the where clause is required")
		return
	}

	rows, _, err := List[T](db, 0, "", wheres...)
	if err != nil {
		return
	}

	switch len(rows) {
	case 0:
		err = fmt.Errorf("not found")
	case 1:
		row = rows[0]
	default:
		err = fmt.Errorf("multiple rows found")
	}

	return
}

// Delete deletes rows from T database table.
func Delete[T any](db *sql.DB, wheres ...Where) (err error) {

	// Where clauses
	var whereArgs []any
	var wnereFields []string
	for _, w := range wheres {
		whereArgs = append(whereArgs, w.Value)
		wnereFields = append(wnereFields, w.Field)
	}

	// Create delete statement
	deleteStmt, err := query.Delete[T](wnereFields...)
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

	// Delete rows
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
func List[T any](db *sql.DB, previous int, orderBy string, wheres ...Where) (
	rows []T, pagination int, err error) {
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
		query.ArgsAppay(&row, args)
		rows = append(rows, row)
	}
	if err = sqlRows.Err(); err != nil {
		return
	}
	pagination = previous + len(rows)

	return
}

// Count returns number of rows from selected in T table from database.
func Count[T any](db *sql.DB, wheres ...Where) (
	count int, err error) {

	var attr = &query.SelectAttr{}
	var selectArgs []any

	// Where clauses
	for _, w := range wheres {
		attr.Wheres = append(attr.Wheres, w.Field+"?")
		selectArgs = append(selectArgs, w.Value)
	}

	// Create select statement
	selectStmt, _ := query.Count[T](attr)
	sqlRows, err := db.Query(selectStmt, selectArgs...)
	if err != nil {
		return
	}
	defer sqlRows.Close()

	// Get row count
	if sqlRows.Next() {
		err = sqlRows.Scan(&count)
		if err != nil {
			return
		}
	}

	return
}
