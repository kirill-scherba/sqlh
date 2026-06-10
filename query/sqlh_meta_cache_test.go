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

// TestGetMeta_dbTableName_sentinelTypes verifies that db_table_name works on
// any sentinel type (_ bool, _ any, _ string). The field is skipped from
// field lists regardless of its Go type.
func TestGetMeta_dbTableName_sentinelTypes(t *testing.T) {
	t.Run("_bool", func(t *testing.T) {
		type T struct {
			_    bool   `db_table_name:"t_bool"`
			Name string `db:"name"`
		}
		meta := getMeta(reflect.TypeOf(T{}))
		if meta.tableName != "t_bool" {
			t.Errorf("tableName = %q, want %q", meta.tableName, "t_bool")
		}
		wantAll := []string{"name"}
		if !reflect.DeepEqual(meta.fieldsAll, wantAll) {
			t.Errorf("fieldsAll = %v, want %v", meta.fieldsAll, wantAll)
		}
		if len(meta.fields) != 2 || !meta.fields[0].skip {
			t.Errorf("expected field 0 (_) to be skipped, got %+v", meta.fields[0])
		}
	})
	t.Run("_any", func(t *testing.T) {
		type T struct {
			_    any    `db_table_name:"t_any"`
			Name string `db:"name"`
		}
		meta := getMeta(reflect.TypeOf(T{}))
		if meta.tableName != "t_any" {
			t.Errorf("tableName = %q, want %q", meta.tableName, "t_any")
		}
		wantAll := []string{"name"}
		if !reflect.DeepEqual(meta.fieldsAll, wantAll) {
			t.Errorf("fieldsAll = %v, want %v", meta.fieldsAll, wantAll)
		}
		if len(meta.fields) != 2 || !meta.fields[0].skip {
			t.Errorf("expected field 0 (_) to be skipped, got %+v", meta.fields[0])
		}
	})
	t.Run("_string", func(t *testing.T) {
		type T struct {
			_    string `db_table_name:"t_string"`
			Name string `db:"name"`
		}
		meta := getMeta(reflect.TypeOf(T{}))
		if meta.tableName != "t_string" {
			t.Errorf("tableName = %q, want %q", meta.tableName, "t_string")
		}
		wantAll := []string{"name"}
		if !reflect.DeepEqual(meta.fieldsAll, wantAll) {
			t.Errorf("fieldsAll = %v, want %v", meta.fieldsAll, wantAll)
		}
		if len(meta.fields) != 2 || !meta.fields[0].skip {
			t.Errorf("expected field 0 (_) to be skipped, got %+v", meta.fields[0])
		}
	})
	t.Run("_any_with_index_tag", func(t *testing.T) {
		// Combined db_table_name + db_key on the same sentinel _ string
		type T struct {
			_    string `db:"-" db_table_name:"t_combined"`
			_    string `db:"-" db_key:"KEY name_idx (name)"`
			Name string `db:"name"`
		}
		meta := getMeta(reflect.TypeOf(T{}))
		if meta.tableName != "t_combined" {
			t.Errorf("tableName = %q, want %q", meta.tableName, "t_combined")
		}
		wantAll := []string{"name"}
		if !reflect.DeepEqual(meta.fieldsAll, wantAll) {
			t.Errorf("fieldsAll = %v, want %v", meta.fieldsAll, wantAll)
		}
		if len(meta.fields) != 3 || !meta.fields[0].skip || !meta.fields[1].skip {
			t.Errorf("expected fields 0 and 1 to be skipped, got %+v", meta.fields)
		}
	})
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

// TestIsAutoIncrement_caseInsensitive verifies that both SQLite-style
// "autoincrement" and MySQL-style "AUTO_INCREMENT" tags are detected, in any
// case. This guards the historical regression where the second Contains
// search was made against an already lower-cased string.
func TestIsAutoIncrement_caseInsensitive(t *testing.T) {
	tests := []struct {
		name   string
		dbKey  string
		expect bool
	}{
		{"sqlite lower", "autoincrement", true},
		{"sqlite upper", "AUTOINCREMENT", true},
		{"sqlite mixed", "AutoIncrement", true},
		{"sqlite combined", "not null primary key autoincrement", true},
		{"mysql lower", "auto_increment", true},
		{"mysql upper", "AUTO_INCREMENT", true},
		{"mysql mixed", "Auto_Increment", true},
		{"mysql combined", "not null primary key AUTO_INCREMENT", true},
		{"none", "not null", false},
		{"empty", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			field := reflect.StructField{
				Tag: reflect.StructTag(`db_key:"` + tc.dbKey + `"`),
			}
			got := isAutoIncrement(field)
			if got != tc.expect {
				t.Errorf("isAutoIncrement(%q) = %v, want %v", tc.dbKey, got, tc.expect)
			}
		})
	}
}

// TestArgsAppay_deprecatedAlias verifies that the misspelled ArgsAppay still
// delegates to ArgsApply for backward compatibility. The alias is scheduled
// for removal in v1.0.0 — this test should be removed at that point.
//
//nolint:staticcheck // intentionally exercises the deprecated alias
func TestArgsAppay_deprecatedAlias(t *testing.T) {
	type Row struct {
		Name string `db:"name"`
		Age  int64  `db:"age"`
	}

	args, err := Args(Row{}, false)
	if err != nil {
		t.Fatalf("Args: %v", err)
	}
	*args[0].(*any) = "Alice"
	*args[1].(*any) = int64(30)

	var dst Row
	if err := ArgsAppay(&dst, args); err != nil {
		t.Fatalf("ArgsAppay (deprecated alias): %v", err)
	}
	if dst.Name != "Alice" || dst.Age != 30 {
		t.Errorf("ArgsAppay did not apply args, got %+v", dst)
	}
}

// TestGetMeta_mysqlAutoIncrement verifies that a struct with MySQL-style
// AUTO_INCREMENT tag has the field excluded from fieldsNoAuto (used for
// INSERT/UPDATE column lists). This is the integration-level guarantee that
// the case-folding fix in isAutoIncrement reaches the meta cache.
func TestGetMeta_mysqlAutoIncrement(t *testing.T) {
	type MySQLRow struct {
		ID   int64  `db:"id" db_key:"not null primary key AUTO_INCREMENT"`
		Name string `db:"name"`
	}

	meta := getMeta(reflect.TypeOf(MySQLRow{}))

	wantAll := []string{"id", "name"}
	if !reflect.DeepEqual(meta.fieldsAll, wantAll) {
		t.Errorf("fieldsAll = %v, want %v", meta.fieldsAll, wantAll)
	}

	wantNoAuto := []string{"name"}
	if !reflect.DeepEqual(meta.fieldsNoAuto, wantNoAuto) {
		t.Errorf("fieldsNoAuto = %v, want %v (id should be excluded as AUTO_INCREMENT)",
			meta.fieldsNoAuto, wantNoAuto)
	}

	if !meta.fields[0].isAutoIncrement {
		t.Error("expected field 0 (ID) to be marked isAutoIncrement for AUTO_INCREMENT tag")
	}
}
