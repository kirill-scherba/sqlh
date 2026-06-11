// Copyright 2024-2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bench

import (
	"fmt"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// BenchmarkGORM_Insert measures inserting a single row using GORM.
func BenchmarkGORM_Insert(b *testing.B) {
	db := newGormDB(b)
	createGormTable(b, db)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		err := db.Create(&GormUser{
			Name:  fmt.Sprintf("user-%06d", i),
			Email: fmt.Sprintf("user-%06d@example.com", i),
		}).Error
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGORM_GetByPK measures retrieving a single row by primary key.
func BenchmarkGORM_GetByPK(b *testing.B) {
	db := newGormDB(b)
	createGormTable(b, db)
	seedGormUsers(b, db, 100)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		id := int64(i%100 + 1)
		var u GormUser
		err := db.First(&u, id).Error
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGORM_ListAll measures selecting and scanning 100 rows.
func BenchmarkGORM_ListAll(b *testing.B) {
	db := newGormDB(b)
	createGormTable(b, db)
	seedGormUsers(b, db, 100)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		var users []GormUser
		err := db.Order("name ASC").Find(&users).Error
		if err != nil {
			b.Fatal(err)
		}
		if len(users) != 100 {
			b.Fatalf("expected 100 users, got %d", len(users))
		}
	}
}

// BenchmarkGORM_ListWithLimit measures paginated selection (10 rows offset 50).
func BenchmarkGORM_ListWithLimit(b *testing.B) {
	db := newGormDB(b)
	createGormTable(b, db)
	seedGormUsers(b, db, 100)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		var users []GormUser
		err := db.Order("name ASC").Limit(10).Offset(50).Find(&users).Error
		if err != nil {
			b.Fatal(err)
		}
		if len(users) != 10 {
			b.Fatalf("expected 10 users, got %d", len(users))
		}
	}
}

// BenchmarkGORM_Update measures updating a single row by primary key.
func BenchmarkGORM_Update(b *testing.B) {
	db := newGormDB(b)
	createGormTable(b, db)
	seedGormUsers(b, db, 100)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		id := int64(i%100 + 1)
		err := db.Model(&GormUser{ID: id}).Update("email", fmt.Sprintf("updated-%d@example.com", i)).Error
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGORM_Delete measures deleting a single row.
// Uses the StopTimer/StartTimer pattern to create a fresh row before each deletion.
func BenchmarkGORM_Delete(b *testing.B) {
	db := newGormDB(b)
	createGormTable(b, db)

	b.ReportAllocs()
	for i := range b.N {
		b.StopTimer()
		u := &GormUser{
			Name:  fmt.Sprintf("del-%06d", i),
			Email: fmt.Sprintf("del-%06d@example.com", i),
		}
		err := db.Create(u).Error
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()

		err = db.Delete(&GormUser{}, u.ID).Error
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ─────────────────── helpers ───────────────────

func newGormDB(tb testing.TB) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	if err != nil {
		tb.Fatal(err)
	}
	return db
}

func createGormTable(tb testing.TB, db *gorm.DB) {
	if err := db.AutoMigrate(&GormUser{}); err != nil {
		tb.Fatal(err)
	}
}

func seedGormUsers(tb testing.TB, db *gorm.DB, count int) {
	for i := range count {
		err := db.Create(&GormUser{
			Name:  fmt.Sprintf("user-%03d", i),
			Email: fmt.Sprintf("user-%03d@example.com", i),
		}).Error
		if err != nil {
			tb.Fatal(err)
		}
	}
}
