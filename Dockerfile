# Этап 1: Сборка приложения на Go
FROM golang:alpine AS builder

WORKDIR /app

# Копируем все необходимые файлы
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код и .env файл
COPY . .
COPY .env .

# Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Этап 2: Минимальный образ для запуска
FROM alpine:latest

WORKDIR /app

# Копируем бинарник и .env файл
COPY --from=builder /app/main .
COPY --from=builder /app/.env .

# Устанавливаем зависимости для SSL
RUN apk --no-c