// Copyright 2024-2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package bench contains comparative benchmarks for sqlh against raw
// database/sql, sqlx, and GORM. It is a separate Go module so that the
// third-party test dependencies do not pollute the main module.
package bench

// RawSQLUser is the raw database/sql benchmark model.
// No struct tags are required because every column is scanned manually.
type RawSQLUser struct {
	ID    int64
	Name  string
	Email string
}

// SqlxUser is the sqlx benchmark model.
// db tags tell sqlx how to map struct fields to/from SQL columns.
type SqlxUser struct {
	ID    int64  `db:"id"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

// GormUser is the GORM benchmark model.
// gorm tags define the primary key, auto-increment, and uniqueness.
type GormUser struct {
	ID    int64  `gorm:"primaryKey;autoIncrement"`
	Name  string `gorm:"unique;not null"`
	Email string
}

// TableName overrides the default table name for GORM.
func (GormUser) TableName() string { return "gorm_users" }

// SqlhUser is the sqlh benchmark model.
// db tags define column names; db_key defines constraints.
type SqlhUser struct {
	ID    int64  `db:"id" db_key:"not null primary key autoincrement"`
	Name  string `db:"name" db_key:"unique"`
	Email string `db:"email"`
}
