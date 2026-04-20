# PostgreSQL Migration — Implementation Plan

Migrating WhoKnows from SQLite to PostgreSQL using `pgx` (driver) and `goose` (migrations). Target deployment: single DigitalOcean droplet (1GB RAM) running Postgres alongside the blue/green app containers.

---

## Phase 0 — Back up existing SQLite data

Before any changes, preserve the current database so we don't lose existing users/pages.

```bash
cp whoknows.db whoknows.db.backup
sqlite3 whoknows.db .dump > whoknows_dump.sql
```

The `.db` file is already gitignored. Keep both files in a safe location (local and a copy off-machine). The `.sql` dump is the source of truth for Phase 4.

---

## Phase 1 — Add Postgres infrastructure

### Production compose (`server_go/deploy/docker-compose.server.yml`)

Add a `postgres` service:

- Image: `postgres:17-alpine`
- Volume: `/opt/whoknows/pgdata:/var/lib/postgresql/data`
- Env: `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` loaded from `.env`
- Healthcheck via `pg_isready`

Both `whoknows-blue` and `whoknows-green` get:

- `depends_on: postgres: { condition: service_healthy }`
- `DATABASE_URL` env var (e.g., `postgres://whoknows:<pw>@postgres:5432/whoknows`)
- Remove the `/opt/whoknows/data` volume (SQLite file no longer needed)

### Dev compose (`server_go/docker-compose.dev.yml`) — new file

Just Postgres, bound to `localhost:5432`, so `go run ./cmd/server` can hit it locally without containerizing the Go app during development.

### Tuning for 1GB RAM

Add a small `postgresql.conf` override or command flags for the postgres service:

- `shared_buffers=128MB`
- `work_mem=4MB`
- `effective_cache_size=256MB`
- `max_connections=20`

---

## Phase 2 — Swap Go dependencies

### `go.mod`

Remove:

- `modernc.org/sqlite`

Add:

- `github.com/jackc/pgx/v5`
- `github.com/jackc/pgx/v5/pgxpool`
- `github.com/pressly/goose/v3`

### `internal/db/db.go`

- Replace `sql.Open("sqlite", ...)` with `pgxpool.New(ctx, os.Getenv("DATABASE_URL"))`
- Change the connection type from `*sql.DB` to `*pgxpool.Pool` throughout the package
- Remove the `ApplyMigrations` helper (goose replaces it)

### `internal/db/users.go`, `internal/db/search.go`

SQL dialect conversion:

- `?` placeholders → `$1`, `$2`, …
- `sql.ErrNoRows` → `errors.Is(err, pgx.ErrNoRows)`
- `LIKE` (case-insensitive in SQLite) → `ILIKE` for the search query
- `INTEGER PRIMARY KEY AUTOINCREMENT` → `BIGSERIAL` (handled in migration)
- Review each query method signature — pgx uses `pool.QueryRow(ctx, sql, args...)` with explicit context

### `cmd/server/main.go`

- Read `DATABASE_URL` instead of `WHOKNOWS_DB_PATH`
- On startup, open a `*sql.DB` handle via `pgx/stdlib` for goose, run `goose.Up(db, "migrations")`, then close it
- Open the `pgxpool.Pool` for the app's actual query workload

### `cmd/dbtest/`

Update or retire — depends on whether it's still useful with Postgres.

---

## Phase 3 — Migrations in goose format

### Replace `server_go/migrations/001_init.sql`

```sql
-- +goose Up
CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  email TEXT NOT NULL UNIQUE,
  password TEXT NOT NULL
);

CREATE TABLE pages (
  title TEXT PRIMARY KEY,
  url TEXT NOT NULL UNIQUE,
  language TEXT NOT NULL CHECK (language IN ('en','da')) DEFAULT 'en',
  last_updated TIMESTAMPTZ,
  content TEXT NOT NULL
);

-- +goose Down
DROP TABLE pages;
DROP TABLE users;
```

