package sqlh

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type User struct {
	Id   int    `db_key:"primary key autoincrement"`
	Name string `db_type:"text"`
}

func TestTableMethods(t *testing.T) {

	// Open in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create table User if not exists
	users, err := CreateTable[User](db)
	require.NoError(t, err)

	// Insert a new user
	err = users.Insert(User{Name: "Bob"})
	require.NoError(t, err)

	// Get user with id=1
	u, err := users.Get(Where{"id=", 1})
	require.NoError(t, err)
	fmt.Println("id:", u.Id)

	// Set (update) user name
	err = users.Set(User{Name: "Alice"}, Where{"id=", 1})
	require.NoError(t, err)

	// Get user with id=1
	u, err = users.Get(Where{"id=", 1})
	require.NoError(t, err)
	fmt.Println("name:", u.Name)

	// Insert a new user
	err = users.Insert(User{Name: "Bob"})
	require.NoError(t, err)

	// Get list of users
	listErrFunc := func(err error) {require.NoError(t, err)	}
	for _, u := range users.List(0, "", "", 0, listErrFunc) {
		fmt.Println(u.Name)
	}
}
