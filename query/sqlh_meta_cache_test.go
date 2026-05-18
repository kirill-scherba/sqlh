// Copyright 2024-2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package query

import (
	"reflect"
	"testing"
	"time"

	"github.com/kirill-scherba/sqlh/query/testtypes/pkg1"
	"github.com/kirill-scherba/sqlh/query/testtypes/pkg2"
)

// TestGetMeta_cacheHit verifies that repeated calls return the same cached
// pointer and that distinct struct types have distinct metadata.
func TestGetMeta_cacheHit(t *testing.T) {
	type UserA struct {
		ID   int    `db:"id"`
		Name string `db:"name"`
	}
	type UserB struct {
		ID    int    `db:"id"`
		Email string `db:"email"`
	}

	metaA1 := getMeta(reflect.TypeOf(UserA{}))
	metaA2 := getMeta(reflect.TypeOf(UserA{}))
	if metaA1 != metaA2 {
		t.Error("expected cache hit for same type")
	}

	metaB := getMeta(reflect.TypeOf(UserB{}))
	if metaA1 == metaB {
		t.Error("expected different metadata for different types")
	}
}

// TestGetMeta_fields verifies that field lists are correctly extracted and
// cached, including autoincrement and skip handling.
func TestGetMeta_fields(t *testing.T) {
	type Item struct {
		ID    int    `db:"id" db_key:"autoincrement"`
		Title string `db:"title"`
		Skip  int    `db:"-"`
	}

	meta := getMeta(reflect.TypeOf(Item{}))

	wantAll := []string{"id", "title"}
	if !reflect.DeepEqual(meta.fieldsAll, wantAll) {
		t.Errorf("fieldsAll = %v, want %v", meta.fieldsAll, wantAll)
	}

	wantNoAuto := []string{"title"}
	if !reflect.DeepEqual(meta.fieldsNoAuto, wantNoAuto) {
		t.Errorf("fieldsNoAuto = %v, want %v", meta.fieldsNoAuto, wantNoAuto)
	}

	if len(meta.fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(meta.fields))
	}
	if meta.fields[0].dbName != "id" || !meta.fields[0].isAutoIncrement {
		t.Error("expected field 0 (ID) to be dbName='id' and autoincrement")
	}
	if meta.fields[1].dbName != "title" {
		t.Errorf("field 1 dbName = %q, want %q", meta.fields[1].dbName, "title")
	}
	if !meta.fields[2].skip {
		t.Error("expected field 2 (Skip) to be skipped")
	}
}

// TestGetMeta_tableName verifies table name resolution: struct name lowercased,
// db_table_name tag, and TableName interface.
func TestGetMeta_tableName(t *testing.T) {
	type MyStruct struct {
		ID int `db:"id"`
	}
	meta1 := getMeta(reflect.TypeOf(MyStruct{}))
	if meta1.tableName != "mystruct" {
		t.Errorf("tableName = %q, want %q", meta1.tableName, "mystruct")
	}

	type TaggedStruct struct {
		TableName string `db_table_name:"custom_table"`
		ID        int    `db:"id"`
	}
	meta2 := getMeta(reflect.TypeOf(TaggedStruct{}))
	if meta2.tableName != "custom_table" {
		t.Errorf("tableName = %q, want %q", meta2.tableName, "custom_table")
	}
}

// TestGetMeta_namedCompositeProjection verifies compatibility with the old
// fields/Name behavior: a named first struct field defines the base table and
// projection for composite JOIN result types.
func TestGetMeta_namedCompositeProjection(t *testing.T) {
	type User struct {
		ID   int64  `db:"id" db_key:"autoincrement"`
		Name string `db:"name"`
	}
	type Profile struct {
		UserID int64  `db:"user_id"`
		Bio    string `db:"bio"`
	}
	type UserWithProfile struct {
		User    User
		Profile Profile
	}

	meta := getMeta(reflect.TypeOf(UserWithProfile{}))

	if meta.tableName != "user" {
		t.Errorf("tableName = %q, want %q", meta.tableName, "user")
	}

	wantAll := []string{"id", "name"}
	if !reflect.DeepEqual(meta.fieldsAll, wantAll) {
		t.Errorf("fieldsAll = %v, want %v", meta.fieldsAll, wantAll)
	}

	wantNoAuto := []string{"name"}
	if !reflect.DeepEqual(meta.fieldsNoAuto, wantNoAuto) {
		t.Errorf("fieldsNoAuto = %v, want %v", meta.fieldsNoAuto, wantNoAuto)
	}

	if len(meta.fields) != 2 {
		t.Fatalf("expected 2 composite scan fields, got %d", len(meta.fields))
	}
}

// TestGetMeta_timeFieldIsNotComposite verifies that ordinary time.Time columns
// are not mistaken for composite JOIN fields.
func TestGetMeta_timeFieldIsNotComposite(t *testing.T) {
	type Event struct {
		ID        int64 `db:"id"`
		CreatedAt time.Time
	}

	meta := getMeta(reflect.TypeOf(Event{}))

	wantAll := []string{"id", "createdat"}
	if !reflect.DeepEqual(meta.fieldsAll, wantAll) {
		t.Errorf("fieldsAll = %v, want %v", meta.fieldsAll, wantAll)
	}
}

// TestGetMeta_sameNameDifferentPackages is the key regression test: two
// structs named User living in different packages must have independent
// metadata because reflect.Type includes the package path.
func TestGetMeta_sameNameDifferentPackages(t *testing.T) {
	meta1 := getMeta(reflect.TypeOf(pkg1.User{}))
	meta2 := getMeta(reflect.TypeOf(pkg2.User{}))

	if meta1 == meta2 {
		t.Fatal("expected different metadata for pkg1.User and pkg2.User")
	}

	// Verify they actually have different fields
	if reflect.DeepEqual(meta1.fieldsAll, meta2.fieldsAll) {
		t.Errorf("expected different fieldsAll, got identical: %v", meta1.fieldsAll)
	}

	want1 := []string{"id", "name"}
	want2 := []string{"id", "email"}
	if !reflect.DeepEqual(meta1.fieldsAll, want1) {
		t.Errorf("pkg1.User fieldsAll = %v, want %v", meta1.fieldsAll, want1)
	}
	if !reflect.DeepEqual(meta2.fieldsAll, want2) {
		t.Errorf("pkg2.User fieldsAll = %v, want %v", meta2.fieldsAll, want2)
	}
}
