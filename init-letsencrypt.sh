#!/bin/bash

# Создаем необходимые директории
mkdir -p certbot/www certbot/conf

# Копируем временную конфигурацию nginx
cp nginx/conf.d/temporary.conf nginx/conf.d/default.conf

# Запускаем nginx и certbot
docker-compose up -d nginx
echo "Waiting for nginx to start..."
sleep 5

docker-compose up certbot
if [ $? -eq 0 ]; then
    echo "Certificates obtained successfully!"
    # Копируем постоянную конфигурацию
    cp nginx/conf.d/permanent.conf nginx/conf.d/default.conf
    # Перезапускаем nginx
    docker-compose restart nginx
    echo "Nginx restarted with SSL configuration"
else
    echo "Failed to obtain certificates. Check your domain configuration."
    exit 1
fi