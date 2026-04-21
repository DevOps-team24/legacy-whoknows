# SQLite → PostgreSQL Cutover Guide

One-time runbook for the first production deploy of the Postgres-backed
version of WhoKnows. After this has been done once, it never has to be done
again — future deploys use the normal `deploy.yaml` blue/green flow and this
guide becomes dead-code documentation.

## What changes on the droplet

| Before                                                | After                                                    |
| ----------------------------------------------------- | -------------------------------------------------------- |
| Single `whoknows` container reading `/opt/whoknows/data/whoknows.db` | `postgres` service + `whoknows-blue` / `whoknows-green` reading `DATABASE_URL` |
| Data persisted as a SQLite file on a bind mount       | Data persisted in the `postgres` container's `pgdata` volume |
| No schema management                                  | Goose migrations applied automatically on app startup    |

## Pre-flight (do these before the cutover)

1. **GitHub org secrets exist and are correct**. Confirm via
   `gh secret list --org DevOps-Team36` that the following are present:
   - `CR_PAT` — PAT with `read:packages` scope
   - `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`
   - `SSH_HOST`, `SSH_USER`, `SSH_PRIVATE_KEY`

   Remember: secrets are write-only. If you don't have the values locally,
   rotate them on GitHub and save the new values to your password manager.

2. **Local prereqs**:
   - Go toolchain on your laptop (`go version`)
   - Ansible installed (`ansible --version`)
   - SSH key authorised on the droplet's `deploy` user
   - Your SSH config / agent resolves `SSH_HOST` to the droplet

3. **Test the GHCR token** without polluting your docker credential cache:
   ```bash
   curl -u devops-team36:$CR_PAT https://ghcr.io/v2/ -w "\n%{http_code}\n"
   ```
   200 = good, 401 = bad token.

4. **Back up the legacy SQLite file** off the droplet — cheap insurance:
   ```bash
   scp deploy@<droplet>:/opt/whoknows/data/whoknows.db ./whoknows-prod-backup.db
   ```

## The cutover

### Step 1 — Export env vars locally

Either `export` them inline, or keep them in a gitignored `server_go/scripts/cutover.env`
and `source` it. Example file:

```bash
# server_go/scripts/cutover.env (gitignored)
export CR_PAT=ghp_...
export SSH_HOST=huw.dk
export SSH_USER=deploy
export POSTGRES_USER=ossas
export POSTGRES_PASSWORD=ossas1234
export POSTGRES_DB=whoknows
```

```bash
source server_go/scripts/cutover.env
```

### Step 2 — Run Ansible to provision Postgres on the droplet

```bash
cd server_go
ansible-playbook deploy/ansible/playbook.yml --skip-tags app
```

The `--skip-tags app` flag makes Ansible stop after Postgres is running — it
does **not** start `whoknows-blue` yet. That prevents blue from serving with an
empty database during the cutover.

What Ansible does in this run:

1. Creates `/opt/whoknows/{pgdata,logs}` on the droplet
2. Writes `/opt/whoknows/docker-compose.yml` (the new blue/green + postgres version)
3. Writes `/opt/whoknows/.env` from your exported secrets
4. Stops and removes the old `whoknows` container (if it still exists from the SQLite era)
5. Logs into GHCR, pulls the new images
6. Starts the `postgres` container

After this completes: Postgres is up, the app is **not yet running**, and there
is no traffic being served. Port 8080 on the droplet is free.

### Step 3 — Run the cutover script

```bash
./scripts/cutover-sqlite-to-postgres.sh
```

What it does:

1. SCPs `/opt/whoknows/data/whoknows.db` from the droplet to a local temp file
2. Opens an SSH tunnel: `localhost:5433` → droplet's `postgres:5432`
3. Applies all goose migrations against prod Postgres (creates the schema)
4. Runs `cmd/import-sqlite` against the tunnel — copies all users and pages
5. Closes the tunnel and deletes the temp SQLite file

Expected output: "users: imported N rows", "pages: imported M rows",
"import complete".

### Step 4 — Merge `dev` → `main`

This triggers `.github/workflows/deploy.yaml`, which:

1. SCPs the updated `docker-compose.server.yml` to the droplet
2. Pulls the latest image
3. Starts `whoknows-blue` (goose sees the schema already exists, runs no migrations)
4. Health-checks blue on port 8080
5. Flips nginx to blue, nobody is affected because blue is already healthy
6. (No old container to stop; Ansible already handled that in Step 2)

### Step 5 — Verify

```bash
# Row counts in prod Postgres
ssh deploy@<droplet> "docker exec whoknows-postgres psql -U $POSTGRES_USER -d $POSTGRES_DB -c 'SELECT COUNT(*) FROM users; SELECT COUNT(*) FROM pages;'"

# App responds
curl -i https://huw.dk/
curl -s https://huw.dk/api/search?q=go | head -c 200

# Log in with a real existing account through the browser
```

## Troubleshooting

**GHCR login fails in Ansible** → `CR_PAT` is wrong or expired. Ansible halts
before starting any containers. Files on disk (`.env`, `docker-compose.yml`)
will have been overwritten but no services are affected. Fix `CR_PAT` and re-run.

**`docker compose up -d whoknows-blue` fails with port in use** → The old
`whoknows` container is still running. Ansible should handle this now via the
"stop legacy" task, but if it didn't, manually:
```bash
ssh deploy@<droplet> "docker stop whoknows && docker rm whoknows"
```

**Cutover script: `scp` fails for the SQLite file** → The droplet might already
have had its `/opt/whoknows/data/` directory cleaned up. Use the local backup
you made in pre-flight:
```bash
go run ./cmd/import-sqlite -sqlite ./whoknows-prod-backup.db
# (manually set up the SSH tunnel first — see the script for the command)
```

**Cutover script: tunnel fails to open** → Port 5433 already in use locally, or
SSH key issues. Either set `TUNNEL_PORT=5434` and re-run, or
`kill $(lsof -ti:5433)` then retry.

**Users can't log in after cutover** → The import preserved user IDs, but if
passwords look wrong check the `password` column isn't empty:
```sql
SELECT id, username, LENGTH(password) FROM users LIMIT 5;
```
All MD5 hashes should be 32 characters.

## Post-cutover cleanup (do this later)

Once prod has been stable on Postgres for a week or two:

1. Delete the legacy SQLite file and its volume:
   ```bash
   ssh deploy@<droplet> "rm -rf /opt/whoknows/data"
   ```
2. Open a cleanup PR that removes:
   - `server_go/cmd/import-sqlite/`
   - `server_go/scripts/cutover-sqlite-to-postgres.sh`
   - `server_go/deploy/CUTOVER-GUIDE.md` (this file)
   - The "stop legacy whoknows" task in `ansible/roles/whoknows-app/tasks/main.yml`
   - The `app` tags on the Ansible tasks (or leave them — they're harmless)
