package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/dhowden/tag"
	"github.com/gopxl/beep/mp3"
)

// TrackMetadata хранит метаданные трека
type TrackMetadata struct {
	Artist string
	Title  string
	Album  string
}

// Функция для получения метаданных из reader
func getMetadataFromReader(reader io.ReadCloser, source string) TrackMetadata {
	// Создаем временный файл для чтения метаданных
	tempFile, err := os.CreateTemp("", "snatcher-*.mp3")
	if err != nil {
		return getDefaultMetadata(source)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Копируем данные в временный файл
	_, err = io.Copy(tempFile, reader)
	if err != nil {
		return getDefaultMetadata(source)
	}

	// Сбрасываем позицию в файле
	if _, err := tempFile.Seek(0, 0); err != nil {
		return getDefaultMetadata(source)
	}

	metadata, err := tag.ReadFrom(tempFile)
	if err != nil {
		// Если не удалось прочитать метаданные, возвращаем значения по умолчанию
		return getDefaultMetadata(source)
	}

	artist := metadata.Artist()
	title := metadata.Title()
	album := metadata.Album()

	// Если метаданные пустые, используем имя файла или URL как название
	if title == "" {
		title = getFileNameFromSource(source)
	}

	return TrackMetadata{
		Artist: artist,
		Title:  title,
		Album:  album,
	}
}

// Функция для получения имени файла из источника (локальный файл или URL)
func getFileNameFromSource(source string) string {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		// Для URL извлекаем имя файла из пути
		parts := strings.Split(source, "/")
		filename := parts[len(parts)-1]
		// Убираем параметры запроса
		if idx := strings.Index(filename, "?"); idx != -1 {
			filename = filename[:idx]
		}
		// Если имя файла пустое или это корневой путь, используем домен
		if filename == "" || filename == "/" {
			// Извлекаем домен из URL
			urlParts := strings.Split(source, "/")
			if len(urlParts) >= 3 {
				filename = urlParts[2] // domain
			} else {
				filename = "online_track"
			}
		}
		return strings.TrimSuffix(filename, ".mp3")
	}
	// Для локального файла
	return getFileNameWithoutExt(source)
}

// Функция для получения метаданных по умолчанию
func getDefaultMetadata(source string) TrackMetadata {
	return TrackMetadata{
		Artist: "Неизвестный исполнитель",
		Title:  getFileNameFromSource(source),
		Album:  "Неизвестный альбом",
	}
}

// Функция для форматирования длительности
func formatDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

// Функция для получения имени файла без расширения
func getFileNameWithoutExt(filepath string) string {
	parts := strings.Split(filepath, "/")
	filename := parts[len(parts)-1]
	return strings.TrimSuffix(filename, ".mp3")
}

// Функция для определения длительности MP3 файла в секундах
func getMP3Duration(filePath string) (time.Duration, error) {
	// Открываем файл
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("ошибка открытия файла: %v", err)
	}
	defer file.Close()

	// Декодируем MP3 для получения длительности
	streamer, format, err := mp3.Decode(file)
	if err != nil {
		return 0, fmt.Errorf("ошибка декодирования MP3: %v", err)
	}
	defer streamer.Close()

	// Получаем длительность трека
	duration := format.SampleRate.D(streamer.Len())

	return duration, nil
}
