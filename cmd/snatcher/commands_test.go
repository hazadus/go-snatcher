package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/hazadus/go-snatcher/internal/config"
	"github.com/hazadus/go-snatcher/internal/data"
)

// captureOutput перехватывает stdout и stderr во время выполнения функции
func captureOutput(t *testing.T, fn func()) string {
	// Сохраняем оригинальные stdout и stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// Создаем временные файлы для перехвата
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Ошибка создания pipe: %v", err)
	}

	// Перенаправляем stdout и stderr
	os.Stdout = w
	os.Stderr = w

	// Выполняем функцию
	fn()

	// Восстанавливаем оригинальные stdout и stderr
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	// Закрываем writer
	w.Close()

	// Читаем результат
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("Ошибка чтения результата: %v", err)
	}

	return buf.String()
}

// TestCmdList проверяет, что команда `list` корректно выводит список треков
func TestCmdList(t *testing.T) {
	// Создаем временную директорию для тестов
	tempDir := t.TempDir()

	// Создаем тестовое приложение
	app := createTestApplication(t, tempDir)

	// Добавляем тестовые треки в данные
	testTrack := data.TrackMetadata{
		Artist:   "Test Artist",
		Title:    "Test Title",
		Album:    "Test Album",
		Length:   180,
		FileSize: 1024000,
		URL:      "https://s3.example.com/test.mp3",
	}
	app.Data.AddTrack(testTrack)

	// Создаем команду list
	listCmd := app.createListCommand()

	// Захватываем вывод с помощью captureOutput
	output := captureOutput(t, func() {
		listCmd.SetArgs([]string{})
		err := listCmd.Execute()
		if err != nil {
			t.Errorf("Ошибка выполнения команды list: %v", err)
		}
	})

	// Проверяем вывод
	expectedStrings := []string{
		"📚 Найдено треков: 1",
		"Test Artist",
		"Test Title",
		"Test Album",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Вывод команды list не содержит ожидаемую строку '%s': %s", expected, output)
		}
	}
}

// TestCmdListEmpty проверяет, что команда `list` корректно обрабатывает пустую библиотеку
func TestCmdListEmpty(t *testing.T) {
	// Создаем временную директорию для тестов
	tempDir := t.TempDir()

	// Создаем тестовое приложение с пустыми данными
	app := createTestApplication(t, tempDir)

	// Создаем команду list
	listCmd := app.createListCommand()

	// Захватываем вывод с помощью captureOutput
	output := captureOutput(t, func() {
		listCmd.SetArgs([]string{})
		err := listCmd.Execute()
		if err != nil {
			t.Errorf("Ошибка выполнения команды list: %v", err)
		}
	})

	// Проверяем вывод для пустой библиотеки
	if !strings.Contains(output, "📚 Библиотека пуста") {
		t.Errorf("Команда list не отобразила сообщение о пустой библиотеке: %s", output)
	}
}

// TestCmdDelete проверяет, что команда `delete` удаляет указанный трек
func TestCmdDelete(t *testing.T) {
	// Создаем временную директорию для тестов
	tempDir := t.TempDir()

	// Создаем тестовое приложение
	app := createTestApplication(t, tempDir)

	// Добавляем тестовые треки
	testTrack1 := data.TrackMetadata{
		Artist: "Artist 1",
		Title:  "Title 1",
		URL:    "https://s3.example.com/test1.mp3",
	}
	testTrack2 := data.TrackMetadata{
		Artist: "Artist 2",
		Title:  "Title 2",
		URL:    "https://s3.example.com/test2.mp3",
	}

	app.Data.AddTrack(testTrack1)
	app.Data.AddTrack(testTrack2)

	// Проверяем, что треки добавлены
	if len(app.Data.Tracks) != 2 {
		t.Fatalf("Ожидалось 2 трека, получено %d", len(app.Data.Tracks))
	}

	// Создаем команду delete
	ctx := context.Background()
	deleteCmd := app.createDeleteCommand(ctx)

	// Захватываем вывод с помощью captureOutput
	output := captureOutput(t, func() {
		deleteCmd.SetArgs([]string{"1"})
		err := deleteCmd.Execute()
		if err != nil {
			t.Errorf("Ошибка выполнения команды delete: %v", err)
		}
	})

	// Проверяем вывод
	if !strings.Contains(output, "🗑️  Удаляем трек: Artist 1 - Title 1") {
		t.Errorf("Команда delete не отобразила ожидаемый вывод: %s", output)
	}

	// Проверяем, что трек был удален из данных
	if len(app.Data.Tracks) != 1 {
		t.Errorf("Ожидался 1 трек после удаления, получено %d", len(app.Data.Tracks))
	}

	// Проверяем, что оставшийся трек правильный
	remainingTrack := app.Data.Tracks[0]
	if remainingTrack.Artist != "Artist 2" {
		t.Errorf("Ожидался Artist: Artist 2, получено: %s", remainingTrack.Artist)
	}
}

