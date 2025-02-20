# Используем многоэтапную сборку для уменьшения размера финального образа
# Этап 1: Сборка приложения на Go
FROM golang:alpine AS builder

WORKDIR /app

# Копируем файлы зависимостей
COPY go.mod go.sum ./

# Скачиваем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN go build -o main .

# Этап 2: Создание финального образа с Nginx
FROM nginx:latest

# Копируем конфигурацию Nginx
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Копируем SSL/TLS сертификаты
COPY /etc/letsencrypt/live/api.get-my-wishlist.ru/ /etc/letsencrypt/live/api.get-my-wishlist.ru/

# Копируем собранное приложение из этапа builder
COPY --from=builder /app/main /usr/local/bin/main

# Открываем порты
EXPOSE 80 443

# Запускаем Nginx и бэкенд
CMD ["sh", "-c", "nginx -g 'daemon off;' & /usr/local/bin/main"]