# Этап 1: Сборка приложения на Go
FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Этап 2: Минимальный образ для запуска
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /app/.env .

# Исправленная команда: добавить --no-cache и ca-certificates
RUN apk add --no-cache ca-certificates

EXPOSE 8080

CMD ["./main"]