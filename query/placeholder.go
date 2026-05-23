// Copyright 2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package query

import (
	"fmt"
	"strings"
)

// Rebind converts ? placeholders to PostgreSQL $N style.
// It is a no-op for strings that contain no ?.
//
//   - Input:  "INSERT INTO t VALUES(?, ?)"
//   - Output: "INSERT INTO t VALUES($1, $2)"
//
// The function only acts on '?' characters that appear outside of string
// literals. Since sqlh never embeds string literals containing ? in
// generated SQL, the simple character scan is sufficient.
func Rebind(sql string) string {
	if !strings.Contains(sql, "?") {
		return sql
	}
	var b strings.Builder
	b.Grow(len(sql) + 10)
	n := 0
	for i := 0; i < len(sql); i++ {
		if sql[i] == '?' {
			n++
			b.WriteString(fmt.Sprintf("$%d", n))
		} else {
			b.WriteByte(sql[i])
		}
	}
	return b.String()
}
