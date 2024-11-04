# Sqlh golang package to easy sql requests using standard golang librarys

Sqlh is a SQL Helper package contains helper functions to execute SQL
requests. It provides such functions as Execute, Select, Insert, Update and
Delete.

Sqlh uses query subpackage. Package query provides helper functions to
generate SQL statement queries. It includes types and functions to construct
SQL SELECT statements with support for pagination, where clauses, and ordering.
The package also includes functionality to generate SQL CREATE TABLE statements
based on Go struct types, mapping struct fields to database fields using struct
tags for customization.

[![GoDoc](https://godoc.org/github.com/kirill-scherba/sqlh?status.svg)](https://godoc.org/github.com/kirill-scherba/sqlh/)
[![Go Report Card](https://goreportcard.com/badge/github.com/kirill-scherba/sqlh)](https://goreportcard.com/report/github.com/kirill-scherba/sqlh)

## Basic usage example

...

## Licence

[BSD](LICENSE)
