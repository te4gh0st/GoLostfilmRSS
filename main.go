package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"github.com/zeebo/bencode"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	debug             = true
	cacheDir          = "cache"
	torrentDir        = "cache/torrents"
	originalRSSPath   = "cache/rss_original.xml"
	processedRSSPath  = "cache/rss.xml"
	rssUpdateInterval = 5 * time.Minute
)

type Config struct {
	RemoteRSS     string
	TrackerID     string
	CookieUID     string
	CookieUSESS   string
	IgnoreQuality []string
	Port          string
	BaseURL       string // Добавлено для формирования абсолютных URL
}

var config Config
var mu sync.Mutex

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title    string `xml:"title"`
	Link     string `xml:"link"`
	Category string `xml:"category"`
	PubDate  string `xml:"pubDate"`
}

func loadConfig() {
	flag.StringVar(&config.RemoteRSS, "rss", os.Getenv("REMOTE_RSS"), "URL удалённой RSS ленты")
	flag.StringVar(&config.TrackerID, "tracker", os.Getenv("TRACKER_ID"), "ID трекера для подмены")
	flag.StringVar(&config.CookieUID, "uid", os.Getenv("COOKIE_UID"), "UID cookie")
	flag.StringVar(&config.CookieUSESS, "usess", os.Getenv("COOKIE_USESS"), "USESS cookie")
	ignoreQ := flag.String("ignore_quality", os.Getenv("IGNORE_QUALITY"),
		"Игнорируемые качества, через запятую (например, \"SD,MP4\")")
	flag.StringVar(&config.Port, "port", os.Getenv("PORT"), "Порт для HTTP сервера")
	flag.StringVar(&config.BaseURL, "base_url", os.Getenv("BASE_URL"),
		"Базовый URL для генерации локальных ссылок (например, http://your_domain.com:8080)")

	flag.Parse()

	if config.RemoteRSS == "" || config.TrackerID == "" || config.CookieUID == "" || config.CookieUSESS == "" {
		log.Fatal("Неверная конфигурация: REMOTE_RSS, TRACKER_ID, COOKIE_UID, COOKIE_USESS обязательны.")
	}

	if config.Port == "" {
		config.Port = "80"
	}

	if config.BaseURL == "" {
		config.BaseURL = fmt.Sprintf("http://localhost:%s", config.Port) // Fallback для BaseURL
		log.Printf("Внимание: BASE_URL не указан. Используется '%s'."+
			" Если сервер доступен извне, установите корректный BASE_URL.", config.BaseURL)
	}
	if *ignoreQ != "" {
		config.IgnoreQuality = strings.Split(*ignoreQ, ",")
	}
}

func fetchOriginalRSS() error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", config.RemoteRSS, nil)
	if err != nil {
		return fmt.Errorf("ошибка создания запроса для RSS: %w", err)
	}
	req.AddCookie(&http.Cookie{Name: "uid", Value: config.CookieUID})
	req.AddCookie(&http.Cookie{Name: "usess", Value: config.CookieUSESS})

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка при получении оригинального RSS: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("получен некорректный статус при загрузке RSS: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения тела ответа RSS: %w", err)
	}

	return os.WriteFile(originalRSSPath, body, 0644)
}

