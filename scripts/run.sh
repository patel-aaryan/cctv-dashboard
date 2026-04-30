#!/usr/bin/env bash
# Single-process run: Go server serves the API, the .mp4 files, AND the
# pre-built SvelteKit SPA. This is what production looks like.
#
# Re-run after editing Svelte source — the SPA is baked into frontend/build.
# For HMR while editing the UI, use ./scripts/dev.sh instead.
#
# Override defaults via env, e.g.:
#   API_PORT=9090 ./scripts/run.sh
#   CCTV_ARCHIVE_PATH=/some/other/dir ./scripts/run.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

mkdir -p data testdata

if [[ ! -d frontend/build ]]; then
  echo "→ frontend/build missing, building once..."
  ( cd frontend && [[ -d node_modules ]] || npm install --no-audit --no-fund )
  ( cd frontend && npm run build )
fi

export PATH="/usr/local/go/bin:$PATH"
export CCTV_ARCHIVE_PATH="${CCTV_ARCHIVE_PATH:-$ROOT/testdata}"
export DB_PATH="${DB_PATH:-$ROOT/data/cctv.db}"
export STATIC_PATH="${STATIC_PATH:-$ROOT/frontend/build}"
export API_PORT="${API_PORT:-8080}"

cat <<EOF
→ open http://localhost:${API_PORT}
   archive: ${CCTV_ARCHIVE_PATH}
   db:      ${DB_PATH}
   static:  ${STATIC_PATH}

EOF

cd backend
exec go run .
