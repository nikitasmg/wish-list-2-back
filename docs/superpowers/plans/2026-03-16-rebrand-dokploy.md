# Rebranding + Dokploy Deployment Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rename all domain references from `get-my-wishlist.ru` to `prosto-namekni.ru` and migrate the deployment stack from Nginx+Certbot to Dokploy-compatible docker-compose with Traefik.

**Architecture:** Remove nginx/certbot/frontend services from docker-compose; add Traefik labels to `server` and `minio`; introduce `MINIO_PUBLIC_URL` env var to decouple MinIO public URL from CORS config; delete obsolete nginx/certbot directories.

**Tech Stack:** Go, Fiber v2, Docker Compose, Dokploy/Traefik, MinIO

**Spec:** `docs/superpowers/specs/2026-03-16-rebrand-dokploy-design.md`

---

## Chunk 1: Go config and MinIO URL fix

### Task 1: Add `MinioPublicURL` to config

**Files:**
- Modify: `config/config.go`

- [ ] **Step 1: Add `MinioPublicURL` field to `AppConfig` struct**

  In `config/config.go`, add the field after `CORSOrigin`:

  ```go
  type AppConfig struct {
      Port           string
      CORSOrigin     string
      MinioPublicURL string
      Env            string
  }
  ```

- [ ] **Step 2: Load `MINIO_PUBLIC_URL` from env and update domain defaults**

  Replace the `App` block in `LoadConfig()`:

  ```go
  App: AppConfig{
      Port:           getEnv("PORT", "8080"),
      CORSOrigin:     getEnv("CORS_ORIGIN", "https://prosto-namekni.ru"),
      MinioPublicURL: getEnv("MINIO_PUBLIC_URL", "https://files.prosto-namekni.ru"),
      Env:            getEnv("APP_ENV", "production"),
  },
  ```

  Also update `Auth.CookieDomain` default:

  ```go
  Auth: AuthConfig{
      JWTSecret:    getEnv("JWT_SECRET", ""),
      BotToken:     getEnv("BOT_TOKEN", ""),
      CookieDomain: getEnv("COOKIE_DOMAIN", "prosto-namekni.ru"),
  },
  ```

- [ ] **Step 3: Verify it compiles**

  ```bash
  go build ./...
  ```

  Expected: no errors.

- [ ] **Step 4: Commit**

  ```bash
  git add config/config.go
  git commit -m "feat: add MinioPublicURL config field, rebrand default domains"
  ```

---

### Task 2: Fix MinIO public URL construction

**Files:**
- Modify: `pkg/minio/minio.go:89`

- [ ] **Step 1: Update the URL format in `Upload`**

  Change line 89 in `pkg/minio/minio.go` from:

  ```go
  url := fmt.Sprintf("%s/minio/%s/%s", s.publicURL, s.bucketName, objectID)
  ```

  To:

  ```go
  url := fmt.Sprintf("%s/%s/%s", s.publicURL, s.bucketName, objectID)
  ```

  This removes the `/minio/` prefix that was an Nginx proxy artifact. New URLs will be: `https://files.prosto-namekni.ru/wish-list-bucket/<uuid>`.

- [ ] **Step 2: Verify it compiles**

  ```bash
  go build ./...
  ```

  Expected: no errors.

