<!--
This file follows the principles of Keep a Changelog (https://keepachangelog.com/en/1.0.0/).
It's intended to be a human-readable history of changes.
-->

# Changelog

## [v0.2.1] - 2025-06-23

### Fixed
- Fixed a bug in the `Set` function where a transaction was not properly closed, potentially causing database locks.
- Corrected the handling of boolean (`bool`) fields in `ArgsAppay`, which were previously misinterpreted as integers.

## [v0.2.0] - 2025-06-21

### Changed
- `INSERT` and `UPDATE` operations now automatically skip fields tagged with `autoincrement` in their `db_key`. This prevents errors when inserting records into tables with auto-generating primary keys.
- Refactored internal argument generation. The `query.Args` function now intelligently handles arguments for both read (`SELECT`) and write (`INSERT`/`UPDATE`) operations.

### Added
- New exported error `sqlh.ErrWhereClauseRequiredForUpdate` for better error handling in `Update` statements.
- Re-exported `sqlh.ErrTypeIsNotStruct` from the `query` package for easier access.
- New internal constants `forWrite` and `forRead` to improve readability when calling `query.Args`.

### Fixed
- Fixed a critical bug in `Delete` where operations were not correctly performed within a transaction due to using `db.Prepare` instead of `tx.Prepare`.
- The `Set` function is now atomic. It uses a transaction to prevent race conditions between checking for a record's existence and performing an `INSERT` or `UPDATE`.

## [v0.1.1] - 2025-06-21

### Added
- New generic function `sqlh.Update[T any]` to update records in the database based on specified conditions.

## [v0.1.0] - 2025-06-05

### Changed

- **Breaking Change:** The `Get` function now returns a pointer (`*T`) to the found struct instead of the struct value (`T`).
  - *Reason:* This provides a clearer way to indicate when a record was not found (by returning `nil`) compared to returning a zero-value struct.
  - *Impact:* Code calling `sqlh.Get` must be updated to expect a pointer (`*T`) and handle the `nil` case.

- **Breaking Change:** Error handling in the `Get` function has been standardized and improved.
  - `Get` now returns the standard `sql.ErrNoRows` error when no record is found. Previously, it returned `fmt.Errorf("not found")`.
  - Two new exported errors have been added: `sqlh.ErrWhereClauseRequired` (returned if `Get` is called without `Where` conditions) and `sqlh.ErrMultipleRowsFound` (returned if `Get` finds more than one record).
  - *Impact:* Code checking for specific errors from `sqlh.Get` (especially using `err.Error() == "not found"`) must be updated to use `errors.Is(err, sql.ErrNoRows)` and potentially check for the new exported errors.

### Added

- Exported errors `ErrWhereClauseRequired` and `ErrMultipleRowsFound` for specific error checking.
- Added a limit of 2 to the internal `ListRows` call within `Get` for minor performance optimization when checking for multiple rows.

### Fixed

- Corrected the return signature and logic of `Get` to consistently return `*T` or `nil` on error/not found.

[Unreleased]: https://github.com/kirill-scherba/sqlh/compare/v0.2.1...HEAD
[v0.2.1]: https://github.com/kirill-scherba/sqlh/compare/v0.2.0...v0.2.1
[v0.2.0]: https://github.com/kirill-scherba/sqlh/compare/v0.1.1...v0.2.0
[v0.1.1]: https://github.com/kirill-scherba/sqlh/compare/v0.1.0...v0.1.1
[v0.1.0]: https://github.com/kirill-scherba/sqlh/releases/tag/v0.1.0
