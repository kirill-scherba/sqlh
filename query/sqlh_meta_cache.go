// Copyright 2024-2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Cache metadata for Structs and Query builder.

package query

import (
	"reflect"
	"strings"
	"sync"
	"time"
)

// structField holds cached metadata for a single struct field.
type structField struct {
	index           int
	dbName          string
	skip            bool
	isAutoIncrement bool
	isComplex       bool
	isTime          bool
	isBytes         bool
	goType          reflect.Type
}

// structMeta holds cached metadata for a struct type.
type structMeta struct {
	typ          reflect.Type
	tableName    string
	fieldsAll    []string
	fieldsNoAuto []string
	fields       []structField
}

var metaCache sync.Map // map[reflect.Type]*structMeta

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

func buildMeta(t reflect.Type) *structMeta {
	// Dereference pointer types to handle embedded pointer structs
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	meta := &structMeta{typ: t}

	// Table name logic
	meta.tableName = strings.ToLower(t.Name())
	for i := range t.NumField() {
		field := t.Field(i)
		if i == 0 && field.Anonymous && isEmbeddedStruct(field) {
			embeddedMeta := buildMeta(field.Type)
			meta.tableName = embeddedMeta.tableName
			break
		}
		if tag := field.Tag.Get("db_table_name"); tag != "" {
			meta.tableName = tag
			break
		}
	}
	// Check TableName interface
	newT := reflect.New(t).Interface()
	if i, ok := newT.(TableName); ok {
		meta.tableName = i.TableName()
	}

	// Fields logic
	for i := range t.NumField() {
		field := t.Field(i)

		// Handle embedded struct (for JOIN composite structs)
		if field.Anonymous && isEmbeddedStruct(field) {
			embeddedMeta := buildMeta(field.Type)

			// For fieldsAll/fieldsNoAuto, only expand the FIRST embedded struct
			// to match original fields() behavior with goto begin
			if i == 0 {
				meta.fieldsAll = append(meta.fieldsAll, embeddedMeta.fieldsAll...)
				meta.fieldsNoAuto = append(meta.fieldsNoAuto, embeddedMeta.fieldsNoAuto...)
			}

			// For Args/ArgsAppay, treat embedded struct as a single field
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

func isEmbeddedStruct(field reflect.StructField) bool {
	return field.Type.Kind() == reflect.Struct ||
		(field.Type.Kind() == reflect.Pointer && field.Type.Elem().Kind() == reflect.Struct)
}