// TestCmdDeleteInvalidID проверяет обработку неверного ID в команде delete
func TestCmdDeleteInvalidID(t *testing.T) {
	// Создаем временную директорию для тестов
	tempDir := t.TempDir()

	// Создаем тестовое приложение
	app := createTestApplication(t, tempDir)

	// Создаем команду delete
	ctx := context.Background()
	deleteCmd := app.createDeleteCommand(ctx)

	// Захватываем вывод с помощью captureOutput
	output := captureOutput(t, func() {
		deleteCmd.SetArgs([]string{"invalid"})
		err := deleteCmd.Execute()
		// Проверяем, что команда не завершилась с ошибкой (обрабатывает неверный ID)
		if err != nil {
			t.Errorf("Команда delete завершилась с ошибкой при неверном ID: %v", err)
		}
	})

	// Проверяем вывод об ошибке
	if !strings.Contains(output, "❌ Ошибка: неверный ID") {
		t.Errorf("Команда delete не отобразила ошибку для неверного ID: %s", output)
	}
}

// TestCmdDownload проверяет, что команда `download` инициирует скачивание
func TestCmdDownload(t *testing.T) {
	// Создаем временную директорию для тестов
	tempDir := t.TempDir()

	// Создаем тестовое приложение
	app := createTestApplication(t, tempDir)

	// Создаем команду download
	ctx := context.Background()
	downloadCmd := app.createDownloadCommand(ctx)

	// Захватываем вывод с помощью captureOutput
	output := captureOutput(t, func() {
		downloadCmd.SetArgs([]string{"https://www.youtube.com/watch?v=dQw4w9WgXcQ"})
		err := downloadCmd.Execute()

		// Проверяем результат
		if err != nil {
			// Ожидаем ошибку, так как скачивание может не удаться в тестовой среде
			// Но команда должна попытаться обработать URL
			if !strings.Contains(err.Error(), "youtube") && !strings.Contains(err.Error(), "network") {
				t.Errorf("Неожиданная ошибка команды download: %v", err)
			}
		}
	})

	// Проверяем, что команда пыталась обработать URL
	if !strings.Contains(output, "Скачиваем аудио для видео ID: dQw4w9WgXcQ") {
		t.Errorf("Команда download не отобразила ожидаемый вывод: %s", output)
	}
}

// TestCmdDownloadInvalidURL проверяет обработку неверного URL в команде download
func TestCmdDownloadInvalidURL(t *testing.T) {
	// Создаем временную директорию для тестов
	tempDir := t.TempDir()

	// Создаем тестовое приложение
	app := createTestApplication(t, tempDir)

	// Создаем команду download
	ctx := context.Background()
	downloadCmd := app.createDownloadCommand(ctx)

	// Захватываем вывод с помощью captureOutput
	output := captureOutput(t, func() {
		downloadCmd.SetArgs([]string{"invalid-url"})
		err := downloadCmd.Execute()

		// Проверяем результат
		if err == nil {
			t.Error("Ожидалась ошибка при выполнении команды download с неверным URL")
		}

		// Проверяем, что ошибка связана с неверным URL или недоступным видео
		if !strings.Contains(err.Error(), "ошибка извлечения ID видео") &&
			!strings.Contains(err.Error(), "This video is unavailable") {
			t.Errorf("Неожиданная ошибка команды download: %v", err)
		}
	})

	// Проверяем, что команда пыталась обработать URL
	if !strings.Contains(output, "Скачиваем аудио для видео ID: invalid-url") {
		t.Errorf("Команда download не отобразила ожидаемый вывод: %s", output)
	}
}

// createTestApplication создает тестовое приложение с временными данными
func createTestApplication(t *testing.T, tempDir string) *Application {
	// Создаем тестовую конфигурацию
	testConfig := &config.Config{
		AwsRegion:     "us-east-1",
		AwsAccessKey:  "test-key",
		AwsSecretKey:  "test-secret",
		AwsEndpoint:   "http://localhost:9000",
		AwsBucketName: "test-bucket",
		DownloadDir:   tempDir,
	}

	// Создаем тестовые данные
	testData := data.NewAppData()

	// Создаем приложение
	app := &Application{
		Config: testConfig,
		Data:   testData,
	}

	return app
}

// TestCmdAddInvalidArgs проверяет обработку неверных аргументов в команде add
func TestCmdAddInvalidArgs(t *testing.T) {
	// Создаем временную директорию для тестов
	tempDir := t.TempDir()

	// Создаем тестовое приложение
	app := createTestApplication(t, tempDir)

	// Создаем команду add
	ctx := context.Background()
	addCmd := app.createAddCommand(ctx)

	// Захватываем вывод
	var buf bytes.Buffer
	addCmd.SetOut(&buf)
	addCmd.SetErr(&buf)

	// Выполняем команду без аргументов
	err := addCmd.Execute()

	// Проверяем, что команда отображает ошибку о неверных аргументах
	if err == nil {
		t.Error("Ожидалась ошибка при выполнении команды add без аргументов")
	}

	// Проверяем вывод об ошибке
	output := buf.String()
	if !strings.Contains(output, "requires exactly 1 arg") && !strings.Contains(output, "accepts 1 arg") {
		t.Errorf("Команда add не отобразила ошибку о неверных аргументах: %s", output)
	}
}
