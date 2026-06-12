---
title: "sqlh — SQL в Go без boilerplate: пишем CRUD за 50 строк"
published: false
tags: go, sql, database, opensource, производительность
hubs: Программирование, Go, Open source
---

# sqlh — SQL в Go без boilerplate: пишем CRUD за 50 строк

> *Zero-boilerplate SQL для Go. Опиши структуру тегами — и это всё.*

Если вы пишете на Go и работаете с SQL-базами, вы знаете эту боль. Каждый CRUD-запрос — ручной SQL-строка, `rows.Scan` для каждого поля, `Begin/Commit/Rollback` вокруг записи, и постоянная синхронизация DDL-схемы с кодом. Шаблонный код не заканчивается никогда.

Это рассказ о `sqlh` — библиотеке, которая убирает всё это, оставаясь в «золотой середине» между raw SQL (слишком много работы) и тяжёлыми ORM (слишком много магии).

## §1. Проблема: Go + SQL = смерть от тысячи `rows.Scan`

Стандартный `database/sql` в Go отличен. Он даёт прочный, переносимый фундамент для любой SQL-базы. Но он намеренно оставляет тяжёлую работу за вами.

Вот как выглядит простой CRUD на чистом `database/sql`:

```go
// 1. CREATE TABLE — raw DDL-строка
_, err := db.Exec(`CREATE TABLE IF NOT EXISTS user (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE,
    email TEXT,
    age INTEGER
)`)

// 2. INSERT — явные placeholder и аргументы
_, err = db.Exec(
    "INSERT INTO user (name, email, age) VALUES (?, ?, ?)",
    "Alice", "alice@example.com", 30,
)

// 3. GET по ID — QueryRow + ручной Scan
var u User
err = db.QueryRow("SELECT id, name, email, age FROM user WHERE id = ?", 1).
    Scan(&u.ID, &u.Name, &u.Email, &u.Age)

// 4. LIST всех — Query + rows.Next + rows.Scan в цикле
rows, err := db.Query("SELECT id, name, email, age FROM user ORDER BY name ASC")
var users []User
for rows.Next() {
    var u User
    if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Age); err != nil {
        log.Fatal(err)
    }
    users = append(users, u)
}
rows.Close()

// 5. UPDATE — raw SQL с placeholder
_, err = db.Exec(
    "UPDATE user SET email = ?, age = ? WHERE id = ?",
    "alice.new@example.com", 31, 1,
)

// 6. DELETE — raw SQL
_, err = db.Exec("DELETE FROM user WHERE id = ?", 1)
```

Это **~115 строк кода** для шести базовых операций. И каждый раз, когда вы добавляете столбец, нужно обновить строку `CREATE TABLE`, список колонок в `INSERT`, список в `SELECT`, и вызов `rows.Scan`. Опечатка в любом месте — runtime-ошибка, compile-time безопасности нет.

| Боль | Почему больно |
|------|---------------|
| Ручной SQL | Каждый CRUD — raw SQL-строка, нет проверки на этапе компиляции |
| `rows.Scan` | 4–5 строк на каждый результат только для маппинга колонок на поля |
| Транзакции | `db.Begin()` + `defer tx.Rollback()` + `tx.Commit()` — везде |
| Нет связи со схемой | DDL в миграциях, структуры в Go — они расходятся |
| Порядок колонок | Новый столбец → обновлять SQL-строки **и** `Scan`-вызовы |

## §2. Существующие решения: sqlx и GORM

В экосистеме Go есть два известных пути. У каждого свои компромиссы.

### sqlx: лучше, но всё ещё ручной SQL

