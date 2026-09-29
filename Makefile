.PHONY: infra run dev stop

# Start infrastructure only (postgres + minio)
infra:
	docker-compose up -d postgres minio

# Run Go server locally (requires infra running)
run:
	go run ./cmd/app/main.go

# Start infra and run server
dev: infra
	go run ./cmd/app/main.go

# Stop infrastructure
stop:
	docker-compose stop postgres minio
