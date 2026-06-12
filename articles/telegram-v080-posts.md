---
title: "sqlh v0.8.0 — Telegram Channel Posts"
description: "Ready-to-post messages for Telegram Go channels"
platform: telegram
---

# Telegram Posts — sqlh v0.8.0

## Channel 1: Go Golang (https://t.me/Golang)

Target: ~50K subscribers, mixed Russian + English audience.
Style: Promotional but technical.
Character limit: In practice, Telegram shows a preview of the first ~300 characters. Ensure the opening line (with 🚀) and at least one feature bullet appear in the preview. The full message can be longer — users will expand it.

```
🚀 sqlh — Zero-boilerplate SQL for Go

Define a struct with tags, get full CRUD:
• sqlh.Create[User](db) — CREATE TABLE from struct
• sqlh.Get[User](db, sqlh.Eq("name", "Alice")) — type-safe Get
• ListRange[User] — Go 1.25 iterator, no rows.Scan

✅ 57% less code vs raw database/sql
✅ 85K inserts/sec (SQLite), auto-transactions
✅ PostgreSQL, MySQL, SQLite — all CI-tested
✅ Native UPSERT, JOINs, pagination

GitHub: https://github.com/kirill-scherba/sqlh
pkg.go.dev: https://pkg.go.dev/github.com/kirill-scherba/sqlh
```

---

## Channel 2: Go News (https://t.me/golang_news)

Target: ~15K subscribers, news-oriented.
Style: Release announcement format.

```
sqlh v0.8.0 released — Go SQL helper with zero-boilerplate CRUD

New in this release:
• Type-safe WHERE helpers: Eq, Ne, Gt, Like, In, IsNull
• Benchmarks vs raw sql, sqlx, GORM
• Animated demo of full CRUD workflow

Uses Go 1.25 generics + struct tags. No rows.Scan, no manual SQL, auto-transactions on every write.

https://github.com/kirill-scherba/sqlh
```

---

## Channel 3: Go Pro (https://t.me/golang_pro)

Target: Professional Go developers.
Style: Architecture-focused, higher bar.
Note: Verify channel is active before posting.

```
sqlh — type-safe SQL helper for Go (generics-first, zero boilerplate)

Architecture highlights for experienced Gophers:
• struct tags → DDL + DML auto-generated
• sync.Map metadata cache by reflect.Type (one-time reflection cost)
• Auto-transactions: every write = BEGIN → EXEC → COMMIT/ROLLBACK
• Database lock retry: 20 attempts × 100ms for SQLite
• Zero-alloc read path via addressable struct pointers (4 allocs/op)
• Go 1.25 iter.Seq2 for lazy ListRange streaming

Benchmarked: 85K inserts/sec, 2.4x faster than GORM.
4 databases supported: SQLite, MySQL, PostgreSQL, SQL Server (exp.)

https://github.com/kirill-scherba/sqlh
```

---

## Posting Instructions

1. Verify channel activity — check last post date.
2. Space messages by 5+ minutes between channels.
3. Do NOT spam or bump messages.
4. Reply to comments/questions if the channel has a discussion group.
5. If channel is admin-only, DM the admin with the message and a polite request to post.
