# Этап 1: Сборка приложения на Go
FROM golang:alpine AS builder

WORKDIR /app

# chai2010/webp requires CGO (libwebp C bindings)
RUN apk add --no-cache gcc musl-dev libwebp-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o main ./cmd/app

# Этап 2: Минимальный образ для запуска
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /app/.env .

# ca-certificates for TLS, libwebp for runtime CGO dependency
RUN apk add --no-cache ca-certificates libwebp

EXPOSE 8080

CMD ["./main"]