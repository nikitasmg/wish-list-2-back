# Rebranding to prosto-namekni.ru + Dokploy Deployment

**Date:** 2026-03-16
**Status:** Approved

## Overview

Migrate the wishlist app from domain `get-my-wishlist.ru` to `prosto-namekni.ru` and switch deployment from a manual docker-compose + Nginx + Certbot setup to self-hosted Dokploy on a VPS.

## Goals

1. Replace all references to the old domain with `prosto-namekni.ru`
2. Remove Nginx and Certbot containers — Dokploy's built-in Traefik handles SSL and routing
3. Restructure `docker-compose.yml` to be Dokploy-compatible
4. Deploy frontend as a separate Dokploy project from its own Git repo

## Domain Mapping

| Service       | Old                      | New                       |
|---------------|--------------------------|---------------------------|
| Frontend      | `get-my-wishlist.ru`     | `prosto-namekni.ru`       |
| API           | `api.get-my-wishlist.ru` | `api.prosto-namekni.ru`   |
| MinIO files   | `/minio/` path on main   | `files.prosto-namekni.ru` |
| MinIO console | internal                 | internal (no public access) |

## Architecture

### Backend Compose project (this repo)

Services:
- **postgres** — internal network only; bind-mount volume `../files/pgdata:/var/lib/postgresql/data/pgdata`
- **minio** — internal + `dokploy-network`; Traefik routes `files.prosto-namekni.ru` → port `9000`; bind-mount volume `../files/minio-storage:/data`
- **server** — internal + `dokploy-network`; Traefik routes `api.prosto-namekni.ru` → port `8080`; reads env via `env_file: .env` (no inline `environment:` block)

Removed services: `nginx`, `certbot`, `frontend`

Networks:
- `internal` — bridge network for inter-service communication (postgres ↔ minio ↔ server)
- `dokploy-network` — external network managed by Dokploy/Traefik

Volumes: bind mounts using `../files/` prefix (Dokploy convention for host-path volumes).

### Frontend project (separate Dokploy project)

- Source: `wish-list-2-front` Git repo
- Traefik routes `prosto-namekni.ru` → port `3000`
- Env: `NEXT_PUBLIC_API_URL=https://api.prosto-namekni.ru`

### SSL / HTTPS

Traefik (built into Dokploy) handles Let's Encrypt certificate provisioning automatically. No Certbot needed. DNS A records for all three domains must point to the VPS IP before deploying.

## MinIO Public URL — Code Change Required

Currently `pkg/minio/minio.go` constructs public file URLs as:

```go
url := fmt.Sprintf("%s/minio/%s/%s", s.publicURL, s.bucketName, objectID)
```

`publicURL` was sourced from `CORSOrigin` (`https://get-my-wishlist.ru`) because Nginx proxied MinIO under `/minio/` on the main domain.

After migration, MinIO is served directly at `https://files.prosto-namekni.ru`. Two changes are required:

1. **New env var `MINIO_PUBLIC_URL`** — added to `Config` struct and loaded from environment. Default: `https://files.prosto-namekni.ru`.
2. **`CORSOrigin` must no longer be passed to `minioPkg.New()`** — the two concerns (CORS, MinIO URL) must be separated.
3. **URL format changes** from `%s/minio/%s/%s` to `%s/%s/%s` — dropping the `/minio/` Nginx prefix but keeping the bucket name in the path (MinIO S3 API always requires `/<bucket>/<object>`). New URLs: `https://files.prosto-namekni.ru/wish-list-bucket/<uuid>`.

4. **`pkg/minio/minio.go` already accepts `publicURL string` as a separate parameter** — only the `fmt.Sprintf` format string needs updating, not the function signature.

5. **Audit callers of `Delete`** — check whether any caller strips the old URL prefix pattern (`/minio/<bucket>/`) to extract the objectID before passing it. If so, update the stripping logic to match the new URL format.

Affected files: `config/config.go` (add `MinioPublicURL` field), `pkg/minio/minio.go` (update `fmt.Sprintf` format string), `app.go` (pass `cfg.App.MinioPublicURL` instead of `cfg.App.CORSOrigin` to `minioPkg.New()`).

