// Copyright 2024 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Query is SQL Helper Query package contains helper functions to generate SQL
// statements query.
package query

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

var ErrTypeIsNotStruct = fmt.Errorf("type is not a struct")

// SelectAttr defines attributes for SELECT statement.
type SelectAttr struct {
	Paginator *Paginator // Offset and limit (optional)
	Wheres    []string   // Where clauses (optional)
	OrderBy   string     // Order by (optional)
}

// Paginator defines attributes for SELECT statement.
type Paginator struct {
	// Get list of rows from this position. In other words: skip the specified
	// number of rows before starting to output rows.
	Offset int

	// Number of rows to get. If 0, all rows will be returned.
	Limit int
}

// Table returns a SQL CREATE TABLE statement for the given struct type.
//
// The table is created if it does not already exist.
// The function returns an error if the given type is not a struct.
//
// Example:
//
//	// Iput struct
//	type Astuct struct {
//		ID   int	`db:"id" db_type:"integer" db_key:"not null primary key"`
//		Name string
//	}
//
//	// Output CREATE TABLE statement
//	"CREATE TABLE IF NOT EXISTS astuct (id integer not null primary key, name text)"
//
// Struct tagas are used to map database fields to struct fields.
// The tag is optional. Next tags may be used:
//   - db:"some_field_name" - set database field name
//   - db_type:"text" - set database field type
//   - db_key:"not null primary key" - set database field key
func Table[T any]() (string, error) {

	// Check if type is struct
	if err := checkType[T](); err != nil {
		return "", err
	}

	t := reflect.TypeOf(new(T)).Elem()

	var dbFields []string
	for i := 0; i < t.NumField(); i++ {

		field := t.Field(i)

		// Get field name
		fieldName, ok := getFieldName(field)
		if !ok {
			continue
		}

		// Get field type
		fieldType, err := getFieldType(field)
		if err != nil {
			return "", err
		}

		dbFields = append(dbFields,
			strings.TrimRight(
				// Remove trailing spaces from the string
				fmt.Sprintf("%s %s %s", strings.ToLower(fieldName), fieldType,
					field.Tag.Get("db_key")),
				" ",
			),
		)
	}

	// Return CREATE TABLE statement
	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s);",
		name[T](),
		strings.Join(dbFields, ", "),
	), nil
}

// Insert returns a SQL INSERT statement for the given struct type.
//
// The struct may be tagged with "db" tags to specify the database field names.
// If the "db" tag is not specified, the field name will be used as the database
// field name. The returned string is a SQL statement that can be executed
// directly.
func Insert[T any]() (string, error) {

	// Check if type is struct
	if err := checkType[T](); err != nil {
		return "", err
	}

	// Get table field names
	fields := fields[T]()

	// Return INSERT statement
	return fmt.Sprintf("INSERT INTO %s(%s) VALUES(%s);",
		name[T](),
		strings.Join(fields, ","),
		strings.TrimRight(strings.Repeat("?,", len(fields)), ","),
	), nil
}

// Update returns a SQL UPDATE statement for the given struct type.
//
// The wheres parameter is an optional list of where clauses. If specified, the
// where clauses will be joined with " and " and added to the SQL statement.
func Update[T any](wheres ...string) (string, error) {

	// Check if type is struct
	if err := checkType[T](); err != nil {
		return "", err
	}

	// Get field names
	fields := fields[T]()

	// Where clause should be set
	if len(wheres) == 0 {
		return "", fmt.Errorf(
			"where clause should be set in the Update statement",
		)
	}

	// Return UPDATE statement
	return fmt.Sprintf("UPDATE %s SET %s WHERE %s;",
		name[T](),
		strings.Join(fields, "=?,")+"=?",
		strings.Join(wheres, "? AND ")+"?",
	), nil
}

