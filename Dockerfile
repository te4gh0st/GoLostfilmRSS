# Используем официальный образ Go для сборки приложения
FROM golang:1.24-alpine AS builder

# Устанавливаем необходимые пакеты
#RUN apk add --no-cache git

# Устанавливаем рабочую директорию внутри контейнера
WORKDIR /app

# Копируем go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код приложения
COPY . .

# Собираем приложение
# CGO_ENABLED=0 и GOOS=linux необходимы для статической сборки без зависимомостей от glibc
# -a - принудительная пересборка всех пакетов
# -installsuffix nocgo - указывает компилятору не использовать CGo
# -ldflags="-s -w" - уменьшает размер бинарника, удаляя отладочную информацию
RUN go build -o /usr/local/bin/go-lostfilm-rss -v -ldflags="-s -w" ./

# Используем минимальный образ Alpine для запуска приложения
FROM alpine:latest

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем скомпилированный бинарник из стадии сборки
COPY --from=builder /usr/local/bin/go-lostfilm-rss /usr/local/bin/go-lostfilm-rss

# Создаем директории для кэша и торрентов
RUN mkdir -p /app/cache/torrents

# Устанавливаем бинарник как исполняемый
RUN chmod +x /usr/local/bin/go-lostfilm-rss

# Открываем порт, указанный в вашей конфигурации (по умолчанию 8080)
EXPOSE 80

# Команда для запуска приложения
# Используем ENV-переменные для передачи настроек
ENTRYPOINT ["/usr/local/bin/go-lostfilm-rss"]