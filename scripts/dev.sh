#!/usr/bin/env bash
# Dual-process dev loop:
#   - Go backend on :8080  (API + /media)
#   - Vite dev server on :5173 (HMR, proxies /api and /media to :8080)
# Browse http://localhost:5173 — edits to Svelte source hot-reload.
#
# Ctrl+C tears down both processes via the EXIT trap.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

mkdir -p data testdata

if [[ ! -d frontend/node_modules ]]; then
  echo "→ installing frontend deps..."
  ( cd frontend && npm install --no-audit --no-fund )
fi

# STATIC_PATH still needs a real dir even though :5173 serves the SPA.
# If frontend/build doesn't exist yet, point at static/ as a placeholder —
# requests to :8080/ will just 404, which is fine because you'll be browsing :5173.
if [[ -d frontend/build ]]; then
  STATIC_DEFAULT="$ROOT/frontend/build"
else
  STATIC_DEFAULT="$ROOT/frontend/static"
fi

export PATH="/usr/local/go/bin:$PATH"
export CCTV_ARCHIVE_PATH="${CCTV_ARCHIVE_PATH:-$ROOT/testdata}"
export DB_PATH="${DB_PATH:-$ROOT/data/cctv.db}"
export STATIC_PATH="${STATIC_PATH:-$STATIC_DEFAULT}"
export API_PORT="${API_PORT:-8080}"

cleanup() {
  echo
  echo "→ shutting down..."
  jobs -p | xargs -I{} kill {} 2>/dev/null || true
  wait 2>/dev/null || true
}
trap cleanup EXIT INT TERM

cat <<EOF
→ frontend (HMR): http://localhost:5173    ← browse here
→ backend (api):  http://localhost:${API_PORT}
   archive:       ${CCTV_ARCHIVE_PATH}
   db:            ${DB_PATH}

EOF

( cd backend && go run . ) &
sleep 1  # let the backend bind before Vite starts proxying

cd frontend
npm run dev
