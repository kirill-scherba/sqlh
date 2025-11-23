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

// Exported errors
var (
	ErrTypeIsNotStruct              = fmt.Errorf("type is not a struct")
	ErrWhereClauseRequiredForUpdate = fmt.Errorf("where clause should be set in the Update statement")
)

// numRows is default number of rows to get in select query. By default, it is 10.
// It can be set by SetNumRows function, and get by GetNumRows function.
var numRows = 10

// SelectAttr defines attributes for SELECT statement.
type SelectAttr struct {
	// Offset and limit (optional). Example: "0, 10"
	Paginator *Paginator

	// Where clauses (optional). Example: "id = ?", "name = ?" joined with " and "
	Wheres []string

	// Join wheres by "or" if true
	WheresJoinOr bool

	// Order by (optional). Example: "id desc, name asc"
	OrderBy string

	// Alias (optional). Table name alias used in the fields and joins conditions
	Alias string

	// Joins (optional). List of joins to other tables used in select
	Joins []Join

	// Name (optional) replaces the table name. By default, the table name is
	// taken from the structure type specified when calling the Select function.
	Name string
}

// Join defines attributes for JOIN statement.
type Join struct {
	Join   string   // Join type: inner, left, right, full
	Name   string   // Table name
	Alias  string   // Alias (optional)
	On     string   // On clause
	Fields []string // Fields
	Select string   // Select clause (optional)
}

// MakeJoin takes a Join struct as input parameter and returns a Join struct with
// the name and fields set from the given struct type T.
//
// If the Name field of the input Join struct is empty, the function sets it
// to the table name of the given struct type T.
//
// The function also sets the Fields field of the output Join struct by iterating
// over all fields of the given struct type T and appending them to the
// Fields field. If the Alias field of the input Join struct is not empty, the
// function prefixes each field with the alias and a dot.
//
// The function returns the output Join struct.
func MakeJoin[T any](join Join) (out Join) {

	// Copy join to out
	out = join

	// Set name from table name
	if len(join.Name) == 0 {
		out.Name = Name[T]()
	}

	// Create join fields
	for _, field := range fields[T](true) {
		if len(join.Alias) > 0 {
			field = join.Alias + "." + field
		}
		out.Fields = append(out.Fields, field)
	}

	return
}

// Paginator defines attributes for SELECT statement.
type Paginator struct {
	// Get list of rows from this position. In other words: skip the specified
	// number of rows before starting to output rows.
	Offset int

	// Number of rows to get. If 0, all rows will be returned.
	Limit int
}

// SetNumRows sets default number of rows returned by List function. By default,
// it is 10.
func SetNumRows(n int) {
	if n <= 1 {
		return
	}
	numRows = n
}

