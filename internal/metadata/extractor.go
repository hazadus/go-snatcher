// Package metadata предоставляет функционал для извлечения метаданных из аудио файлов
package metadata

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
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

// FileInfo содержит информацию о файле
type FileInfo struct {
	Size     int64
	Duration time.Duration
}

// Extractor извлекает метаданные из аудио файлов
type Extractor struct{}

// NewExtractor создает новый экстрактор метаданных
func NewExtractor() *Extractor {
	return &Extractor{}
}

// ExtractFromReader извлекает метаданные из io.Reader
func (e *Extractor) ExtractFromReader(reader io.ReadSeeker, source string) TrackMetadata {
	// Сбрасываем reader в начало
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return e.getDefaultMetadata(source)
	}

	metadata, err := tag.ReadFrom(reader)
	if err != nil {
		return e.getDefaultMetadata(source)
	}

	return TrackMetadata{
		Artist: metadata.Artist(),
		Title:  metadata.Title(),
		Album:  metadata.Album(),
	}
}

// ExtractFromFile извлекает метаданные из файла
func (e *Extractor) ExtractFromFile(filePath string) TrackMetadata {
	file, err := os.Open(filePath)
	if err != nil {
		return e.getDefaultMetadata(filePath)
	}
	defer file.Close()

	return e.ExtractFromReader(file, filePath)
}

// GetDuration получает длительность MP3 файла
func (e *Extractor) GetDuration(filePath string) (time.Duration, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer file.Close()

	streamer, format, err := mp3.Decode(file)
	if err != nil {
		return 0, fmt.Errorf("ошибка декодирования MP3: %w", err)
	}
	defer streamer.Close()

	// Вычисляем длительность
	return format.SampleRate.D(streamer.Len()), nil
}

// GetFileInfo получает информацию о файле (размер и длительность)
func (e *Extractor) GetFileInfo(filePath string) (*FileInfo, error) {
	// Получаем размер файла
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения информации о файле: %w", err)
	}

	// Получаем длительность
	duration, err := e.GetDuration(filePath)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения длительности: %w", err)
	}

	return &FileInfo{
		Size:     fileInfo.Size(),
		Duration: duration,
	}, nil
}

// getDefaultMetadata возвращает метаданные по умолчанию на основе имени файла
func (e *Extractor) getDefaultMetadata(source string) TrackMetadata {
	fileName := filepath.Base(source)
	nameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))

	// Пытаемся разобрать имя файла в формате "Artist - Title"
	parts := strings.Split(nameWithoutExt, " - ")
	if len(parts) >= 2 {
		return TrackMetadata{
			Artist: strings.TrimSpace(parts[0]),
			Title:  strings.TrimSpace(strings.Join(parts[1:], " - ")),
			Album:  "",
		}
	}

	// Если не удалось разобрать, используем имя файла как название
	return TrackMetadata{
		Artist: "Unknown Artist",
		Title:  nameWithoutExt,
		Album:  "",
	}
}
