// Copyright 2024 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package query

import (
	"testing"
	"time"
)

func TestSQLQuery(t *testing.T) {

	t.Run("TestQueryArgs", func(t *testing.T) {

		type SomeStruct struct {
			Name string    `db:"name"`
			Cost float64   `db:"cost"`
			Age  int32     `db:"age"`
			Time time.Time `db:"time"`
		}

		var someStruct = SomeStruct{
			Name: "John",
			Cost: 100.0,
			Age:  20,
			Time: time.Now(),
		}

		// Create args
		args, err := Args(someStruct, false)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("someStruct: %+v", someStruct)

		// Update args
		*args[0].(*any) = "Jane"
		*args[1].(*any) = float32(200.0)
		*args[2].(*any) = int8(30)
		*args[3].(*any) = time.Now()

		// Applay args
		err = ArgsAppay(&someStruct, args)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("someStruct: %+v", someStruct)
	})

}
