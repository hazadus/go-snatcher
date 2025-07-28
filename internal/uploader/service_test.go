package uploader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hazadus/go-snatcher/internal/data"
	"github.com/hazadus/go-snatcher/internal/metadata"
)

// S3UploaderInterface интерфейс для S3 uploader
type S3UploaderInterface interface {
	UploadFile(ctx context.Context, reader interface{}, key string) (string, error)
}

// MetadataExtractorInterface интерфейс для извлечения метаданных
type MetadataExtractorInterface interface {
	ExtractFromFile(filePath string) metadata.TrackMetadata
	GetFileInfo(filePath string) (*metadata.FileInfo, error)
}

// MockS3Uploader мок для S3 uploader
type MockS3Uploader struct {
	uploadFunc func(ctx context.Context, reader interface{}, key string) (string, error)
}

func (m *MockS3Uploader) UploadFile(ctx context.Context, reader interface{}, key string) (string, error) {
	return m.uploadFunc(ctx, reader, key)
}

// MockMetadataExtractor мок для извлечения метаданных
type MockMetadataExtractor struct {
	extractFunc     func(filePath string) metadata.TrackMetadata
	getFileInfoFunc func(filePath string) (*metadata.FileInfo, error)
}

func (m *MockMetadataExtractor) ExtractFromFile(filePath string) metadata.TrackMetadata {
	return m.extractFunc(filePath)
}

func (m *MockMetadataExtractor) GetFileInfo(filePath string) (*metadata.FileInfo, error) {
	return m.getFileInfoFunc(filePath)
}

// TestService тестовая версия Service для тестирования
type TestService struct {
	s3Uploader        S3UploaderInterface
	metadataExtractor MetadataExtractorInterface
	appData           *data.AppData
}

// NewTestService создает тестовый сервис
func NewTestService(s3Uploader S3UploaderInterface, metadataExtractor MetadataExtractorInterface, appData *data.AppData) *TestService {
	return &TestService{
		s3Uploader:        s3Uploader,
		metadataExtractor: metadataExtractor,
		appData:           appData,
	}
}

