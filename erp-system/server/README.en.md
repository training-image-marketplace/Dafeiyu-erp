# erp-system / server

Backend of Dafeiyu ERP: Go 1.25 + Gin + MySQL 8.0.16+, listening on `:8889`, database `pt_erp`.
It also serves the built frontend (`web/`), so production needs only this one process.

Architecture, business rules and the endpoint list live elsewhere:
[backend architecture](../../doc/01-后端架构.md), [database](../../doc/04-数据库设计.md),
[API](../../doc/05-API设计.md) (all in Chinese).

## Layout

```
cmd/server/            entry point, manual DI
internal/handler/      HTTP parsing and responses
internal/service/      business rules, transactions
internal/repository/   SQL, migrations
internal/domain/       entities and domain errors
internal/perm/         permission codes, page/button tree
internal/middleware/   auth, CSRF, rate limit, audit
internal/httpserver/   routes and static assets
config/                example config (config.yaml is gitignored)
```

## Quick start

```bash
# Create the database first (the app only runs migrations):
#   CREATE DATABASE pt_erp DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

cp config/config.example.yaml config/config.yaml   # point it at your MySQL
go run ./cmd/server                                # :8889
```

Default account `admin / admin123` — change it after the first login. Demo accounts are listed in
the [repository README](../../README.md).

## Environment variables

Precedence: environment > config file > built-in defaults.

| Variable | Default | Purpose |
| --- | --- | --- |
| `APP_ADDR` | `:8889` | listen address |
| `APP_ENV` | `prod` | `dev` enables gin debug |
| `APP_DB_HOST` / `PORT` / `USER` / `PASSWORD` / `NAME` | — | database (UTC) |
| `APP_WEB_DIR` | `web` | directory of the built frontend |
| `APP_UPLOAD_DIR` | `uploads` | upload directory |
| `APP_SECURE_COOKIE` | false | must be true behind HTTPS |
| `APP_TRUSTED_PROXIES` | empty | required behind nginx |
| `INTERNAL_TOKEN` | empty | token for `/internal/*`; empty allows 127.0.0.1 only |

See `config/config.example.yaml` for the full list.

## Tests

```bash
export TEST_MYSQL_DSN='root:***@tcp(127.0.0.1:23306)/'
go test ./... -count=1          # 207 cases; integration tests skip without the DSN
```

Each case creates its own random database and drops it afterwards.

## Conventions (read before changing code)

- Layering: handlers never touch `*sql.DB`, services never build SQL; wrap errors with
  `domain.ErrXxx` and let `httpx` map them to status codes + i18n keys.
- Every route declares a permission code; new codes must also join `internal/perm/buttons.go`
  and get `perm.*` entries in both dictionaries.
- A new page needs a `web/src/pages/**.svelte` file, a `sys_menus` seed row and `menu.*` keys.
- Migrations only move forward: `NNNN_description.sql`, bump `migrationCount` in
  `repository/migrate_test.go`.
- Bill and stock state changes run inside a transaction with `SELECT ... FOR UPDATE` and write
  an operation record.

## License

MIT — see [LICENSE](../../LICENSE).