## `docker-compose.dev.yml`

Out of scope — local dev compose is unchanged.

## Files to Change

| File | Change |
|------|--------|
| `docker-compose.yml` | Full rewrite: remove nginx/certbot/frontend, add Traefik labels, bind-mount volumes, `env_file` for server, update networks |
| `config/config.go` | Update defaults: `CORS_ORIGIN` → `https://prosto-namekni.ru`, `COOKIE_DOMAIN` → `prosto-namekni.ru`; add `MinioPublicURL` field reading `MINIO_PUBLIC_URL` |
| `pkg/minio/minio.go` | Accept `publicURL` as a separate param instead of deriving from `CORSOrigin`; update URL format to `%s/%s` |
| `main.go` / `app.go` | Pass `cfg.App.MinioPublicURL` to `minioPkg.New()` instead of `cfg.App.CORSOrigin` |
| `.env.example` | Update `CORS_ORIGIN`, `COOKIE_DOMAIN`; add `MINIO_PUBLIC_URL`; remove unused `PROD_URL` |
| `CLAUDE.md` | Update domain references; fix config path from `common/config/config.go` to `config/config.go` |

## Files to Delete

| Path | Reason |
|------|--------|
| `nginx/` (entire directory) | Nginx replaced by Traefik |
| `certbot/` (entire directory) | SSL handled by Traefik |
| `init-letsencrypt.sh` | No longer needed |

## New `docker-compose.yml` skeleton

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:17-alpine
    container_name: postgres_container
    environment:
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: ${DB_NAME}
      PGDATA: /var/lib/postgresql/data/pgdata
    volumes:
      - ../files/pgdata:/var/lib/postgresql/data/pgdata
    networks:
      - internal
    restart: unless-stopped

  minio:
    container_name: minio
    image: minio/minio:latest
    command: server --console-address ":9001" /data/
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
    volumes:
      - ../files/minio-storage:/data
    networks:
      - internal
      - dokploy-network
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.wishlist-files.rule=Host(`files.prosto-namekni.ru`)"
      - "traefik.http.routers.wishlist-files.entrypoints=websecure"
      - "traefik.http.routers.wishlist-files.tls.certResolver=letsencrypt"
      - "traefik.http.services.wishlist-files.loadbalancer.server.port=9000"
    restart: unless-stopped

  server:
    container_name: server
    build:
      context: .
    env_file:
      - .env
    depends_on:
      - postgres
      - minio
    networks:
      - internal
      - dokploy-network
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.wishlist-api.rule=Host(`api.prosto-namekni.ru`)"
      - "traefik.http.routers.wishlist-api.entrypoints=websecure"
      - "traefik.http.routers.wishlist-api.tls.certResolver=letsencrypt"
      - "traefik.http.services.wishlist-api.loadbalancer.server.port=8080"
    restart: unless-stopped

networks:
  internal:
    driver: bridge
  dokploy-network:
    external: true
```

## Environment Variables (Dokploy UI)

Set in Dokploy project env editor, consumed via `env_file: .env`:

```
JWT_SECRET=...
BOT_TOKEN=...
CORS_ORIGIN=https://prosto-namekni.ru
COOKIE_DOMAIN=prosto-namekni.ru
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=...
DB_NAME=wishlist_db
DB_SSLMODE=disable
MINIO_ENDPOINT=minio:9000
MINIO_PUBLIC_URL=https://files.prosto-namekni.ru
MINIO_BUCKET_NAME=wish-list-bucket
MINIO_ROOT_USER=...
MINIO_ROOT_PASSWORD=...
MINIO_USE_SSL=false
```

## Pre-Deploy Checklist

- [ ] DNS A records for `prosto-namekni.ru`, `api.prosto-namekni.ru`, `files.prosto-namekni.ru` point to VPS IP
- [ ] Dokploy installed and running on VPS
- [ ] `dokploy-network` exists (created automatically by Dokploy on install)
- [ ] Env variables set in Dokploy UI before first deploy
- [ ] Frontend deployed as separate Dokploy project with `NEXT_PUBLIC_API_URL=https://api.prosto-namekni.ru`
