// Вот корректный пример использования `sqlh.List` с JOIN, основанный на анализе исходного кода:

package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/kirill-scherba/sqlh"
	"github.com/kirill-scherba/sqlh/query"
	_ "modernc.org/sqlite" // Import the sqlite3 driver
)

// User — основная таблица
type User struct {
	ID   int64  `db:"id" db_key:"not null primary key autoincrement"`
	Name string `db:"name" db_key:"unique"`
}

// Profile — присоединяемая таблица
type Profile struct {
	ID     int64  `db:"id" db_key:"not null primary key autoincrement"`
	UserID int64  `db:"user_id"`
	Bio    string `db:"bio"`
}

// UserProfile — составная структура для JOIN.
// Важно: первое поле — основная структура (users),
// второе — присоединяемая (profiles).
// Имена полей не важны, используется порядок.
type UserProfile struct {
	User    User
	Profile Profile
}

func main() {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Создаём таблицы
	if err := sqlh.Create[User](db); err != nil {
		log.Fatal(err)
	}
	if err := sqlh.Create[Profile](db); err != nil {
		log.Fatal(err)
	}

	// Вставляем данные
	sqlh.Insert(db, User{Name: "Alice"})
	sqlh.Insert(db, User{Name: "Bob"})
	sqlh.Insert(db, Profile{UserID: 1, Bio: "Alice's bio"})
	sqlh.Insert(db, Profile{UserID: 2, Bio: "Bob's bio"})

	// List с LEFT JOIN
	users, _, err := sqlh.ListRows[UserProfile](db, 0, "", "", 10,
		// Set main table alias to use in Joins
		sqlh.SetAlias("t"),
		// Join profile table
		query.MakeJoin[Profile](query.Join{On: "t.id = o.user_id", Alias: "o"}),
	)
	if err != nil {
		log.Fatal(err)
	}

	for _, up := range users {
		fmt.Printf("User: %s, Bio: %s\n", up.User.Name, up.Profile.Bio)
	}
}

/**

**Ключевые моменты:**

1. **Составная структура `UserProfile`** — первое поле `User` (основная таблица), второе `Profile` (JOIN-таблица). Функция `fields()` рекурсивно обходит вложенные структуры: если `i == 0` и поле — структура, она "ныряет" внутрь неё.

2. **`query.Join`** передаётся как variadic attribute:
   - `Join: "LEFT"` → генерируется `"LEFT join"` (Join поле конкатенируется с `" join"`)
   - `Name` — имя таблицы через `query.Name[Profile]()` (возвращает `"profile"`)
   - `Alias` — алиас для условий и префиксов полей
   - `On` — условие соединения
   - `Fields` — поля из JOIN-таблицы; `query.MakeJoin[Profile](query.Join{Alias: "p"})` автоматически заполняет поля с префиксом алиаса

3. **Сгенерированный SQL:**
   ```sql
   SELECT user.id, user.name, p.id, p.user_id, p.bio
   FROM user LEFT join profile p on user.id = p.user_id
   ```

4. **Важно:** Без `MakeJoin` можно указать поля вручную:
   ```go
   Fields: []string{"p.id", "p.user_id", "p.bio"},
*/
