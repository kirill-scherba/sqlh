module github.com/kirill-scherba/sqlh/bench

go 1.25.2

require (
	github.com/jmoiron/sqlx v1.4.0
	github.com/kirill-scherba/sqlh v0.0.0
	github.com/mattn/go-sqlite3 v1.14.44
	gorm.io/driver/sqlite v1.5.7
	gorm.io/gorm v1.25.12
)

require (
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace github.com/kirill-scherba/sqlh => ../
