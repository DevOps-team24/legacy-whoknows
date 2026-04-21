#!/usr/bin/env bash
# One-time cutover: migrate users and pages from the droplet's legacy
# whoknows.db (SQLite) into the freshly-provisioned Postgres container.
#
# Run this ONCE, after Postgres is up on the droplet and before the
# blue/green app containers start serving. It:
#   1. copies the droplet's SQLite file down to a local temp path
#   2. opens an SSH tunnel to prod Postgres on localhost:5433
#   3. applies goose migrations against prod (creates the schema)
#   4. runs cmd/import-sqlite to copy all rows into prod Postgres
#   5. tears the tunnel down
#
# Prereqs on your laptop:
#   - Go toolchain (for `go run`)
#   - SSH key that's authorised on the droplet's deploy user
#
# Usage:
#   SSH_HOST=huw.dk SSH_USER=deploy \
#   POSTGRES_USER=... POSTGRES_PASSWORD=... POSTGRES_DB=whoknows \
#   ./scripts/cutover-sqlite-to-postgres.sh

set -euo pipefail

: "${SSH_HOST:?SSH_HOST required (e.g. huw.dk)}"
: "${SSH_USER:?SSH_USER required (e.g. deploy)}"
: "${POSTGRES_USER:?POSTGRES_USER required}"
: "${POSTGRES_PASSWORD:?POSTGRES_PASSWORD required}"
: "${POSTGRES_DB:=whoknows}"

TUNNEL_PORT="${TUNNEL_PORT:-5433}"
REMOTE_SQLITE="${REMOTE_SQLITE:-/opt/whoknows/data/whoknows.db}"
SERVER_GO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
LOCAL_SQLITE="$(mktemp --suffix=.db)"
CONTROL_SOCKET="$(mktemp -u)"

cleanup() {
  echo "→ Cleaning up"
  if ssh -S "$CONTROL_SOCKET" -O check "$SSH_USER@$SSH_HOST" 2>/dev/null; then
    ssh -S "$CONTROL_SOCKET" -O exit "$SSH_USER@$SSH_HOST" 2>/dev/null || true
  fi
  rm -f "$LOCAL_SQLITE"
}
trap cleanup EXIT

echo "→ Copying $REMOTE_SQLITE from droplet"
scp "$SSH_USER@$SSH_HOST:$REMOTE_SQLITE" "$LOCAL_SQLITE"

echo "→ Opening SSH tunnel: localhost:$TUNNEL_PORT → droplet 127.0.0.1:5432"
ssh -fN -M -S "$CONTROL_SOCKET" \
    -L "$TUNNEL_PORT:127.0.0.1:5432" \
    "$SSH_USER@$SSH_HOST"

export DATABASE_URL="postgres://$POSTGRES_USER:$POSTGRES_PASSWORD@localhost:$TUNNEL_PORT/$POSTGRES_DB?sslmode=disable"

echo "→ Applying goose migrations against prod Postgres"
(cd "$SERVER_GO_DIR" && go run github.com/pressly/goose/v3/cmd/goose \
    -dir migrations postgres "$DATABASE_URL" up)

echo "→ Importing users and pages from legacy SQLite"
(cd "$SERVER_GO_DIR" && go run ./cmd/import-sqlite -sqlite "$LOCAL_SQLITE")

echo ""
echo "✓ Cutover complete. Verify row counts on the droplet:"
echo "  ssh $SSH_USER@$SSH_HOST \"docker exec whoknows-postgres psql -U $POSTGRES_USER -d $POSTGRES_DB -c 'SELECT count(*) FROM users; SELECT count(*) FROM pages;'\""
