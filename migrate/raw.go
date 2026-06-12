// Copyright 2026 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package migrate

// Raw creates a migration step from explicit SQL.
//
// The SQL is executed exactly as provided — no dialect transformation or
// placeholder rebinding is performed. Callers are responsible for writing
// dialect-specific SQL or using query.Rebind() when PostgreSQL $N placeholders
// are needed.
func Raw(name string, v Version, sql string) Migration {
	return &rawMigration{name: name, version: v, sql: sql}
}

type rawMigration struct {
	name    string
	version Version
	sql     string
}

func (r *rawMigration) Name() string       { return r.name }
func (r *rawMigration) Version() Version   { return r.version }
func (r *rawMigration) SQL(_ Dialect) (string, error) { return r.sql, nil }
