package metadata

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractMetadata(t *testing.T) {
	// Создаем временный тестовый файл
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test.mp3")

	// Создаем простой файл для тестирования
	content := []byte("fake mp3 content for testing")
	err := os.WriteFile(testFilePath, content, 0644)
	if err != nil {
		t.Fatalf("Ошибка создания тестового файла: %v", err)
	}

	extractor := NewExtractor()
	metadata := extractor.ExtractFromFile(testFilePath)

	// Проверяем, что метаданные извлечены (даже если файл не содержит реальных метаданных)
	if metadata.Artist == "" && metadata.Title == "" {
		t.Error("Ожидались извлеченные метаданные, получены пустые значения")
	}
}

func TestExtractFromNoMetadataFile(t *testing.T) {
	// Создаем временный файл без метаданных
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "Artist - Title.mp3")

	// Создаем файл с именем в формате "Artist - Title"
	content := []byte("fake content")
	err := os.WriteFile(testFilePath, content, 0644)
	if err != nil {
		t.Fatalf("Ошибка создания тестового файла: %v", err)
	}

	extractor := NewExtractor()
	metadata := extractor.ExtractFromFile(testFilePath)

	// Проверяем, что метаданные извлечены из имени файла
	if metadata.Artist != "Artist" {
		t.Errorf("Ожидался Artist: Artist, получено: %s", metadata.Artist)
	}
	if metadata.Title != "Title" {
		t.Errorf("Ожидался Title: Title, получено: %s", metadata.Title)
	}
}

func TestExtractFromCorruptedFile(t *testing.T) {
	// Создаем временный файл с некорректными данными
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "Unknown - Track.mp3")

	// Создаем файл с некорректными данными
	corruptedContent := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}
	err := os.WriteFile(testFilePath, corruptedContent, 0644)
	if err != nil {
		t.Fatalf("Ошибка создания тестового файла: %v", err)
	}

	extractor := NewExtractor()
	metadata := extractor.ExtractFromFile(testFilePath)

	// Проверяем, что метаданные извлечены из имени файла при ошибке
	if metadata.Artist != "Unknown" {
		t.Errorf("Ожидался Artist: Unknown, получено: %s", metadata.Artist)
	}
	if metadata.Title != "Track" {
		t.Errorf("Ожидался Title: Track, получено: %s", metadata.Title)
	}
}

func TestGetDefaultMetadata(t *testing.T) {
	extractor := NewExtractor()

	// Тестируем файл с форматом "Artist - Title"
	source1 := "/path/to/Artist - Title.mp3"
	metadata1 := extractor.ExtractFromFile(source1)

	if metadata1.Artist != "Artist" {
		t.Errorf("Ожидался Artist: Artist, получено: %s", metadata1.Artist)
	}
	if metadata1.Title != "Title" {
		t.Errorf("Ожидался Title: Title, получено: %s", metadata1.Title)
	}

	// Тестируем файл с простым именем
	source2 := "/path/to/SimpleTrack.mp3"
	metadata2 := extractor.ExtractFromFile(source2)

	if metadata2.Artist != "Unknown Artist" {
		t.Errorf("Ожидался Artist: Unknown Artist, получено: %s", metadata2.Artist)
	}
	if metadata2.Title != "SimpleTrack" {
		t.Errorf("Ожидался Title: SimpleTrack, получено: %s", metadata2.Title)
	}

	// Тестируем файл с несколькими дефисами
	source3 := "/path/to/Artist - Album - Title.mp3"
	metadata3 := extractor.ExtractFromFile(source3)

	if metadata3.Artist != "Artist" {
		t.Errorf("Ожидался Artist: Artist, получено: %s", metadata3.Artist)
	}
	if metadata3.Title != "Album - Title" {
		t.Errorf("Ожидался Title: Album - Title, получено: %s", metadata3.Title)
	}
}

func TestExtractFromReader(t *testing.T) {
	// Создаем временный файл
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "Test - Song.mp3")

	content := []byte("test content")
	err := os.WriteFile(testFilePath, content, 0644)
	if err != nil {
		t.Fatalf("Ошибка создания тестового файла: %v", err)
	}

	// Открываем файл для чтения
	file, err := os.Open(testFilePath)
	if err != nil {
		t.Fatalf("Ошибка открытия файла: %v", err)
	}
	defer file.Close()

	extractor := NewExtractor()
	metadata := extractor.ExtractFromReader(file, testFilePath)

	// Проверяем, что метаданные извлечены
	if metadata.Artist != "Test" {
		t.Errorf("Ожидался Artist: Test, получено: %s", metadata.Artist)
	}
	if metadata.Title != "Song" {
		t.Errorf("Ожидался Title: Song, получено: %s", metadata.Title)
	}
}

func TestGetFileInfo(t *testing.T) {
	// Создаем временный файл
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test.mp3")

	content := []byte("test content for file info")
	err := os.WriteFile(testFilePath, content, 0644)
	if err != nil {
		t.Fatalf("Ошибка создания тестового файла: %v", err)
	}

	extractor := NewExtractor()
	fileInfo, err := extractor.GetFileInfo(testFilePath)

	// Ожидаем ошибку, так как файл не является валидным MP3
	if err == nil {
		t.Error("Ожидалась ошибка для некорректного MP3 файла")
		return
	}

	// Проверяем, что fileInfo равен nil при ошибке
	if fileInfo != nil {
		t.Error("fileInfo должен быть nil при ошибке")
	}

	// Проверяем сообщение об ошибке
	if !strings.Contains(err.Error(), "ошибка получения длительности") {
		t.Errorf("Неожиданное сообщение об ошибке: %v", err)
	}
}

func TestGetFileInfoNonExistentFile(t *testing.T) {
	extractor := NewExtractor()
	_, err := extractor.GetFileInfo("/non/existent/file.mp3")

	if err == nil {
		t.Error("Ожидалась ошибка для несуществующего файла")
	}

	if !strings.Contains(err.Error(), "ошибка получения информации о файле") {
		t.Errorf("Неожиданное сообщение об ошибке: %v", err)
	}
}

func TestGetDuration(t *testing.T) {
	// Создаем временный файл
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test.mp3")

	content := []byte("test content")
	err := os.WriteFile(testFilePath, content, 0644)
	if err != nil {
		t.Fatalf("Ошибка создания тестового файла: %v", err)
	}

	extractor := NewExtractor()
	duration, err := extractor.GetDuration(testFilePath)

	// Ожидаем ошибку, так как файл не является валидным MP3
	if err == nil {
		t.Error("Ожидалась ошибка для некорректного MP3 файла")
	}

	if !strings.Contains(err.Error(), "ошибка декодирования MP3") {
		t.Errorf("Неожиданное сообщение об ошибке: %v", err)
	}

	// Проверяем, что длительность равна 0 при ошибке
	if duration != 0 {
		t.Errorf("Ожидалась длительность 0 при ошибке, получено: %v", duration)
	}
}

func TestGetDurationNonExistentFile(t *testing.T) {
	extractor := NewExtractor()
	_, err := extractor.GetDuration("/non/existent/file.mp3")

	if err == nil {
		t.Error("Ожидалась ошибка для несуществующего файла")
	}

	if !strings.Contains(err.Error(), "ошибка открытия файла") {
		t.Errorf("Неожиданное сообщение об ошибке: %v", err)
	}
}
