# syntax=docker/dockerfile:1.7

# ---------------------------------------------------------------------
# Stage 1 — build the SvelteKit SPA (static output)
# ---------------------------------------------------------------------
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend

COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install --no-audit --no-fund

COPY frontend/ ./
RUN npm run build

# ---------------------------------------------------------------------
# Stage 2 — build the Go binary (CGO required by mattn/go-sqlite3)
# ---------------------------------------------------------------------
FROM golang:1.26-bookworm AS backend-builder
WORKDIR /src

RUN apt-get update \
 && apt-get install -y --no-install-recommends build-essential libsqlite3-dev \
 && rm -rf /var/lib/apt/lists/*

# Resolve dependencies first for layer caching. go.sum is generated on the
# fly if it is not committed yet — `go mod download` populates it.
COPY backend/go.mod ./
COPY backend/go.sum* ./
RUN go mod download

COPY backend/ ./

ENV CGO_ENABLED=1 GOOS=linux GOARCH=amd64
RUN go build -trimpath -ldflags="-s -w" -o /out/cctv-dashboard ./

# ---------------------------------------------------------------------
# Stage 3 — minimal runtime image
# ---------------------------------------------------------------------
FROM debian:bookworm-slim AS runtime

RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates libsqlite3-0 tzdata \
 && rm -rf /var/lib/apt/lists/* \
 && groupadd --system cctv \
 && useradd  --system --gid cctv --home /app --shell /usr/sbin/nologin cctv \
 && mkdir -p /app /data \
 && chown -R cctv:cctv /app /data

WORKDIR /app

COPY --from=backend-builder  /out/cctv-dashboard      /usr/local/bin/cctv-dashboard
COPY --from=frontend-builder /app/frontend/build      /app/web

USER cctv
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/cctv-dashboard"]