// Select returns a SQL SELECT statement for the given struct type.
//
// The struct may be tagged with "db" tags to specify the database field names.
// If the "db" tag is not specified, the field name will be used as the database
// field name. The returned string is a SQL statement that can be executed
// directly.
//
// The wheres parameter is an optional list of where clauses. If specified, the
// where clauses will be joined with " and " and added to the SQL statement.
func Select[T any](attr *SelectAttr) (string, error) {

	// Check if type is struct
	if err := checkType[T](); err != nil {
		return "", err
	}

	// Make where clause and offset limit from attr struct
	var where string
	var limit string
	var orderby string
	if attr != nil {
		// Where clauses
		if len(attr.Wheres) > 0 {
			where = strings.Join(attr.Wheres, " and ")
		}
		if len(where) > 0 {
			where = fmt.Sprintf(" where %s", where)
		}

		// Order by
		if len(attr.OrderBy) > 0 {
			orderby = fmt.Sprintf(" ORDER BY %s", attr.OrderBy)
		}

		// Offset and limit
		if attr.Paginator != nil {
			switch {
			// No limit or offset
			case !(attr.Paginator.Limit > 0 && attr.Paginator.Offset > 0):

			// Limit is set
			case attr.Paginator.Limit > 0:
				limit = fmt.Sprintf(" LIMIT %d, %d", attr.Paginator.Offset,
					attr.Paginator.Limit)

			// Limit is not set - get all rows
			default:
				limit = fmt.Sprintf(" LIMIT %d, ~0", attr.Paginator.Offset)
			}
		}
	}

	// Return the complete SELECT statement
	return fmt.Sprintf("SELECT * from %s%s%s%s;",
		name[T](),
		where,
		orderby,
		limit,
	), nil
}

// Count returns a SQL COUNT statement for the given struct type.
//
// The struct may be tagged with "db" tags to specify the database field names.
// If the "db" tag is not specified, the field name will be used as the database
// field name. The returned string is a SQL statement that can be executed
// directly.
//
// The wheres parameter is an optional list of where clauses. If specified, the
// where clauses will be joined with " and " and added to the SQL statement.
func Count[T any](attr *SelectAttr) (string, error) {

	// Check if type is struct
	if err := checkType[T](); err != nil {
		return "", err
	}

	// Make where clause and offset limit from attr struct
	var where string
	if attr != nil {
		// Where clauses
		if len(attr.Wheres) > 0 {
			where = strings.Join(attr.Wheres, " and ")
		}
		if len(where) > 0 {
			where = fmt.Sprintf(" where %s", where)
		}
	}

	// Return the complete SELECT statement
	return fmt.Sprintf("SELECT count(*) from %s%s;", name[T](), where), nil
}

// Delete returns a SQL DELETE statement for the given struct type.
//
// The struct may be tagged with "db" tags to specify the database field names.
// If the "db" tag is not specified, the field name will be used as the database
// field name. The returned string is a SQL statement that can be executed
// directly.
//
// The wheres parameter is an optional list of where clauses. If specified, the
// where clauses will be joined with " and " and added to the SQL statement.
func Delete[T any](wheres ...string) (string, error) {

	// Check if type is struct
	if err := checkType[T](); err != nil {
		return "", err
	}

	// Join the where statements with " and "
	var where string
	if len(wheres) > 0 {
		where = strings.Join(wheres, "? AND ")
	}

	// Add the where statement to the SQL query
	if len(where) > 0 {
		where = fmt.Sprintf(" where %s?", where)
	}

	// Return the complete DELETE statement
	return fmt.Sprintf("DELETE from %s%s;", name[T](), where), nil
}

// Args returns the arguments array for the given struct type. The given struct
// may be a pointer to struct or struct.
//
// It loops through the given struct fields and get field values.
// Supported types are string, float64, time.Time, int64 and bool.
// If unsupported type is found, it returns an error.
func Args(row any) ([]interface{}, error) {

	// Get row value and type from the given row
	rowVal := reflect.ValueOf(row)
	rowType := rowVal.Type()
	if rowVal.Kind() == reflect.Ptr {
		rowVal = rowVal.Elem()
		rowType = rowType.Elem()
	}

	// Check if row is struct
	if rowVal.Kind() != reflect.Struct {
		return nil, ErrTypeIsNotStruct
	}

	// Make arguments array for the given struct
	args := make([]interface{}, 0, rowVal.NumField())
	for i := 0; i < rowVal.NumField(); i++ {

		// Skip not db fields tagged with "-"
		if rowType.Field(i).Tag.Get("db") == "-" {
			continue
		}

		arg := rowVal.Field(i).Interface()
		args = append(args, &arg)
	}

	return args, nil
}