func rewriteTorrent(link, id string) error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return fmt.Errorf("ошибка создания запроса для торрента %s: %w", id, err)
	}
	req.AddCookie(&http.Cookie{Name: "uid", Value: config.CookieUID})
	req.AddCookie(&http.Cookie{Name: "usess", Value: config.CookieUSESS})

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка при получении торрента %s: %w", id, err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("получен некорректный статус при загрузке торрента %s: %s", id, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения тела ответа торрента %s: %w", id, err)
	}

	if !bytes.HasPrefix(data, []byte("d8:announce")) && !bytes.HasPrefix(data, []byte("d10:announce")) {
		return fmt.Errorf("полученные данные для торрента %s не похожи на .torrent файл", id)
	}

	// Декодируем Bencode
	// Используем map[string]interface{} для универсального декодирования
	var torrentData map[string]interface{}
	err = bencode.DecodeBytes(data, &torrentData) // Используем DecodeBytes
	if err != nil {
		return fmt.Errorf("ошибка декодирования торрент файла %s: %w", id, err)
	}

	// Регулярное выражение для поиска и замены части "tracker.php//announce".
	re := regexp.MustCompile(`(tracker\.php)(/+)announce`)
	replacementFormat := "${1}/%s/announce" // Формат для замены, %s будет заменен на config.TrackerID

	// Функция для обработки одного URL трекера (возвращает string)
	processTrackerURL := func(trackerURL string) string {
		return re.ReplaceAllString(trackerURL, fmt.Sprintf(replacementFormat, config.TrackerID))
	}

	// Обрабатываем announce поле
	if announce, ok := torrentData["announce"]; ok {
		if announceBytes, isBytes := announce.([]byte); isBytes {
			torrentData["announce"] = []byte(processTrackerURL(string(announceBytes)))
		} else if announceString, isString := announce.(string); isString {
			torrentData["announce"] = processTrackerURL(announceString)
		}
	}

	// Обрабатываем announce-list
	if announceList, ok := torrentData["announce-list"]; ok {
		if listInterface, isList := announceList.([]interface{}); isList {
			newAnnounceList := make([]interface{}, len(listInterface))
			for i, tier := range listInterface {
				if tierSlice, isSlice := tier.([]interface{}); isSlice {
					newTierSlice := make([]interface{}, len(tierSlice))
					for j, tracker := range tierSlice {
						if trackerURLBytes, isBytes := tracker.([]byte); isBytes {
							newTierSlice[j] = []byte(processTrackerURL(string(trackerURLBytes)))
						} else if trackerURLString, isString := tracker.(string); isString {
							newTierSlice[j] = processTrackerURL(trackerURLString)
						} else {
							newTierSlice[j] = tracker // Оставить как есть, если не строка/ []byte
						}
					}
					newAnnounceList[i] = newTierSlice
				} else {
					newAnnounceList[i] = tier // Оставить как есть, если не слайс (неправильный формат, но на всякий случай)
				}
			}
			torrentData["announce-list"] = newAnnounceList
		}
	}

	// Кодируем обратно в Bencode
	patchedData, err := bencode.EncodeBytes(torrentData) // Используем EncodeBytes
	if err != nil {
		return fmt.Errorf("ошибка кодирования торрент файла %s: %w", id, err)
	}

	path := filepath.Join(torrentDir, id+".torrent")
	return os.WriteFile(path, patchedData, 0644)
}

