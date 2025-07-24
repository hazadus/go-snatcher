package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"time"
)

// StreamingReader представляет буферизованный поток для чтения данных порциями
type StreamingReader struct {
	reader     *bufio.Reader
	resp       *http.Response
	bufferSize int
}

// NewStreamingReader создает новый потоковый ридер
func NewStreamingReader(url string, bufferSize int) (*StreamingReader, error) {
	// Создаем HTTP клиент без таймаута для длительного потокового чтения
	client := &http.Client{
		// Убираем общий таймаут, оставляем только таймауты соединения
		Transport: &http.Transport{
			// Настройки для оптимального потокового чтения
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			// Таймаут для TLS handshake
			TLSHandshakeTimeout: 10 * time.Second,
			// Таймаут ожидания заголовков ответа
			ResponseHeaderTimeout: 30 * time.Second,
			// Время жизни соединения в пуле
			IdleConnTimeout: 300 * time.Second, // 5 минут
			// Максимальное время жизни соединения
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 2,
			// Отключаем ограничение на время ожидания между чтениями
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	// Создаем запрос с заголовками для потокового чтения
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Добавляем заголовки для оптимизации потокового чтения
	req.Header.Set("Accept-Encoding", "identity")   // Отключаем сжатие для потока
	req.Header.Set("Range", "bytes=0-")             // Указываем, что хотим читать с начала
	req.Header.Set("Connection", "keep-alive")      // Поддерживаем соединение
	req.Header.Set("User-Agent", "go-snatcher/1.0") // Идентифицируем клиент

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		resp.Body.Close()
		return nil, fmt.Errorf("ошибка HTTP: %s", resp.Status)
	}

	return &StreamingReader{
		reader:     bufio.NewReaderSize(resp.Body, bufferSize),
		resp:       resp,
		bufferSize: bufferSize,
	}, nil
}

// Read реализует интерфейс io.Reader для потокового чтения
func (sr *StreamingReader) Read(p []byte) (n int, err error) {
	return sr.reader.Read(p)
}

// Close закрывает соединение
func (sr *StreamingReader) Close() error {
	return sr.resp.Body.Close()
}

// getStreamStatus возвращает текстовое описание состояния потока
func getStreamStatus(stuckCount int) string {
	switch {
	case stuckCount == 0:
		return "Потоковое воспроизведение"
	case stuckCount <= 3:
		return "Буферизация..."
	case stuckCount <= 5:
		return "Медленная загрузка"
	default:
		return "Возможная проблема с соединением"
	}
}
