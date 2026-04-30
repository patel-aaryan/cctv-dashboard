# CCTV Dashboard

A small self-hosted dashboard for browsing CCTV clips stored as `.mp4` files on disk. A **Go** backend indexes footage into **SQLite**, watches the archive for new or removed files, and serves the **API**, **video streams**, and a static **SvelteKit** UI in one process. Use it on a NAS or any machine that can read your recording folder.

## What it does

- Scans an archive directory for `.mp4` files and keeps a SQLite index (camera name + timestamp from each filename).
- Watches the filesystem so new clips appear and removed/renamed clips drop out of the list without a restart.
- Serves a dark-themed grid UI with HTML5 video playback.
- Exposes a JSON API and a `/health` check suitable for Docker or reverse proxies.

## Clip filename format

Only files whose names parse correctly are indexed. Pattern:

```text
<camera_name>_YYYY-MM-DD_HH-MM-SS.mp4
```

- **Camera name** can contain underscores; the last two underscore-separated segments are always the date and time.
- Example: `front-door_2026-04-30_14-30-00.mp4` → camera `front-door`, local time `2026-04-30 14:30:00`.

Files that do not match are skipped (logged at debug level during scans).

## Architecture

| Layer | Role |
|--------|------|
| `backend/` | HTTP server: `/api/videos`, `/media/…` (`.mp4` only, path-safe), `/health`, and static SPA from `STATIC_PATH`. |
| `frontend/` | SvelteKit app built with `@sveltejs/adapter-static`; production assets are embedded in the Docker image under `/app/web`. |
| `data/` | Default location for `cctv.db` when using the helper scripts (create with scripts or compose). |

## Requirements

- **Go** 1.23+ (module `go 1.23`) and **CGO** with SQLite dev libraries for native builds (`mattn/go-sqlite3`).
- **Node.js** 22+ (Dockerfile) or compatible with the frontend lockfile for UI builds / dev.
- **Linux/macOS** (or WSL): filesystem watcher (`fsnotify`) is used for live updates.

## Quick start (local)

1. Clone the repo and ensure `CCTV_ARCHIVE_PATH` points at a directory of `.mp4` files (or use the default `testdata/` from the scripts).

2. **Production-style single process** (Go serves API + media + built SPA):

   ```bash
   chmod +x scripts/run.sh
   ./scripts/run.sh
   ```

   Opens `http://localhost:8080` (override with `API_PORT`). The script builds `frontend/` once if `frontend/build` is missing.

3. **Development** (Vite HMR on `:5173`, Go API on `:8080`):

   ```bash
   cp frontend/.env.example frontend/.env   # optional; defaults proxy to localhost:8080
   chmod +x scripts/dev.sh
   ./scripts/dev.sh
   ```

   Browse **http://localhost:5173**. API and `/media` requests are proxied to the backend.

### Manual backend run

From repo root, after building the frontend to `frontend/build`:

```bash
export CCTV_ARCHIVE_PATH=/path/to/mp4s
export DB_PATH=./data/cctv.db
export STATIC_PATH=./frontend/build
export API_PORT=8080
cd backend && go run .
```

`CCTV_ARCHIVE_PATH` and `DB_PATH` are required. `STATIC_PATH` defaults to `/app/web` if unset (Docker layout).

## Configuration

| Variable | Required | Description |
|----------|----------|-------------|
| `CCTV_ARCHIVE_PATH` | Yes | Absolute path to the read-only (in production) folder containing `.mp4` clips. |
| `DB_PATH` | Yes | Path to the SQLite file (directory must exist or be creatable). |
| `API_PORT` | No | Listen port (default `8080`). |
| `STATIC_PATH` | No | Directory with the built SvelteKit site (default `/app/web`). |
| `CORS_ALLOWED_ORIGIN` | No | If set, enables CORS for that origin (e.g. `http://localhost:5173` during Vite dev). Leave empty when the UI is served from the same origin as the API. |

Host-side variables for **Docker Compose** are documented in `.env.example` (`HOST_CCTV_PATH`, `HOST_DATA_PATH`, `HOST_PORT`, etc.).

## HTTP API

- **`GET /health`** — JSON `{"status":"ok"}` if the process and DB are healthy; `503` if the database ping fails.
- **`GET /api/videos`** — JSON array of up to 500 clips, newest first: `id`, `camera_name`, `timestamp` (RFC3339), `media_url` (path under `/media/…`).
- **`GET /media/<relative-path>.mp4`** — Streams a file from the archive; rejects traversal and non-`.mp4` extensions.

## Docker

Build a multi-stage image (frontend static build + Go binary) from the repository root:

```bash
docker build -t cctv-dashboard:local .
```

Run with the same env vars as the binary; mount the archive read-only and persist `/data` for SQLite. See `docker-compose.yml` and `.env.example` for a NAS-oriented layout (`HOST_*` paths and `CCTV_ARCHIVE_PATH` / `DB_PATH` inside the container).

The image runs as a non-root user and includes a `HEALTHCHECK` that runs `cctv-dashboard -healthcheck`.

### CI image

Pushes to **GitHub Container Registry** are defined in `.github/workflows/docker.yml` when relevant paths change. Ensure the image name/tag in that workflow matches the image you reference in `docker-compose.yml` (compose uses `ghcr.io/<username>/cctv-dashboard:latest` as a template).

## Project layout

```text
backend/          Go module (server, SQLite, watcher)
frontend/         SvelteKit SPA (adapter-static)
scripts/          run.sh (single process), dev.sh (dual process)
Dockerfile        Production image
docker-compose.yml
.env.example      Container + host env template
```