func processRSS() error {
	mu.Lock()
	defer mu.Unlock()

	data, err := os.ReadFile(originalRSSPath)
	if err != nil {
		return fmt.Errorf("ошибка чтения оригинального RSS файла: %w", err)
	}

	rss := RSS{}
	err = xml.Unmarshal(data, &rss)
	if err != nil {
		return fmt.Errorf("ошибка парсинга оригинального RSS: %w", err)
	}

	var processedItems []Item // Создаем новый слайс для элементов, которые будут включены в обработанный RSS

	for i := range rss.Channel.Items {
		item := rss.Channel.Items[i] // Копируем элемент, чтобы избежать модификации в цикле без указателя
		ignore := false

		for _, q := range config.IgnoreQuality {
			if strings.Contains(item.Category, q) {
				ignore = true
				break
			}
		}
		if ignore {
			if debug {
				log.Printf("Игнорируем торрент: %s (качество: %s)\n", item.Title, item.Category)
			}
			continue
		}

		idMatch := regexp.MustCompile(`id=(\d+)(?:&|$)`).FindStringSubmatch(item.Link)
		if len(idMatch) < 2 {
			if debug {
				log.Printf("Не удалось извлечь ID из ссылки: %s\n", item.Link)
			}
			continue
		}
		id := idMatch[1]

		torrentFilePath := filepath.Join(torrentDir, id+".torrent")
		if _, err := os.Stat(torrentFilePath); os.IsNotExist(err) {
			if debug {
				log.Printf("Загружаем и перезаписываем торрент: %s (ID: %s)\n", item.Title, id)
			}
			err := rewriteTorrent(item.Link, id)
			if err != nil {
				log.Printf("Ошибка при перезаписи торрента %s (ID: %s): %v\n", item.Title, id, err)
				continue
			}
		} else if err != nil {
			log.Printf("Ошибка при проверке файла торрента %s: %v\n", torrentFilePath, err)
			continue
		} else {
			if debug {
				log.Printf("Торрент уже существует: %s (ID: %s)\n", item.Title, id)
			}
		}

		// ИСПРАВЛЕНИЕ: Формирование URL вручную или с использованием url.URL.JoinPaths
		// чтобы избежать проблем с filepath.Join и обратными слешами в URL.
		parsedBaseURL, err := url.Parse(config.BaseURL)
		if err != nil {
			log.Printf("Ошибка парсинга BaseURL '%s': %v. Используется localhost.", config.BaseURL, err)
			item.Link = fmt.Sprintf("http://localhost:%s/torrents/%s.torrent", config.Port, id)
		} else {
			// Использование url.JoinPath для корректного формирования URL пути с прямыми слешами
			// без необходимости ручной обработки filepath.ToSlash
			item.Link = parsedBaseURL.JoinPath("torrents", id+".torrent").String()
		}

		processedItems = append(processedItems, item)
	}

	rss.Channel.Items = processedItems
	rss.Channel.Title = "Lostfilm RSS"
	rss.Channel.Link = "https://github.com/te4gh0st/GoLostfilmRSS"
	file, err := os.Create(processedRSSPath)
	if err != nil {
		return fmt.Errorf("ошибка создания файла обработанного RSS: %w", err)
	}
	defer file.Close()

	encoder := xml.NewEncoder(file)
	encoder.Indent("", "  ")
	return encoder.Encode(rss)
}

func updateLoop() {
	if debug {
		log.Println("[UPDATE] Выполняем первоначальное обновление RSS...")
	}
	err := fetchOriginalRSS()
	if err != nil {
		log.Printf("Ошибка первоначального обновления оригинального RSS: %v\n", err)
	} else {
		err = processRSS()
		if err != nil {
			log.Printf("Ошибка первоначальной обработки RSS: %v\n", err)
		}
	}

	ticker := time.NewTicker(rssUpdateInterval)
	defer ticker.Stop()

	for range ticker.C {
		if debug {
			log.Println("[UPDATE] Обновление RSS...")
		}
		err := fetchOriginalRSS()
		if err != nil {
			log.Printf("Ошибка обновления оригинального RSS: %v\n", err)
		} else {
			err = processRSS()
			if err != nil {
				log.Printf("Ошибка обработки RSS: %v\n", err)
			}
		}
	}
}

func serveRSS(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	data, err := os.ReadFile(processedRSSPath)
	if err != nil {
		log.Printf("Ошибка чтения обработанного RSS файла: %v\n", err)
		http.Error(w, "RSS не доступен", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Write(data)
}

func serveTorrent(w http.ResponseWriter, r *http.Request) {
	file := filepath.Base(r.URL.Path)
	path := filepath.Join(torrentDir, file)

	if strings.Contains(file, "..") {
		http.Error(w, "Недопустимый запрос файла", http.StatusBadRequest)
		return
	}

	if debug {
		log.Printf("Попытка отдать торрент: %s\n", path)
	}

	http.ServeFile(w, r, path)
}

func main() {
	loadConfig()
	_ = os.MkdirAll(cacheDir, 0755)
	_ = os.MkdirAll(torrentDir, 0755)

	go updateLoop()

	http.HandleFunc("/rss", serveRSS)
	http.HandleFunc("/torrents/", serveTorrent)
	log.Printf("Сервер запущен на порту %s. Доступ к RSS по адресу %s/rss\n", config.Port, config.BaseURL)
	log.Fatal(http.ListenAndServe(":"+config.Port, nil))
}
