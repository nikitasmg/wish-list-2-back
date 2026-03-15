# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run full stack
docker-compose up --build

# Run only backend services (postgres, minio, server)
docker-compose up -d postgres minio server

# Build Go binary manually
go build -o main .

# Run locally (requires .env)
go run main.go
```

Environment variables are loaded from `.env` (see `common/config/config.go` for all required keys: `JWT_SECRET`, `PROD_URL`, `BOT_TOKEN`, `MINIO_ROOT_USER`, `MINIO_ROOT_PASSWORD`, `MINIO_ENDPOINT`, `MINIO_BUCKET_NAME`, `MINIO_USE_SSL`).

## Architecture

Go backend using **Fiber v2** web framework with **GORM + PostgreSQL** and **MinIO** for file storage.

**Request flow:** Nginx → Fiber router (`router/router.go`) → Handler (`handler/`) → Service (`services/`) → GORM (`database/database.go`)

**Layers:**
- `main.go` — bootstraps config, DB, MinIO, router
- `router/router.go` — defines all routes; protected routes require JWT middleware
- `handler/` — HTTP layer (user auth, wishlist CRUD, present CRUD)
- `services/` — business logic for wishlists and presents
- `model/` — GORM models: `User`, `Wishlist`, `Present`
- `pkg/minio/` — MinIO client and file upload/download helpers
- `helpers/` — shared utilities (JWT user extraction, error responses, cookie clearing)
- `common/config/` — environment config struct loaded with godotenv

**Auth:** JWT tokens (30-day expiry) stored in HTTPOnly cookies. Telegram OAuth also supported via `handler/user.go`.

**File storage:** Images uploaded to MinIO and served via Nginx proxy at `/minio/[bucket]/[uuid]`.

**Database:** GORM auto-migrates all models on startup. PostgreSQL 17.

## Related Projects

- **Frontend (Windows):** `C:\Users\nvsma\OneDrive\Документы\projects\wish-list-2-front`
- **Frontend (macOS):** `/Users/nvsmagin/WebstormProjects/wish-list-2-front`

## Infrastructure

- `docker-compose.yml` — 6 services: postgres, minio, server, frontend (wish-list-2-front), nginx, certbot
- `nginx/conf.d/default.conf` — reverse proxy with SSL; `api.get-my-wishlist.ru` → Go server, `/minio/` → MinIO
- `Dockerfile` — multi-stage build (golang:1.22 builder → alpine runtime)