// GetNumRows returns default number of rows. It may be set by SetNumRows. By
// default, it is 10.
func GetNumRows() int {
	return numRows
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
	for i := range t.NumField() {

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

		// Use db_key text only if field name is "_"
		if fieldName == "_" {
			dbFields = append(dbFields,
				strings.TrimRight(field.Tag.Get("db_key"), " "),
			)
			continue
		}

		dbFields = append(
			dbFields,
			strings.TrimRight(
				fmt.Sprintf("%s %s %s", strings.ToLower(fieldName), fieldType,
					field.Tag.Get("db_key")),
				" ",
			),
		)
	}

	// Return CREATE TABLE statement
	q := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s);",
		Name[T](),
		strings.Join(dbFields, ", "),
	)

	return q, nil
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
		Name[T](),
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
		return "", ErrWhereClauseRequiredForUpdate
	}

	// Return UPDATE statement
	return fmt.Sprintf("UPDATE %s SET %s WHERE %s;",
		Name[T](),
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
	var joins string
	var joinsFields []string
	var where string
	var limit string
	var orderby string
	var name = Name[T]()
	var fields = fields[T](true)

	// Check attributes
	if attr != nil {

		// Alias
		if len(attr.Alias) > 0 {
			name = name + " " + attr.Alias
		}

		// Joins
		for _, join := range attr.Joins {

			// Make join table
			table := join.Name
			if len(join.Select) > 0 {
				table = "(" + join.Select + ")"
			}

			// Add join
			joins = joins + " " +
				strings.TrimSpace(join.Join+" join") + " " + table + " " +
				join.Alias + " on " + join.On

			// Add join fields
			joinsFields = append(joinsFields, join.Fields...)
		}

		// Where clauses
		if len(attr.Wheres) > 0 {
			// Join wheres by "and" or "or"
			var sep = " and "
			if attr.WheresJoinOr {
				sep = " or "
			}
			where = " where " + strings.Join(attr.Wheres, sep)
		}
		// if len(where) > 0 {
		// 	where = fmt.Sprintf(" where %s", where)
		// }

		// Order by
		if len(attr.OrderBy) > 0 {
			orderby = fmt.Sprintf(" ORDER BY %s", attr.OrderBy)
		}

		// Offset and limit
		if attr.Paginator != nil {
			switch {
			// No limit and offset - get all rows
			case attr.Paginator.Limit <= 0 && attr.Paginator.Offset <= 0:

			// Set limit and offset in request. If limit is not set, set it to
			default:
				// sqlh.SetNumRows(1)
				// limit = fmt.Sprintf(" OFFSET %d", attr.Paginator.Offset)
				n := numRows
				if attr.Paginator.Limit > 0 {
					n = attr.Paginator.Limit
				}
				limit = fmt.Sprintf(" LIMIT %d OFFSET %d",
					n, attr.Paginator.Offset)
			}
		}

		// Make fields
		if len(attr.Alias) > 0 {
			for i := range fields {
				fields[i] = attr.Alias + "." + fields[i]
			}
		}
		// Append joins fields
		fields = append(fields, joinsFields...)
	}

	// Make select fields string
	fieldsStr := strings.Join(fields, ", ")
	// fieldsStr = "*"

	// Return the complete SELECT statement
	return fmt.Sprintf("SELECT %s from %s%s%s%s%s;",
		fieldsStr,
		name,
		joins,
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
	return fmt.Sprintf("SELECT count(*) from %s%s;", Name[T](), where), nil
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
	return fmt.Sprintf("DELETE from %s%s;", Name[T](), where), nil
}

// Args returns the arguments array for the given struct type.
// The given struct may be a pointer to struct or struct.
//
// The forWrite parameter controls the behavior:
//   - If forWrite is true, it returns a slice of values for INSERT/UPDATE,
//     skipping autoincrement fields.
//   - If forWrite is false, it returns a slice of pointers to copies of field values for
//     SELECT (for sql.Scan). These pointers are then used with ArgsAppay to populate the struct.
func Args(row any, forWrite bool) ([]any, error) {

	// Get row value and type from the given row
	rowVal, rowType := getRowVal(row)

	// Check if row is not struct
	if rowVal.Kind() != reflect.Struct {
		return nil, ErrTypeIsNotStruct
	}

	// Make arguments array for the given struct
	args := make([]any, 0, rowVal.NumField())
	for i := range rowVal.NumField() {
		field := rowType.Field(i)

		// For write operations, skip autoincrement fields.
		if forWrite && isAutoIncrement(field) {
			continue
		}

		// Always skip fields tagged with db:"-" or has name "_"
		if field.Tag.Get("db") == "-" || field.Name == "_" {
			continue
		}

		if forWrite {
			// For writing, get the value of the field.
			args = append(args, rowVal.Field(i).Interface())
		} else {
			// For reading/scanning, get a pointer to a copy of the field's value.
			arg := rowVal.Field(i).Interface()
			args = append(args, &arg)
		}
	}

	return args, nil
}

// isAutoIncrement returns true if the given struct field is tagged with
// "autoincrement" or "AUTO_INCREMENT". It is used to skip autoincrement fields
// in INSERT and UPDATE operations.
func isAutoIncrement(field reflect.StructField) bool {
	return strings.Contains(strings.ToLower(field.Tag.Get("db_key")), "autoincrement") ||
		strings.Contains(strings.ToLower(field.Tag.Get("db_key")), "AUTO_INCREMENT")
}

// ArgsAppay sets fields values of the given pointer to struct row from the args
// array.
//
// It loops through the given struct fields and sets field values from the
// corresponding arguments in the given args array.
// Supported types are string, float64, time.Time, int64 and bool.
// If unsupported type is found, it returns an error.
func ArgsAppay(row any, args []any) (err error) {

	// Get row value and type
	rowVal, rowType := getRowValPtr(row)

	// Check if the given value is a struct
	if rowVal.Kind() != reflect.Struct {
		return ErrTypeIsNotStruct
	}

	// Loop through the struct fields
	for i := range rowVal.NumField() {

		// Skip not db fields tagged with "-"
		if rowType.Field(i).Tag.Get("db") == "-" || rowType.Field(i).Name == "_" {
			continue
		}

		// Get the current field and its value
		f := rowVal.Field(i)
		arg := reflect.ValueOf(args[i]).Elem().Interface()

		// Set the field value based on the type of the argument
		switch v := arg.(type) {
		case string:
			f.SetString(v)

		case time.Time:
			f.Set(reflect.ValueOf(v))

		case bool:
			f.SetBool(v)

		case float64:
			f.SetFloat(v)
		case float32:
			f.SetFloat(float64(v))

		// case int:
		// 	f.SetInt(int64(v))
		// case int8:
		// 	f.SetInt(int64(v))
		// case int16:
		// 	f.SetInt(int64(v))
		// case int32:
		// 	f.SetInt(int64(v))
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

		// case uint:
		// 	f.SetUint(uint64(v))
		// case uint8:
		// 	f.SetUint(uint64(v))
		// case uint16:
		// 	f.SetUint(uint64(v))
		// case uint32:
		// 	f.SetUint(uint64(v))
		// case uint64:
		// 	f.SetUint(v)

		case complex64:
			f.SetComplex(complex128(v))
		case complex128:
			f.SetComplex(v)

		case []byte:
			switch {
			// Ensure the target field f in the struct is also []byte
			case f.Kind() == reflect.Slice && f.Type().Elem().Kind() == reflect.Uint8:
				f.SetBytes(v)

			// If the target field is a string, convert []byte to string
			case f.Kind() == reflect.String:
				rawString := string(v)
				f.SetString(rawString)

			// Return an error in other cases
			default:
				err = fmt.Errorf("type mismatch for field %s: "+
					"expected []byte for DB type []byte, but struct field is %s",
					rowType.Field(i).Name, f.Type().String(),
				)
				return
			}

		default:
			// When unsupported type is found, usali it may be nil, we set zero
			// to this field (zero value of v's type)
			f.SetZero()
		}
	}

	return
}

// Name returns table Name from struct Name.
//
// It takes type T as an argument and returns the table Name as a string.
// The table Name is the lower case version of the struct Name.
func Name[T any]() string {
	// Get the type of the struct
	t := reflect.TypeOf(new(T)).Elem()

	// If the type is a pointer, get the type of the struct it points to
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	// Return the table name as the lower case version of the struct name
	return strings.ToLower(t.Name())
}

// getRowVal returns the reflect.Value and reflect.Type of the given row.
//
// It takes the given row as an argument and returns the reflect.Value and
// reflect.Type of the row. If the row is a pointer, it gets the type of the
// struct it points to.
func getRowVal(row any) (rowVal reflect.Value, rowType reflect.Type) {
	rowVal = reflect.ValueOf(row)
	rowType = rowVal.Type()
	if rowVal.Kind() == reflect.Pointer {
		rowVal = rowVal.Elem()
		rowType = rowType.Elem()
	}
	return
}

// getRowValPtr returns the reflect.Value and reflect.Type of the given row.
//
// It takes the given row as an argument and returns the reflect.Value and
// reflect.Type of the row. If the row is a pointer, it gets the type of the
// struct it points to.
//
// It is similar to getRowVal, but it doesn't dereference the type if it is a
// pointer. Instead, it returns the type of the pointer itself.
//
// This function is useful when you want to check the type of the row itself,
// instead of the type of the struct it points to.
func getRowValPtr(row any) (rowVal reflect.Value, rowType reflect.Type) {
	rowVal = reflect.ValueOf(row).Elem()
	rowType = reflect.TypeOf(row).Elem()
	if rowVal.Kind() == reflect.Pointer {
		rowVal = rowVal.Elem()
		rowType = rowType.Elem()
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
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	// Check if the type is a struct
	if t.Kind() != reflect.Struct {
		// Return an error if the type is not a struct
		err = ErrTypeIsNotStruct
	}
	return
}

// fields returns a list of struct field names.
//
// It takes type T as an argument and returns a slice of strings.
// The slice contains the names of the struct fields.
// The names are determined by the db tag in the struct field.
// If the db tag is not specified, the field name is used as the
// table field name.
func fields[T any](alls ...bool) (fields []string) {
	t := reflect.TypeOf(new(T)).Elem()

	// If the type is a pointer, get the type of the struct it points to
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	// Check parameter all
	var all bool
	if len(alls) > 0 && alls[0] {
		all = true
	}

	// Loop through the struct fields
	for i := range t.NumField() {
		// Get the field
		field := t.Field(i)

		// Skip autoincrement fields if all is false
		if !all && isAutoIncrement(field) {
			continue
		}

		// If the field name is not empty and the db tag is not set to "-"
		// add the field name to the slice
		if fieldName, ok := getFieldName(field); ok && fieldName != "_" {
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
		case reflect.Slice:
			// Check if it's a slice of bytes ([]byte)
			if field.Type.Elem().Kind() == reflect.Uint8 {
				fieldType = "blob"
			} else {
				err = fmt.Errorf("unsupported slice type: %s", field.Type)
			}
		case reflect.Struct:
			// Check if it's time.Time
			if field.Type == reflect.TypeOf(time.Time{}) {
				fieldType = "timestamp"
			} else {
				err = fmt.Errorf("unsupported struct type: %s", field.Type)
			}
		default:
			// If the type is not supported, return an error
			err = fmt.Errorf("unsupported type: %s", field.Type.Kind())
		}
	}

	return
}