Future migrations follow the naming convention `NNN_description.sql`, created via `goose create <name> sql`.

---

## Phase 4 — Data import

Create `cmd/import-sqlite/main.go` — a one-off program that:

1. Opens `whoknows.db.backup` with the old sqlite driver in read-only mode
2. Reads every row from `users` and `pages`
3. Inserts them into Postgres via pgx, preserving IDs

Run it once locally after Phase 3 migrations have been applied to the target Postgres instance. After production cutover, the command can be removed or moved to `scripts/` for reference.

---

## Phase 5 — Update tests

The existing tests in `internal/db/*_test.go` use in-memory SQLite. Replace with a shared Postgres test helper:

- Add `internal/db/testhelper_test.go` with a `newTestPool(t)` helper that:

  - Connects to a test Postgres (from `TEST_DATABASE_URL` env var)
  - Creates a unique schema per test run (or per test)
  - Applies all migrations
  - Returns a `*pgxpool.Pool` and a cleanup function

- In CI, a Postgres service container provides `TEST_DATABASE_URL`.
- For local runs, devs run `docker compose -f docker-compose.dev.yml up -d` first.

---

## Phase 6 — CI workflow

Update `.github/workflows/ci.yml`:

### `build-and-test` job

Add a Postgres service:

```yaml
services:
  postgres:
    image: postgres:17-alpine
    env:
      POSTGRES_PASSWORD: test
      POSTGRES_DB: whoknows_test
    ports: ["5432:5432"]
    options: >-
      --health-cmd pg_isready --health-interval 5s --health-retries 5
```

Set `TEST_DATABASE_URL=postgres://postgres:test@localhost:5432/whoknows_test?sslmode=disable` for the test step.

### New `migration-test` job

Spins up a fresh Postgres, installs goose, applies all migrations from scratch. Catches broken SQL, missing semicolons, bad ordering, etc. before PRs merge.

### `e2e.yml`

Needs a Postgres service too, plus the app needs `DATABASE_URL` pointing at it.

---

## Phase 7 — Deploy workflow & Ansible

### `.github/workflows/deploy.yaml`

Before the blue/green block:

```bash
docker compose up -d postgres
```

Idempotent — if Postgres is already healthy, nothing happens. The blue/green logic for the app containers stays the same.

### `server_go/deploy/ansible/roles/whoknows-app/tasks/main.yml`

- Remove lines 65–74 (the SQLite bootstrap)
- Add a task to create `/opt/whoknows/pgdata` directory with correct permissions

### `server_go/deploy/ansible/roles/whoknows-app/templates/env.j2`

Add:

- `DATABASE_URL`
- `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` (for the postgres container)

---

## Phase 8 — Docs

Update `server_go/deploy/README.md` with:

- How to start Postgres locally (dev compose)
- How to run migrations manually (`goose -dir migrations postgres "..." up`)
- Environment variable reference

---

## Commit order (one feature branch, multiple commits)

1. Add Postgres to production compose + new dev compose
2. Swap Go deps, convert `internal/db/` to pgx, add goose startup
3. Convert `001_init.sql` to goose format, remove `ApplyMigrations`
4. Add `cmd/import-sqlite/` data migration command
5. Convert tests to Postgres helper
6. Update CI (`ci.yml`, `e2e.yml`) with Postgres services
7. Update `deploy.yaml` + Ansible role + env template
8. Docs

---

## Constraints & notes

- **Blue/green + migrations**: only ship additive (backward-compatible) migrations. CI's `migration-test` job catches broken SQL before merge.
- **Single instance**: goose runs in-process on app startup. No multi-replica coordination needed.
- **RAM**: Postgres + Go app on 1GB is tight but workable with the tuning above.
- **Data backup**: before production cutover, run `pg_dump` against the droplet's Postgres and store the result alongside `whoknows.db.backup`.