- [ ] **Step 3: Run existing tests**

  ```bash
  go test ./...
  ```

  Expected: all pass (upload tests use mocks, so they're unaffected by the format change).

- [ ] **Step 4: Commit**

  ```bash
  git add pkg/minio/minio.go
  git commit -m "fix: remove /minio/ nginx prefix from MinIO public URL"
  ```

---

### Task 3: Pass `MinioPublicURL` to `minioPkg.New()` in `app.go`

**Files:**
- Modify: `internal/app/app.go:46`

- [ ] **Step 1: Replace `cfg.App.CORSOrigin` with `cfg.App.MinioPublicURL`**

  Change line 46 in `internal/app/app.go` from:

  ```go
  fileStorage, err := minioPkg.New(cfg.Minio, cfg.App.CORSOrigin)
  ```

  To:

  ```go
  fileStorage, err := minioPkg.New(cfg.Minio, cfg.App.MinioPublicURL)
  ```

- [ ] **Step 2: Verify it compiles and tests pass**

  ```bash
  go build ./... && go test ./...
  ```

  Expected: no errors, all tests pass.

- [ ] **Step 3: Commit**

  ```bash
  git add internal/app/app.go
  git commit -m "fix: pass MinioPublicURL to minio client instead of CORSOrigin"
  ```

---

## Chunk 2: Infra and config files

> **Depends on Chunk 1 being complete.** `MINIO_PUBLIC_URL` appears in `.env.example` and `CLAUDE.md` — these reference the config field and app wiring added in Tasks 1–3.

### Task 4: Update `.env.example`

**Files:**
- Modify: `.env.example`

- [ ] **Step 1: Rewrite `.env.example`**

  Replace the entire file content:

  ```
  # App
  PORT=8080
  APP_ENV=dev
  CORS_ORIGIN=https://prosto-namekni.ru
  COOKIE_DOMAIN=prosto-namekni.ru

  # DB
  # In docker-compose (production): DB_HOST=postgres, DB_PORT=5432
  # In local dev (outside docker): DB_HOST=localhost, DB_PORT=5430
  DB_HOST=localhost
  DB_PORT=5430
  DB_USER=postgres
  DB_PASSWORD=postgres
  DB_NAME=wishlist_db
  DB_SSLMODE=disable

  # Auth
  JWT_SECRET=your-secret-key
  BOT_TOKEN=your-telegram-bot-token

  # MinIO
  # In docker-compose (production): MINIO_ENDPOINT=minio:9000
  # In local dev (outside docker): MINIO_ENDPOINT=localhost:9000
  MINIO_ENDPOINT=localhost:9000
  MINIO_PUBLIC_URL=https://files.prosto-namekni.ru
  MINIO_BUCKET_NAME=wish-list-bucket
  MINIO_ROOT_USER=your-minio-user
  MINIO_ROOT_PASSWORD=your-minio-password
  MINIO_USE_SSL=false
  ```

- [ ] **Step 2: Commit**

  ```bash
  git add .env.example
  git commit -m "chore: update .env.example for new domain and MINIO_PUBLIC_URL"
  ```

---

### Task 5: Rewrite `docker-compose.yml` for Dokploy

**Files:**
- Modify: `docker-compose.yml`

- [ ] **Step 1: Replace `docker-compose.yml` entirely**

  ```yaml
  version: '3.8'

  services:
    minio:
      container_name: minio
      image: minio/minio:latest
      command: server --console-address ":9001" /data/
      environment:
        MINIO_ROOT_USER: "${MINIO_ROOT_USER}"
        MINIO_ROOT_PASSWORD: "${MINIO_ROOT_PASSWORD}"
      volumes:
        - ../files/minio-storage:/data
      healthcheck:
        test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
        interval: 30s
        timeout: 20s
        retries: 3
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

    postgres:
      image: postgres:17-alpine
      container_name: postgres_container
      environment:
        POSTGRES_USER: "${DB_USER}"
        POSTGRES_PASSWORD: "${DB_PASSWORD}"
        POSTGRES_DB: "${DB_NAME}"
        PGDATA: /var/lib/postgresql/data/pgdata
      volumes:
        - ../files/pgdata:/var/lib/postgresql/data/pgdata
      deploy:
        resources:
          limits:
            cpus: '0.50'
            memory: 512M
          reservations:
            cpus: '0.25'
            memory: 256M
      command: >
        postgres -c max_connections=1000
                 -c shared_buffers=256MB
                 -c effective_cache_size=768MB
                 -c maintenance_work_mem=64MB
                 -c checkpoint_completion_target=0.7
                 -c wal_buffers=16MB
                 -c default_statistics_target=100
      healthcheck:
        test: ["CMD-SHELL", "pg_isready -U ${DB_USER} -d ${DB_NAME}"]
        interval: 30s
        timeout: 10s
        retries: 5
      networks:
        - internal
      restart: unless-stopped

    server:
      container_name: server
      restart: unless-stopped
      build:
        context: .
      env_file:
        - .env
      networks:
        - internal
        - dokploy-network
      depends_on:
        - postgres
        - minio
      labels:
        - "traefik.enable=true"
        - "traefik.http.routers.wishlist-api.rule=Host(`api.prosto-namekni.ru`)"
        - "traefik.http.routers.wishlist-api.entrypoints=websecure"
        - "traefik.http.routers.wishlist-api.tls.certResolver=letsencrypt"
        - "traefik.http.services.wishlist-api.loadbalancer.server.port=8080"

  networks:
    internal:
      driver: bridge
    dokploy-network:
      external: true
  ```

- [ ] **Step 2: Note on bind-mount directories**

  Dokploy creates `../files/` on first deploy. If deploying manually, create the directories first to avoid permission issues:

  ```bash
  mkdir -p ../files/pgdata ../files/minio-storage
  ```

- [ ] **Step 3: Commit**

  ```bash
  git add docker-compose.yml
  git commit -m "feat: rewrite docker-compose for Dokploy (Traefik, remove nginx/certbot/frontend)"
  ```

---

### Task 6: Delete obsolete files

**Files:**
- Delete: `nginx/` directory
- Delete: `certbot/` directory
- Delete: `init-letsencrypt.sh`

- [ ] **Step 1: Remove obsolete directories and script**

  ```bash
  git rm -r nginx/ certbot/ init-letsencrypt.sh
  ```

- [ ] **Step 2: Commit**

  ```bash
  git commit -m "chore: remove nginx, certbot, and init-letsencrypt.sh (replaced by Traefik)"
  ```

---

### Task 7: Update `CLAUDE.md`

**Files:**
- Modify: `CLAUDE.md`

- [ ] **Step 1: Update domain references and fix config path**

  Make the following specific changes in `CLAUDE.md`:

  1. Replace `get-my-wishlist.ru` → `prosto-namekni.ru` and `api.get-my-wishlist.ru` → `api.prosto-namekni.ru` everywhere.

  2. Fix the config path: `common/config/config.go` → `config/config.go`.

  3. Update the required env keys list — replace:
     `JWT_SECRET`, `PROD_URL`, `BOT_TOKEN`, `MINIO_ROOT_USER`, `MINIO_ROOT_PASSWORD`, `MINIO_ENDPOINT`, `MINIO_BUCKET_NAME`, `MINIO_USE_SSL`
     with:
     `JWT_SECRET`, `BOT_TOKEN`, `CORS_ORIGIN`, `COOKIE_DOMAIN`, `MINIO_ROOT_USER`, `MINIO_ROOT_PASSWORD`, `MINIO_ENDPOINT`, `MINIO_BUCKET_NAME`, `MINIO_USE_SSL`, `MINIO_PUBLIC_URL`

  4. Update request flow line from:
     `Nginx → Fiber router` → `Traefik → Fiber router`

  5. Update file storage description from:
     `Images uploaded to MinIO and served via Nginx proxy at /minio/[bucket]/[uuid]`
     to:
     `Images uploaded to MinIO and served directly at https://files.prosto-namekni.ru/[bucket]/[uuid]`

  6. Update docker-compose services list: remove `nginx`, `certbot`, `frontend` (6 services → 3 services). Add note: frontend is deployed as a separate Dokploy project from `wish-list-2-front` repo.

- [ ] **Step 2: Commit**

  ```bash
  git add CLAUDE.md
  git commit -m "docs: update CLAUDE.md for new domain and Dokploy deployment"
  ```

---

## Pre-Deploy Notes (not code tasks)

Before deploying to Dokploy:

1. **DNS** — Create A records pointing to VPS IP:
   - `prosto-namekni.ru`
   - `api.prosto-namekni.ru`
   - `files.prosto-namekni.ru`

2. **Dokploy UI** — Create a new Compose project from this Git repo, set all env vars from `.env.example` in the env editor.

3. **Frontend** — Deploy `wish-list-2-front` as a separate Dokploy project with:
   - `NEXT_PUBLIC_API_URL=https://api.prosto-namekni.ru`
   - Traefik domain: `prosto-namekni.ru`