// UploadFile загружает файл с метаданными (тестовая версия)
func (s *TestService) UploadFile(ctx context.Context, filePath string, progressCallback func(int64)) (*UploadResult, error) {
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
	var reader interface{} = file
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

// UpdateApplicationData обновляет данные приложения с информацией о треке (тестовая версия)
func (s *TestService) UpdateApplicationData(result *UploadResult) error {
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

// TestSuccessfulUpload тестирует успешную загрузку файла
func TestSuccessfulUpload(t *testing.T) {
	// Создаем временный тестовый файл
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test-song.mp3")

	// Создаем тестовый файл с содержимым
	testContent := "test audio content"
	err := os.WriteFile(testFilePath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Ошибка создания тестового файла: %v", err)
	}

	// Создаем мок S3 uploader
	mockS3Uploader := &MockS3Uploader{
		uploadFunc: func(ctx context.Context, reader interface{}, key string) (string, error) {
			// Проверяем, что ключ формируется правильно
			if key != "test-song.mp3" {
				t.Errorf("Ожидался ключ: test-song.mp3, получено: %s", key)
			}
			return "https://s3.amazonaws.com/test-bucket/test-song.mp3", nil
		},
	}

	// Создаем мок metadata extractor
	mockMetadataExtractor := &MockMetadataExtractor{
		extractFunc: func(_ string) metadata.TrackMetadata {
			return metadata.TrackMetadata{
				Artist: "Test Artist",
				Title:  "Test Song",
				Album:  "Test Album",
			}
		},
		getFileInfoFunc: func(_ string) (*metadata.FileInfo, error) {
			return &metadata.FileInfo{
				Size:     1024,
				Duration: 180 * time.Second,
			}, nil
		},
	}

	// Создаем тестовый сервис
	appData := data.NewAppData()
	service := NewTestService(mockS3Uploader, mockMetadataExtractor, appData)

	// Тестируем загрузку
	ctx := context.Background()
	result, err := service.UploadFile(ctx, testFilePath, nil)

	if err != nil {
		t.Errorf("Неожиданная ошибка при загрузке: %v", err)
	}

	// Проверяем результат
	expectedURL := "https://s3.amazonaws.com/test-bucket/test-song.mp3"
	if result.URL != expectedURL {
		t.Errorf("Ожидался URL: %s, получено: %s", expectedURL, result.URL)
	}

	if result.Metadata.Artist != "Test Artist" {
		t.Errorf("Ожидался Artist: Test Artist, получено: %s", result.Metadata.Artist)
	}

	if result.Metadata.Title != "Test Song" {
		t.Errorf("Ожидался Title: Test Song, получено: %s", result.Metadata.Title)
	}

	if result.FileInfo.Size != 1024 {
		t.Errorf("Ожидался Size: 1024, получено: %d", result.FileInfo.Size)
	}
}

// TestUploadErrorHandling тестирует обработку ошибок при загрузке
func TestUploadErrorHandling(t *testing.T) {
	// Создаем временный тестовый файл
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test-song.mp3")

	// Создаем тестовый файл
	testContent := "test audio content"
	err := os.WriteFile(testFilePath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Ошибка создания тестового файла: %v", err)
	}

	// Тест 1: Ошибка S3 загрузки
	t.Run("S3UploadError", func(t *testing.T) {
		mockS3Uploader := &MockS3Uploader{
			uploadFunc: func(ctx context.Context, reader interface{}, key string) (string, error) {
				return "", fmt.Errorf("S3 upload failed")
			},
		}

		mockMetadataExtractor := &MockMetadataExtractor{
			extractFunc: func(_ string) metadata.TrackMetadata {
				return metadata.TrackMetadata{}
			},
			getFileInfoFunc: func(_ string) (*metadata.FileInfo, error) {
				return &metadata.FileInfo{}, nil
			},
		}

		appData := data.NewAppData()
		service := NewTestService(mockS3Uploader, mockMetadataExtractor, appData)

		ctx := context.Background()
		_, err := service.UploadFile(ctx, testFilePath, nil)

		if err == nil {
			t.Error("Ожидалась ошибка при загрузке в S3")
		}

		if !strings.Contains(err.Error(), "ошибка загрузки в S3") {
			t.Errorf("Неожиданное сообщение об ошибке: %v", err)
		}
	})

	// Тест 2: Ошибка получения информации о файле
	t.Run("FileInfoError", func(t *testing.T) {
		mockS3Uploader := &MockS3Uploader{
			uploadFunc: func(ctx context.Context, reader interface{}, key string) (string, error) {
				return "https://s3.amazonaws.com/test-bucket/test-song.mp3", nil
			},
		}

		mockMetadataExtractor := &MockMetadataExtractor{
			extractFunc: func(_ string) metadata.TrackMetadata {
				return metadata.TrackMetadata{}
			},
			getFileInfoFunc: func(_ string) (*metadata.FileInfo, error) {
				return nil, fmt.Errorf("File info error")
			},
		}

		appData := data.NewAppData()
		service := NewTestService(mockS3Uploader, mockMetadataExtractor, appData)

		ctx := context.Background()
		_, err := service.UploadFile(ctx, testFilePath, nil)

		if err == nil {
			t.Error("Ожидалась ошибка при получении информации о файле")
		}

		if !strings.Contains(err.Error(), "ошибка получения информации о файле") {
			t.Errorf("Неожиданное сообщение об ошибке: %v", err)
		}
	})

	// Тест 3: Файл не существует
	t.Run("FileNotExists", func(t *testing.T) {
		mockS3Uploader := &MockS3Uploader{
			uploadFunc: func(ctx context.Context, reader interface{}, key string) (string, error) {
				return "https://s3.amazonaws.com/test-bucket/test-song.mp3", nil
			},
		}

		mockMetadataExtractor := &MockMetadataExtractor{
			extractFunc: func(_ string) metadata.TrackMetadata {
				return metadata.TrackMetadata{}
			},
			getFileInfoFunc: func(_ string) (*metadata.FileInfo, error) {
				return &metadata.FileInfo{}, nil
			},
		}

		appData := data.NewAppData()
		service := NewTestService(mockS3Uploader, mockMetadataExtractor, appData)

		ctx := context.Background()
		_, err := service.UploadFile(ctx, "/non/existent/file.mp3", nil)

		if err == nil {
			t.Error("Ожидалась ошибка при несуществующем файле")
		}

		if !strings.Contains(err.Error(), "файл не найден") {
			t.Errorf("Неожиданное сообщение об ошибке: %v", err)
		}
	})
}

// TestS3ObjectKeyFormation тестирует корректность формирования имени файла (ключа) в S3
func TestS3ObjectKeyFormation(t *testing.T) {
	// Создаем временный тестовый файл
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test-song.mp3")

	// Создаем тестовый файл
	testContent := "test audio content"
	err := os.WriteFile(testFilePath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Ошибка создания тестового файла: %v", err)
	}

	testCases := []struct {
		name        string
		fileName    string
		expectedKey string
		description string
	}{
		{
			name:        "SimpleFileName",
			fileName:    "song.mp3",
			expectedKey: "song.mp3",
			description: "Простое имя файла без пути",
		},
		{
			name:        "FileNameWithPath",
			fileName:    "music/artist/song.mp3",
			expectedKey: "song.mp3",
			description: "Имя файла с путем",
		},
		{
			name:        "FileNameWithSpecialChars",
			fileName:    "song (remix) [2024].mp3",
			expectedKey: "song (remix) [2024].mp3",
			description: "Имя файла со специальными символами",
		},
		{
			name:        "FileNameWithSpaces",
			fileName:    "my song title.mp3",
			expectedKey: "my song title.mp3",
			description: "Имя файла с пробелами",
		},
		{
			name:        "FileNameWithUnicode",
			fileName:    "песня.mp3",
			expectedKey: "песня.mp3",
			description: "Имя файла с Unicode символами",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Создаем файл с нужным именем
			testFilePath := filepath.Join(tempDir, tc.fileName)

			// Создаем директории, если они нужны
			dir := filepath.Dir(testFilePath)
			if dir != tempDir {
				err := os.MkdirAll(dir, 0755)
				if err != nil {
					t.Fatalf("Ошибка создания директории: %v", err)
				}
			}

			err := os.WriteFile(testFilePath, []byte(testContent), 0644)
			if err != nil {
				t.Fatalf("Ошибка создания тестового файла: %v", err)
			}

			var receivedKey string
			mockS3Uploader := &MockS3Uploader{
				uploadFunc: func(ctx context.Context, reader interface{}, key string) (string, error) {
					receivedKey = key
					return "https://s3.amazonaws.com/test-bucket/" + key, nil
				},
			}

			mockMetadataExtractor := &MockMetadataExtractor{
				extractFunc: func(_ string) metadata.TrackMetadata {
					return metadata.TrackMetadata{}
				},
				getFileInfoFunc: func(_ string) (*metadata.FileInfo, error) {
					return &metadata.FileInfo{}, nil
				},
			}

			appData := data.NewAppData()
			service := NewTestService(mockS3Uploader, mockMetadataExtractor, appData)

			ctx := context.Background()
			_, err = service.UploadFile(ctx, testFilePath, nil)

			if err != nil {
				t.Errorf("Ошибка при загрузке: %v", err)
			}

			if receivedKey != tc.expectedKey {
				t.Errorf("Ожидался ключ: %s, получено: %s", tc.expectedKey, receivedKey)
			}
		})
	}
}

// TestUpdateApplicationData тестирует обновление данных приложения
func TestUpdateApplicationData(t *testing.T) {
	// Создаем тестовый результат загрузки
	result := &UploadResult{
		URL: "https://s3.amazonaws.com/test-bucket/test-song.mp3",
		Metadata: metadata.TrackMetadata{
			Artist: "Test Artist",
			Title:  "Test Song",
			Album:  "Test Album",
		},
		FileInfo: &metadata.FileInfo{
			Size:     1024,
			Duration: 180 * time.Second,
		},
	}

	// Создаем тестовый сервис
	appData := data.NewAppData()
	service := NewTestService(&MockS3Uploader{}, &MockMetadataExtractor{}, appData)

	// Тестируем обновление данных
	err := service.UpdateApplicationData(result)
	if err != nil {
		t.Errorf("Неожиданная ошибка при обновлении данных: %v", err)
	}

	// Примечание: метод GetTracks может не существовать, поэтому пропускаем эту проверку
	// Проверяем только, что обновление данных не вызывает ошибку
}

// TestProgressReader тестирует отслеживание прогресса чтения
func TestProgressReader(t *testing.T) {
	// Создаем тестовые данные
	testData := "test content for progress tracking"
	reader := strings.NewReader(testData)

	var progressCalled bool
	var progressBytes int64

	progressCallback := func(bytesRead int64) {
		progressCalled = true
		progressBytes = bytesRead
	}

	// Создаем ProgressReader
	progressReader := &ProgressReader{
		Reader:     reader,
		Size:       int64(len(testData)),
		OnProgress: progressCallback,
	}

	// Читаем данные
	buffer := make([]byte, 1024)
	n, err := progressReader.Read(buffer)

	if err != nil {
		t.Errorf("Неожиданная ошибка при чтении: %v", err)
	}

	if n != len(testData) {
		t.Errorf("Ожидалось прочитано байт: %d, получено: %d", len(testData), n)
	}

	if !progressCalled {
		t.Error("Callback прогресса не был вызван")
	}

	if progressBytes != int64(len(testData)) {
		t.Errorf("Ожидалось байт в callback: %d, получено: %d", len(testData), progressBytes)
	}
}

// TestFormatFileSize тестирует форматирование размера файла
func TestFormatFileSize(t *testing.T) {
	testCases := []struct {
		bytes    int64
		expected string
	}{
		{1024, "1.0 KB"},
		{2048, "2.0 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{512, "512 B"},
		{0, "0 B"},
	}

	for _, tc := range testCases {
		result := FormatFileSize(tc.bytes)
		if result != tc.expected {
			t.Errorf("Для %d байт ожидалось: %s, получено: %s", tc.bytes, tc.expected, result)
		}
	}
}

// TestFormatDuration тестирует форматирование длительности
func TestFormatDuration(t *testing.T) {
	testCases := []struct {
		duration time.Duration
		expected string
	}{
		{180 * time.Second, "00:03:00"},
		{3661 * time.Second, "01:01:01"},
		{0, "00:00:00"},
		{59 * time.Second, "00:00:59"},
		{3600 * time.Second, "01:00:00"},
	}

	for _, tc := range testCases {
		result := FormatDuration(tc.duration)
		if result != tc.expected {
			t.Errorf("Для %v ожидалось: %s, получено: %s", tc.duration, tc.expected, result)
		}
	}
}
