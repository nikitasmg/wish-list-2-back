#!/bin/bash

# Создаем необходимые директории
mkdir -p certbot/www certbot/conf

# Копируем временную конфигурацию nginx
cp nginx/conf.d/temporary.conf nginx/conf.d/default.conf

# Запускаем nginx
echo "Starting nginx with temporary configuration..."
docker-compose up -d nginx

# Ждем запуска nginx
echo "Waiting for nginx to start..."
sleep 5

# Запускаем certbot для получения сертификатов
echo "Requesting SSL certificates from Let's Encrypt..."
docker-compose run --rm certbot certonly --webroot -w /var/www/certbot \
  --email nvsmagin@mail.ru \
  -d get-my-wishlist.ru -d api.get-my-wishlist.ru \
  --agree-tos --noninteractive --keep-until-expiring

if [ $? -eq 0 ]; then
    echo "Certificates obtained successfully!"
    # Копируем постоянную конфигурацию
    cp nginx/conf.d/permanent.conf nginx/conf.d/default.conf
    # Перезапускаем nginx
    echo "Restarting nginx with SSL configuration..."
    docker-compose restart nginx
    echo "Nginx restarted with SSL configuration"
else
    echo "Failed to obtain certificates. Check your domain configuration."
    exit 1
fi