// ArgsAppay sets fields values of the given pointer to struct row from the args
// array.
//
// It loops through the given struct fields and sets field values from the
// corresponding arguments in the given args array.
// Supported types are string, float64, time.Time, int64 and bool.
// If unsupported type is found, it returns an error.
func ArgsAppay(row any, args []interface{}) (err error) {

	rowVal := reflect.ValueOf(row).Elem()
	rowType := reflect.TypeOf(row).Elem()
	if rowVal.Kind() == reflect.Ptr {
		rowVal = rowVal.Elem()
		rowType = rowType.Elem()
	}

	// Check if the given value is a struct
	if rowVal.Kind() != reflect.Struct {
		return ErrTypeIsNotStruct
	}

	// Loop through the struct fields
	for i := 0; i < rowVal.NumField(); i++ {

		// Skip not db fields tagged with "-"
		if rowType.Field(i).Tag.Get("db") == "-" {
			continue
		}

		// Get the current field and its value
		f := rowVal.Field(i)
		arg := reflect.ValueOf(args[i]).Elem().Interface()

		// Set the field value based on the type of the argument
		switch v := arg.(type) {
		case string:
			f.SetString(v)
		case float64:
			f.SetFloat(v)
		case time.Time:
			f.Set(reflect.ValueOf(v))
		case int64:
			// Set the field value based on the type of the field
			switch f.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				f.SetInt(v)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				f.SetUint(uint64(v))
			case reflect.Bool:
				f.SetBool(v == 1)
			}
		default:
			// Return an error if unsupported type is found
			err = fmt.Errorf(
				"unknown value type for field %s: %T",
				rowVal.Type().Field(i).Name, v,
			)
		}
	}

	return
}

// checkType checks if the type T is a struct or a pointer to a struct.
//
// It takes the type T as an argument and returns an error if the type is not a
// struct or a pointer to a struct.
func checkType[T any]() (err error) {
	// Get the type of the struct
	t := reflect.TypeOf(new(T)).Elem()

	// If the type is a pointer, get the type of the struct it points to
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Check if the type is a struct
	if t.Kind() != reflect.Struct {
		// Return an error if the type is not a struct
		err = ErrTypeIsNotStruct
	}
	return
}

// name returns table name from struct name.
//
// It takes type T as an argument and returns the table name as a string.
// The table name is the lower case version of the struct name.
func name[T any]() string {
	// Get the type of the struct
	t := reflect.TypeOf(new(T)).Elem()

	// If the type is a pointer, get the type of the struct it points to
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Return the table name as the lower case version of the struct name
	return strings.ToLower(t.Name())
}

// fields returns a list of struct field names.
//
// It takes type T as an argument and returns a slice of strings.
// The slice contains the names of the struct fields.
// The names are determined by the db tag in the struct field.
// If the db tag is not specified, the field name is used as the
// table field name.
func fields[T any]() (fields []string) {
	t := reflect.TypeOf(new(T)).Elem()

	// If the type is a pointer, get the type of the struct it points to
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Loop through the struct fields
	for i := 0; i < t.NumField(); i++ {
		// Get the field
		field := t.Field(i)

		// If the field name is not empty and the db tag is not set to "-"
		// add the field name to the slice
		if fieldName, ok := getFieldName(field); ok {
			fields = append(fields, fieldName)
		}
	}
	return
}

// getFieldName returns a SQL fields name using db tag.
//
// It takes a reflect.StructField as an argument and returns a string
// and a boolean. The string is the name of the SQL field. The boolean
// indicates if the field name was set successfully.
//
// The function first checks if the field has the db tag set.
// If the tag is set, the function returns the value of the tag as the
// field name.
// If the tag is not set, the function returns the name of the field
// as the field name by calling strings.ToLower on the field name.
// If the tag is set to "-", the function returns an empty string and
// false indicating that the field name was not set successfully.
func getFieldName(field reflect.StructField) (fieldName string, ok bool) {
	fieldName = field.Tag.Get("db")
	switch fieldName {
	case "":
		fieldName = strings.ToLower(field.Name)
	case "-":
		return
	}
	ok = true
	return
}

// getFieldType returns a SQL field type using db_type tag.
//
// If the db_type tag is not set, the function tries to infer the type from
// the Go type of the field. The mapping between Go types and SQL types is
// as follows:
//
//	int, int8, int16, int32, int64: "integer"
//	uint8: "tinyint"
//	uint, uint16, uint32, uint64: "bigint"
//	float32, float64: "double"
//	bool: "bit"
//	string: "text"
//
// If the type is not supported, the function returns an error.
func getFieldType(field reflect.StructField) (fieldType string, err error) {

	fieldType = field.Tag.Get("db_type")
	if fieldType == "" {
		switch field.Type.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			// Sql does not support all integer types, so we map them all to "integer"
			fieldType = "integer"
		case reflect.Uint8:
			fieldType = "tinyint"
		case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			fieldType = "bigint"
		case reflect.Float32, reflect.Float64:
			fieldType = "double"
		case reflect.Bool:
			fieldType = "bit"
		case reflect.String:
			fieldType = "text"
		default:
			// If the type is not supported, return an error
			err = fmt.Errorf("unsupported type: %s", field.Type.Kind())
		}
	}

	return
}
