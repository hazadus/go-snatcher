// Package uploader предоставляет функционал для загрузки файлов с метаданными
package uploader

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hazadus/go-snatcher/internal/data"
	"github.com/hazadus/go-snatcher/internal/metadata"
	"github.com/hazadus/go-snatcher/internal/s3"
)

// Service управляет процессом загрузки файлов
type Service struct {
	s3Uploader        *s3.Uploader
	metadataExtractor *metadata.Extractor
	appData           *data.AppData
}

// NewService создает новый сервис загрузки
func NewService(s3Uploader *s3.Uploader, appData *data.AppData) *Service {
	return &Service{
		s3Uploader:        s3Uploader,
		metadataExtractor: metadata.NewExtractor(),
		appData:           appData,
	}
}

// UploadResult содержит результат загрузки
type UploadResult struct {
	URL      string
	Metadata metadata.TrackMetadata
	FileInfo *metadata.FileInfo
}

// UploadFile загружает файл с метаданными
func (s *Service) UploadFile(ctx context.Context, filePath string, progressCallback func(int64)) (*UploadResult, error) {
	// Проверяем существование файла
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("файл не найден: %s", filePath)
	}

	// Получаем информацию о файле
	fileInfo, err := s.metadataExtractor.GetFileInfo(filePath)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения информации о файле: %w", err)
	}

	// Извлекаем метаданные
	trackMetadata := s.metadataExtractor.ExtractFromFile(filePath)

	// Открываем файл для загрузки
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer file.Close()

	// Создаем reader с отслеживанием прогресса
	var reader io.Reader = file
	if progressCallback != nil {
		reader = &ProgressReader{
			Reader:     file,
			Size:       fileInfo.Size,
			OnProgress: progressCallback,
		}
	}

	// Формируем ключ для S3
	fileName := getFileNameWithoutExt(filePath)
	s3Key := fileName + ".mp3"

	// Загружаем файл с контекстом
	url, err := s.s3Uploader.UploadFile(ctx, reader, s3Key)
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки в S3: %w", err)
	}

	return &UploadResult{
		URL:      url,
		Metadata: trackMetadata,
		FileInfo: fileInfo,
	}, nil
}

// UpdateApplicationData обновляет данные приложения с информацией о треке
func (s *Service) UpdateApplicationData(result *UploadResult) error {
	track := data.TrackMetadata{
		Artist:   result.Metadata.Artist,
		Title:    result.Metadata.Title,
		Album:    result.Metadata.Album,
		Length:   int(result.FileInfo.Duration.Seconds()),
		FileSize: result.FileInfo.Size,
		URL:      result.URL,
	}

	s.appData.AddTrack(track)
	return nil
}

// ProgressReader структура для отслеживания прогресса чтения
type ProgressReader struct {
	io.Reader
	Size       int64
	OnProgress func(int64)
	bytesRead  int64
}

func (pr *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = pr.Reader.Read(p)
	pr.bytesRead += int64(n)
	if pr.OnProgress != nil {
		pr.OnProgress(pr.bytesRead)
	}
	return n, err
}

// getFileNameWithoutExt возвращает имя файла без расширения
func getFileNameWithoutExt(filePath string) string {
	fileName := filepath.Base(filePath)
	return strings.TrimSuffix(fileName, filepath.Ext(fileName))
}

// FormatFileSize форматирует размер файла в читаемом виде
func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatDuration форматирует длительность времени
func FormatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
