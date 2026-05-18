// Copyright 2026 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package query provides SQL query building helpers with cached struct metadata.
// This file implements a thread-safe cache that uses Go reflection to inspect
// struct types once and reuse the extracted metadata for subsequent operations.
package query

import (
	"reflect"
	"strings"
	"sync"
	"time"
)

// structField holds cached metadata for a single struct field.
type structField struct {
	index           int          // field index in the parent struct
	dbName          string       // database column name (from "db" tag or lower-cased field name)
	skip            bool         // true if the field should be skipped (tag "-" or name "_")
	isAutoIncrement bool         // true if the field is marked as auto-increment
	isComplex       bool         // true for complex64 / complex128 kinds
	isTime          bool         // true when the field type is time.Time
	isBytes         bool         // true when the field type is []byte
	goType          reflect.Type // original Go type of the field
}

// structMeta holds cached metadata for a struct type.
type structMeta struct {
	typ          reflect.Type // the struct type this metadata describes
	tableName    string       // resolved database table name
	fieldsAll    []string     // list of all DB column names
	fieldsNoAuto []string     // list of column names excluding auto-increment fields
	fields       []structField // per-field metadata preserving struct order
}

// metaCache is a concurrency-safe cache mapping reflect.Type to *structMeta.
// It avoids repeated expensive reflection work when the same type is used
// in multiple queries.
var metaCache sync.Map // map[reflect.Type]*structMeta

// getMeta returns cached metadata for the provided type.
// If t is a pointer it is automatically dereferenced before lookup.
// The result is built once by buildMeta and then stored in metaCache.
func getMeta(t reflect.Type) *structMeta {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if cached, ok := metaCache.Load(t); ok {
		return cached.(*structMeta)
	}
	meta := buildMeta(t)
	metaCache.Store(t, meta)
	return meta
}

// buildMeta creates a fresh structMeta for type t by walking every field.
// Pointer types are dereferenced so that both T and *T share the same entry.
// Table name resolution follows this priority:
//
//  1. If the first field is an embedded struct (projection base), use that
//     embedded struct's table name.
//  2. A "db_table_name" struct tag on any field overrides the name.
//  3. If the struct implements the TableName interface, call TableName().
//  4. Default to the lower-cased Go type name.
//
// Embedded structs are flattened for column lists (fieldsAll/fieldsNoAuto)
// but kept as a single composite field in the fields slice so that
// Args/ArgsAppend can recurse into them when reading values.
func buildMeta(t reflect.Type) *structMeta {
	// Dereference pointer types to handle embedded pointer structs.
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	meta := &structMeta{typ: t}
	hasProjectionBase := false

	// Table name logic.
	meta.tableName = strings.ToLower(t.Name())
	for i := range t.NumField() {
		field := t.Field(i)
		if isProjectionBaseField(i, field) {
			hasProjectionBase = true
			embeddedMeta := buildMeta(field.Type)
			meta.tableName = embeddedMeta.tableName
			break
		}
		if tag := field.Tag.Get("db_table_name"); tag != "" {
			meta.tableName = tag
			break
		}
	}
	// Check TableName interface.
	newT := reflect.New(t).Interface()
	if i, ok := newT.(TableName); ok {
		meta.tableName = i.TableName()
	}

	// Fields logic.
	for i := range t.NumField() {
		field := t.Field(i)

		// Handle nested struct (for JOIN composite structs).
		if (hasProjectionBase || field.Anonymous) && isEmbeddedStruct(field) {
			embeddedMeta := buildMeta(field.Type)

			// For fieldsAll/fieldsNoAuto, only expand the FIRST embedded struct
			// to match original fields() behavior with goto begin.
			if i == 0 {
				meta.fieldsAll = append(meta.fieldsAll, embeddedMeta.fieldsAll...)
				meta.fieldsNoAuto = append(meta.fieldsNoAuto, embeddedMeta.fieldsNoAuto...)
			}

			// For Args/ArgsAppend, treat embedded struct as a single field.
			meta.fields = append(meta.fields, structField{
				index:  i,
				dbName: "",
				skip:   false,
				goType: field.Type,
			})
			continue
		}

		db := field.Tag.Get("db")
		if db == "-" || field.Name == "_" {
			meta.fields = append(meta.fields, structField{index: i, skip: true})
			continue
		}

		dbName := db
		if dbName == "" {
			dbName = strings.ToLower(field.Name)
		}

		isAuto := isAutoIncrement(field)

		meta.fieldsAll = append(meta.fieldsAll, dbName)
		if !isAuto {
			meta.fieldsNoAuto = append(meta.fieldsNoAuto, dbName)
		}

		ft := field.Type
		meta.fields = append(meta.fields, structField{
			index:           i,
			dbName:          dbName,
			skip:            false,
			isAutoIncrement: isAuto,
			isComplex:       ft.Kind() == reflect.Complex64 || ft.Kind() == reflect.Complex128,
			isTime:          ft == reflect.TypeOf(time.Time{}),
			isBytes:         ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Uint8,
			goType:          ft,
		})
	}

	return meta
}

// isEmbeddedStruct reports whether field is an embedded struct (or pointer to
// struct). time.Time is explicitly excluded so it is treated as a scalar value.
func isEmbeddedStruct(field reflect.StructField) bool {
	t := field.Type
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Kind() == reflect.Struct && t != reflect.TypeOf(time.Time{})
}

// isProjectionBaseField reports whether field at index 0 is an embedded struct
// used as the projection base. When true, the outer struct inherits the
// embedded struct's table name and flattens its columns.
func isProjectionBaseField(index int, field reflect.StructField) bool {
	return index == 0 && isEmbeddedStruct(field)
}