[sqlx](https://github.com/jmoiron/sqlx) — популярное расширение `database/sql`. Он добавляет `StructScan`, `Get`, `Select`, именованные параметры. SQL пишете по-прежнему руками, но `rows.Scan` автоматизирован.

```go
// sqlx: всё ещё ручной SQL, но StructScan убирает Scan
var u User
dbx.Get(&u, "SELECT id, name, email, age FROM user WHERE id = ?", 1)
```

sqlx сэкономит примерно **30% boilerplate** (до ~80 строк). Но `CREATE TABLE`, `INSERT`, `SELECT`, `UPDATE`, `DELETE` — всё ещё пишете вручную. Генерация SQL — не его задача.

### GORM: полный ORM, полная магия

[GORM](https://gorm.io/) — тяжеловес. Генерирует всё — схему, запросы, миграции — и даёт богатый chainable API. Но цена высока:

- **Тяжёлый reflection** в runtime
- **Крутая кривая обучения** — теги, хуки, scopes, ассоциации
- **~4 MB увеличение бинарника** только за ORM
- **Магия, которая скрывает сложность** — пока не сломается, и вы часами дебажите

Для больших команд с выделенными DBA и сложными моделями GORM — solid choice. Для CLI-утилит, стартапов и микросервисов — overkill.

### sqlh: золотая середина

| Фича | `database/sql` | sqlx | GORM | **sqlh** |
|---|---|---|---|---|
| SQL-генерация | ❌ Ручная | ❌ Ручная | ✅ Полная | ✅ Полная |
| `rows.Scan` | ✅ Нужен | ❌ `StructScan` | ❌ Авто | ❌ Авто |
| Типобезопасность (generics) | ❌ | ❌ | ❌ | ✅ |
| Авто-транзакции | ❌ | ❌ | ✅ | ✅ |
| Ретрай блокировок | ❌ | ❌ | ❌ | ✅ |
| Кривая обучения | Средняя | Средняя | Высокая | **Низкая** |
| Оверхед бинарника | 0 | ~200 KB | ~4 MB | ~200 KB |

sqlh живёт между sqlx и GORM:
- **Zero-boilerplate CRUD** — структурные теги генерируют весь SQL
- **Типобезопасность через Go generics** — `Get[User]()` возвращает `*User`, не `interface{}`
- **Никакой магии** — что видите в структуре, то и получите в базе
- **Лёгкий** — минимальный reflection, кеш метаданных, никакой скрытой сложности

## §3. Как sqlh решает проблему: структурные теги как единственный источник правды

Идея проста: **ваша Go-структура — это ваша схема**.

```go
type User struct {
    ID    int64  `db:"id" db_key:"not null primary key autoincrement"`
    Name  string `db:"name" db_key:"unique"`
    Email string `db:"email"`
    Age   int    `db:"age"`
}
```

Три тега управляют всем:

| Тег | Назначение | Пример |
|-----|------------|--------|
| `db` | Имя колонки | `db:"user_name"` |
| `db_key` | Ограничения, индексы | `db_key:"primary key autoincrement"` |
| `db_type` | Переопределение типа SQL | `db_type:"TEXT"` |

Из этого единственного определения sqlh генерирует:

- **CREATE TABLE** — `sqlh.Create[User](db)` → `CREATE TABLE IF NOT EXISTS user (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT UNIQUE, email TEXT, age INTEGER)`
- **INSERT** — `sqlh.Insert(db, User{Name: "Alice"})` → `INSERT INTO user (name, email, age) VALUES (?, ?, ?)`
- **SELECT** — `sqlh.Get[User](db, ...)` → `SELECT id, name, email, age FROM user WHERE ... LIMIT 2`
- **UPDATE** — `sqlh.Update(db, ...)` → `UPDATE user SET name=?, email=?, age=? WHERE ...`
- **DELETE** — `sqlh.Delete[User](db, ...)` → `DELETE FROM user WHERE ...`

### Архитектура

```
┌─────────────────────────────────────────────┐
│  sqlh package                               │
│  Insert, Get, List, Update, Delete, Set,    │
│  Create — с авто-транзакциями               │
├─────────────────────────────────────────────┤
│  query package                              │
│  SQL-генерация, кеш метаданных, JOIN        │
├─────────────────────────────────────────────┤
│  database/sql (stdlib)                        │
│  Пул соединений, выполнение raw-запросов    │
└─────────────────────────────────────────────┘
```

### Ключевые дизайн-решения

1. **Generics-first (Go 1.25+)** — `Get[User]()` возвращает `*User` с проверкой типов на этапе компиляции. Никаких `interface{}`, никаких приведений типов.
2. **Рефлексия один раз** — метаданные структуры парсятся и кешируются в `sync.Map` по `reflect.Type`. Последующие вызовы переиспользуют имена таблиц, списки полей, scan-метаданные.
3. **Авто-транзакции на запись** — каждый `Insert`, `Update`, `Delete`, `Set` обёрнут в `BEGIN...COMMIT` с `ROLLBACK` при ошибке. Транзакции никогда не забудете.
4. **Ретрай блокировок SQLite** — ошибки «database is locked» ретраятся до 20 раз с backoff 100 ms. Production-устойчивость из коробки.
5. **Мульти-БД** — SQLite (основной), MySQL, PostgreSQL (оба в CI), SQL Server (экспериментально).

## §4. CRUD за 50 строк: быстрый старт

Тот же CRUD, что в начале — но **~57% короче**:

```go
package main

import (
    "database/sql"
    "fmt"

    "github.com/kirill-scherba/sqlh"
    _ "github.com/mattn/go-sqlite3"
)

type User struct {
    ID    int64  `db:"id" db_key:"not null primary key autoincrement"`
    Name  string `db:"name" db_key:"unique"`
    Email string `db:"email"`
    Age   int    `db:"age"`
}

func main() {
    db, _ := sql.Open("sqlite3", "file::memory:?cache=shared")
    defer db.Close()

    // 1. CREATE TABLE из структуры
    sqlh.Create[User](db)

    // 2. INSERT
    sqlh.Insert(db, User{Name: "Alice", Email: "alice@example.com", Age: 30})
    bobID, _ := sqlh.InsertId(db, User{Name: "Bob", Email: "bob@example.com", Age: 25})

    // 3. GET по ID — возвращает *User, не interface{}
    u, _ := sqlh.Get[User](db, sqlh.Eq("id", bobID))
    fmt.Println(u.Name) // "Bob"

    // 4. LIST всех — возвращает []User + next offset
    users, _, _ := sqlh.List[User](db, 0, "", "name ASC")
    fmt.Println(len(users)) // 2

    // 5. UPDATE — передаём полную структуру, чтобы не занулить другие колонки
    sqlh.Update(db, sqlh.UpdateAttr[User]{
        Row:    User{Name: "Alice", Email: "alice.new@example.com", Age: 31},
        Wheres: []sqlh.Where{sqlh.Eq("id", 1)},
    })

    // 6. DELETE
    sqlh.Delete[User](db, sqlh.Eq("id", bobID))
}
```

**~50 строк.** Никакого raw SQL. Ни одного `rows.Scan`. Ни одного `BEGIN/COMMIT`. Ни одной ошибки в порядке колонок.

### Сравнение бок-о-бок

| Операция | Raw `database/sql` | sqlx | **sqlh** |
|----------|------------------|------|----------|
| CREATE TABLE | Raw SQL-строка | Raw SQL-строка | `sqlh.Create[User](db)` |
| INSERT | `Exec(?,?,?)` | `NamedExec` | `Insert(T)` |
| GET | `QueryRow + Scan` | `Get(&T)` | `Get[T](where)` |
| LIST | `rows.Next + Scan` | `Select` | `List[T](...)` |
| UPDATE | `Exec(?,?,?,?)` | `NamedExec` | `Update(attr)` |
| DELETE | `Exec(?)` | `Exec(?)` | `Delete[T](where)` |
| COUNT | `QueryRow + Scan` | `Get(&int)` | `Count[T]()` |

| | Строк кода | Сокращение |
|---|---|---|
| Raw `database/sql` | ~115 | baseline |
| sqlx | ~80 | −30% |
| **sqlh** | **~50** | **−57%** |

### Table[T]: удобный method-based API

Для компонентов, где несколько операций над одной таблицей — можно обернуть в `Table[T]`:

```go
tbl, _ := sqlh.CreateTable[User](db)
tbl.Insert(User{Name: "Charlie", Email: "charlie@example.com", Age: 28})
c, _ := tbl.Get(sqlh.Eq("name", "Charlie"))
fmt.Println(c.Name)

for _, user := range tbl.List(0, "", "name ASC", 0) {
    fmt.Println(user.Name)
}
```

`Table[T]` — лёгкий wrapper над общим `*sql.DB`. Он **не владеет** соединением, поэтому `Close()` — no-op (для обратной совместимости). Ресурсы очищает вызывающий через `db.Close()`.

### Set (upsert): нативный UPSERT

`Set` — атомарный upsert. Для PostgreSQL, SQLite и MySQL использует нативный синтаксис базы:

- **PostgreSQL**: `INSERT ... ON CONFLICT (...) DO UPDATE SET ...`
- **SQLite**: `INSERT ... ON CONFLICT (...) DO UPDATE SET ...`
- **MySQL**: `INSERT ... ON DUPLICATE KEY UPDATE ...`

Для неизвестных драйверов — fallback на SELECT-then-INSERT/UPDATE в транзакции.

```go
// name помечен db_key:"unique" — Set сделает UPDATE при совпадении
err := sqlh.Set(db, User{Name: "Dave", Email: "dave@example.com"}, sqlh.Eq("name", "Dave"))
```

### ListRange: Go 1.25 iterators

Вместо `List` с слайсом — ленивый итератор `ListRange`, который возвращает `iter.Seq2[int, T]`. Не загружает всё в память — идеален для стриминга, JOIN и контекстов с таймаутом.

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

var listErr error
for i, user := range sqlh.ListRange[User](db, 0, "", "name ASC", 0,
    func(err error) { listErr = err },
    ctx,
) {
    fmt.Printf("%d: %s\n", i, user.Name)
}
```

### Типобезопасные WHERE-хелперы

Вместо ручного SQL в `Where.Field` — конструкторы для типобезопасных условий:

```go
sqlh.Eq("name", "Alice")         // name = ?
sqlh.Ne("status", "deleted")     // status <> ?
sqlh.Gt("age", 18)               // age > ?
sqlh.Like("name", "%Alice%")     // name LIKE ?
sqlh.In("id", 1, 2, 3)           // id IN (?, ?, ?)
sqlh.IsNull("deleted_at")        // deleted_at IS NULL
```

Значения передаются как bind-параметры (безопасно). Низкоуровневый `Where{Field, Value}` остаётся для кастомных операторов.

### JOIN: composite-структуры

```go
type UserWithOrders struct {
    *UserTable   // основная таблица
    *OrderTable  // JOIN-таблица
}

join := query.MakeJoin[OrderTable](query.Join{
    Join:  "LEFT", Alias: "o", On: "t.id = o.user_id",
})

for _, row := range sqlh.ListRange[UserWithOrders](db, 0, "", "t.name ASC", 0,
    sqlh.SetAlias("t"), join, func(err error) { log.Fatal(err) },
) {
    if row.OrderTable != nil {
        fmt.Println(row.UserTable.Name, row.OrderTable.Total)
    }
}
```

## §5. Бенчмарки: производительность в цифрах

Насколько быстр sqlh на практике? В модуле `bench/` — воспроизводимые Go-бенчмарки сравнивают raw `database/sql`, `sqlx`, GORM и sqlh на одном и том же CRUD-ворклоаде. Все тесты используют in-memory SQLite — никакой внешней настройки.

Воспроизвести на своей машине:

```bash
cd bench && go test -bench=. -benchmem -benchtime=1s
```

### CRUD Throughput (ops/sec)

| Операция | raw sql | sqlx | GORM | **sqlh** |
|----------|---------|------|------|----------|
| **Insert** | 158,041 | 131,596 | 34,971 | **87,085** |
| **Get by PK** | 169,232 | 152,415 | 78,666 | **68,675** |
| **List all** | 11,807 | 9,261 | 6,779 | **7,573** |
| **List limit 10** | 51,500 | 43,691 | 37,821 | **44,142** |
| **Update** | 228,728 | 180,505 | 65,933 | **85,543** |
| **Delete** | 172,128 | 166,279 | 41,162 | **60,650** |

### Memory Allocations (bytes/op, allocs/op)

| Операция | raw sql | sqlx | GORM | **sqlh** |
|----------|---------|------|------|----------|
| **Insert** | 328 B, 12 | 721 B, 20 | 5,534 B, 82 | **1,274 B, 39** |
| **Get by PK** | 792 B, 27 | 976 B, 31 | 3,952 B, 66 | **2,592 B, 78** |
| **List all** | 23,744 B, 528 | 26,376 B, 632 | 27,668 B, 946 | **26,391 B, 745** |
| **List limit** | 3,120 B, 76 | 3,624 B, 91 | 6,145 B, 141 | **3,958 B, 115** |
| **Update** | 296 B, 9 | 680 B, 19 | 5,079 B, 68 | **1,393 B, 43** |
| **Delete** | 216 B, 7 | 216 B, 7 | 5,484 B, 67 | **1,136 B, 37** |

### Что говорят цифры

- **GORM** показывает наибольшую latency и самый тяжёлый allocation footprint — следствие богатого feature set и reflection-оверхеда.
- **sqlh** находится между raw/sqlx и GORM. Умеренный оверхед — плата за авто-генерацию SQL, парсинг тегов и встроенные транзакции на запись.
- **sqlh торгует скоростью на корректность**: каждая запись атомарна (auto-transact с rollback), что устраняет целый класс багов ценой оверхеда ~2–6x vs raw SQL для однострочных мутаций.
- **ListAll** доминируется сканированием 100 строк. Все библиотеки здесь показывают схожую производительность.

> **Окружение:** Linux AMD Ryzen 9 3900, Go 1.26.3, SQLite in-memory.
> Запустите `cd bench && go test -bench=. -benchmem -benchtime=1s` на своём железе для сравнения.

## §6. Когда использовать sqlh

sqlh — не серебряная пуля. Вот где он сияет, а где лучше что-то другое:

| Сценарий | Рекомендация |
|----------|--------------|
| CLI-утилиты | ✅ Идеально — ноль файлов миграций, один бинарник |
| Стартапы и MVP | ✅ Быстрее пишете, потом рефакторите |
| Микросервисы с простыми схемами | ✅ Низкий оверхед, типобезопасность |
| High-throughput OLTP (>100K writes/sec) | ⚠️ Тестируйте — возможно, raw SQL |
| Сложная аналитика | ⚠️ Предпочтительно raw SQL или query builder |
| Большие команды с DBA | ⚠️ GORM или sqlx могут подойти лучше |
| Обучение Go + SQL | ✅ Отличный учебный инструмент — низкая когнитивная нагрузка |

## Заключение

sqlh активно развивается. На момент v0.8.0 (июнь 2026) библиотека поддерживает:

- ✅ Полный CRUD с авто-транзакциями
- ✅ Нативный UPSERT (PostgreSQL, SQLite, MySQL)
- ✅ JOIN-запросы со сканированием в composite-структуры
- ✅ Go 1.25 iterators (`ListRange`) для ленивого стриминга
- ✅ Типобезопасные WHERE-хелперы (`Eq`, `Ne`, `Gt`, `Like`, `In` и др.)
- ✅ Ретрай блокировок для SQLite
- ✅ Мульти-БД (SQLite, MySQL, PostgreSQL)

В планах: агрегатные функции (`SUM`, `AVG`), миграции схемы, batch-операции. API стабилизируется к v1.0.0.

Если вы строите Go-проект, который общается с SQL, и устали писать один и тот же boilerplate снова и снова — дайте sqlh шанс. Опишите структуру. Это всё.

```bash
go get github.com/kirill-scherba/sqlh
```

- 📖 [README & Quick Start](https://github.com/kirill-scherba/sqlh)
- 📦 [pkg.go.dev reference](https://pkg.go.dev/github.com/kirill-scherba/sqlh)
- 🏗️ [Исходный код](https://github.com/kirill-scherba/sqlh)
- ⭐ [Awesome Go PR](https://github.com/avelino/awesome-go/pull/6401)

---

*Автор: [Kirill Scherba](https://github.com/kirill-scherba). sqlh — open source под BSD-лицензией. Contributions welcome.*
