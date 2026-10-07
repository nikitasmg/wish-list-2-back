.PHONY: infra run dev stop tg-webhook

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

# Регистрирует вебхук бота Санты. Нужны SANTA_BOT_TOKEN (или BOT_TOKEN),
# TELEGRAM_WEBHOOK_SECRET и API_PUBLIC_URL в окружении:
#   SANTA_BOT_TOKEN=… TELEGRAM_WEBHOOK_SECRET=… API_PUBLIC_URL=https://api.prosto-namekni.ru make tg-webhook
tg-webhook:
	@curl -fsS "https://api.telegram.org/bot$${SANTA_BOT_TOKEN:-$$BOT_TOKEN}/setWebhook" \
		--data-urlencode "url=$${API_PUBLIC_URL}/api/v1/telegram/webhook" \
		--data-urlencode "secret_token=$${TELEGRAM_WEBHOOK_SECRET}" \
		--data-urlencode 'allowed_updates=["message"]'
	@echo
