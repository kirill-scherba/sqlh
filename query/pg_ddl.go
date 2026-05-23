// Copyright 2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package query

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

// TablePG returns a PostgreSQL-compatible CREATE TABLE statement for the
// given struct type T.
//
// Key differences from the default Table():
//   - Auto-increment fields (detected via isAutoIncrement) use SERIAL /
//     BIGSERIAL type instead of integer / bigint, and the autoincrement
//     keyword is removed from the column definition (PostgreSQL does not
//     support AUTOINCREMENT as a modifier).
//   - Type mapping uses PostgreSQL-compatible types:
//     tinyint → smallint, bit → boolean, blob → bytea, double → double precision
//
// If the struct uses db_type:"SERIAL" (or BIGSERIAL / SMALLSERIAL) the
// field is already PG-compatible and is left unchanged.
func TablePG[T any]() (string, error) {
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
		fieldType, err := getFieldTypePG(field)
		if err != nil {
			return "", err
		}

		// Get db_key tag
		dbKey := field.Tag.Get("db_key")

		// Use db_key text only if field name is "_"
		if fieldName == "_" {
			if len(dbKey) > 0 {
				dbFields = append(dbFields, strings.TrimRight(dbKey, " "))
			}
			continue
		}

		// Handle autoincrement fields: use SERIAL/BIGSERIAL and remove
		// autoincrement keywords from db_key
		if isAutoIncrement(field) {
			// Determine original non-PG type to pick SERIAL vs BIGSERIAL
			origType, _ := getFieldType(field)
			if dbType := field.Tag.Get("db_type"); dbType == "" {
				// No explicit db_type — convert to SERIAL based on original type
				switch origType {
				case "integer":
					fieldType = "serial"
				case "bigint":
					fieldType = "bigserial"
				case "tinyint":
					fieldType = "smallserial"
				}
			}
			// Remove autoincrement keywords from db_key (PG doesn't support
			// them as column modifiers; SERIAL handles auto-generation)
			dbKeyLower := strings.ToLower(dbKey)
			dbKeyLower = strings.ReplaceAll(dbKeyLower, "autoincrement", "")
			dbKeyLower = strings.ReplaceAll(dbKeyLower, "auto_increment", "")
			dbKey = strings.TrimSpace(dbKeyLower)
		}

		dbFields = append(
			dbFields,
			strings.TrimRight(
				fmt.Sprintf("%s %s %s", strings.ToLower(fieldName), fieldType, dbKey),
				" ",
			),
		)
	}

	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s);",
		Name[T](),
		strings.Join(dbFields, ", "),
	), nil
}

// getFieldTypePG returns a PostgreSQL-compatible SQL type for the field.
// It respects db_type tag if set; otherwise uses PG-friendly defaults.
func getFieldTypePG(field reflect.StructField) (fieldType string, err error) {
	fieldType = field.Tag.Get("db_type")
	if fieldType == "" {
		switch field.Type.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			fieldType = "integer"
		case reflect.Uint8:
			fieldType = "smallint"
		case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			fieldType = "bigint"
		case reflect.Float32, reflect.Float64:
			fieldType = "double precision"
		case reflect.Bool:
			fieldType = "boolean"
		case reflect.String:
			fieldType = "text"
		case reflect.Slice:
			if field.Type.Elem().Kind() == reflect.Uint8 {
				fieldType = "bytea"
			} else {
				err = fmt.Errorf("unsupported slice type: %s", field.Type)
			}
		case reflect.Struct:
			if field.Type == reflect.TypeOf(time.Time{}) {
				fieldType = "timestamp"
			} else {
				err = fmt.Errorf("unsupported struct type: %s", field.Type)
			}
		case reflect.Complex64, reflect.Complex128:
			fieldType = "bytea"
		default:
			err = fmt.Errorf("unsupported type: %s", field.Type.Kind())
		}
	}
	return
}
