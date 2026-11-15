# Yandex Object Storage deployment

The production stack runs the Go API and PostgreSQL. Images live in an external S3 bucket; no MinIO server or files-domain router is deployed. The existing minio-go package is an S3-compatible client, not a MinIO server dependency.

## Cloud setup

- Bucket: `prosto-namekni-prod`, Standard storage.
- Allow anonymous object reads for images shown on public wishlist pages. Keep object listing and bucket settings private. Do not use this public bucket for database backups.
- Create a dedicated service account; grant `storage.editor` on this bucket only, then generate a static access key.
- Endpoint: `storage.yandexcloud.net`; signing region: `ru-central1`.
- Public bucket URL: `https://storage.yandexcloud.net/prosto-namekni-prod`.

## Dokploy Environment

Keep existing DB, JWT, CORS, cookie and notification settings. Replace the old MINIO_* settings with:

```dotenv
S3_ENDPOINT=storage.yandexcloud.net
S3_REGION=ru-central1
S3_BUCKET_NAME=prosto-namekni-prod
S3_ACCESS_KEY_ID=REPLACE_IN_DOKPLOY
S3_SECRET_ACCESS_KEY=REPLACE_IN_DOKPLOY
S3_PUBLIC_URL=https://storage.yandexcloud.net/prosto-namekni-prod
S3_USE_SSL=true
S3_AUTO_CREATE_BUCKET=false
```

`S3_ENDPOINT` is a hostname without `https://` or paths. `S3_PUBLIC_URL` is the full HTTPS bucket URL: uploads append only the object key, not a second bucket name. The API checks bucket access at startup with a 10-second timeout and fails if the bucket is absent or inaccessible; it never creates buckets or changes their permissions in production. Uploads/deletes also have a 10-second timeout.

Do not put real keys in Git, patches, screenshots or chat. `.env` is supplied at runtime with Compose, never copied into the image.

After these repository changes are published to the branch selected in Dokploy, remove the superseded patches for `docker-compose.yml`, `Dockerfile`, `.dockerignore` and any MinIO build Dockerfile. Otherwise patches can restore the obsolete stack. Keep the existing PostgreSQL data mount; no volumes or directories need deleting. Save Environment before deploying. Neither commits nor deployment are performed by this change.

The `files.prosto-namekni.ru` DNS record is not used by new uploads. No DNS change is needed to use the Yandex URL. If switching to a custom public domain later, configure its HTTPS and object path handling before changing S3_PUBLIC_URL; previously saved links are not automatically rewritten.

## Verification after deployment

1. Confirm API startup logs show no S3 errors and PostgreSQL is healthy.
2. Request `https://api.prosto-namekni.ru/health`.
3. Register/sign in, create a wishlist and upload an image.
4. Open its returned Yandex HTTPS URL in a private browser window: it must load without credentials, and its bucket name must occur once.
5. Check that an anonymous bucket listing is denied. Verify a service-account test upload/delete independently; deleting a gift currently does not promise to delete its object.
6. Restart the service and verify the image and wishlist persist.

Object storage on another service survives a VPS disk loss, but is not a complete backup strategy. Keep database dumps in a separate private backup bucket and test restoration.

## Local development

Legacy MINIO_* settings remain supported when S3_* settings are unset. `docker-compose.dev.yml` passes .env at runtime and explicitly enables local bucket provisioning. Its pre-existing MinIO image reference is historical and may only work with a cached image; a fresh local environment should use a separate external development bucket and an adapted Compose instead of production credentials. Do not reuse the production bucket for local experiments.

## Local validation

Without a C compiler: `CGO_ENABLED=0 go test ./...`, `CGO_ENABLED=0 go vet ./...`, `CGO_ENABLED=0 go build ./...`. Production uses CGO for WebP; validate the Docker build on a Docker-enabled host. Integration tests require a Docker daemon and the integration build tag. Live Yandex access requires credentials and is checked only after deployment.
