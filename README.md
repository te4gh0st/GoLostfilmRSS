# GoLostfilmRSS

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Language](https://img.shields.io/badge/language-Go-brightgreen)

## Прокси для RSS-ленты LostFilm.TV

Этот проект представляет собой Go-приложение, которое работает как прокси для RSS-ленты LostFilm.TV.
Оно позволяет получать RSS-ленту, скачивать торрент-файлы, подменять в них URL трекера на свой собственный и 
предоставлять эти торренты и измененную RSS-ленту по локальным URL.


## Особенности

* **Проксирование RSS:** Загружает оригинальную RSS-ленту с LostFilm.TV.
* **Скачивание торрентов:** Загружает `.torrent` файлы по ссылкам из RSS.
* **Подмена трекера:** Изменяет URL трекера в скачанных `.torrent` файлах для использования вашего TrackerID.
* **Локальные ссылки:** Генерирует локальные ссылки на торрент-файлы в обработанной RSS-ленте.
* **Фильтрация по качеству:** Возможность игнорировать торренты определенного качества (например, SD, MP4).
* **Автоматическое обновление:** Периодически обновляет RSS-ленту и торренты.
* **Docker-поддержка:** Удобное развертывание с использованием Docker и Docker Compose.

## Настройка и Запуск

### 1. Клонирование репозитория

```bash
git clone [https://github.com/te4gh0st/GoLostfilmRSS.git](https://github.com/te4gh0st/GoLostfilmRSS.git)
cd GoLostfilmRSS
````

### 2. Получение необходимых данных

Для работы приложения вам потребуются следующие данные:

* **`uid`** – Ваш ID на сайте lostfilm.tv. Его можно найти в настройках вашего личного кабинета.
* **`usess`** – Ваша юзер-сессия. Её можно найти в окне [insearch.site](http://insearch.site), внизу слева от значка RSS. Кликните на надпись `usess`, и вам будет показано ваше значение.
* **`REMOTE_RSS`** – URL удалённой RSS-ленты LostFilm.TV. Обычно это что-то вроде `http://insearch.site/rssdd.xml` или другой URL, который вы используете для RSS.
* **`TRACKER_ID`** – Значение, которое будет вставлено в URL трекера в торрент-файлах. Чтобы получить это значение, скачайте любой Torrent-файл с LostFilm, добавьте его в торрент-клиент, затем скопируйте выделенное значение из URL трекера, которое выглядит примерно так:
  `http://bt.tracktor.in/tracker.php/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx/announce`
  В данном примере **`xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`** и будет вашим `TRACKER_ID`.

### 3. Настройка Docker Compose

Откройте файл `docker-compose.yml` и заполните значения в разделе `environment` (переменные окружения):

```yaml
services:
  lostfilm-rss-proxy:
    build: .
    container_name: lostfilm-rss-proxy
    restart: unless-stopped
    ports:
      - "8080:80" # Порт для доступа к RSS и торрентам. Измените 8080 на нужный, если занят.
    environment:
      # ОБЯЗАТЕЛЬНЫЕ ПАРАМЕТРЫ:
      - REMOTE_RSS=УДАЛЕННЫЙ_RSS_URL # Пример: "https://insearch.site/rssdd.xml"
      - TRACKER_ID=ВАШ_TRACKER_ID
      - COOKIE_UID=ВАШ_UID              # Ваш UID с lostfilm.tv
      - COOKIE_USESS=ВАШ_USESS          # Ваш USESS (указан в окне insearch.site, внизу слева от значка RSS)
      # НЕОБЯЗАТЕЛЬНЫЕ ПАРАМЕТРЫ:
      #- PORT=80 # Порт, на котором будет работать HTTP сервер внутри контейнера (должен совпадать с EXPOSE в Dockerfile)
      - BASE_URL=http://ВАШ_IP_ИЛИ_ДОМЕН:8080 # Базовый URL для генерации локальных ссылок (ВАЖНО для внешнего доступа)
      - IGNORE_QUALITY=SD,MP4           # Игнорируемые качества, через запятую (например, SD,MP4,1080p)
#    volumes:
#      - lostfilm-cache:/app/cache # хранилище для RSS и торрентов

#volumes:
#  lostfilm-cache:
```

### 4. Запуск с помощью Docker Compose

Перейдите в директорию проекта, где находятся `Dockerfile` и `docker-compose.yml`, и выполните:

```bash
docker compose up -d
```

Эта команда соберет Docker-образ, создаст контейнер и запустит его в фоновом режиме.

### 5. Доступ к RSS-ленте

После успешного запуска ваш прокси будет доступен по адресу, указанному в `BASE_URL` с добавлением `/rss`.

Например, если ваш `BASE_URL` был `http://localhost:80`, то адрес RSS-ленты будет:
`http://localhost:80/rss`. Добавьте этот URL в ваш торрент-клиент (например, qBittorrent, Transmission и т.д.) для автоматического отслеживания новых серий.
-----

### 6. Так же можно запустить напрямую

```shell
  > ./go-lostfilm-rss -h
  
  -base_url string
        Базовый URL для генерации локальных ссылок (например, http://your_domain.com:8080)
  -ignore_quality string
        Игнорируемые качества, через запятую (например, "SD,MP4")
  -port string
        Порт для HTTP сервера
  -rss string
        URL удалённой RSS ленты
  -tracker string
        ID трекера для подмены
  -uid string
        UID cookie
  -usess string
        USESS cookie

```

---

## Пример RSS Ленты

```xml
<rss version="0.91">
    <channel>
        <title>Lostfilm RSS</title>
        <link>https://github.com/te4gh0st/GoLostfilmRSS</link>
        <description>Новинки от LostFilm.TV</description>
        <item>
            <title>Ходячие мертвецы: Город мертвых (The Walking Dead: Dead City). Если бы история была пожаром (S02E08) [1080p]</title>
            <link>http://localhost:80/torrents/65256.torrent</link>
            <category>[1080p]</category>
            <pubDate>Tue, 24 Jun 2025 20:57:00 +0000</pubDate>
        </item>
        <item>
            <title>Сюрриэлторы (SurrealEstate). (S03E999) [1080p]</title>
            <link>http://localhost:80/torrents/65253.torrent</link>
            <category>[1080p]</category>
            <pubDate>Tue, 24 Jun 2025 15:31:49 +0000</pubDate>
        </item><item>
            <title>Гангстерлэнд (MobLand). (S01E999) [1080p]</title>
            <link>http://localhost:80/torrents/65250.torrent</link>
            <category>[1080p]</category>
            <pubDate>Tue, 24 Jun 2025 15:27:57 +0000</pubDate>
        </item>
        <item>
            <title>Фубар (Fubar). Погружение в бездну (S02E05) [1080p]</title>
            <link>http://localhost:80/torrents/65247.torrent</link>
            <category>[1080p]</category>
            <pubDate>Mon, 23 Jun 2025 20:33:44 +0000</pubDate>
        </item>
        <item>
            <title>Больница Питт (The Pitt). 17:00 (S01E11) [1080p]</title>
            <link>http://localhost:80/torrents/65244.torrent</link>
            <category>[1080p]</category>
            <pubDate>Mon, 23 Jun 2025 18:53:18 +0000</pubDate>
        </item>
    </channel>
</rss>

```

---
MIT License

Copyright (c) 2025 (te4gh0st) Vitaly Timtsurak