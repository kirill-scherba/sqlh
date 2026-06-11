module github.com/kirill-scherba/sqlh/examples/comparison

go 1.25.2

replace github.com/kirill-scherba/sqlh => ../../

require (
	github.com/jmoiron/sqlx v1.4.0
	github.com/kirill-scherba/sqlh v0.5.1
	github.com/mattn/go-sqlite3 v1.14.44
)
